package repository

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// RepositoryTestSuite provides a test suite for repository operations
type RepositoryTestSuite struct {
	suite.Suite
	db   *sql.DB
	repo InventoryRepository
	ctx  context.Context
}

func (suite *RepositoryTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	// Create in-memory SQLite database for testing
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(suite.T(), err)

	suite.db = db

	// Apply migrations
	err = suite.applyMigrations()
	require.NoError(suite.T(), err)

	// Create repository
	suite.repo = NewInventoryRepository(db)

	// Seed test data
	err = suite.seedTestData()
	require.NoError(suite.T(), err)
}

func (suite *RepositoryTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *RepositoryTestSuite) SetupTest() {
	// Clean up data before each test but keep schema
	_, err := suite.db.Exec("DELETE FROM reservations")
	require.NoError(suite.T(), err)
	_, err = suite.db.Exec("DELETE FROM idempotency_keys")
	require.NoError(suite.T(), err)
	_, err = suite.db.Exec("DELETE FROM inventory_items")
	require.NoError(suite.T(), err)
	_, err = suite.db.Exec("DELETE FROM products")
	require.NoError(suite.T(), err)

	// Re-seed basic test data
	err = suite.seedTestData()
	require.NoError(suite.T(), err)
}

func (suite *RepositoryTestSuite) applyMigrations() error {
	// Read and apply the migration schema
	migrationSQL := `
	CREATE TABLE products (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE inventory_items (
		id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
		product_id TEXT NOT NULL UNIQUE,
		available_stock INTEGER NOT NULL DEFAULT 0,
		reserved_stock INTEGER NOT NULL DEFAULT 0,
		version INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE
	);

	\tCREATE TABLE reservations (\n\t\tid TEXT PRIMARY KEY,\n\t\tproduct_id TEXT NOT NULL,\n\t\tquantity INTEGER NOT NULL,\n\t\trequest_id TEXT NOT NULL UNIQUE,\n\t\tstatus TEXT NOT NULL DEFAULT 'pending',\n\t\texpires_at DATETIME NOT NULL,\n\t\tcreated_at DATETIME DEFAULT CURRENT_TIMESTAMP,\n\t\tupdated_at DATETIME DEFAULT CURRENT_TIMESTAMP,\n\t\tFOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE\n\t);

	CREATE TABLE idempotency_keys (
		request_id TEXT PRIMARY KEY,
		operation_type TEXT NOT NULL,
		response_data BLOB,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL
	);

	CREATE INDEX idx_inventory_product_id ON inventory_items(product_id);
	CREATE INDEX idx_reservations_product_id ON reservations(product_id);
	CREATE INDEX idx_reservations_status ON reservations(status);
	CREATE INDEX idx_reservations_expires_at ON reservations(expires_at);
	CREATE INDEX idx_idempotency_expires_at ON idempotency_keys(expires_at);
	`

	_, err := suite.db.Exec(migrationSQL)
	return err
}

func (suite *RepositoryTestSuite) seedTestData() error {
	// Create test products
	_, err := suite.db.Exec(`
		INSERT INTO products (id, name, description) VALUES 
		('product-1', 'Test Product 1', 'Description for product 1'),
		('product-2', 'Test Product 2', 'Description for product 2')
	`)
	if err != nil {
		return err
	}

	// Create test inventory
	_, err = suite.db.Exec(`
		INSERT INTO inventory_items (id, product_id, available_stock, reserved_stock, version) VALUES 
		('inv-1', 'product-1', 100, 0, 1),
		('inv-2', 'product-2', 50, 10, 1)
	`)
	return err
}

// TestOptimisticLockingSuccess tests successful operations with optimistic locking
func (suite *RepositoryTestSuite) TestOptimisticLockingSuccess() {
	// Get current inventory
	inventory, err := suite.repo.GetInventoryForUpdate(suite.ctx, "product-1")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), int64(100), inventory.AvailableStock)
	require.Equal(suite.T(), int64(1), inventory.Version)

	// Reserve stock with correct version
	req := ReserveStockRequest{
		ProductID: "product-1",
		Quantity:  20,
		Version:   inventory.Version,
		RequestID: "test-request-1",
	}

	updated, err := suite.repo.ReserveStock(suite.ctx, req)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(80), updated.AvailableStock)
	assert.Equal(suite.T(), int64(20), updated.ReservedStock)
	assert.Equal(suite.T(), int64(2), updated.Version) // Version incremented
}

// TestOptimisticLockingVersionConflict tests version conflict handling
func (suite *RepositoryTestSuite) TestOptimisticLockingVersionConflict() {
	// Get current inventory
	inventory, err := suite.repo.GetInventoryForUpdate(suite.ctx, "product-1")
	require.NoError(suite.T(), err)

	// First operation succeeds
	req1 := ReserveStockRequest{
		ProductID: "product-1",
		Quantity:  20,
		Version:   inventory.Version,
		RequestID: "test-request-1",
	}

	_, err = suite.repo.ReserveStock(suite.ctx, req1)
	require.NoError(suite.T(), err)

	// Second operation with same version should fail with version conflict
	req2 := ReserveStockRequest{
		ProductID: "product-1",
		Quantity:  10,
		Version:   inventory.Version, // Old version
		RequestID: "test-request-2",
	}

	_, err = suite.repo.ReserveStock(suite.ctx, req2)
	require.Error(suite.T(), err)
	assert.True(suite.T(), IsVersionConflict(err))

	var repoErr *RepositoryError
	require.ErrorAs(suite.T(), err, &repoErr)
	assert.Equal(suite.T(), "optimistic_update", repoErr.Op)
	assert.Equal(suite.T(), "inventory", repoErr.Entity)
	assert.Equal(suite.T(), "product-1", repoErr.ID)
}

// TestInsufficientStockError tests insufficient stock handling
func (suite *RepositoryTestSuite) TestInsufficientStockError() {
	// Get current inventory
	inventory, err := suite.repo.GetInventoryForUpdate(suite.ctx, "product-1")
	require.NoError(suite.T(), err)

	// Try to reserve more than available
	req := ReserveStockRequest{
		ProductID: "product-1",
		Quantity:  150, // More than the 100 available
		Version:   inventory.Version,
		RequestID: "test-request-1",
	}

	_, err = suite.repo.ReserveStock(suite.ctx, req)
	require.Error(suite.T(), err)
	assert.True(suite.T(), IsInsufficientStock(err))

	var repoErr *RepositoryError
	require.ErrorAs(suite.T(), err, &repoErr)
	assert.Equal(suite.T(), "reserve_stock", repoErr.Op)
	assert.NotNil(suite.T(), repoErr.Context)
	assert.Equal(suite.T(), int64(150), repoErr.Context["requested"])
	assert.Equal(suite.T(), int64(100), repoErr.Context["available"])
}

// TestConcurrentReservations tests concurrent operations
func (suite *RepositoryTestSuite) TestConcurrentReservations() {
	const numGoroutines = 10
	const reserveQuantity = 15

	// Channel to collect results
	results := make(chan error, numGoroutines)
	var wg sync.WaitGroup

	// Launch concurrent reservation attempts
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// Get fresh inventory state
			inventory, err := suite.repo.GetInventoryForUpdate(suite.ctx, "product-1")
			if err != nil {
				results <- err
				return
			}

			req := ReserveStockRequest{
				ProductID: "product-1",
				Quantity:  reserveQuantity,
				Version:   inventory.Version,
				RequestID: fmt.Sprintf("concurrent-request-%d", id),
			}

			_, err = suite.repo.ReserveStock(suite.ctx, req)
			results <- err
		}(i)
	}

	wg.Wait()
	close(results)

	// Collect results
	var successes, versionConflicts, otherErrors int
	for err := range results {
		if err == nil {
			successes++
		} else if IsVersionConflict(err) {
			versionConflicts++
		} else {
			otherErrors++
			suite.T().Logf("Unexpected error: %v", err)
		}
	}

	// We should have some successes and some version conflicts due to concurrent access
	// With 100 stock and 15 per reservation, max 6 can succeed
	assert.True(suite.T(), successes > 0, "Should have some successful reservations")
	assert.True(suite.T(), successes <= 6, "Cannot reserve more than stock allows")
	assert.True(suite.T(), versionConflicts > 0, "Should have version conflicts due to concurrency")
	assert.Equal(suite.T(), 0, otherErrors, "Should not have other types of errors")

	// Verify final state
	final, err := suite.repo.GetInventoryByProduct(suite.ctx, "product-1")
	require.NoError(suite.T(), err)
	expectedReserved := int64(successes * reserveQuantity)
	assert.Equal(suite.T(), expectedReserved, final.ReservedStock)
	assert.Equal(suite.T(), 100-expectedReserved, final.AvailableStock)
}

// TestReservationLifecycle tests complete reservation lifecycle
func (suite *RepositoryTestSuite) TestReservationLifecycle() {
	expiresAt := time.Now().Add(1 * time.Hour)

	// Create reservation
	req := CreateReservationRequest{
		ProductID: "product-1",
		Quantity:  25,
		RequestID: "lifecycle-test",
		ExpiresAt: expiresAt,
	}

	reservation, err := suite.repo.CreateReservation(suite.ctx, req)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "product-1", reservation.ProductID)
	assert.Equal(suite.T(), int64(25), reservation.Quantity)
	assert.Equal(suite.T(), "lifecycle-test", reservation.RequestID)
	assert.Equal(suite.T(), "active", reservation.Status)

	// Get reservation by ID
	retrieved, err := suite.repo.GetReservation(suite.ctx, reservation.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), reservation.ID, retrieved.ID)

	// Get reservation by request ID (idempotency)
	retrievedByRequest, err := suite.repo.GetReservationByRequestID(suite.ctx, req.RequestID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), reservation.ID, retrievedByRequest.ID)

	// Update reservation status
	updated, err := suite.repo.UpdateReservationStatus(suite.ctx, reservation.ID, "consumed")
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "consumed", updated.Status)
}

// TestIdempotencySupport tests idempotency key operations
func (suite *RepositoryTestSuite) TestIdempotencySupport() {
	expiresAt := time.Now().Add(1 * time.Hour)
	responseData := []byte(`{"status": "success", "id": "test-123"}`)

	req := IdempotencyRequest{
		RequestID:     "idempotency-test-1",
		OperationType: "reserve_stock",
		ResponseData:  responseData,
		ExpiresAt:     expiresAt,
	}

	// Store idempotency key
	key, err := suite.repo.StoreIdempotencyKey(suite.ctx, req)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), req.RequestID, key.RequestID)
	assert.Equal(suite.T(), req.OperationType, key.OperationType)
	assert.Equal(suite.T(), string(responseData), key.ResponseData.String)

	// Retrieve idempotency key
	retrieved, err := suite.repo.GetIdempotencyKey(suite.ctx, req.RequestID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), key.RequestID, retrieved.RequestID)
	assert.Equal(suite.T(), string(responseData), retrieved.ResponseData.String)

	// Test non-existent key
	_, err = suite.repo.GetIdempotencyKey(suite.ctx, "non-existent")
	require.Error(suite.T(), err)
	assert.True(suite.T(), IsNotFound(err))
}

// TestTransactionSupport tests transaction operations
func (suite *RepositoryTestSuite) TestTransactionSupport() {
	// Test successful transaction
	err := suite.repo.WithTransaction(suite.ctx, func(repo InventoryRepository) error {
		// Create product in transaction
		product, err := repo.CreateProduct(suite.ctx, CreateProductRequest{
			Name:        "Transactional Product",
			Description: "Created in transaction",
		})
		if err != nil {
			return err
		}

		// Verify product exists within transaction
		retrieved, err := repo.GetProduct(suite.ctx, product.ID)
		if err != nil {
			return err
		}
		assert.Equal(suite.T(), product.ID, retrieved.ID)

		return nil
	})
	require.NoError(suite.T(), err)

	// Test transaction rollback
	var productID string
	err = suite.repo.WithTransaction(suite.ctx, func(repo InventoryRepository) error {
		// Create product
		product, err := repo.CreateProduct(suite.ctx, CreateProductRequest{
			Name:        "Rollback Product",
			Description: "Should be rolled back",
		})
		if err != nil {
			return err
		}
		productID = product.ID

		// Force an error to trigger rollback
		return fmt.Errorf("forced error for rollback test")
	})
	require.Error(suite.T(), err)

	// Product should not exist after rollback
	_, err = suite.repo.GetProduct(suite.ctx, productID)
	require.Error(suite.T(), err)
	assert.True(suite.T(), IsNotFound(err))
}

// TestExpiredReservationCleanup tests cleanup of expired reservations
func (suite *RepositoryTestSuite) TestExpiredReservationCleanup() {
	// Create reservations with different expiration times
	pastTime := time.Now().Add(-1 * time.Hour)
	futureTime := time.Now().Add(1 * time.Hour)

	// Expired reservation
	expiredReq := CreateReservationRequest{
		ProductID: "product-1",
		Quantity:  10,
		RequestID: "expired-reservation",
		ExpiresAt: pastTime,
	}

	expired, err := suite.repo.CreateReservation(suite.ctx, expiredReq)
	require.NoError(suite.T(), err)

	// Active reservation
	activeReq := CreateReservationRequest{
		ProductID: "product-1",
		Quantity:  15,
		RequestID: "active-reservation",
		ExpiresAt: futureTime,
	}

	active, err := suite.repo.CreateReservation(suite.ctx, activeReq)
	require.NoError(suite.T(), err)

	// Run cleanup
	err = suite.repo.CleanupExpiredReservations(suite.ctx, 100)
	require.NoError(suite.T(), err)

	// Check that expired reservation is marked as expired
	expiredAfterCleanup, err := suite.repo.GetReservation(suite.ctx, expired.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "expired", expiredAfterCleanup.Status)

	// Check that active reservation is unchanged
	activeAfterCleanup, err := suite.repo.GetReservation(suite.ctx, active.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "active", activeAfterCleanup.Status)
}

// Test data validation
func (suite *RepositoryTestSuite) TestValidationErrors() {
	inventory, err := suite.repo.GetInventoryForUpdate(suite.ctx, "product-1")
	require.NoError(suite.T(), err)

	// Test negative quantity
	req := ReserveStockRequest{
		ProductID: "product-1",
		Quantity:  -10,
		Version:   inventory.Version,
		RequestID: "negative-test",
	}

	_, err = suite.repo.ReserveStock(suite.ctx, req)
	require.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "invalid quantity")

	// Test zero quantity
	req.Quantity = 0
	_, err = suite.repo.ReserveStock(suite.ctx, req)
	require.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "invalid quantity")
}

// TestReleaseStock tests stock release operations
func (suite *RepositoryTestSuite) TestReleaseStock() {
	// First reserve some stock
	inventory, err := suite.repo.GetInventoryForUpdate(suite.ctx, "product-2")
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(10), inventory.ReservedStock)

	// Release some reserved stock
	releaseReq := ReleaseStockRequest{
		ProductID: "product-2",
		Quantity:  5,
		Version:   inventory.Version,
		RequestID: "release-test",
	}

	updated, err := suite.repo.ReleaseStock(suite.ctx, releaseReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(55), updated.AvailableStock) // 50 + 5
	assert.Equal(suite.T(), int64(5), updated.ReservedStock)   // 10 - 5
	assert.Equal(suite.T(), int64(2), updated.Version)         // Incremented

	// Try to release more than reserved
	releaseReq2 := ReleaseStockRequest{
		ProductID: "product-2",
		Quantity:  10, // More than the 5 reserved
		Version:   updated.Version,
		RequestID: "release-test-2",
	}

	_, err = suite.repo.ReleaseStock(suite.ctx, releaseReq2)
	require.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "insufficient reserved stock")
}

// TestUpdateStock tests stock update operations
func (suite *RepositoryTestSuite) TestUpdateStock() {
	// Get current inventory
	inventory, err := suite.repo.GetInventoryForUpdate(suite.ctx, "product-1")
	require.NoError(suite.T(), err)

	// Update stock
	updateReq := UpdateStockRequest{
		ProductID:  "product-1",
		TotalStock: 200,
		Version:    inventory.Version,
		RequestID:  "update-test",
	}

	updated, err := suite.repo.UpdateStock(suite.ctx, updateReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(200), updated.AvailableStock) // New total since no reserved stock
	assert.Equal(suite.T(), int64(0), updated.ReservedStock)
	assert.Equal(suite.T(), int64(2), updated.Version)
}

// TestProductOperations tests basic product CRUD
func (suite *RepositoryTestSuite) TestProductOperations() {
	// Create product
	createReq := CreateProductRequest{
		Name:        "New Test Product",
		Description: "A new product for testing",
	}

	product, err := suite.repo.CreateProduct(suite.ctx, createReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), createReq.Name, product.Name)
	assert.Equal(suite.T(), createReq.Description, product.Description.String)
	assert.NotEmpty(suite.T(), product.ID)

	// Get product
	retrieved, err := suite.repo.GetProduct(suite.ctx, product.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), product.ID, retrieved.ID)
	assert.Equal(suite.T(), product.Name, retrieved.Name)

	// List products (should include our new one plus the seeded ones)
	products, err := suite.repo.ListProducts(suite.ctx)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(products), 3) // At least our new one + 2 seeded

	// Test not found
	_, err = suite.repo.GetProduct(suite.ctx, "non-existent")
	require.Error(suite.T(), err)
	assert.True(suite.T(), IsNotFound(err))
}

// Run the test suite
func TestRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(RepositoryTestSuite))
}

// Benchmark tests for performance
func BenchmarkReserveStock(b *testing.B) {
	// Setup
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(b, err)
	defer db.Close()

	suite := &RepositoryTestSuite{db: db}
	err = suite.applyMigrations()
	require.NoError(b, err)
	err = suite.seedTestData()
	require.NoError(b, err)

	repo := NewInventoryRepository(db)
	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Get current state
		inventory, err := repo.GetInventoryForUpdate(ctx, "product-1")
		if err != nil {
			b.Fatal(err)
		}

		req := ReserveStockRequest{
			ProductID: "product-1",
			Quantity:  1,
			Version:   inventory.Version,
			RequestID: fmt.Sprintf("benchmark-%d", i),
		}

		_, err = repo.ReserveStock(ctx, req)
		if err != nil && !IsVersionConflict(err) && !IsInsufficientStock(err) {
			b.Fatal(err)
		}
	}
}

// Helper function to run tests
func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()
	os.Exit(code)
}
