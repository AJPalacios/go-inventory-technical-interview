package repository

import (
	"errors"
	"fmt"
)

// Core repository errors
var (
	// Concurrency errors
	ErrVersionConflict    = fmt.Errorf("repository: version conflict: optimistic lock failed")
	ErrMaxRetriesExceeded = fmt.Errorf("repository: maximum retries exceeded")

	// Business logic errors
	ErrInsufficientStock = fmt.Errorf("repository: insufficient stock available")
	ErrInvalidQuantity   = fmt.Errorf("repository: invalid quantity: must be positive")
	ErrInvalidVersion    = fmt.Errorf("repository: invalid version: must be positive")

	// Entity not found errors
	ErrProductNotFound     = fmt.Errorf("repository: product not found")
	ErrInventoryNotFound   = fmt.Errorf("repository: inventory item not found")
	ErrReservationNotFound = fmt.Errorf("repository: reservation not found")
	ErrIdempotencyNotFound = fmt.Errorf("repository: idempotency key not found")

	// State errors
	ErrReservationExpired  = fmt.Errorf("repository: reservation has expired")
	ErrReservationReleased = fmt.Errorf("repository: reservation already released")
	ErrStockBelowReserved  = fmt.Errorf("repository: stock cannot be below reserved amount")
)

// RepositoryError wraps errors with additional context
type RepositoryError struct {
	Op      string                 // Operation being performed
	Entity  string                 // Entity type (product, inventory, reservation)
	ID      string                 // Entity ID for context
	Err     error                  // Underlying error
	Context map[string]interface{} // Additional context
}

func (e *RepositoryError) Error() string {
	if e.Context != nil && len(e.Context) > 0 {
		return fmt.Sprintf("repository.%s failed for %s[%s]: %v (context: %+v)",
			e.Op, e.Entity, e.ID, e.Err, e.Context)
	}
	return fmt.Sprintf("repository.%s failed for %s[%s]: %v",
		e.Op, e.Entity, e.ID, e.Err)
}

func (e *RepositoryError) Unwrap() error {
	return e.Err
}

// Error constructors with context
func NewRepositoryError(op, entity, id string, err error) *RepositoryError {
	return &RepositoryError{
		Op:     op,
		Entity: entity,
		ID:     id,
		Err:    err,
	}
}

func NewRepositoryErrorWithContext(op, entity, id string, err error, context map[string]interface{}) *RepositoryError {
	return &RepositoryError{
		Op:      op,
		Entity:  entity,
		ID:      id,
		Err:     err,
		Context: context,
	}
}

// Specific error constructors for common scenarios
func NewVersionConflictError(entity, id string, expectedVersion, actualVersion int64) *RepositoryError {
	return NewRepositoryErrorWithContext(
		"optimistic_update",
		entity,
		id,
		ErrVersionConflict,
		map[string]interface{}{
			"expected_version": expectedVersion,
			"actual_version":   actualVersion,
		},
	)
}

func NewInsufficientStockError(productID string, requested, available int64) *RepositoryError {
	return NewRepositoryErrorWithContext(
		"reserve_stock",
		"inventory",
		productID,
		ErrInsufficientStock,
		map[string]interface{}{
			"requested": requested,
			"available": available,
			"shortfall": requested - available,
		},
	)
}

// NewMaxRetriesError creates an error for when max retries are exceeded
func NewMaxRetriesError(op, entity, id string, attempts int, lastErr error) *RepositoryError {
	// Wrap the last error so errors.Is() still works
	wrappedErr := fmt.Errorf("%w: %s", ErrMaxRetriesExceeded, lastErr.Error())

	// If the last error was a RepositoryError, preserve its context
	var repoErr *RepositoryError
	if errors.As(lastErr, &repoErr) {
		// Keep the original error wrapped so IsVersionConflict still works
		if errors.Is(lastErr, ErrVersionConflict) {
			wrappedErr = fmt.Errorf("%w (after %d retries)", repoErr.Err, attempts)
		}

		return NewRepositoryErrorWithContext(
			repoErr.Op,
			repoErr.Entity,
			repoErr.ID,
			wrappedErr,
			map[string]interface{}{
				"attempts":        attempts,
				"last_error":      lastErr.Error(),
				"original_op":     repoErr.Op,
				"original_entity": repoErr.Entity,
				"original_id":     repoErr.ID,
				"original_error":  repoErr.Err.Error(),
			},
		)
	}

	// Fallback for non-RepositoryError cases
	return NewRepositoryErrorWithContext(
		op,
		entity,
		id,
		ErrMaxRetriesExceeded,
		map[string]interface{}{
			"attempts":   attempts,
			"last_error": lastErr.Error(),
		},
	)
}

// Error checking helpers
func IsVersionConflict(err error) bool {
	return errors.Is(err, ErrVersionConflict)
}

func IsInsufficientStock(err error) bool {
	return errors.Is(err, ErrInsufficientStock)
}

func IsNotFound(err error) bool {
	return errors.Is(err, ErrProductNotFound) ||
		errors.Is(err, ErrInventoryNotFound) ||
		errors.Is(err, ErrReservationNotFound) ||
		errors.Is(err, ErrIdempotencyNotFound)
}

func IsRetryable(err error) bool {
	return IsVersionConflict(err)
}
