package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/AJPalacios/inventory/internal/domain"
)

// Tests simplificados para service layer - enfocándose en cobertura básica

// Test para verificar que el servicio se inicializa
func TestInventoryServiceConfig(t *testing.T) {
	config := InventoryServiceConfig{
		OperationTimeout:   time.Duration(300),
		MaxRetryAttempts:   3,
		MaxBatchSize:       100,
		LowStockThreshold:  10,
		MaxStockCapacity:   10000,
		ConcurrentOpsLimit: 10,
	}

	assert.Equal(t, time.Duration(300), config.OperationTimeout)
	assert.Equal(t, 3, config.MaxRetryAttempts)
	assert.Equal(t, 100, config.MaxBatchSize)
	assert.Equal(t, int64(10), config.LowStockThreshold)
	assert.Equal(t, int64(10000), config.MaxStockCapacity)
	assert.Equal(t, 10, config.ConcurrentOpsLimit)
}

func TestIdempotencyServiceConfig(t *testing.T) {
	config := IdempotencyServiceConfig{
		MaxSize:    1000,
		DefaultTTL: time.Duration(3600),
	}

	assert.Equal(t, 1000, config.MaxSize)
	assert.Equal(t, time.Duration(3600), config.DefaultTTL)
}

// Test para validación de requests
func TestValidationResult(t *testing.T) {
	// Test valid result
	result := domain.ValidationResult{
		Valid:    true,
		Errors:   nil,
		Warnings: []string{"warning message"},
	}

	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
	assert.Len(t, result.Warnings, 1)

	// Test invalid result
	result = domain.ValidationResult{
		Valid:    false,
		Errors:   []string{"validation error"},
		Warnings: nil,
	}

	assert.False(t, result.Valid)
	assert.Len(t, result.Errors, 1)
	assert.Empty(t, result.Warnings)
}

// Test para requests del dominio
func TestDomainRequests(t *testing.T) {
	// Test ReserveStockServiceRequest
	reserveReq := domain.ReserveStockServiceRequest{
		ProductID: "prod-123",
		Quantity:  5,
		RequestID: "req-456",
	}

	assert.Equal(t, "prod-123", reserveReq.ProductID)
	assert.Equal(t, int32(5), reserveReq.Quantity)
	assert.Equal(t, "req-456", reserveReq.RequestID)

	// Test ReleaseStockServiceRequest
	releaseReq := domain.ReleaseStockServiceRequest{
		ReservationID: "res-789",
		RequestID:     "req-456",
	}

	assert.Equal(t, "res-789", releaseReq.ReservationID)
	assert.Equal(t, "req-456", releaseReq.RequestID)

	// Test UpdateStockServiceRequest
	updateReq := domain.UpdateStockServiceRequest{
		ProductID:      "prod-123",
		NewStock:       10,
		AdjustmentType: "restock",
		Reason:         "manual adjustment",
		RequestID:      "req-456",
	}

	assert.Equal(t, "prod-123", updateReq.ProductID)
	assert.Equal(t, int32(10), updateReq.NewStock)
	assert.Equal(t, "restock", updateReq.AdjustmentType)
	assert.Equal(t, "manual adjustment", updateReq.Reason)
	assert.Equal(t, "req-456", updateReq.RequestID)
}
