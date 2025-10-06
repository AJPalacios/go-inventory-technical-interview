-- Drop triggers
DROP TRIGGER IF EXISTS update_reservations_timestamp;
DROP TRIGGER IF EXISTS update_inventory_items_timestamp;
DROP TRIGGER IF EXISTS update_products_timestamp;

-- Drop indexes
DROP INDEX IF EXISTS idx_products_name;
DROP INDEX IF EXISTS idx_idempotency_expires;
DROP INDEX IF EXISTS idx_reservations_status;
DROP INDEX IF EXISTS idx_reservations_product_status;
DROP INDEX IF EXISTS idx_reservations_request_id;
DROP INDEX IF EXISTS idx_inventory_product_id;

-- Drop tables (reverse order due to foreign keys)
DROP TABLE IF EXISTS idempotency_keys;
DROP TABLE IF EXISTS reservations;
DROP TABLE IF EXISTS inventory_items;
DROP TABLE IF EXISTS products;