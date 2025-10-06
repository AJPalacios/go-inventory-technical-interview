// Package domain contains core business entities and value objects.
//
// This package defines the domain models that represent the core business
// concepts of the inventory management system. It follows Domain-Driven Design
// principles and is independent of external frameworks or infrastructure.
package domain

import (
	"time"
)

// ReservationResult represents the result of a stock reservation operation.
//
// This value object encapsulates all the information about a completed
// reservation, including its status, timing, and associated metadata.
type ReservationResult struct {
	ReservationID string            `json:"reservation_id"`
	ProductID     string            `json:"product_id"`
	Quantity      int64             `json:"quantity"`
	Status        ReservationStatus `json:"status"`
	ExpiresAt     time.Time         `json:"expires_at"`
	CreatedAt     time.Time         `json:"created_at"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// StockInfo represents comprehensive stock information for a product.
//
// This value object provides a complete view of stock levels, including
// available, reserved, and total quantities, along with versioning
// information for optimistic locking.
type StockInfo struct {
	ProductID      string    `json:"product_id"`
	AvailableStock int64     `json:"available_stock"`
	ReservedStock  int64     `json:"reserved_stock"`
	TotalStock     int64     `json:"total_stock"`
	Version        int64     `json:"version"`
	LastUpdated    time.Time `json:"last_updated"`
}

// ReservationStatus represents the current state of a stock reservation.
type ReservationStatus string

const (
	// ReservationStatusActive indicates the reservation is currently holding stock
	ReservationStatusActive ReservationStatus = "active"
	// ReservationStatusReleased indicates the reservation was released back to available stock
	ReservationStatusReleased ReservationStatus = "released"
	// ReservationStatusExpired indicates the reservation expired without being consumed
	ReservationStatusExpired ReservationStatus = "expired"
	// ReservationStatusConsumed indicates the reservation was used in a transaction
	ReservationStatusConsumed ReservationStatus = "consumed"
)

// ServiceHealth represents the health status of a service component.
type ServiceHealth struct {
	Status           string                 `json:"status"`
	Timestamp        time.Time              `json:"timestamp"`
	Version          string                 `json:"version"`
	Uptime           time.Duration          `json:"uptime"`
	ComponentsHealth map[string]interface{} `json:"components"`
}

// ValidationResult represents the outcome of business rule validation.
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// BatchSummary provides summary statistics for batch operations.
type BatchSummary struct {
	Total     int `json:"total"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
}

// Request types for service layer
type ReserveStockServiceRequest struct {
	ProductID      string            `json:"product_id" validate:"required,uuid"`
	Quantity       int32             `json:"quantity" validate:"required,min=1"`
	RequestID      string            `json:"request_id" validate:"required"`
	TimeoutSeconds int32             `json:"timeout_seconds,omitempty"`
	Reason         string            `json:"reason,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	ClientID       string            `json:"client_id,omitempty"`
}

type ReleaseStockServiceRequest struct {
	ReservationID string `json:"reservation_id" validate:"required,uuid"`
	Reason        string `json:"reason" validate:"required,oneof=cancelled purchased timeout expired admin"`
	RequestID     string `json:"request_id" validate:"required"`
}

type UpdateStockServiceRequest struct {
	ProductID      string            `json:"product_id" validate:"required,uuid"`
	NewStock       int32             `json:"new_stock" validate:"min=0"`
	AdjustmentType string            `json:"adjustment_type" validate:"required,oneof=restock adjustment return correction"`
	Reason         string            `json:"reason" validate:"required"`
	Reference      string            `json:"reference,omitempty"`
	RequestID      string            `json:"request_id" validate:"required"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// Additional request and response types for service layer compatibility

// ReserveStockRequest represents a request to reserve stock
type ReserveStockRequest struct {
	ProductID      string            `json:"product_id" validate:"required,uuid"`
	Quantity       int64             `json:"quantity" validate:"required,min=1"`
	RequestID      string            `json:"request_id" validate:"required"`
	TimeoutSeconds int32             `json:"timeout_seconds,omitempty"`
	Reason         string            `json:"reason,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	ClientID       string            `json:"client_id,omitempty"`
}

// ReleaseReservationRequest represents a request to release a reservation
type ReleaseReservationRequest struct {
	ReservationID string `json:"reservation_id" validate:"required,uuid"`
	Reason        string `json:"reason" validate:"required,oneof=cancelled purchased timeout expired admin"`
	RequestID     string `json:"request_id" validate:"required"`
}

// UpdateStockRequest represents a request to update stock levels
type UpdateStockRequest struct {
	ProductID   string            `json:"product_id" validate:"required,uuid"`
	NewQuantity int64             `json:"new_quantity" validate:"min=0"`
	Operation   StockOperation    `json:"operation" validate:"required"`
	Reason      string            `json:"reason" validate:"required"`
	Reference   string            `json:"reference,omitempty"`
	RequestID   string            `json:"request_id" validate:"required"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// StockOperation represents the type of stock operation to perform
type StockOperation string

const (
	StockOperationSet    StockOperation = "set"
	StockOperationAdjust StockOperation = "adjust"
)

// ReleaseResult represents the result of releasing a reservation
type ReleaseResult struct {
	ReservationID string            `json:"reservation_id"`
	ProductID     string            `json:"product_id"`
	Quantity      int64             `json:"quantity"`
	Status        ReservationStatus `json:"status"`
	ReleasedAt    time.Time         `json:"released_at"`
	Reason        string            `json:"reason,omitempty"`
}
