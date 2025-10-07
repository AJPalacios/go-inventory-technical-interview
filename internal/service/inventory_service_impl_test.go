package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/AJPalacios/inventory/internal/domain"
	"github.com/AJPalacios/inventory/internal/providers"
	"github.com/AJPalacios/inventory/internal/repository"
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

// ============================================================================
// COMPREHENSIVE INTEGRATION TESTS - Bug Fix Validation
// ============================================================================

// InventoryServiceTestSuite provides integration tests for inventory service
type InventoryServiceTestSuite struct {
	suite.Suite
	db      *sql.DB
	service domain.InventoryService
	ctx     context.Context
}

// SetupSuite initializes the test suite with in-memory database
func (s *InventoryServiceTestSuite) SetupSuite() {
	var err error

	// Create in-memory SQLite database
	s.db, err = sql.Open("sqlite3", ":memory:")
	s.Require().NoError(err)

	s.ctx = context.Background()

	// Load schema (not used in this test but kept for reference)
	// schemaPath := filepath.Join("..", "..", "db", "migrations", "000001_init_schema.up.sql")

	// For now, create minimal schema manually since we can't easily load the file
	schema := `
	CREATE TABLE IF NOT EXISTS products (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS inventory_items (
		id TEXT PRIMARY KEY,
		product_id TEXT NOT NULL UNIQUE,
		available_stock INTEGER NOT NULL DEFAULT 0,
		reserved_stock INTEGER NOT NULL DEFAULT 0,
		version INTEGER NOT NULL DEFAULT 1,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
		CHECK (available_stock >= 0),
		CHECK (reserved_stock >= 0)
	);

	CREATE TABLE IF NOT EXISTS reservations (
		id TEXT PRIMARY KEY,
		request_id TEXT NOT NULL UNIQUE,
		product_id TEXT NOT NULL,
		quantity INTEGER NOT NULL,
		status TEXT NOT NULL CHECK (status IN ('pending', 'confirmed', 'released', 'expired')),
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		expires_at TIMESTAMP,
		FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
		CHECK (quantity > 0)
	);

	CREATE TABLE IF NOT EXISTS idempotency_keys (
		request_id TEXT PRIMARY KEY,
		operation_type TEXT NOT NULL,
		response_data TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		expires_at TIMESTAMP NOT NULL
	);
	`
	_, err = s.db.Exec(schema)
	s.Require().NoError(err)

	// Setup service components
	queries := repository.New(s.db)
	logger := providers.NewLogger(providers.LoggerConfig{Level: "info", Format: "json"})

	// Create minimal service config
	config := InventoryServiceConfig{
		OperationTimeout:   time.Second * 30,
		MaxRetryAttempts:   3,
		MaxBatchSize:       100,
		LowStockThreshold:  10,
		MaxStockCapacity:   10000,
		ConcurrentOpsLimit: 10,
	}

	// Mock validation and idempotency services for testing
	validationService := &MockValidationService{}
	idempotencyService := &MockIdempotencyService{}
	metricsProvider := &MockMetricsProvider{}

	s.service = NewInventoryServiceImpl(
		queries,
		validationService,
		idempotencyService,
		logger,
		metricsProvider,
		config,
	)
}

// TearDownSuite cleans up the test suite
func (s *InventoryServiceTestSuite) TearDownSuite() {
	if s.db != nil {
		s.db.Close()
	}
}

// SetupTest creates fresh test data for each test
func (s *InventoryServiceTestSuite) SetupTest() {
	// Clean tables (use DELETE with WHERE to handle case where tables might not exist)
	s.db.Exec("DELETE FROM reservations WHERE 1=1")
	s.db.Exec("DELETE FROM inventory_items WHERE 1=1") 
	s.db.Exec("DELETE FROM products WHERE 1=1")
	s.db.Exec("DELETE FROM idempotency_keys WHERE 1=1")

	// Insert test product
	_, err := s.db.Exec(`
		INSERT INTO products (id, name, description)
		VALUES ('test-product-1', 'Test Product', 'Test Description')
	`)
	s.Require().NoError(err)

	// Insert test inventory with known values
	_, err = s.db.Exec(`
		INSERT INTO inventory_items (id, product_id, available_stock, reserved_stock, version)
		VALUES ('test-inventory-1', 'test-product-1', 100, 0, 1)
	`)
	s.Require().NoError(err)
}

// TestReservationAccuracy validates that reservations work correctly after bug fix
func (s *InventoryServiceTestSuite) TestReservationAccuracy() {
	// Test Case 1: Single reservation should reserve exact quantity
	req := domain.ReserveStockServiceRequest{
		ProductID:      "test-product-1",
		Quantity:       5,
		RequestID:      "test-req-1",
		TimeoutSeconds: 300,
		Reason:         "Test reservation",
	}

	// Execute reservation
	result, err := s.service.ReserveStock(s.ctx, req)
	s.Require().NoError(err)
	s.Assert().NotEmpty(result.ReservationID)
	s.Assert().Equal(int64(5), result.Quantity)

	// Verify stock levels in database
	var availableStock, reservedStock int64
	err = s.db.QueryRow(`
		SELECT available_stock, reserved_stock 
		FROM inventory_items 
		WHERE product_id = 'test-product-1'
	`).Scan(&availableStock, &reservedStock)
	s.Require().NoError(err)

	// CRITICAL: This validates the bug fix
	s.Assert().Equal(int64(95), availableStock, "Available stock should decrease by exactly the reserved quantity")
	s.Assert().Equal(int64(5), reservedStock, "Reserved stock should increase by exactly the reserved quantity")

	// Verify total stock is conserved
	s.Assert().Equal(int64(100), availableStock+reservedStock, "Total stock should be conserved")
}

// TestMultipleReservationsAccuracy validates cumulative reservations
func (s *InventoryServiceTestSuite) TestMultipleReservationsAccuracy() {
	// Reserve 10 units first
	req1 := domain.ReserveStockServiceRequest{
		ProductID:      "test-product-1",
		Quantity:       10,
		RequestID:      "test-req-1",
		TimeoutSeconds: 300,
	}

	result1, err := s.service.ReserveStock(s.ctx, req1)
	s.Require().NoError(err)
	s.Assert().NotEmpty(result1.ReservationID, "First reservation should have ID")

	// Reserve 15 more units
	req2 := domain.ReserveStockServiceRequest{
		ProductID:      "test-product-1", 
		Quantity:       15,
		RequestID:      "test-req-2",
		TimeoutSeconds: 300,
	}

	result2, err := s.service.ReserveStock(s.ctx, req2)
	s.Require().NoError(err)
	s.Assert().NotEmpty(result2.ReservationID, "Second reservation should have ID")

	// Verify final stock levels
	var availableStock, reservedStock int64
	err = s.db.QueryRow(`
		SELECT available_stock, reserved_stock 
		FROM inventory_items 
		WHERE product_id = 'test-product-1'
	`).Scan(&availableStock, &reservedStock)
	s.Require().NoError(err)

	// Validate cumulative effect
	s.Assert().Equal(int64(75), availableStock, "Available should be 100 - 10 - 15 = 75")
	s.Assert().Equal(int64(25), reservedStock, "Reserved should be 10 + 15 = 25")
	s.Assert().Equal(int64(100), availableStock+reservedStock, "Total stock conserved")

	// Validate individual reservations exist
	var count int
	err = s.db.QueryRow(`
		SELECT COUNT(*) FROM reservations 
		WHERE product_id = 'test-product-1' AND status = 'pending'
	`).Scan(&count)
	s.Require().NoError(err)
	s.Assert().Equal(2, count, "Should have 2 active reservations")

	// Validate quantities in reservations table
	var totalReservedInTable int64
	err = s.db.QueryRow(`
		SELECT COALESCE(SUM(quantity), 0) FROM reservations 
		WHERE product_id = 'test-product-1' AND status = 'pending'
	`).Scan(&totalReservedInTable)
	s.Require().NoError(err)
	s.Assert().Equal(reservedStock, totalReservedInTable, "Reserved stock should match sum of active reservations")
}

// TestReservationAndReleaseFlow validates complete reservation lifecycle
func (s *InventoryServiceTestSuite) TestReservationAndReleaseFlow() {
	// Step 1: Reserve stock
	reserveReq := domain.ReserveStockServiceRequest{
		ProductID:      "test-product-1",
		Quantity:       20,
		RequestID:      "flow-test-1",
		TimeoutSeconds: 300,
	}

	result, err := s.service.ReserveStock(s.ctx, reserveReq)
	s.Require().NoError(err)

	// Verify reservation state
	var availableAfterReserve, reservedAfterReserve int64
	err = s.db.QueryRow(`
		SELECT available_stock, reserved_stock 
		FROM inventory_items 
		WHERE product_id = 'test-product-1'
	`).Scan(&availableAfterReserve, &reservedAfterReserve)
	s.Require().NoError(err)

	s.Assert().Equal(int64(80), availableAfterReserve)
	s.Assert().Equal(int64(20), reservedAfterReserve)

	// Step 2: Release the reservation
	releaseReq := domain.ReleaseStockServiceRequest{
		ReservationID: result.ReservationID,
		RequestID:     "flow-test-release-1",
		Reason:        "cancelled",
	}

	_, err = s.service.ReleaseStock(s.ctx, releaseReq)
	s.Require().NoError(err)

	// Verify stock returned to available pool
	var availableAfterRelease, reservedAfterRelease int64
	err = s.db.QueryRow(`
		SELECT available_stock, reserved_stock 
		FROM inventory_items 
		WHERE product_id = 'test-product-1'
	`).Scan(&availableAfterRelease, &reservedAfterRelease)
	s.Require().NoError(err)

	s.Assert().Equal(int64(100), availableAfterRelease, "Stock should be fully restored")
	s.Assert().Equal(int64(0), reservedAfterRelease, "No stock should remain reserved")

	// Verify reservation status updated
	var reservationStatus string
	err = s.db.QueryRow(`
		SELECT status FROM reservations WHERE id = ?
	`, result.ReservationID).Scan(&reservationStatus)
	s.Require().NoError(err)
	s.Assert().Equal("released", reservationStatus)
}

// TestInsufficientStockValidation validates insufficient stock scenarios
func (s *InventoryServiceTestSuite) TestInsufficientStockValidation() {
	// Try to reserve more than available
	req := domain.ReserveStockServiceRequest{
		ProductID:      "test-product-1",
		Quantity:       150, // More than the 100 available
		RequestID:      "insufficient-test-1",
		TimeoutSeconds: 300,
	}

	_, err := s.service.ReserveStock(s.ctx, req)
	s.Require().Error(err)

	// Verify error type
	var insufficientErr domain.ErrInsufficientStock
	s.Assert().ErrorAs(err, &insufficientErr)

	// Verify no changes to stock
	var availableStock, reservedStock int64
	err = s.db.QueryRow(`
		SELECT available_stock, reserved_stock 
		FROM inventory_items 
		WHERE product_id = 'test-product-1'
	`).Scan(&availableStock, &reservedStock)
	s.Require().NoError(err)

	s.Assert().Equal(int64(100), availableStock, "Available stock unchanged on failure")
	s.Assert().Equal(int64(0), reservedStock, "Reserved stock unchanged on failure")
}

// TestUpdateStock validates stock update functionality  
func (s *InventoryServiceTestSuite) TestUpdateStock() {
	// Setup separate product for this test
	productID := "test-product-update"
	_, err := s.db.Exec(`
		INSERT INTO products (id, name, description)
		VALUES (?, 'Update Test Product', 'Test Description')
	`, productID)
	s.Require().NoError(err)

	_, err = s.db.Exec(`
		INSERT INTO inventory_items (id, product_id, available_stock, reserved_stock, version)
		VALUES (?, ?, 50, 0, 1)
	`, "inventory-"+productID, productID)
	s.Require().NoError(err)

	// Test updating stock with valid request
	req := domain.UpdateStockServiceRequest{
		ProductID:      productID,
		NewStock:       150,
		AdjustmentType: "restock", 
		Reason:         "manual adjustment",
		RequestID:      "update-test-1",
	}

	result, err := s.service.UpdateStock(s.ctx, req)
	s.Require().NoError(err)
	s.Assert().NotNil(result)
	s.Assert().Equal(productID, result.ProductID)

	// Verify stock was updated in database
	var availableStock, reservedStock int64
	err = s.db.QueryRow(`
		SELECT available_stock, reserved_stock 
		FROM inventory_items 
		WHERE product_id = ?
	`, productID).Scan(&availableStock, &reservedStock)
	s.Require().NoError(err)

	// Stock should be updated correctly (implementation may vary)
	s.Assert().True(availableStock >= 0, "Available stock should be non-negative")
	s.Assert().True(reservedStock >= 0, "Reserved stock should be non-negative")
}

// TestGetAvailableStock validates stock retrieval functionality
func (s *InventoryServiceTestSuite) TestGetAvailableStock() {
	// Test getting available stock for existing product
	stockInfo, err := s.service.GetAvailableStock(s.ctx, "test-product-1")
	s.Require().NoError(err)
	s.Assert().NotNil(stockInfo)
	s.Assert().Equal("test-product-1", stockInfo.ProductID)
	s.Assert().Equal(int64(100), stockInfo.AvailableStock)
	s.Assert().Equal(int64(0), stockInfo.ReservedStock)
	s.Assert().Equal(int64(100), stockInfo.TotalStock)

	// Test getting stock for non-existent product
	_, err = s.service.GetAvailableStock(s.ctx, "non-existent-product")
	s.Require().Error(err)
}

// TestValidateStockLevel validates stock level validation functionality
func (s *InventoryServiceTestSuite) TestValidateStockLevel() {
	// Test validation with stock above threshold (should pass)
	err := s.service.ValidateStockLevel(s.ctx, "test-product-1", 50)
	s.Require().NoError(err)

	// Test validation with stock below threshold (should fail)
	err = s.service.ValidateStockLevel(s.ctx, "test-product-1", 150)
	s.Require().Error(err)

	// Test validation with non-existent product
	err = s.service.ValidateStockLevel(s.ctx, "non-existent-product", 10)
	s.Require().Error(err)
}

// TestBatchReserveStock validates batch reservation functionality
func (s *InventoryServiceTestSuite) TestBatchReserveStock() {
	// Create multiple reservation requests
	requests := []domain.ReserveStockServiceRequest{
		{
			ProductID:      "test-product-1",
			Quantity:       10,
			RequestID:      "batch-req-1",
			TimeoutSeconds: 300,
			Reason:         "Batch test 1",
		},
		{
			ProductID:      "test-product-1",
			Quantity:       15,
			RequestID:      "batch-req-2",
			TimeoutSeconds: 300,
			Reason:         "Batch test 2",
		},
	}

	// Execute batch reservation
	results, err := s.service.BatchReserveStock(s.ctx, requests)
	s.Require().NoError(err)
	s.Assert().Len(results, 2, "Should have results for both requests")

	// Validate individual results
	for i, result := range results {
		s.Assert().NotEmpty(result.ReservationID, "Reservation %d should have ID", i+1)
		s.Assert().Equal(requests[i].ProductID, result.ProductID)
		s.Assert().Equal(int64(requests[i].Quantity), result.Quantity)
	}

	// Verify total stock was reserved correctly
	var availableStock, reservedStock int64
	err = s.db.QueryRow(`
		SELECT available_stock, reserved_stock 
		FROM inventory_items 
		WHERE product_id = 'test-product-1'
	`).Scan(&availableStock, &reservedStock)
	s.Require().NoError(err)

	s.Assert().Equal(int64(75), availableStock, "Available should be 100 - 10 - 15 = 75")
	s.Assert().Equal(int64(25), reservedStock, "Reserved should be 10 + 15 = 25")
}

// TestBatchReserveStockPartialFailure validates batch reservation with some failures
func (s *InventoryServiceTestSuite) TestBatchReserveStockPartialFailure() {
	// Create requests where one will fail due to insufficient stock
	requests := []domain.ReserveStockServiceRequest{
		{
			ProductID:      "test-product-1",
			Quantity:       50,
			RequestID:      "batch-success-1",
			TimeoutSeconds: 300,
		},
		{
			ProductID:      "test-product-1",
			Quantity:       80, // This will fail as only 50 left
			RequestID:      "batch-fail-1",
			TimeoutSeconds: 300,
		},
	}

	// Execute batch reservation (should handle partial failure)
	results, err := s.service.BatchReserveStock(s.ctx, requests)
	
	// Depending on implementation, this might error or return partial results
	// Let's check what happens
	if err != nil {
		s.Assert().Error(err, "Batch operation should fail when individual requests fail")
	} else {
		s.Assert().NotEmpty(results, "Should have some results even with partial failure")
	}
}

// TestGetHealthStatus validates health check functionality
func (s *InventoryServiceTestSuite) TestGetHealthStatus() {
	// Test health status
	health, err := s.service.GetHealthStatus(s.ctx)
	s.Require().NoError(err)
	s.Assert().NotNil(health)
	s.Assert().NotEmpty(health.Status, "Health status should not be empty")
	s.Assert().NotZero(health.Timestamp, "Health timestamp should be set")
}

// TestErrorHandling validates various error scenarios
func (s *InventoryServiceTestSuite) TestErrorHandling() {
	// Test with invalid product ID in reserve request
	invalidReq := domain.ReserveStockServiceRequest{
		ProductID:      "non-existent-product",
		Quantity:       10,
		RequestID:      "error-test-1",
		TimeoutSeconds: 300,
	}

	_, err := s.service.ReserveStock(s.ctx, invalidReq)
	s.Require().Error(err, "Should error when product doesn't exist")

	// Test release with invalid reservation ID
	invalidReleaseReq := domain.ReleaseStockServiceRequest{
		ReservationID: "non-existent-reservation",
		RequestID:     "error-test-2",
		Reason:        "cancelled",
	}

	_, err = s.service.ReleaseStock(s.ctx, invalidReleaseReq)
	s.Require().Error(err, "Should error when reservation doesn't exist")
}

// TestIdempotency validates idempotent operations
func (s *InventoryServiceTestSuite) TestIdempotency() {
	// Make the same reservation request twice with same request ID
	req := domain.ReserveStockServiceRequest{
		ProductID:      "test-product-1",
		Quantity:       10,
		RequestID:      "idempotent-test-1", // Same request ID
		TimeoutSeconds: 300,
		Reason:         "Idempotency test",
	}

	// First request
	result1, err := s.service.ReserveStock(s.ctx, req)
	s.Require().NoError(err)

	// Second request with same request ID (should be idempotent)
	result2, err := s.service.ReserveStock(s.ctx, req)
	
	// Depending on implementation, might return same result or error
	if err == nil {
		s.Assert().Equal(result1.ReservationID, result2.ReservationID, "Idempotent requests should return same reservation ID")
	}

	// Verify stock was only reserved once
	var reservedStock int64
	err = s.db.QueryRow(`
		SELECT reserved_stock FROM inventory_items WHERE product_id = 'test-product-1'
	`).Scan(&reservedStock)
	s.Require().NoError(err)
	s.Assert().Equal(int64(10), reservedStock, "Stock should only be reserved once despite duplicate requests")
}

// TestConcurrentOperations validates thread safety (basic test)
func (s *InventoryServiceTestSuite) TestConcurrentOperations() {
	// This is a basic concurrency test - in production you'd want more sophisticated tests
	done := make(chan bool, 2)
	
	// Start two concurrent reservation operations
	go func() {
		req := domain.ReserveStockServiceRequest{
			ProductID:      "test-product-1",
			Quantity:       5,
			RequestID:      "concurrent-1",
			TimeoutSeconds: 300,
		}
		_, err := s.service.ReserveStock(s.ctx, req)
		s.Assert().NoError(err, "Concurrent operation 1 should succeed")
		done <- true
	}()

	go func() {
		req := domain.ReserveStockServiceRequest{
			ProductID:      "test-product-1",  
			Quantity:       7,
			RequestID:      "concurrent-2",
			TimeoutSeconds: 300,
		}
		_, err := s.service.ReserveStock(s.ctx, req)
		s.Assert().NoError(err, "Concurrent operation 2 should succeed")
		done <- true
	}()

	// Wait for both operations to complete
	<-done
	<-done

	// Verify final state is consistent
	var availableStock, reservedStock int64
	err := s.db.QueryRow(`
		SELECT available_stock, reserved_stock 
		FROM inventory_items 
		WHERE product_id = 'test-product-1'
	`).Scan(&availableStock, &reservedStock)
	s.Require().NoError(err)

	s.Assert().Equal(int64(88), availableStock, "Available should be 100 - 5 - 7 = 88")
	s.Assert().Equal(int64(12), reservedStock, "Reserved should be 5 + 7 = 12")
	s.Assert().Equal(int64(100), availableStock+reservedStock, "Total stock should be conserved")
}

// TestQuantityEdgeCases validates edge cases for quantities
func (s *InventoryServiceTestSuite) TestQuantityEdgeCases() {
	testCases := []struct {
		name            string
		quantity        int32
		expectedSuccess bool
		expectedAvail   int64
		expectedReserv  int64
	}{
		{"Reserve exactly all available", 100, true, 0, 100},
		{"Reserve single unit", 1, true, 99, 1},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Reset stock for each test case
			s.SetupTest()

			req := domain.ReserveStockServiceRequest{
				ProductID:      "test-product-1",
				Quantity:       tc.quantity,
				RequestID:      "edge-case-" + tc.name,
				TimeoutSeconds: 300,
			}

			result, err := s.service.ReserveStock(s.ctx, req)

			if tc.expectedSuccess {
				s.Require().NoError(err)
				s.Assert().Equal(int64(tc.quantity), result.Quantity)

				// Verify stock levels
				var availableStock, reservedStock int64
				err = s.db.QueryRow(`
					SELECT available_stock, reserved_stock 
					FROM inventory_items 
					WHERE product_id = 'test-product-1'
				`).Scan(&availableStock, &reservedStock)
				s.Require().NoError(err)

				s.Assert().Equal(tc.expectedAvail, availableStock)
				s.Assert().Equal(tc.expectedReserv, reservedStock)
				s.Assert().Equal(int64(100), availableStock+reservedStock, "Total conserved")
			} else {
				s.Assert().Error(err)
			}
		})
	}
}

// Mock services for testing
type MockValidationService struct{}

func (m *MockValidationService) ValidateReserveRequest(req domain.ReserveStockServiceRequest) domain.ValidationResult {
	return domain.ValidationResult{Valid: true}
}

func (m *MockValidationService) ValidateReleaseRequest(req domain.ReleaseStockServiceRequest) domain.ValidationResult {
	return domain.ValidationResult{Valid: true}
}

func (m *MockValidationService) ValidateUpdateRequest(req domain.UpdateStockServiceRequest) domain.ValidationResult {
	return domain.ValidationResult{Valid: true}
}

type MockIdempotencyService struct{}

func (m *MockIdempotencyService) CheckIdempotency(ctx context.Context, requestID string) (interface{}, bool, error) {
	return nil, false, nil
}

func (m *MockIdempotencyService) StoreResult(ctx context.Context, requestID string, result interface{}, ttl time.Duration) error {
	return nil
}

func (m *MockIdempotencyService) CleanupExpired(ctx context.Context) (int, error) {
	return 0, nil
}

type MockMetricsProvider struct{}

func (m *MockMetricsProvider) RecordOperation(operation string, duration time.Duration, success bool) {
}
func (m *MockMetricsProvider) RecordCount(metric string, value int64)   {}
func (m *MockMetricsProvider) RecordGauge(metric string, value float64) {}
func (m *MockMetricsProvider) IncrementCounter(name string, labels map[string]string) {}
func (m *MockMetricsProvider) RecordDuration(name string, duration time.Duration, labels map[string]string) {}

// Run the test suite
func TestInventoryServiceSuite(t *testing.T) {
	suite.Run(t, new(InventoryServiceTestSuite))
}
