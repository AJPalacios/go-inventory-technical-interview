package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// InventoryRepository defines enhanced inventory operations with optimistic locking
type InventoryRepository interface {
	// Stock Operations with Optimistic Locking
	ReserveStock(ctx context.Context, req ReserveStockRequest) (*InventoryItem, error)
	ReleaseStock(ctx context.Context, req ReleaseStockRequest) (*InventoryItem, error)
	UpdateStock(ctx context.Context, req UpdateStockRequest) (*InventoryItem, error)

	// Inventory Queries
	GetInventoryByProduct(ctx context.Context, productID string) (*InventoryItem, error)
	GetInventoryForUpdate(ctx context.Context, productID string) (*InventoryItem, error)

	// Reservation Operations
	CreateReservation(ctx context.Context, req CreateReservationRequest) (*Reservation, error)
	UpdateReservationStatus(ctx context.Context, reservationID string, status string) (*Reservation, error)
	GetReservation(ctx context.Context, reservationID string) (*Reservation, error)
	GetReservationByRequestID(ctx context.Context, requestID string) (*Reservation, error)
	CleanupExpiredReservations(ctx context.Context, limit int32) error

	// Idempotency Support
	StoreIdempotencyKey(ctx context.Context, req IdempotencyRequest) (*IdempotencyKey, error)
	GetIdempotencyKey(ctx context.Context, requestID string) (*IdempotencyKey, error)

	// Product Operations
	CreateProduct(ctx context.Context, req CreateProductRequest) (*Product, error)
	GetProduct(ctx context.Context, productID string) (*Product, error)
	ListProducts(ctx context.Context) ([]Product, error)

	// Atomic Composite Operations (ACID-compliant)
	AtomicReserveAndCreateReservation(ctx context.Context, req ReserveStockRequest, reservationReq CreateReservationRequest) (*InventoryItem, *Reservation, error)
	AtomicReleaseAndUpdateReservation(ctx context.Context, releaseReq ReleaseStockRequest, reservationID, newStatus string) (*InventoryItem, *Reservation, error)

	// Transaction Management
	WithTransaction(ctx context.Context, fn func(repo InventoryRepository) error) error
	WithTransactionIsolation(ctx context.Context, level TransactionIsolationLevel, fn func(repo InventoryRepository) error) error

	// ACID Validation
	ValidateACIDCompliance(ctx context.Context) error
}

// Request types for better API design
type ReserveStockRequest struct {
	ProductID string
	Quantity  int64
	Version   int64
	RequestID string // For idempotency
}

type ReleaseStockRequest struct {
	ProductID string
	Quantity  int64
	Version   int64
	RequestID string
}

type UpdateStockRequest struct {
	ProductID  string
	TotalStock int64
	Version    int64
	RequestID  string
}

type CreateReservationRequest struct {
	ProductID string
	Quantity  int64
	RequestID string
	ExpiresAt time.Time
}

type CreateProductRequest struct {
	Name        string
	Description string
}

type IdempotencyRequest struct {
	RequestID     string
	OperationType string
	ResponseData  []byte
	ExpiresAt     time.Time
}

// inventoryRepository implements InventoryRepository with enhanced error handling and retry logic
type inventoryRepository struct {
	queries *Queries
	db      *sql.DB
}

// NewInventoryRepository creates a new enhanced inventory repository
func NewInventoryRepository(db *sql.DB) InventoryRepository {
	return &inventoryRepository{
		queries: New(db),
		db:      db,
	}
}

// ReserveStock implements optimistic locking with retry logic
// It checks for version conflicts and insufficient stock
// The operation is idempotent based on RequestID
// It returns a RepositoryError with context if any validation fails
func (r *inventoryRepository) ReserveStock(ctx context.Context, req ReserveStockRequest) (*InventoryItem, error) {
	if req.Quantity <= 0 {
		return nil, NewRepositoryError("reserve_stock", "inventory", req.ProductID, ErrInvalidQuantity)
	}

	var result *InventoryItem

	err := StandardRetry.Execute(ctx, func() error {
		params := ReserveStockOptimisticParams{
			AvailableStock: req.Quantity,
			ProductID:      req.ProductID,
			Version:        req.Version,
		}

		item, err := r.queries.ReserveStockOptimistic(ctx, params)
		if err != nil {
			if err == sql.ErrNoRows {
				// Check if it's version conflict or insufficient stock
				currentItem, getErr := r.queries.GetInventoryForUpdate(ctx, req.ProductID)
				if getErr == sql.ErrNoRows {
					return NewRepositoryError("reserve_stock", "inventory", req.ProductID, ErrInventoryNotFound)
				}
				if getErr != nil {
					return NewRepositoryError("reserve_stock", "inventory", req.ProductID, getErr)
				}

				// Check version conflict
				if currentItem.Version != req.Version {
					return NewVersionConflictError("inventory", req.ProductID, req.Version, currentItem.Version)
				}

				// Must be insufficient stock
				return NewInsufficientStockError(req.ProductID, req.Quantity, currentItem.AvailableStock)
			}
			return NewRepositoryError("reserve_stock", "inventory", req.ProductID, err)
		}

		result = &item
		return nil
	})

	return result, err
}

// ReleaseStock implements optimistic stock release with validation
func (r *inventoryRepository) ReleaseStock(ctx context.Context, req ReleaseStockRequest) (*InventoryItem, error) {
	if req.Quantity <= 0 {
		return nil, NewRepositoryError("release_stock", "inventory", req.ProductID, ErrInvalidQuantity)
	}

	var result *InventoryItem

	err := StandardRetry.Execute(ctx, func() error {
		params := ReleaseStockOptimisticParams{
			AvailableStock: req.Quantity,
			ProductID:      req.ProductID,
			Version:        req.Version,
		}

		item, err := r.queries.ReleaseStockOptimistic(ctx, params)
		if err != nil {
			if err == sql.ErrNoRows {
				// Check if it's version conflict or insufficient reserved stock
				currentItem, getErr := r.queries.GetInventoryForUpdate(ctx, req.ProductID)
				if getErr == sql.ErrNoRows {
					return NewRepositoryError("release_stock", "inventory", req.ProductID, ErrInventoryNotFound)
				}
				if getErr != nil {
					return NewRepositoryError("release_stock", "inventory", req.ProductID, getErr)
				}

				// Check version conflict
				if currentItem.Version != req.Version {
					return NewVersionConflictError("inventory", req.ProductID, req.Version, currentItem.Version)
				}

				// Must be insufficient reserved stock
				return NewRepositoryErrorWithContext(
					"release_stock", "inventory", req.ProductID,
					fmt.Errorf("insufficient reserved stock: requested=%d, available=%d", req.Quantity, currentItem.ReservedStock),
					map[string]interface{}{
						"requested": req.Quantity,
						"reserved":  currentItem.ReservedStock,
					},
				)
			}
			return NewRepositoryError("release_stock", "inventory", req.ProductID, err)
		}

		result = &item
		return nil
	})

	return result, err
}

// UpdateStock updates total stock with optimistic locking
func (r *inventoryRepository) UpdateStock(ctx context.Context, req UpdateStockRequest) (*InventoryItem, error) {
	if req.TotalStock < 0 {
		return nil, NewRepositoryError("update_stock", "inventory", req.ProductID, ErrInvalidQuantity)
	}

	var result *InventoryItem

	err := StandardRetry.Execute(ctx, func() error {
		params := UpdateStockOptimisticParams{
			ReservedStock: req.TotalStock,
			ProductID:     req.ProductID,
			Version:       req.Version,
		}

		item, err := r.queries.UpdateStockOptimistic(ctx, params)
		if err != nil {
			if err == sql.ErrNoRows {
				// Check the specific reason for failure
				currentItem, getErr := r.queries.GetInventoryForUpdate(ctx, req.ProductID)
				if getErr == sql.ErrNoRows {
					return NewRepositoryError("update_stock", "inventory", req.ProductID, ErrInventoryNotFound)
				}
				if getErr != nil {
					return NewRepositoryError("update_stock", "inventory", req.ProductID, getErr)
				}

				// Check version conflict
				if currentItem.Version != req.Version {
					return NewVersionConflictError("inventory", req.ProductID, req.Version, currentItem.Version)
				}

				// Must be trying to set total below reserved
				return NewRepositoryErrorWithContext(
					"update_stock", "inventory", req.ProductID,
					ErrStockBelowReserved,
					map[string]interface{}{
						"total_stock":    req.TotalStock,
						"reserved_stock": currentItem.ReservedStock,
					},
				)
			}
			return NewRepositoryError("update_stock", "inventory", req.ProductID, err)
		}

		result = &item
		return nil
	})

	return result, err
}

// GetInventoryByProduct retrieves inventory item by product ID
func (r *inventoryRepository) GetInventoryByProduct(ctx context.Context, productID string) (*InventoryItem, error) {
	item, err := r.queries.GetInventoryItem(ctx, productID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewRepositoryError("get_inventory", "inventory", productID, ErrInventoryNotFound)
		}
		return nil, NewRepositoryError("get_inventory", "inventory", productID, err)
	}
	return &item, nil
}

// GetInventoryForUpdate retrieves inventory with current version for optimistic locking
func (r *inventoryRepository) GetInventoryForUpdate(ctx context.Context, productID string) (*InventoryItem, error) {
	item, err := r.queries.GetInventoryForUpdate(ctx, productID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewRepositoryError("get_inventory_for_update", "inventory", productID, ErrInventoryNotFound)
		}
		return nil, NewRepositoryError("get_inventory_for_update", "inventory", productID, err)
	}
	return &item, nil
}

// CreateReservation creates a new reservation with timeout
func (r *inventoryRepository) CreateReservation(ctx context.Context, req CreateReservationRequest) (*Reservation, error) {
	id := uuid.New().String()

	params := CreateReservationWithTimeoutParams{
		ID:        id,
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
		RequestID: req.RequestID,
		ExpiresAt: sql.NullTime{Time: req.ExpiresAt, Valid: true},
	}

	reservation, err := r.queries.CreateReservationWithTimeout(ctx, params)
	if err != nil {
		return nil, NewRepositoryError("create_reservation", "reservation", id, err)
	}

	return &reservation, nil
}

// UpdateReservationStatus updates reservation status optimistically
func (r *inventoryRepository) UpdateReservationStatus(ctx context.Context, reservationID string, status string) (*Reservation, error) {
	params := UpdateReservationStatusByIdParams{
		Status: status,
		ID:     reservationID,
	}

	reservation, err := r.queries.UpdateReservationStatusById(ctx, params)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewRepositoryError("update_reservation_status", "reservation", reservationID, ErrReservationNotFound)
		}
		return nil, NewRepositoryError("update_reservation_status", "reservation", reservationID, err)
	}

	return &reservation, nil
}

// GetReservation retrieves a reservation by ID
func (r *inventoryRepository) GetReservation(ctx context.Context, reservationID string) (*Reservation, error) {
	reservation, err := r.queries.GetReservation(ctx, reservationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewRepositoryError("get_reservation", "reservation", reservationID, ErrReservationNotFound)
		}
		return nil, NewRepositoryError("get_reservation", "reservation", reservationID, err)
	}

	return &reservation, nil
}

// GetReservationByRequestID retrieves a reservation by request ID (for idempotency)
func (r *inventoryRepository) GetReservationByRequestID(ctx context.Context, requestID string) (*Reservation, error) {
	reservation, err := r.queries.GetReservationByRequestID(ctx, requestID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewRepositoryError("get_reservation_by_request", "reservation", requestID, ErrReservationNotFound)
		}
		return nil, NewRepositoryError("get_reservation_by_request", "reservation", requestID, err)
	}

	return &reservation, nil
}

// CleanupExpiredReservations marks expired reservations and returns count
func (r *inventoryRepository) CleanupExpiredReservations(ctx context.Context, limit int32) error {
	// Get expired reservations first
	expired, err := r.queries.GetExpiredReservations(ctx, int64(limit))
	if err != nil {
		return NewRepositoryError("cleanup_expired_reservations", "reservation", "batch", err)
	}

	if len(expired) == 0 {
		return nil // Nothing to cleanup
	}

	// Mark them as expired
	err = r.queries.MarkReservationsExpired(ctx)
	if err != nil {
		return NewRepositoryError("cleanup_expired_reservations", "reservation", "batch", err)
	}

	return nil
}

// StoreIdempotencyKey stores an idempotency key with response data
func (r *inventoryRepository) StoreIdempotencyKey(ctx context.Context, req IdempotencyRequest) (*IdempotencyKey, error) {
	params := StoreIdempotencyKeyParams{
		RequestID:     req.RequestID,
		OperationType: req.OperationType,
		ResponseData:  sql.NullString{String: string(req.ResponseData), Valid: len(req.ResponseData) > 0},
		ExpiresAt:     req.ExpiresAt,
	}

	key, err := r.queries.StoreIdempotencyKey(ctx, params)
	if err != nil {
		return nil, NewRepositoryError("store_idempotency_key", "idempotency", req.RequestID, err)
	}

	return &key, nil
}

// GetIdempotencyKey retrieves a valid (non-expired) idempotency key
func (r *inventoryRepository) GetIdempotencyKey(ctx context.Context, requestID string) (*IdempotencyKey, error) {
	key, err := r.queries.GetValidIdempotencyKey(ctx, requestID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewRepositoryError("get_idempotency_key", "idempotency", requestID, ErrIdempotencyNotFound)
		}
		return nil, NewRepositoryError("get_idempotency_key", "idempotency", requestID, err)
	}

	return &key, nil
}

// CreateProduct creates a new product
func (r *inventoryRepository) CreateProduct(ctx context.Context, req CreateProductRequest) (*Product, error) {
	id := uuid.New().String()

	params := CreateProductParams{
		ID:          id,
		Name:        req.Name,
		Description: sql.NullString{String: req.Description, Valid: req.Description != ""},
	}

	product, err := r.queries.CreateProduct(ctx, params)
	if err != nil {
		return nil, NewRepositoryError("create_product", "product", id, err)
	}

	return &product, nil
}

// GetProduct retrieves a product by ID
func (r *inventoryRepository) GetProduct(ctx context.Context, productID string) (*Product, error) {
	product, err := r.queries.GetProduct(ctx, productID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewRepositoryError("get_product", "product", productID, ErrProductNotFound)
		}
		return nil, NewRepositoryError("get_product", "product", productID, err)
	}

	return &product, nil
}

// ListProducts retrieves all products
func (r *inventoryRepository) ListProducts(ctx context.Context) ([]Product, error) {
	products, err := r.queries.ListProducts(ctx)
	if err != nil {
		return nil, NewRepositoryError("list_products", "product", "all", err)
	}

	return products, nil
}

// WithTransaction executes a function within a database transaction
// This is useful to group multiple operations atomically
// with Begin, Commit, and Rollback handling
func (r *inventoryRepository) WithTransaction(ctx context.Context, fn func(repo InventoryRepository) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return NewRepositoryError("begin_transaction", "transaction", "", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Create a new repository instance with the transaction
	txRepo := &inventoryRepository{
		queries: r.queries.WithTx(tx),
		db:      r.db,
	}

	if err := fn(txRepo); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return NewRepositoryError("commit_transaction", "transaction", "", err)
	}

	return nil
}
