// Package service provides business logic implementations for inventory management.
//
// This package implements ACID-compliant inventory operations with comprehensive
// error handling, idempotency support, and integration with agnostic providers.
package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/AJPalacios/inventory/internal/domain"
	"github.com/AJPalacios/inventory/internal/repository"
	"github.com/google/uuid"
)

// inventoryService provides inventory operations.
type inventoryService struct {
	repo               repository.Querier
	validationService  domain.ValidationService
	idempotencyService domain.IdempotencyService
	logger             domain.Logger
	metrics            domain.MetricsProvider
	cache              domain.CacheProvider
	circuitBreaker     domain.CircuitBreaker
	config             InventoryServiceConfig
}

// NewInventoryService creates a new inventory service instance.
func NewInventoryService(
	repo repository.Querier,
	validationService domain.ValidationService,
	idempotencyService domain.IdempotencyService,
	logger domain.Logger,
	metrics domain.MetricsProvider,
	cache domain.CacheProvider,
	circuitBreaker domain.CircuitBreaker,
	config InventoryServiceConfig,
) domain.InventoryService {
	return &inventoryService{
		repo:               repo,
		validationService:  validationService,
		idempotencyService: idempotencyService,
		logger:             logger,
		metrics:            metrics,
		cache:              cache,
		circuitBreaker:     circuitBreaker,
		config:             config,
	}
}

// ReserveStock reserves stock for a product with idempotency support.
func (s *inventoryService) ReserveStock(ctx context.Context, req domain.ReserveStockServiceRequest) (*domain.ReservationResult, error) {
	// Validate request
	validationResult := s.validationService.ValidateReserveRequest(req)
	if !validationResult.Valid {
		err := fmt.Errorf("validation failed: %v", validationResult.Errors)
		s.logger.Error("Invalid reserve stock request", err, map[string]interface{}{
			"request_id": req.RequestID,
			"product_id": req.ProductID,
		})
		return nil, err
	}

	// Check idempotency
	if existing, found, _ := s.idempotencyService.CheckIdempotency(ctx, req.RequestID); found {
		s.logger.Info("Returning cached result", map[string]interface{}{
			"request_id": req.RequestID,
		})
		if result, ok := existing.(*domain.ReservationResult); ok {
			return result, nil
		}
	}

	// Generate reservation ID
	reservationID := uuid.New().String()

	// Get current inventory
	item, err := s.repo.GetInventoryItem(ctx, req.ProductID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrProductNotFound{ProductID: req.ProductID}
		}
		return nil, fmt.Errorf("failed to get inventory: %w", err)
	}

	// Check stock availability
	quantityInt64 := int64(req.Quantity)
	if item.AvailableStock < quantityInt64 {
		return nil, domain.ErrInsufficientStock{
			ProductID: req.ProductID,
			Requested: quantityInt64,
			Available: item.AvailableStock,
		}
	}

	// Reserve stock optimistically
	newAvailable := item.AvailableStock - quantityInt64
	_, err = s.repo.ReserveStockOptimistic(ctx, repository.ReserveStockOptimisticParams{
		ProductID:      req.ProductID,
		AvailableStock: newAvailable,
		Version:        item.Version,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to reserve stock: %w", err)
	}

	// Create reservation
	expiresAt := sql.NullTime{
		Time:  time.Now().Add(time.Duration(req.TimeoutSeconds) * time.Second),
		Valid: req.TimeoutSeconds > 0,
	}
	if req.TimeoutSeconds == 0 {
		expiresAt = sql.NullTime{
			Time:  time.Now().Add(5 * time.Minute),
			Valid: true,
		}
	}

	reservation, err := s.repo.CreateReservation(ctx, repository.CreateReservationParams{
		ID:        reservationID,
		RequestID: req.RequestID,
		ProductID: req.ProductID,
		Quantity:  quantityInt64,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		// Try to rollback stock reservation
		rollbackAvailable := item.AvailableStock // back to original
		s.repo.ReleaseStockOptimistic(ctx, repository.ReleaseStockOptimisticParams{
			ProductID:      req.ProductID,
			AvailableStock: rollbackAvailable,
			Version:        item.Version + 1,
		})
		return nil, fmt.Errorf("failed to create reservation: %w", err)
	}

	// Create result
	result := &domain.ReservationResult{
		ReservationID: reservation.ID,
		ProductID:     reservation.ProductID,
		Quantity:      reservation.Quantity,
		Status:        domain.ReservationStatusActive,
		ExpiresAt:     reservation.ExpiresAt.Time,
		CreatedAt:     reservation.CreatedAt,
		Metadata:      req.Metadata,
	}

	// Store idempotency result
	s.idempotencyService.StoreResult(req.RequestID, result, time.Hour*24)

	s.logger.Info("Stock reserved successfully", map[string]interface{}{
		"request_id":     req.RequestID,
		"reservation_id": reservation.ID,
		"product_id":     req.ProductID,
		"quantity":       req.Quantity,
	})

	return result, nil
}

// ReleaseReservation releases a stock reservation.
func (s *inventoryService) ReleaseReservation(ctx context.Context, req *domain.ReleaseReservationRequest) (*domain.ReleaseResult, error) {
	// Validate request
	if err := s.validationService.ValidateReleaseReservationRequest(req); err != nil {
		return nil, err
	}

	// Check idempotency
	if existing, found := s.idempotencyService.GetResult(req.RequestID); found {
		if result, ok := existing.(*domain.ReleaseResult); ok {
			return result, nil
		}
	}

	// Get reservation
	reservation, err := s.repo.GetReservation(ctx, req.ReservationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrReservationNotFound{ReservationID: req.ReservationID}
		}
		return nil, fmt.Errorf("failed to get reservation: %w", err)
	}

	// Check if reservation is active
	if reservation.Status != "active" {
		return nil, domain.ErrReservationNotActive{
			ReservationID: req.ReservationID,
			Status:        reservation.Status,
		}
	}

	// Get current inventory
	item, err := s.repo.GetInventoryItem(ctx, reservation.ProductID)
	if err != nil {
		return nil, fmt.Errorf("failed to get inventory: %w", err)
	}

	// Release reserved stock
	_, err = s.repo.ReleaseStockOptimistic(ctx, repository.ReleaseStockOptimisticParams{
		ProductID: reservation.ProductID,
		Quantity:  reservation.Quantity,
		Version:   item.Version,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to release stock: %w", err)
	}

	// Update reservation status
	_, err = s.repo.UpdateReservationStatusById(ctx, repository.UpdateReservationStatusByIdParams{
		Status: "released",
		ID:     req.ReservationID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update reservation: %w", err)
	}

	result := &domain.ReleaseResult{
		ReservationID: req.ReservationID,
		ProductID:     reservation.ProductID,
		Quantity:      reservation.Quantity,
		Status:        domain.ReservationStatusReleased,
		ReleasedAt:    time.Now(),
	}

	// Store idempotency result
	s.idempotencyService.StoreResult(req.RequestID, result, time.Hour*24)

	return result, nil
}

// UpdateStock updates the total stock for a product.
func (s *inventoryService) UpdateStock(ctx context.Context, req *domain.UpdateStockRequest) (*domain.StockInfo, error) {
	// Validate request
	if err := s.validationService.ValidateUpdateStockRequest(req); err != nil {
		return nil, err
	}

	// Check idempotency
	if existing, found := s.idempotencyService.GetResult(req.RequestID); found {
		if result, ok := existing.(*domain.StockInfo); ok {
			return result, nil
		}
	}

	// Get current inventory
	item, err := s.repo.GetInventoryItem(ctx, req.ProductID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrProductNotFound{ProductID: req.ProductID}
		}
		return nil, fmt.Errorf("failed to get inventory: %w", err)
	}

	// Calculate new total stock
	newTotal := req.NewQuantity
	if req.Operation == domain.StockOperationAdjust {
		newTotal = item.AvailableStock + item.ReservedStock + req.NewQuantity
	}

	// Validate that we don't go below reserved stock
	if newTotal < item.ReservedStock {
		return nil, domain.ErrInvalidStockOperation{
			ProductID: req.ProductID,
			Operation: string(req.Operation),
			Reason:    "cannot reduce total stock below reserved quantity",
		}
	}

	newAvailable := newTotal - item.ReservedStock

	// Update stock
	updatedItem, err := s.repo.UpdateStockOptimistic(ctx, repository.UpdateStockOptimisticParams{
		ProductID:      req.ProductID,
		AvailableStock: newAvailable,
		Version:        item.Version,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update stock: %w", err)
	}

	result := &domain.StockInfo{
		ProductID:      updatedItem.ProductID,
		AvailableStock: updatedItem.AvailableStock,
		ReservedStock:  updatedItem.ReservedStock,
		TotalStock:     updatedItem.AvailableStock + updatedItem.ReservedStock,
		Version:        updatedItem.Version,
		LastUpdated:    updatedItem.UpdatedAt,
	}

	// Store idempotency result
	s.idempotencyService.StoreResult(req.RequestID, result, time.Hour*24)

	return result, nil
}

// GetStockInfo returns current stock information for a product.
func (s *inventoryService) GetStockInfo(ctx context.Context, productID string) (*domain.StockInfo, error) {
	item, err := s.repo.GetInventoryItem(ctx, productID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrProductNotFound{ProductID: productID}
		}
		return nil, fmt.Errorf("failed to get inventory: %w", err)
	}

	return &domain.StockInfo{
		ProductID:      item.ProductID,
		AvailableStock: item.AvailableStock,
		ReservedStock:  item.ReservedStock,
		TotalStock:     item.AvailableStock + item.ReservedStock,
		Version:        item.Version,
		LastUpdated:    item.UpdatedAt,
	}, nil
}

// GetLowStockProducts returns products with stock below the threshold.
func (s *inventoryService) GetLowStockProducts(ctx context.Context, threshold int64) ([]domain.StockInfo, error) {
	items, err := s.repo.GetLowStockProducts(ctx, threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to get low stock products: %w", err)
	}

	result := make([]domain.StockInfo, len(items))
	for i, item := range items {
		result[i] = domain.StockInfo{
			ProductID:      item.ProductID,
			AvailableStock: item.AvailableStock,
			ReservedStock:  item.ReservedStock,
			TotalStock:     item.AvailableStock + item.ReservedStock,
			LastUpdated:    item.UpdatedAt,
		}
	}

	return result, nil
}

// HealthCheck performs a health check on the inventory service.
func (s *inventoryService) HealthCheck(ctx context.Context) error {
	// Simple health check - try to get inventory summary
	_, err := s.repo.GetInventorySummary(ctx)
	return err
}
