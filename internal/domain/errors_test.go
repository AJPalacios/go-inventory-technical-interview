package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrInsufficientStock_Error(t *testing.T) {
	err := ErrInsufficientStock{
		ProductID: "product-1",
		Requested: 10,
		Available: 5,
	}

	expected := "insufficient stock for product product-1: requested 10, available 5"
	assert.Equal(t, expected, err.Error())
}

func TestErrReservationExpired_Error(t *testing.T) {
	err := ErrReservationExpired{
		ReservationID: "res-123",
		ExpiredAt:     "2023-10-06T10:00:00Z",
	}

	expected := "reservation res-123 expired at 2023-10-06T10:00:00Z"
	assert.Equal(t, expected, err.Error())
}

func TestErrInvalidBusinessRule_Error(t *testing.T) {
	err := ErrInvalidBusinessRule{
		Rule:    "quantity_validation",
		Details: "quantity must be positive",
	}

	expected := "business rule violation [quantity_validation]: quantity must be positive"
	assert.Equal(t, expected, err.Error())
}

func TestErrServiceUnavailable_Error(t *testing.T) {
	err := ErrServiceUnavailable{
		Service: "inventory",
		Reason:  "database connection failed",
	}

	expected := "service inventory unavailable: database connection failed"
	assert.Equal(t, expected, err.Error())
}

func TestErrDuplicateRequest_Error(t *testing.T) {
	err := ErrDuplicateRequest{
		RequestID: "req-123",
	}

	expected := "duplicate request ID: req-123"
	assert.Equal(t, expected, err.Error())
}

func TestErrProductNotFound_Error(t *testing.T) {
	err := ErrProductNotFound{
		ProductID: "product-1",
	}

	expected := "product not found: product-1"
	assert.Equal(t, expected, err.Error())
}

func TestErrReservationNotFound_Error(t *testing.T) {
	err := ErrReservationNotFound{
		ReservationID: "res-123",
	}

	expected := "reservation not found: res-123"
	assert.Equal(t, expected, err.Error())
}

func TestErrReservationNotActive_Error(t *testing.T) {
	err := ErrReservationNotActive{
		ReservationID: "res-123",
		Status:        "expired",
	}

	expected := "reservation res-123 is not active, current status: expired"
	assert.Equal(t, expected, err.Error())
}

func TestErrInvalidStockOperation_Error(t *testing.T) {
	err := ErrInvalidStockOperation{
		ProductID: "product-1",
		Operation: "negative_adjustment",
		Reason:    "would result in negative stock",
	}

	expected := "invalid stock operation for product product-1 [negative_adjustment]: would result in negative stock"
	assert.Equal(t, expected, err.Error())
}

func TestIsInsufficientStockError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "insufficient stock error",
			err:      ErrInsufficientStock{ProductID: "product-1", Requested: 10, Available: 5},
			expected: true,
		},
		{
			name:     "other error",
			err:      ErrProductNotFound{ProductID: "product-1"},
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsInsufficientStockError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsReservationExpiredError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "reservation expired error",
			err:      ErrReservationExpired{ReservationID: "res-123", ExpiredAt: "2023-10-06T10:00:00Z"},
			expected: true,
		},
		{
			name:     "other error",
			err:      ErrProductNotFound{ProductID: "product-1"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsReservationExpiredError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsBusinessRuleError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "business rule error",
			err:      ErrInvalidBusinessRule{Rule: "test", Details: "test details"},
			expected: true,
		},
		{
			name:     "other error",
			err:      ErrProductNotFound{ProductID: "product-1"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsBusinessRuleError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsServiceUnavailableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "service unavailable error",
			err:      ErrServiceUnavailable{Service: "test", Reason: "test reason"},
			expected: true,
		},
		{
			name:     "other error",
			err:      ErrProductNotFound{ProductID: "product-1"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsServiceUnavailableError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsDuplicateRequestError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "duplicate request error",
			err:      ErrDuplicateRequest{RequestID: "req-123"},
			expected: true,
		},
		{
			name:     "other error",
			err:      ErrProductNotFound{ProductID: "product-1"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDuplicateRequestError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsProductNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "product not found error",
			err:      ErrProductNotFound{ProductID: "product-1"},
			expected: true,
		},
		{
			name:     "other error",
			err:      ErrInsufficientStock{ProductID: "product-1", Requested: 10, Available: 5},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsProductNotFoundError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsReservationNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "reservation not found error",
			err:      ErrReservationNotFound{ReservationID: "res-123"},
			expected: true,
		},
		{
			name:     "other error",
			err:      ErrProductNotFound{ProductID: "product-1"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsReservationNotFoundError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
