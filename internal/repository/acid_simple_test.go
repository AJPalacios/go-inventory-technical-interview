package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestACIDAnalysis(t *testing.T) {
	// Test ACID compliance analysis
	analysis := GetACIDCompliance()
	require.Len(t, analysis, 4)

	// Verify all ACID properties are covered
	properties := make(map[string]bool)
	for _, item := range analysis {
		properties[item.Property] = true
		assert.Equal(t, "FULL", item.Compliance)
		assert.NotEmpty(t, item.Implementation)
		assert.NotEmpty(t, item.Mitigations)
	}

	assert.True(t, properties["Atomicity"])
	assert.True(t, properties["Consistency"])
	assert.True(t, properties["Isolation"])
	assert.True(t, properties["Durability"])
}

func TestDeadlockAnalysis(t *testing.T) {
	// Test deadlock prevention analysis
	analysis := GetDeadlockAnalysis()
	require.GreaterOrEqual(t, len(analysis), 3)

	for _, scenario := range analysis {
		assert.NotEmpty(t, scenario.Scenario)
		assert.Contains(t, []string{"LOW", "MEDIUM", "HIGH"}, scenario.Risk)
		assert.NotEmpty(t, scenario.Mitigation)
		assert.NotEmpty(t, scenario.Prevention)
	}
}

func TestAtomicityWithTransactions(t *testing.T) {
	// Simple atomicity test
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
		product_id TEXT NOT NULL,
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
	_, err = db.Exec(`INSERT INTO inventory_items (id, product_id, available_stock, version) VALUES ('inv-1', 'product-1', 100, 1)`)
	require.NoError(t, err)

	repo := NewInventoryRepository(db)
	ctx := context.Background()

	// Get initial state
	initial, err := repo.GetInventoryByProduct(ctx, "product-1")
	require.NoError(t, err)
	initialStock := initial.AvailableStock

	// Test atomicity - transaction that fails should rollback
	err = repo.WithTransaction(ctx, func(txRepo InventoryRepository) error {
		// Reserve stock
		req := ReserveStockRequest{
			ProductID: "product-1",
			Quantity:  10,
			Version:   initial.Version,
			RequestID: "atomicity-test",
		}
		_, err := txRepo.ReserveStock(ctx, req)
		if err != nil {
			return err
		}
		// Force failure
		return fmt.Errorf("forced failure")
	})

	// Should have failed
	require.Error(t, err)
	assert.Contains(t, err.Error(), "forced failure")

	// Stock should be unchanged (atomicity)
	final, err := repo.GetInventoryByProduct(ctx, "product-1")
	require.NoError(t, err)
	assert.Equal(t, initialStock, final.AvailableStock)
	assert.Equal(t, initial.Version, final.Version)
}

func TestConsistencyValidation(t *testing.T) {
	// Simple consistency test
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Setup
	_, err = db.Exec(`CREATE TABLE products (id TEXT PRIMARY KEY, name TEXT NOT NULL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP)`)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE inventory_items (id TEXT PRIMARY KEY, product_id TEXT NOT NULL, available_stock INTEGER NOT NULL DEFAULT 0, reserved_stock INTEGER NOT NULL DEFAULT 0, version INTEGER NOT NULL DEFAULT 1, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO products (id, name) VALUES ('product-1', 'Test')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO inventory_items (id, product_id, available_stock, reserved_stock, version) VALUES ('inv-1', 'product-1', 50, 0, 1)`)
	require.NoError(t, err)

	repo := NewInventoryRepository(db)
	ctx := context.Background()

	inventory, err := repo.GetInventoryByProduct(ctx, "product-1")
	require.NoError(t, err)

	// Test consistency - cannot reserve more than available
	req := ReserveStockRequest{
		ProductID: "product-1",
		Quantity:  100, // More than the 50 available
		Version:   inventory.Version,
		RequestID: "consistency-test",
	}

	_, err = repo.ReserveStock(ctx, req)
	require.Error(t, err)
	assert.True(t, IsInsufficientStock(err))

	// Test consistency - cannot reserve negative quantity
	req.Quantity = -10
	_, err = repo.ReserveStock(ctx, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid quantity")
}

func TestTransactionIsolation(t *testing.T) {
	// Simple isolation level test
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Setup
	_, err = db.Exec(`CREATE TABLE products (id TEXT PRIMARY KEY, name TEXT NOT NULL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP)`)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE inventory_items (id TEXT PRIMARY KEY, product_id TEXT NOT NULL, available_stock INTEGER NOT NULL DEFAULT 0, reserved_stock INTEGER NOT NULL DEFAULT 0, version INTEGER NOT NULL DEFAULT 1, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO products (id, name) VALUES ('product-1', 'Test')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO inventory_items (id, product_id, available_stock, reserved_stock, version) VALUES ('inv-1', 'product-1', 100, 0, 1)`)
	require.NoError(t, err)

	repo := NewInventoryRepository(db)
	ctx := context.Background()

	// Test different isolation levels work
	err = repo.WithTransactionIsolation(ctx, ReadCommitted, func(txRepo InventoryRepository) error {
		_, err := txRepo.GetInventoryByProduct(ctx, "product-1")
		return err
	})
	require.NoError(t, err)

	err = repo.WithTransactionIsolation(ctx, Serializable, func(txRepo InventoryRepository) error {
		_, err := txRepo.GetInventoryByProduct(ctx, "product-1")
		return err
	})
	require.NoError(t, err)
}
