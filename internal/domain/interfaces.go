package domain

import (
	"context"
	"time"

	"github.com/AJPalacios/inventory/internal/repository"
)

// InventoryService defines core inventory business operations.
//
// This interface provides business logic for inventory management,
// including stock reservations, releases, and updates with proper
// validation and error handling.
type InventoryService interface {
	// Core inventory operations
	ReserveStock(ctx context.Context, req ReserveStockServiceRequest) (*ReservationResult, error)
	ReleaseStock(ctx context.Context, req ReleaseStockServiceRequest) (*repository.InventoryItem, error)
	UpdateStock(ctx context.Context, req UpdateStockServiceRequest) (*repository.InventoryItem, error)

	// Business operations
	GetAvailableStock(ctx context.Context, productID string) (*StockInfo, error)
	ValidateStockLevel(ctx context.Context, productID string, minThreshold int32) error

	// Batch operations
	BatchReserveStock(ctx context.Context, requests []ReserveStockServiceRequest) ([]ReservationResult, error)

	// Health and monitoring
	GetHealthStatus(ctx context.Context) (*ServiceHealth, error)
}

// ReservationService manages stock reservations lifecycle.
type ReservationService interface {
	CreateReservation(ctx context.Context, productID string, quantity int32, requestID string) (*ReservationResult, error)
	CancelReservation(ctx context.Context, reservationID string, reason string) error
	ExtendReservation(ctx context.Context, reservationID string, newExpiration time.Time) error
	GetReservationStatus(ctx context.Context, reservationID string) (*ReservationResult, error)
	CleanupExpired(ctx context.Context) (int, error)
}

// IdempotencyService handles request deduplication.
type IdempotencyService interface {
	CheckIdempotency(ctx context.Context, requestID string) (interface{}, bool, error)
	StoreResult(ctx context.Context, requestID string, result interface{}, ttl time.Duration) error
	CleanupExpired(ctx context.Context) (int, error)
}

// ValidationService handles business rule validation.
type ValidationService interface {
	ValidateReserveRequest(req ReserveStockServiceRequest) ValidationResult
	ValidateReleaseRequest(req ReleaseStockServiceRequest) ValidationResult
	ValidateUpdateRequest(req UpdateStockServiceRequest) ValidationResult
}

// AGNOSTIC INTERFACES FOR EXTERNAL DEPENDENCIES

// MetricsProvider defines interface for metrics collection (agnostic).
//
// This interface allows switching between different metrics providers
// (Prometheus, StatsD, DataDog, etc.) without changing business logic.
type MetricsProvider interface {
	// Counters
	IncrementCounter(name string, labels map[string]string)
	IncrementCounterBy(name string, value float64, labels map[string]string)

	// Gauges
	SetGauge(name string, value float64, labels map[string]string)
	AddGauge(name string, value float64, labels map[string]string)

	// Histograms
	RecordHistogram(name string, value float64, labels map[string]string)

	// Timing
	RecordDuration(name string, duration time.Duration, labels map[string]string)
}

// Logger defines interface for structured logging (agnostic).
//
// This interface allows switching between different logging providers
// (Zap, Logrus, etc.) without changing business logic.
type Logger interface {
	Debug(msg string, fields ...map[string]interface{})
	Info(msg string, fields ...map[string]interface{})
	Warn(msg string, fields ...map[string]interface{})
	Error(msg string, err error, fields ...map[string]interface{})
	With(fields map[string]interface{}) Logger
}

// CacheProvider defines interface for caching (agnostic).
//
// This interface allows switching between different cache providers
// (Redis, Memcached, in-memory, etc.) without changing business logic.
type CacheProvider interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}

// CircuitBreaker defines interface for fault tolerance (agnostic).
type CircuitBreaker interface {
	Execute(operation func() (interface{}, error)) (interface{}, error)
	GetState() string
	Reset()
}
