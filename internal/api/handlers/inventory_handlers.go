// Package handlers implements HTTP handlers for the inventory API.
//
// This package provides RESTful API endpoints for inventory management
// operations including stock reservations, releases, and updates.
package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/AJPalacios/inventory/internal/domain"
	"github.com/AJPalacios/inventory/pkg/httputil"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// InventoryHandler handles HTTP requests for inventory operations.
//
// This handler provides RESTful endpoints for inventory management
// with proper validation, error handling, and response formatting.
type InventoryHandler struct {
	inventoryService domain.InventoryService
	logger           domain.Logger
}

// NewInventoryHandler creates a new inventory handler instance.
func NewInventoryHandler(
	inventoryService domain.InventoryService,
	logger domain.Logger,
) *InventoryHandler {
	return &InventoryHandler{
		inventoryService: inventoryService,
		logger:           logger,
	}
}

// ReserveStockRequest represents the request body for stock reservation.
type ReserveStockRequest struct {
	ProductID      string            `json:"product_id" binding:"required,uuid"`
	Quantity       int32             `json:"quantity" binding:"required,min=1,max=100000"`
	TimeoutSeconds int32             `json:"timeout_seconds,omitempty" binding:"omitempty,min=60,max=3600"`
	Reason         string            `json:"reason,omitempty" binding:"omitempty,max=500"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	ClientID       string            `json:"client_id,omitempty" binding:"omitempty,min=1,max=100"`
}

// ReleaseStockRequest represents the request body for stock release.
type ReleaseStockRequest struct {
	ReservationID string `json:"reservation_id" binding:"required,uuid"`
	Reason        string `json:"reason" binding:"required,min=3,max=500"`
}

// UpdateStockRequest represents the request body for stock updates.
type UpdateStockRequest struct {
	ProductID      string            `json:"product_id" binding:"required,uuid"`
	NewStock       int32             `json:"new_stock" binding:"required,min=0,max=100000"`
	AdjustmentType string            `json:"adjustment_type" binding:"required,oneof=restock adjustment return correction"`
	Reason         string            `json:"reason" binding:"required,min=3,max=500"`
	Reference      string            `json:"reference,omitempty" binding:"omitempty,max=100"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// BatchReserveStockRequest represents the request body for batch reservations.
type BatchReserveStockRequest struct {
	Requests []ReserveStockRequest `json:"requests" binding:"required,min=1,max=100,dive"`
}

// APIResponse represents a standard API response format.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
	Meta    *APIMeta    `json:"meta,omitempty"`
}

// APIError represents an API error response.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// APIMeta represents API response metadata.
type APIMeta struct {
	RequestID string `json:"request_id"`
	Timestamp string `json:"timestamp"`
	Version   string `json:"version"`
}

// ReserveStock handles POST /api/v1/inventory/reserve
//
// @Summary Reserve stock for a product
// @Description Reserve a specified quantity of stock for a product
// @Tags inventory
// @Accept json
// @Produce json
// @Param request body ReserveStockRequest true "Reserve stock request"
// @Success 200 {object} APIResponse{data=domain.ReservationResult}
// @Failure 400 {object} APIResponse{error=APIError}
// @Failure 409 {object} APIResponse{error=APIError}
// @Failure 500 {object} APIResponse{error=APIError}
// @Router /api/v1/inventory/reserve [post]
func (h *InventoryHandler) ReserveStock(c *gin.Context) {
	var req ReserveStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.handleValidationError(c, err)
		return
	}

	// Generate request ID for idempotency
	requestID := httputil.GetOrGenerateRequestID(c)

	// Convert to service request
	serviceReq := domain.ReserveStockServiceRequest{
		ProductID:      req.ProductID,
		Quantity:       req.Quantity,
		RequestID:      requestID,
		TimeoutSeconds: req.TimeoutSeconds,
		Reason:         req.Reason,
		Metadata:       req.Metadata,
		ClientID:       req.ClientID,
	}

	// Call service
	result, err := h.inventoryService.ReserveStock(c.Request.Context(), serviceReq)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	h.logger.Info("stock reservation successful", map[string]interface{}{
		"product_id":     req.ProductID,
		"quantity":       req.Quantity,
		"reservation_id": result.ReservationID,
		"request_id":     requestID,
	})

	h.sendResponse(c, http.StatusOK, result, requestID)
}

// ReleaseStock handles POST /api/v1/inventory/release
//
// @Summary Release a stock reservation
// @Description Release a previously made stock reservation back to available stock
// @Tags inventory
// @Accept json
// @Produce json
// @Param request body ReleaseStockRequest true "Release stock request"
// @Success 200 {object} APIResponse{data=repository.InventoryItem}
// @Failure 400 {object} APIResponse{error=APIError}
// @Failure 404 {object} APIResponse{error=APIError}
// @Failure 500 {object} APIResponse{error=APIError}
// @Router /api/v1/inventory/release [post]
func (h *InventoryHandler) ReleaseStock(c *gin.Context) {
	var req ReleaseStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.handleValidationError(c, err)
		return
	}

	// Generate request ID for idempotency
	requestID := httputil.GetOrGenerateRequestID(c)

	// Convert to service request
	serviceReq := domain.ReleaseStockServiceRequest{
		ReservationID: req.ReservationID,
		Reason:        req.Reason,
		RequestID:     requestID,
	}

	// Call service
	result, err := h.inventoryService.ReleaseStock(c.Request.Context(), serviceReq)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	h.logger.Info("stock release successful", map[string]interface{}{
		"reservation_id": req.ReservationID,
		"product_id":     result.ProductID,
		"reason":         req.Reason,
		"request_id":     requestID,
	})

	h.sendResponse(c, http.StatusOK, result, requestID)
}

// UpdateStock handles PUT /api/v1/inventory/stock
//
// @Summary Update stock level for a product
// @Description Update the stock level for a product with proper tracking
// @Tags inventory
// @Accept json
// @Produce json
// @Param request body UpdateStockRequest true "Update stock request"
// @Success 200 {object} APIResponse{data=repository.InventoryItem}
// @Failure 400 {object} APIResponse{error=APIError}
// @Failure 404 {object} APIResponse{error=APIError}
// @Failure 500 {object} APIResponse{error=APIError}
// @Router /api/v1/inventory/stock [put]
func (h *InventoryHandler) UpdateStock(c *gin.Context) {
	var req UpdateStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.handleValidationError(c, err)
		return
	}

	// Generate request ID for idempotency
	requestID := httputil.GetOrGenerateRequestID(c)

	// Convert to service request
	serviceReq := domain.UpdateStockServiceRequest{
		ProductID:      req.ProductID,
		NewStock:       req.NewStock,
		AdjustmentType: req.AdjustmentType,
		Reason:         req.Reason,
		Reference:      req.Reference,
		RequestID:      requestID,
		Metadata:       req.Metadata,
	}

	// Call service
	result, err := h.inventoryService.UpdateStock(c.Request.Context(), serviceReq)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	h.logger.Info("stock update successful", map[string]interface{}{
		"product_id":      req.ProductID,
		"new_stock":       req.NewStock,
		"adjustment_type": req.AdjustmentType,
		"request_id":      requestID,
	})

	h.sendResponse(c, http.StatusOK, result, requestID)
}

// GetStock handles GET /api/v1/inventory/stock/:productId
//
// @Summary Get stock information for a product
// @Description Retrieve current stock levels and information for a product
// @Tags inventory
// @Produce json
// @Param productId path string true "Product ID (UUID format)"
// @Success 200 {object} APIResponse{data=domain.StockInfo}
// @Failure 400 {object} APIResponse{error=APIError}
// @Failure 404 {object} APIResponse{error=APIError}
// @Failure 500 {object} APIResponse{error=APIError}
// @Router /api/v1/inventory/stock/{productId} [get]
func (h *InventoryHandler) GetStock(c *gin.Context) {
	productID := c.Param("productId")
	if productID == "" {
		h.handleError(c, http.StatusBadRequest, "MISSING_PRODUCT_ID", "Product ID is required", "")
		return
	}

	// Validate UUID format using Gin's validator
	if _, err := uuid.Parse(productID); err != nil {
		h.handleError(c, http.StatusBadRequest, "INVALID_PRODUCT_ID", "Product ID must be a valid UUID", err.Error())
		return
	}

	requestID := httputil.GetOrGenerateRequestID(c)

	// Call service
	result, err := h.inventoryService.GetAvailableStock(c.Request.Context(), productID)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	h.sendResponse(c, http.StatusOK, result, requestID)
}

// BatchReserveStock handles POST /api/v1/inventory/batch/reserve
//
// @Summary Reserve stock for multiple products in batch
// @Description Reserve stock for multiple products in a single batch operation
// @Tags inventory
// @Accept json
// @Produce json
// @Param request body BatchReserveStockRequest true "Batch reserve stock request"
// @Success 200 {object} APIResponse{data=[]domain.ReservationResult}
// @Failure 400 {object} APIResponse{error=APIError}
// @Failure 409 {object} APIResponse{error=APIError}
// @Failure 500 {object} APIResponse{error=APIError}
// @Router /api/v1/inventory/batch/reserve [post]
func (h *InventoryHandler) BatchReserveStock(c *gin.Context) {
	var req BatchReserveStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.handleValidationError(c, err)
		return
	}

	requestID := httputil.GetOrGenerateRequestID(c)

	// Convert to service requests
	serviceReqs := make([]domain.ReserveStockServiceRequest, len(req.Requests))
	for i, r := range req.Requests {
		serviceReqs[i] = domain.ReserveStockServiceRequest{
			ProductID:      r.ProductID,
			Quantity:       r.Quantity,
			RequestID:      requestID + "_" + strconv.Itoa(i), // Unique per item
			TimeoutSeconds: r.TimeoutSeconds,
			Reason:         r.Reason,
			Metadata:       r.Metadata,
			ClientID:       r.ClientID,
		}
	}

	// Call service
	results, err := h.inventoryService.BatchReserveStock(c.Request.Context(), serviceReqs)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	h.logger.Info("batch stock reservation successful", map[string]interface{}{
		"batch_size": len(req.Requests),
		"results":    len(results),
		"request_id": requestID,
	})

	h.sendResponse(c, http.StatusOK, results, requestID)
}

// Helper methods

// sendResponse sends a successful API response.
func (h *InventoryHandler) sendResponse(c *gin.Context, status int, data interface{}, requestID string) {
	response := APIResponse{
		Success: true,
		Data:    data,
		Meta: &APIMeta{
			RequestID: requestID,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Version:   "v1",
		},
	}
	c.JSON(status, response)
}

// handleError handles API errors with proper formatting.
func (h *InventoryHandler) handleError(c *gin.Context, status int, code, message, details string) {
	requestID := httputil.GetOrGenerateRequestID(c)

	response := APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
		Meta: &APIMeta{
			RequestID: requestID,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Version:   "v1",
		},
	}

	h.logger.Error("API error", nil, map[string]interface{}{
		"status":     status,
		"code":       code,
		"message":    message,
		"details":    details,
		"request_id": requestID,
		"path":       c.Request.URL.Path,
		"method":     c.Request.Method,
	})

	c.JSON(status, response)
}

// handleValidationError handles validation errors from Gin binding.
func (h *InventoryHandler) handleValidationError(c *gin.Context, err error) {
	requestID := httputil.GetOrGenerateRequestID(c)

	// Extract validation errors
	var validationErrors []string
	if validationErrs, ok := err.(validator.ValidationErrors); ok {
		for _, fieldErr := range validationErrs {
			validationErrors = append(validationErrors, h.formatValidationError(fieldErr))
		}
	} else {
		validationErrors = append(validationErrors, err.Error())
	}

	details := strings.Join(validationErrors, "; ")

	response := APIResponse{
		Success: false,
		Error: &APIError{
			Code:    "VALIDATION_ERROR",
			Message: "Request validation failed",
			Details: details,
		},
		Meta: &APIMeta{
			RequestID: requestID,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Version:   "v1",
		},
	}

	h.logger.Warn("Validation error", map[string]interface{}{
		"errors":     validationErrors,
		"request_id": requestID,
		"path":       c.Request.URL.Path,
		"method":     c.Request.Method,
	})

	c.JSON(http.StatusBadRequest, response)
}

// formatValidationError formats a single validation error into a user-friendly message.
func (h *InventoryHandler) formatValidationError(err validator.FieldError) string {
	field := err.Field()
	tag := err.Tag()

	switch tag {
	case "required":
		return fmt.Sprintf("field '%s' is required", field)
	case "uuid":
		return fmt.Sprintf("field '%s' must be a valid UUID", field)
	case "min":
		return fmt.Sprintf("field '%s' must be at least %s", field, err.Param())
	case "max":
		return fmt.Sprintf("field '%s' must be at most %s", field, err.Param())
	case "oneof":
		return fmt.Sprintf("field '%s' must be one of: %s", field, err.Param())
	case "dive":
		return fmt.Sprintf("invalid item in array '%s'", field)
	default:
		return fmt.Sprintf("field '%s' failed validation '%s'", field, tag)
	}
}

// handleServiceError handles errors from the service layer.
func (h *InventoryHandler) handleServiceError(c *gin.Context, err error) {
	// Map domain errors to HTTP status codes
	switch {
	case domain.IsProductNotFoundError(err):
		h.handleError(c, http.StatusNotFound, "PRODUCT_NOT_FOUND", "Product not found", err.Error())
	case domain.IsReservationNotFoundError(err):
		h.handleError(c, http.StatusNotFound, "RESERVATION_NOT_FOUND", "Reservation not found", err.Error())
	case domain.IsInsufficientStockError(err):
		h.handleError(c, http.StatusConflict, "INSUFFICIENT_STOCK", "Insufficient stock available", err.Error())
	case domain.IsReservationExpiredError(err):
		h.handleError(c, http.StatusGone, "RESERVATION_EXPIRED", "Reservation has expired", err.Error())
	case domain.IsBusinessRuleError(err):
		h.handleError(c, http.StatusBadRequest, "BUSINESS_RULE_VIOLATION", "Business rule violation", err.Error())
	case domain.IsServiceUnavailableError(err):
		h.handleError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Service temporarily unavailable", err.Error())
	case domain.IsDuplicateRequestError(err):
		h.handleError(c, http.StatusConflict, "DUPLICATE_REQUEST", "Duplicate request ID", err.Error())
	default:
		h.handleError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", err.Error())
	}
}
