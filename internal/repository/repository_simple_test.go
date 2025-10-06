package repository

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimpleRepository(t *testing.T) {
	// Create in-memory database
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create simple tables
	_, err = db.Exec(`CREATE TABLE products (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	require.NoError(t, err)

	_, err = db.Exec(`CREATE TABLE inventory_items (
		id TEXT PRIMARY KEY,
		product_id TEXT NOT NULL UNIQUE,
		available_stock INTEGER NOT NULL DEFAULT 0,
		reserved_stock INTEGER NOT NULL DEFAULT 0,
		version INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	require.NoError(t, err)

	// Create repository
	repo := NewInventoryRepository(db)
	ctx := context.Background()

	// Test product creation
	product, err := repo.CreateProduct(ctx, CreateProductRequest{
		Name:        "Test Product",
		Description: "A test product",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, product.ID)
	assert.Equal(t, "Test Product", product.Name)

	// Test product retrieval
	retrieved, err := repo.GetProduct(ctx, product.ID)
	require.NoError(t, err)
	assert.Equal(t, product.ID, retrieved.ID)
	assert.Equal(t, product.Name, retrieved.Name)

	// Test list products
	products, err := repo.ListProducts(ctx)
	require.NoError(t, err)
	assert.Len(t, products, 1)
	assert.Equal(t, product.ID, products[0].ID)
}

func TestInventoryOperations(t *testing.T) {
	// Create in-memory database
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create tables
	_, err = db.Exec(`CREATE TABLE products (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	require.NoError(t, err)

	_, err = db.Exec(`CREATE TABLE inventory_items (
		id TEXT PRIMARY KEY,
		product_id TEXT NOT NULL UNIQUE,
		available_stock INTEGER NOT NULL DEFAULT 0,
		reserved_stock INTEGER NOT NULL DEFAULT 0,
		version INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	require.NoError(t, err)

	// Seed data
	_, err = db.Exec(`INSERT INTO products (id, name) VALUES ('product-1', 'Test Product')`)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO inventory_items (id, product_id, available_stock, version) 
		VALUES ('inv-1', 'product-1', 100, 1)`)
	require.NoError(t, err)

	// Create repository
	repo := NewInventoryRepository(db)
	ctx := context.Background()

	// Test get inventory
	inventory, err := repo.GetInventoryByProduct(ctx, "product-1")
	require.NoError(t, err)
	assert.Equal(t, int64(100), inventory.AvailableStock)
	assert.Equal(t, int64(0), inventory.ReservedStock)
	assert.Equal(t, int64(1), inventory.Version)

	// Test get inventory for update
	inventoryForUpdate, err := repo.GetInventoryForUpdate(ctx, "product-1")
	require.NoError(t, err)
	assert.Equal(t, inventory.ID, inventoryForUpdate.ID)
	assert.Equal(t, inventory.Version, inventoryForUpdate.Version)
}

func TestReserveStockOptimistic(t *testing.T) {
	// Create in-memory database
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create tables
	_, err = db.Exec(`CREATE TABLE products (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	require.NoError(t, err)

	_, err = db.Exec(`CREATE TABLE inventory_items (
		id TEXT PRIMARY KEY,
		product_id TEXT NOT NULL UNIQUE,
		available_stock INTEGER NOT NULL DEFAULT 0,
		reserved_stock INTEGER NOT NULL DEFAULT 0,
		version INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	require.NoError(t, err)

	// Seed data
	_, err = db.Exec(`INSERT INTO products (id, name) VALUES ('product-1', 'Test Product')`)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO inventory_items (id, product_id, available_stock, version) 
		VALUES ('inv-1', 'product-1', 100, 1)`)
	require.NoError(t, err)

	// Create repository
	repo := NewInventoryRepository(db)
	ctx := context.Background()

	// Get current inventory
	inventory, err := repo.GetInventoryForUpdate(ctx, "product-1")
	require.NoError(t, err)

	// Reserve stock
	req := ReserveStockRequest{
		ProductID: "product-1",
		Quantity:  20,
		Version:   inventory.Version,
		RequestID: "test-request-1",
	}

	updated, err := repo.ReserveStock(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, int64(80), updated.AvailableStock)
	assert.Equal(t, int64(20), updated.ReservedStock)
	assert.Equal(t, int64(2), updated.Version)
}

func TestVersionConflictHandling(t *testing.T) {
	// Create in-memory database
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create tables
	_, err = db.Exec(`CREATE TABLE products (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	require.NoError(t, err)

	_, err = db.Exec(`CREATE TABLE inventory_items (
		id TEXT PRIMARY KEY,
		product_id TEXT NOT NULL UNIQUE,
		available_stock INTEGER NOT NULL DEFAULT 0,
		reserved_stock INTEGER NOT NULL DEFAULT 0,
		version INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	require.NoError(t, err)

	// Seed data
	_, err = db.Exec(`INSERT INTO products (id, name) VALUES ('product-1', 'Test Product')`)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO inventory_items (id, product_id, available_stock, version) 
		VALUES ('inv-1', 'product-1', 100, 1)`)
	require.NoError(t, err)

	// Create repository
	repo := NewInventoryRepository(db)
	ctx := context.Background()

	// Get current inventory
	inventory, err := repo.GetInventoryForUpdate(ctx, "product-1")
	require.NoError(t, err)

	// First reservation succeeds
	req1 := ReserveStockRequest{
		ProductID: "product-1",
		Quantity:  20,
		Version:   inventory.Version,
		RequestID: "test-request-1",
	}

	_, err = repo.ReserveStock(ctx, req1)
	require.NoError(t, err)

	// Second reservation with same version should fail
	req2 := ReserveStockRequest{
		ProductID: "product-1",
		Quantity:  10,
		Version:   inventory.Version, // Old version
		RequestID: "test-request-2",
	}

	_, err = repo.ReserveStock(ctx, req2)
	require.Error(t, err)

	// With retry logic, we expect max retries error, not direct version conflict
	var repoErr *RepositoryError
	require.ErrorAs(t, err, &repoErr)

	// The retry logic will exhaust attempts and return max retries error
	// But it should preserve context from the original version conflict
	assert.Contains(t, err.Error(), "maximum retries exceeded")
	assert.NotNil(t, repoErr.Context)
	assert.Contains(t, repoErr.Context, "attempts")
	assert.Contains(t, repoErr.Context, "last_error")

	// The last error should indicate version conflict
	lastError := repoErr.Context["last_error"].(string)
	assert.Contains(t, lastError, "version conflict")
}
