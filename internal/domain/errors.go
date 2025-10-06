package domain

import (
	"fmt"
)

// Domain-specific errors for business logic layer

// ErrInsufficientStock represents insufficient stock for requested operation.
type ErrInsufficientStock struct {
	ProductID string
	Requested int64
	Available int64
}

func (e ErrInsufficientStock) Error() string {
	return fmt.Sprintf("insufficient stock for product %s: requested %d, available %d",
		e.ProductID, e.Requested, e.Available)
}

// ErrReservationExpired represents an expired reservation.
type ErrReservationExpired struct {
	ReservationID string
	ExpiredAt     string
}

func (e ErrReservationExpired) Error() string {
	return fmt.Sprintf("reservation %s expired at %s", e.ReservationID, e.ExpiredAt)
}

// ErrInvalidBusinessRule represents a business rule violation.
type ErrInvalidBusinessRule struct {
	Rule    string
	Details string
}

func (e ErrInvalidBusinessRule) Error() string {
	return fmt.Sprintf("business rule violation [%s]: %s", e.Rule, e.Details)
}

// ErrServiceUnavailable represents service unavailability.
type ErrServiceUnavailable struct {
	Service string
	Reason  string
}

func (e ErrServiceUnavailable) Error() string {
	return fmt.Sprintf("service %s unavailable: %s", e.Service, e.Reason)
}

// ErrDuplicateRequest represents a duplicate request ID.
type ErrDuplicateRequest struct {
	RequestID string
}

func (e ErrDuplicateRequest) Error() string {
	return fmt.Sprintf("duplicate request ID: %s", e.RequestID)
}

// Error type checkers
func IsInsufficientStockError(err error) bool {
	_, ok := err.(ErrInsufficientStock)
	return ok
}

func IsReservationExpiredError(err error) bool {
	_, ok := err.(ErrReservationExpired)
	return ok
}

func IsBusinessRuleError(err error) bool {
	_, ok := err.(ErrInvalidBusinessRule)
	return ok
}

func IsServiceUnavailableError(err error) bool {
	_, ok := err.(ErrServiceUnavailable)
	return ok
}

func IsDuplicateRequestError(err error) bool {
	_, ok := err.(ErrDuplicateRequest)
	return ok
}

// ErrProductNotFound represents a product that doesn't exist.
type ErrProductNotFound struct {
	ProductID string
}

func (e ErrProductNotFound) Error() string {
	return fmt.Sprintf("product not found: %s", e.ProductID)
}

// ErrReservationNotFound represents a reservation that doesn't exist.
type ErrReservationNotFound struct {
	ReservationID string
}

func (e ErrReservationNotFound) Error() string {
	return fmt.Sprintf("reservation not found: %s", e.ReservationID)
}

// ErrReservationNotActive represents a reservation that is not active.
type ErrReservationNotActive struct {
	ReservationID string
	Status        string
}

func (e ErrReservationNotActive) Error() string {
	return fmt.Sprintf("reservation %s is not active, current status: %s", e.ReservationID, e.Status)
}

// ErrInvalidStockOperation represents an invalid stock operation.
type ErrInvalidStockOperation struct {
	ProductID string
	Operation string
	Reason    string
}

func (e ErrInvalidStockOperation) Error() string {
	return fmt.Sprintf("invalid stock operation for product %s [%s]: %s", e.ProductID, e.Operation, e.Reason)
}
