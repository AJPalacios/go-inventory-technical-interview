-- Products queries
-- name: CreateProduct :one
INSERT INTO products (id, name, description)
VALUES (?, ?, ?)
RETURNING *;

-- name: GetProduct :one
SELECT * FROM products
WHERE id = ? LIMIT 1;

-- name: ListProducts :many
SELECT * FROM products
ORDER BY name;

-- Inventory Items queries (adapting to existing schema)
-- name: CreateInventoryItem :one
INSERT INTO inventory_items (id, product_id, available_stock, reserved_stock)
VALUES (?, ?, ?, 0)
RETURNING *;

-- name: GetInventoryItem :one
SELECT * FROM inventory_items
WHERE product_id = ? LIMIT 1;

-- name: GetInventoryItemWithVersion :one
SELECT * FROM inventory_items
WHERE product_id = ? AND version = ? LIMIT 1;

-- Optimistic locking: Update available stock with version check
-- name: UpdateAvailableStockWithVersion :one
UPDATE inventory_items 
SET 
    available_stock = ?,
    version = version + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE 
    product_id = ? 
    AND version = ?
RETURNING *;

-- Reserve stock (decrease available, increase reserved)
-- name: ReserveStockWithVersion :one
UPDATE inventory_items 
SET 
    available_stock = available_stock - ?,
    reserved_stock = reserved_stock + ?,
    version = version + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE 
    product_id = ? 
    AND version = ?
    AND available_stock >= ?
RETURNING *;

-- Release reserved stock (decrease reserved, increase available)
-- name: ReleaseReservedStock :one
UPDATE inventory_items 
SET 
    available_stock = available_stock + ?,
    reserved_stock = reserved_stock - ?,
    version = version + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE 
    product_id = ? 
    AND reserved_stock >= ?
RETURNING *;

-- Convert reservation to sale (decrease reserved, don't change available)
-- name: ConvertReservationToSale :one
UPDATE inventory_items 
SET 
    reserved_stock = reserved_stock - ?,
    version = version + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE 
    product_id = ? 
    AND reserved_stock >= ?
RETURNING *;

-- Reservations queries (adapting to existing schema)
-- name: CreateReservation :one
INSERT INTO reservations (id, product_id, quantity, request_id, status, expires_at)
VALUES (?, ?, ?, ?, 'pending', ?)
RETURNING *;

-- name: GetReservation :one
SELECT * FROM reservations
WHERE id = ? LIMIT 1;

-- name: GetReservationByRequestID :one
SELECT * FROM reservations
WHERE request_id = ? LIMIT 1;

-- name: UpdateReservationStatusById :one
UPDATE reservations
SET 
    status = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: ListActiveReservations :many
SELECT * FROM reservations
WHERE status = 'active'
ORDER BY created_at;

-- name: ListExpiredReservations :many
SELECT * FROM reservations
WHERE status = 'active' AND expires_at < CURRENT_TIMESTAMP
ORDER BY expires_at;

-- name: GetReservationsByProduct :many
SELECT * FROM reservations
WHERE product_id = ? AND status = 'active'
ORDER BY created_at;

-- Idempotency queries (adapting to existing schema)
-- name: CreateIdempotencyKey :one
INSERT INTO idempotency_keys (request_id, operation_type, response_data, expires_at)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetIdempotencyKey :one
SELECT * FROM idempotency_keys
WHERE request_id = ? LIMIT 1;

-- name: DeleteExpiredIdempotencyKeys :exec
DELETE FROM idempotency_keys
WHERE expires_at < CURRENT_TIMESTAMP;

-- Inventory reporting queries (adapting to existing schema)
-- name: GetInventorySummary :one
SELECT 
    COUNT(*) as total_products,
    SUM(available_stock + reserved_stock) as total_stock,
    SUM(reserved_stock) as total_reserved,
    SUM(available_stock) as total_available
FROM inventory_items;

-- name: GetLowStockProducts :many
SELECT 
    p.id, p.name,
    i.available_stock, i.reserved_stock, i.available_stock as available
FROM products p
JOIN inventory_items i ON p.id = i.product_id
WHERE i.available_stock <= ?
ORDER BY available ASC;

-- Transaction support queries
-- name: BeginTransaction :exec
BEGIN TRANSACTION;

-- name: CommitTransaction :exec
COMMIT;

-- name: RollbackTransaction :exec
ROLLBACK;

-- =====================================================
-- ENHANCED OPTIMISTIC LOCKING QUERIES
-- =====================================================

-- Critical: Optimistic stock reservation with comprehensive validation
-- name: ReserveStockOptimistic :one
UPDATE inventory_items 
SET 
    available_stock = available_stock - ?1,
    reserved_stock = reserved_stock + ?1,
    version = version + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE 
    product_id = ?2 
    AND version = ?3
    AND (available_stock - ?1) >= 0  -- Prevent negative available stock
    AND (available_stock >= ?1)      -- Ensure sufficient stock
RETURNING *;

-- Critical: Optimistic stock release with validation
-- name: ReleaseStockOptimistic :one
UPDATE inventory_items 
SET 
    available_stock = available_stock + ?1,
    reserved_stock = reserved_stock - ?1,
    version = version + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE 
    product_id = ?2 
    AND version = ?3
    AND reserved_stock >= ?1  -- Ensure sufficient reserved stock
RETURNING *;

-- Critical: Update total stock with version check
-- name: UpdateStockOptimistic :one
UPDATE inventory_items 
SET 
    available_stock = ?1 - reserved_stock,  -- New available = total - reserved
    version = version + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE 
    product_id = ?2 
    AND version = ?3
    AND ?1 >= reserved_stock  -- Ensure new total >= reserved
RETURNING *;

-- Get inventory with version for optimistic locking
-- name: GetInventoryForUpdate :one
SELECT id, product_id, available_stock, reserved_stock, version, created_at, updated_at
FROM inventory_items
WHERE product_id = ?1
LIMIT 1;

-- Create reservation with timeout and validation
-- name: CreateReservationWithTimeout :one
INSERT INTO reservations (id, product_id, quantity, request_id, status, expires_at, created_at)
VALUES (?1, ?2, ?3, ?4, 'active', ?5, CURRENT_TIMESTAMP)
RETURNING *;

-- Update reservation status with optimistic approach\n-- name: UpdateReservationStatus :one\nUPDATE reservations\nSET \n    status = ?1,\n    updated_at = CURRENT_TIMESTAMP\nWHERE id = ?2 AND status = 'active'  -- Only update active reservations\nRETURNING *;

-- Get active reservations for cleanup
-- name: GetExpiredReservations :many
SELECT id, product_id, quantity, request_id, expires_at, created_at
FROM reservations
WHERE status = 'active' 
AND expires_at < CURRENT_TIMESTAMP
ORDER BY expires_at ASC
LIMIT ?1;

-- Batch release expired reservations
-- name: MarkReservationsExpired :exec
UPDATE reservations
SET
    status = 'expired',
    updated_at = datetime('now')
WHERE status = 'active'
AND datetime(expires_at) < datetime('now');

-- =====================================================
-- IDEMPOTENCY QUERIES
-- =====================================================

-- Store idempotency key with response
-- name: StoreIdempotencyKey :one
INSERT INTO idempotency_keys (request_id, operation_type, response_data, expires_at, created_at)
VALUES (?1, ?2, ?3, ?4, CURRENT_TIMESTAMP)
ON CONFLICT(request_id) DO UPDATE SET
    response_data = excluded.response_data,
    expires_at = excluded.expires_at
RETURNING *;

-- Get idempotency key if not expired
-- name: GetValidIdempotencyKey :one
SELECT request_id, operation_type, response_data, created_at, expires_at
FROM idempotency_keys
WHERE request_id = ?1
AND datetime(expires_at) > datetime('now')
LIMIT 1;

-- Cleanup expired idempotency keys
-- name: CleanupExpiredIdempotencyKeys :exec
DELETE FROM idempotency_keys
WHERE expires_at < CURRENT_TIMESTAMP;