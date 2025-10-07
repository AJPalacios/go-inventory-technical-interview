package domain

import (
	"strings"
	"testing"

	"github.com/AJPalacios/inventory/pkg/validator"
	"github.com/stretchr/testify/assert"
)

func TestNewValidationService(t *testing.T) {
	service := NewValidationService()
	assert.NotNil(t, service)
}

func TestValidateReserveRequest(t *testing.T) {
	service := NewValidationService()

	tests := []struct {
		name    string
		req     ReserveStockServiceRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			req: ReserveStockServiceRequest{
				ProductID: "550e8400-e29b-41d4-a716-446655440000",
				Quantity:  10,
				RequestID: "req-123",
			},
			wantErr: false,
		},
		{
			name: "empty product ID",
			req: ReserveStockServiceRequest{
				ProductID: "",
				Quantity:  10,
				RequestID: "req-123",
			},
			wantErr: true,
			errMsg:  "product_id is required",
		},
		{
			name: "invalid UUID format",
			req: ReserveStockServiceRequest{
				ProductID: "invalid-uuid",
				Quantity:  10,
				RequestID: "req-123",
			},
			wantErr: true,
			errMsg:  "product_id must be a valid UUID",
		},
		{
			name: "zero quantity",
			req: ReserveStockServiceRequest{
				ProductID: "550e8400-e29b-41d4-a716-446655440000",
				Quantity:  0,
				RequestID: "req-123",
			},
			wantErr: true,
			errMsg:  "quantity must be greater than 0",
		},
		{
			name: "negative quantity",
			req: ReserveStockServiceRequest{
				ProductID: "550e8400-e29b-41d4-a716-446655440000",
				Quantity:  -5,
				RequestID: "req-123",
			},
			wantErr: true,
			errMsg:  "quantity must be greater than 0",
		},
		{
			name: "large quantity warning",
			req: ReserveStockServiceRequest{
				ProductID: "550e8400-e29b-41d4-a716-446655440000",
				Quantity:  1001,
				RequestID: "req-123",
			},
			wantErr: false, // Should have warnings but be valid
		},
		{
			name: "empty request ID",
			req: ReserveStockServiceRequest{
				ProductID: "550e8400-e29b-41d4-a716-446655440000",
				Quantity:  10,
				RequestID: "",
			},
			wantErr: true,
			errMsg:  "request_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.ValidateReserveRequest(tt.req)
			if tt.wantErr {
				assert.False(t, result.Valid)
				if tt.errMsg != "" {
					assert.Contains(t, strings.Join(result.Errors, " "), tt.errMsg)
				}
			} else {
				assert.True(t, result.Valid)
			}
		})
	}
}

func TestValidateReleaseRequest(t *testing.T) {
	service := NewValidationService()

	tests := []struct {
		name    string
		req     ReleaseStockServiceRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			req: ReleaseStockServiceRequest{
				ReservationID: "550e8400-e29b-41d4-a716-446655440000",
				Reason:        "cancelled",
				RequestID:     "req-123",
			},
			wantErr: false,
		},
		{
			name: "empty reservation ID",
			req: ReleaseStockServiceRequest{
				ReservationID: "",
				Reason:        "cancelled",
				RequestID:     "req-123",
			},
			wantErr: true,
			errMsg:  "reservation_id is required",
		},
		{
			name: "invalid UUID format",
			req: ReleaseStockServiceRequest{
				ReservationID: "invalid-uuid",
				Reason:        "cancelled",
				RequestID:     "req-123",
			},
			wantErr: true,
			errMsg:  "reservation_id must be a valid UUID",
		},
		{
			name: "empty reason",
			req: ReleaseStockServiceRequest{
				ReservationID: "550e8400-e29b-41d4-a716-446655440000",
				Reason:        "",
				RequestID:     "req-123",
			},
			wantErr: true,
			errMsg:  "reason is required",
		},
		{
			name: "invalid reason",
			req: ReleaseStockServiceRequest{
				ReservationID: "550e8400-e29b-41d4-a716-446655440000",
				Reason:        "invalid_reason",
				RequestID:     "req-123",
			},
			wantErr: true,
			errMsg:  "reason must be one of",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.ValidateReleaseRequest(tt.req)
			if tt.wantErr {
				assert.False(t, result.Valid)
				if tt.errMsg != "" {
					assert.Contains(t, strings.Join(result.Errors, " "), tt.errMsg)
				}
			} else {
				assert.True(t, result.Valid)
			}
		})
	}
}

func TestValidateUpdateRequest(t *testing.T) {
	service := NewValidationService()

	tests := []struct {
		name    string
		req     UpdateStockServiceRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			req: UpdateStockServiceRequest{
				ProductID:      "550e8400-e29b-41d4-a716-446655440000",
				NewStock:       100,
				AdjustmentType: "restock",
				Reason:         "inventory replenishment",
				RequestID:      "req-123",
			},
			wantErr: false,
		},
		{
			name: "empty product ID",
			req: UpdateStockServiceRequest{
				ProductID:      "",
				NewStock:       100,
				AdjustmentType: "restock",
				Reason:         "inventory replenishment",
				RequestID:      "req-123",
			},
			wantErr: true,
			errMsg:  "product_id is required",
		},
		{
			name: "invalid UUID format",
			req: UpdateStockServiceRequest{
				ProductID:      "invalid-uuid",
				NewStock:       100,
				AdjustmentType: "restock",
				Reason:         "inventory replenishment",
				RequestID:      "req-123",
			},
			wantErr: true,
			errMsg:  "product_id must be a valid UUID",
		},
		{
			name: "negative stock",
			req: UpdateStockServiceRequest{
				ProductID:      "550e8400-e29b-41d4-a716-446655440000",
				NewStock:       -10,
				AdjustmentType: "restock",
				Reason:         "inventory replenishment",
				RequestID:      "req-123",
			},
			wantErr: true,
			errMsg:  "new_stock cannot be negative",
		},
		{
			name: "invalid adjustment type",
			req: UpdateStockServiceRequest{
				ProductID:      "550e8400-e29b-41d4-a716-446655440000",
				NewStock:       100,
				AdjustmentType: "invalid_type",
				Reason:         "inventory replenishment",
				RequestID:      "req-123",
			},
			wantErr: true,
			errMsg:  "adjustment_type must be one of",
		},
		{
			name: "empty reason",
			req: UpdateStockServiceRequest{
				ProductID:      "550e8400-e29b-41d4-a716-446655440000",
				NewStock:       100,
				AdjustmentType: "restock",
				Reason:         "",
				RequestID:      "req-123",
			},
			wantErr: true,
			errMsg:  "reason is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.ValidateUpdateRequest(tt.req)
			if tt.wantErr {
				assert.False(t, result.Valid)
				if tt.errMsg != "" {
					assert.Contains(t, strings.Join(result.Errors, " "), tt.errMsg)
				}
			} else {
				assert.True(t, result.Valid)
			}
		})
	}
}

func TestIsValidUUID(t *testing.T) {
	tests := []struct {
		name     string
		uuid     string
		expected bool
	}{
		{
			name:     "valid UUID v4",
			uuid:     "550e8400-e29b-41d4-a716-446655440000",
			expected: true,
		},
		{
			name:     "valid UUID v1",
			uuid:     "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			expected: true,
		},
		{
			name:     "empty string",
			uuid:     "",
			expected: false,
		},
		{
			name:     "invalid format - no hyphens",
			uuid:     "550e8400e29b41d4a716446655440000",
			expected: false,
		},
		{
			name:     "invalid format - wrong length",
			uuid:     "550e8400-e29b-41d4-a716-44665544000",
			expected: false,
		},
		{
			name:     "invalid format - non-hex characters",
			uuid:     "550e8400-e29b-41d4-a716-44665544000g",
			expected: false,
		},
		{
			name:     "invalid format - wrong number of hyphens",
			uuid:     "550e8400-e29b-41d4-a716446655440000",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.IsValidUUID(tt.uuid)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContains(t *testing.T) {
	slice := []string{"apple", "banana", "cherry"}

	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{
			name:     "item exists",
			slice:    slice,
			item:     "banana",
			expected: true,
		},
		{
			name:     "item does not exist",
			slice:    slice,
			item:     "grape",
			expected: false,
		},
		{
			name:     "empty slice",
			slice:    []string{},
			item:     "apple",
			expected: false,
		},
		{
			name:     "case insensitive match",
			slice:    slice,
			item:     "APPLE",
			expected: true, // contains uses EqualFold
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.Contains(tt.slice, tt.item)
			assert.Equal(t, tt.expected, result)
		})
	}
}
