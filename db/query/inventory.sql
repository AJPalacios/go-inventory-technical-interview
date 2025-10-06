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

-- name: UpdateReservationStatus :one
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

-- name: CleanupExpiredIdempotencyKeys :exec
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