package domain

import (
	"regexp"
	"strings"
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
	if !isValidUUID(req.ProductID) {
		result.Valid = false
		result.Errors = append(result.Errors, "product_id must be a valid UUID")
	}

	// Timeout validation
	if req.TimeoutSeconds > 0 && req.TimeoutSeconds < 60 {
		result.Warnings = append(result.Warnings, "timeout less than 60 seconds may cause premature expiration")
	}

	if req.TimeoutSeconds > 3600 {
		result.Valid = false
		result.Errors = append(result.Errors, "timeout cannot exceed 1 hour (3600 seconds)")
	}

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
	if !isValidUUID(req.ReservationID) {
		result.Valid = false
		result.Errors = append(result.Errors, "reservation_id must be a valid UUID")
	}

	// Valid reason validation
	validReasons := []string{"cancelled", "purchased", "timeout", "expired", "admin"}
	if !contains(validReasons, req.Reason) {
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
	if !isValidUUID(req.ProductID) {
		result.Valid = false
		result.Errors = append(result.Errors, "product_id must be a valid UUID")
	}

	// Valid adjustment type validation
	validTypes := []string{"restock", "adjustment", "return", "correction"}
	if !contains(validTypes, req.AdjustmentType) {
		result.Valid = false
		result.Errors = append(result.Errors, "adjustment_type must be one of: restock, adjustment, return, correction")
	}

	return result
}

// Helper functions
func isValidUUID(uuid string) bool {
	uuidRegex := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	return uuidRegex.MatchString(uuid)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}
