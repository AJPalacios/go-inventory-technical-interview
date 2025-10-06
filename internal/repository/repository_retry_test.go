package repository

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetryBehaviorWithVersionConflict(t *testing.T) {
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

	// Second reservation with same version should fail with retry exhaustion
	// because it will keep retrying the version conflict
	req2 := ReserveStockRequest{
		ProductID: "product-1",
		Quantity:  10,
		Version:   inventory.Version, // Old version
		RequestID: "test-request-2",
	}

	_, err = repo.ReserveStock(ctx, req2)
	require.Error(t, err)

	// The error should indicate max retries exceeded
	var repoErr *RepositoryError
	require.ErrorAs(t, err, &repoErr)

	// Since retry logic is working, we get a max retries error
	// But the underlying cause should be version conflict
	assert.Contains(t, err.Error(), "maximum retries exceeded")
	assert.NotNil(t, repoErr.Context)
	assert.Contains(t, repoErr.Context, "attempts")
	assert.Contains(t, repoErr.Context, "last_error")

	// The last error should contain version conflict information
	lastError := repoErr.Context["last_error"].(string)
	assert.Contains(t, lastError, "version conflict")
}

func TestVersionConflictWithoutRetry(t *testing.T) {
	// This test bypasses retry logic to test direct version conflict detection
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

	// Test direct SQLC call without retry logic
	queries := New(db)
	ctx := context.Background()

	// Update version to 2 directly in database
	_, err = db.Exec(`UPDATE inventory_items SET version = 2 WHERE product_id = 'product-1'`)
	require.NoError(t, err)

	// Try to reserve with old version (1) - should return sql.ErrNoRows
	params := ReserveStockOptimisticParams{
		AvailableStock: 10,
		ProductID:      "product-1",
		Version:        1, // Old version
	}

	_, err = queries.ReserveStockOptimistic(ctx, params)
	assert.Equal(t, sql.ErrNoRows, err)

	// This confirms the SQL query works correctly for version conflicts
}
