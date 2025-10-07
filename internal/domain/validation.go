package domain

import (
	"github.com/AJPalacios/inventory/pkg/validator"
)

// ValidationService implementation for business rules
type validationService struct{}

// NewValidationService creates a new validation service instance.
func NewValidationService() ValidationService {
	return &validationService{}
}

// ValidateReserveRequest validates reserve stock request according to business rules.
func (v *validationService) ValidateReserveRequest(req ReserveStockServiceRequest) ValidationResult {
	result := ValidationResult{Valid: true}

	// Required field validation
	if req.ProductID == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "product_id is required")
	}

	if req.RequestID == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "request_id is required")
	}

	// Business rule validation
	if req.Quantity <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "quantity must be greater than 0")
	}

	if req.Quantity > 1000 {
		result.Warnings = append(result.Warnings, "large quantity reservation (>1000 units)")
	}

	// UUID format validation
	if !validator.IsValidUUID(req.ProductID) {
		result.Valid = false
		result.Errors = append(result.Errors, "product_id must be a valid UUID")
	}

	// Timeout validation is now handled at the handler layer

	return result
}

// ValidateReleaseRequest validates release stock request according to business rules.
func (v *validationService) ValidateReleaseRequest(req ReleaseStockServiceRequest) ValidationResult {
	result := ValidationResult{Valid: true}

	// Required field validation
	if req.ReservationID == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "reservation_id is required")
	}

	if req.RequestID == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "request_id is required")
	}

	if req.Reason == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "reason is required")
	}

	// UUID format validation
	if !validator.IsValidUUID(req.ReservationID) {
		result.Valid = false
		result.Errors = append(result.Errors, "reservation_id must be a valid UUID")
	}

	// Valid reason validation
	validReasons := []string{"cancelled", "purchased", "timeout", "expired", "admin"}
	if !validator.Contains(validReasons, req.Reason) {
		result.Valid = false
		result.Errors = append(result.Errors, "reason must be one of: cancelled, purchased, timeout, expired, admin")
	}

	return result
}

// ValidateUpdateRequest validates update stock request according to business rules.
func (v *validationService) ValidateUpdateRequest(req UpdateStockServiceRequest) ValidationResult {
	result := ValidationResult{Valid: true}

	// Required field validation
	if req.ProductID == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "product_id is required")
	}

	if req.RequestID == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "request_id is required")
	}

	if req.Reason == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "reason is required")
	}

	// Business rule validation
	if req.NewStock < 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "new_stock cannot be negative")
	}

	if req.NewStock > 100000 {
		result.Warnings = append(result.Warnings, "very large stock quantity (>100,000 units)")
	}

	// UUID format validation
	if !validator.IsValidUUID(req.ProductID) {
		result.Valid = false
		result.Errors = append(result.Errors, "product_id must be a valid UUID")
	}

	// Valid adjustment type validation
	validTypes := []string{"restock", "adjustment", "return", "correction"}
	if !validator.Contains(validTypes, req.AdjustmentType) {
		result.Valid = false
		result.Errors = append(result.Errors, "adjustment_type must be one of: restock, adjustment, return, correction")
	}

	return result
}
