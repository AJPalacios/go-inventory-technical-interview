package service

import "time"

// InventoryServiceConfig holds configuration for the inventory service.
type InventoryServiceConfig struct {
	OperationTimeout   time.Duration
	MaxRetryAttempts   int
	MaxBatchSize       int
	LowStockThreshold  int64
	MaxStockCapacity   int64
	ConcurrentOpsLimit int
}

// IdempotencyServiceConfig holds configuration for the idempotency service.
type IdempotencyServiceConfig struct {
	MaxSize    int
	DefaultTTL time.Duration
}
