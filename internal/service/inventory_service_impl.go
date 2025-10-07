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

// inventoryServiceImpl implements the InventoryService interface.
type inventoryServiceImpl struct {
	repo               repository.Querier
	validationService  domain.ValidationService
	idempotencyService domain.IdempotencyService
	logger             domain.Logger
	metrics            domain.MetricsProvider
	config             InventoryServiceConfig
}

// NewInventoryServiceImpl creates a new inventory service instance.
func NewInventoryServiceImpl(
	repo repository.Querier,
	validationService domain.ValidationService,
	idempotencyService domain.IdempotencyService,
	logger domain.Logger,
	metrics domain.MetricsProvider,
	config InventoryServiceConfig,
) domain.InventoryService {
	return &inventoryServiceImpl{
		repo:               repo,
		validationService:  validationService,
		idempotencyService: idempotencyService,
		logger:             logger,
		metrics:            metrics,
		config:             config,
	}
}

// ReserveStock implements InventoryService.ReserveStock
func (s *inventoryServiceImpl) ReserveStock(ctx context.Context, req domain.ReserveStockServiceRequest) (*domain.ReservationResult, error) {
	// Validate request
	validation := s.validationService.ValidateReserveRequest(req)
	if !validation.Valid {
		return nil, fmt.Errorf("validation failed: %v", validation.Errors)
	}

	// Check idempotency
	if existing, found, _ := s.idempotencyService.CheckIdempotency(ctx, req.RequestID); found {
		if result, ok := existing.(*domain.ReservationResult); ok {
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

	// Check availability
	quantityInt64 := int64(req.Quantity)
	if item.AvailableStock < quantityInt64 {
		return nil, domain.ErrInsufficientStock{
			ProductID: req.ProductID,
			Requested: quantityInt64,
			Available: item.AvailableStock,
		}
	}

	// Reserve stock
	_, err = s.repo.ReserveStockOptimistic(ctx, repository.ReserveStockOptimisticParams{
		ProductID:      req.ProductID,
		AvailableStock: quantityInt64, // NOTE: Despite name, this is actually the quantity to reserve (SQL uses ?1 for both operations)
		Version:        item.Version,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to reserve stock: %w", err)
	}

	// Create reservation
	reservationID := uuid.New().String()
	expiresAt := sql.NullTime{
		Time:  time.Now().Add(time.Duration(req.TimeoutSeconds) * time.Second),
		Valid: req.TimeoutSeconds > 0,
	}
	if req.TimeoutSeconds == 0 {
		expiresAt = sql.NullTime{Time: time.Now().Add(5 * time.Minute), Valid: true}
	}

	reservation, err := s.repo.CreateReservation(ctx, repository.CreateReservationParams{
		ID:        reservationID,
		RequestID: req.RequestID,
		ProductID: req.ProductID,
		Quantity:  quantityInt64,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create reservation: %w", err)
	}

	result := &domain.ReservationResult{
		ReservationID: reservation.ID,
		ProductID:     reservation.ProductID,
		Quantity:      reservation.Quantity,
		Status:        domain.ReservationStatusActive,
		ExpiresAt:     reservation.ExpiresAt.Time,
		CreatedAt:     reservation.CreatedAt,
		Metadata:      req.Metadata,
	}

	// Store idempotency
	s.idempotencyService.StoreResult(ctx, req.RequestID, result, time.Hour*24)

	return result, nil
}

// ReleaseStock implements InventoryService.ReleaseStock
func (s *inventoryServiceImpl) ReleaseStock(ctx context.Context, req domain.ReleaseStockServiceRequest) (*repository.InventoryItem, error) {
	// Validate
	validation := s.validationService.ValidateReleaseRequest(req)
	if !validation.Valid {
		return nil, fmt.Errorf("validation failed: %v", validation.Errors)
	}

	// Get reservation
	reservation, err := s.repo.GetReservation(ctx, req.ReservationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrReservationNotFound{ReservationID: req.ReservationID}
		}
		return nil, fmt.Errorf("failed to get reservation: %w", err)
	}

	// Check active or pending (both are valid for release)
	if reservation.Status != "active" && reservation.Status != "pending" {
		return nil, domain.ErrReservationNotActive{
			ReservationID: req.ReservationID,
			Status:        reservation.Status,
		}
	}

	// Get inventory
	item, err := s.repo.GetInventoryItem(ctx, reservation.ProductID)
	if err != nil {
		return nil, fmt.Errorf("failed to get inventory: %w", err)
	}

	// Release stock
	updatedItem, err := s.repo.ReleaseStockOptimistic(ctx, repository.ReleaseStockOptimisticParams{
		ProductID:      reservation.ProductID,
		AvailableStock: reservation.Quantity, // This should be the quantity to release, not the new available stock
		Version:        item.Version,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to release stock: %w", err)
	}

	// Update reservation
	_, err = s.repo.UpdateReservationStatusById(ctx, repository.UpdateReservationStatusByIdParams{
		Status: "released",
		ID:     req.ReservationID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update reservation: %w", err)
	}

	return &updatedItem, nil
}

// UpdateStock implements InventoryService.UpdateStock
func (s *inventoryServiceImpl) UpdateStock(ctx context.Context, req domain.UpdateStockServiceRequest) (*repository.InventoryItem, error) {
	// Validate
	validation := s.validationService.ValidateUpdateRequest(req)
	if !validation.Valid {
		return nil, fmt.Errorf("validation failed: %v", validation.Errors)
	}

	// Get inventory
	item, err := s.repo.GetInventoryItem(ctx, req.ProductID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrProductNotFound{ProductID: req.ProductID}
		}
		return nil, fmt.Errorf("failed to get inventory: %w", err)
	}

	// Calculate new reserved stock (keeping it unchanged for now)
	updatedItem, err := s.repo.UpdateStockOptimistic(ctx, repository.UpdateStockOptimisticParams{
		ProductID:     req.ProductID,
		ReservedStock: item.ReservedStock,
		Version:       item.Version,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update stock: %w", err)
	}

	return &updatedItem, nil
}

// GetAvailableStock implements InventoryService.GetAvailableStock
func (s *inventoryServiceImpl) GetAvailableStock(ctx context.Context, productID string) (*domain.StockInfo, error) {
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

// ValidateStockLevel implements InventoryService.ValidateStockLevel
func (s *inventoryServiceImpl) ValidateStockLevel(ctx context.Context, productID string, minThreshold int32) error {
	item, err := s.repo.GetInventoryItem(ctx, productID)
	if err != nil {
		return err
	}

	if item.AvailableStock < int64(minThreshold) {
		return domain.ErrInsufficientStock{
			ProductID: productID,
			Requested: int64(minThreshold),
			Available: item.AvailableStock,
		}
	}

	return nil
}

// BatchReserveStock implements InventoryService.BatchReserveStock
func (s *inventoryServiceImpl) BatchReserveStock(ctx context.Context, requests []domain.ReserveStockServiceRequest) ([]domain.ReservationResult, error) {
	results := make([]domain.ReservationResult, 0, len(requests))

	for _, req := range requests {
		result, err := s.ReserveStock(ctx, req)
		if err != nil {
			s.logger.Error("Batch reserve failed for request", err, map[string]interface{}{
				"request_id": req.RequestID,
				"product_id": req.ProductID,
			})
			continue
		}
		results = append(results, *result)
	}

	return results, nil
}

// GetHealthStatus implements InventoryService.GetHealthStatus
func (s *inventoryServiceImpl) GetHealthStatus(ctx context.Context) (*domain.ServiceHealth, error) {
	_, err := s.repo.GetInventorySummary(ctx)
	if err != nil {
		return &domain.ServiceHealth{
			Status:    "unhealthy",
			Timestamp: time.Now(),
		}, err
	}

	return &domain.ServiceHealth{
		Status:    "healthy",
		Timestamp: time.Now(),
	}, nil
}
