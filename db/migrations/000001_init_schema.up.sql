-- ============================================================================
-- INVENTORY MANAGEMENT DATABASE SCHEMA
-- ============================================================================
-- This schema provides a complete inventory management system with:
-- - Product catalog management
-- - Real-time inventory tracking with optimistic locking
-- - Reservation system for stock allocation
-- - Idempotency support for safe retries
-- ============================================================================

-- ============================================================================
-- PRODUCTS TABLE
-- ============================================================================
-- Stores the product catalog with basic product information.
-- 
-- Fields:
--   - id: Unique identifier for the product (UUID)
--   - name: Product name (required)
--   - description: Optional product description
--   - created_at: Timestamp when the product was created
--   - updated_at: Timestamp of last update (auto-updated via trigger)
--
-- Indexes:
--   - Primary key on id
--   - Index on name for search performance
-- ============================================================================
CREATE TABLE IF NOT EXISTS products (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================================
-- INVENTORY ITEMS TABLE
-- ============================================================================
-- Tracks available and reserved stock levels for each product.
-- Uses optimistic locking (version field) to prevent race conditions during
-- concurrent stock updates.
--
-- Fields:
--   - id: Unique identifier for the inventory record (UUID)
--   - product_id: Reference to the product (one-to-one relationship)
--   - available_stock: Current stock available for reservation (≥ 0)
--   - reserved_stock: Stock currently reserved but not yet fulfilled (≥ 0)
--   - version: Optimistic locking version number (incremented on each update)
--   - created_at: Timestamp when the inventory record was created
--   - updated_at: Timestamp of last update (auto-updated via trigger)
--
-- Constraints:
--   - product_id must be unique (one inventory record per product)
--   - available_stock and reserved_stock must be non-negative
--   - Foreign key to products with CASCADE delete
--
-- Note: Total stock = available_stock + reserved_stock
-- ============================================================================
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

-- ============================================================================
-- RESERVATIONS TABLE
-- ============================================================================
-- Tracks stock reservations for products. Supports temporary holds on inventory
-- with expiration and status tracking.
--
-- Fields:
--   - id: Unique identifier for the reservation (UUID)
--   - request_id: Idempotency key for the reservation request (unique)
--   - product_id: Reference to the product being reserved
--   - quantity: Number of units reserved (must be > 0)
--   - status: Current state of the reservation
--   - created_at: Timestamp when the reservation was created
--   - updated_at: Timestamp of last update (auto-updated via trigger)
--   - expires_at: Optional expiration timestamp for auto-release
--
-- Status values:
--   - pending: Reservation created but not yet confirmed
--   - confirmed: Reservation confirmed and stock allocated
--   - released: Reservation released, stock returned to available
--   - expired: Reservation expired without confirmation
--
-- Indexes:
--   - Unique index on request_id for idempotency
--   - Composite index on (product_id, status) for queries
--   - Index on status for bulk operations
-- ============================================================================
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

-- ============================================================================
-- IDEMPOTENCY KEYS TABLE
-- ============================================================================
-- Provides idempotency support for API operations, allowing safe retries
-- without duplicate effects. Stores request results for a limited time.
--
-- Fields:
--   - request_id: Unique identifier for the request (primary key)
--   - operation_type: Type of operation (e.g., 'reserve', 'release', 'add_stock')
--   - response_data: Cached response data (JSON format)
--   - created_at: Timestamp when the key was created
--   - expires_at: Expiration timestamp (for cleanup)
--
-- Usage:
--   1. Client sends request with unique request_id
--   2. Server checks if request_id exists
--   3. If exists, return cached response_data
--   4. If not, process request and store result
--
-- Indexes:
--   - Primary key on request_id for fast lookups
--   - Index on expires_at for efficient cleanup of old records
-- ============================================================================
CREATE TABLE IF NOT EXISTS idempotency_keys (
    request_id TEXT PRIMARY KEY,
    operation_type TEXT NOT NULL,
    response_data TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_inventory_product_id ON inventory_items(product_id);
CREATE INDEX IF NOT EXISTS idx_reservations_request_id ON reservations(request_id);
CREATE INDEX IF NOT EXISTS idx_reservations_product_status ON reservations(product_id, status);
CREATE INDEX IF NOT EXISTS idx_reservations_status ON reservations(status);
CREATE INDEX IF NOT EXISTS idx_idempotency_expires ON idempotency_keys(expires_at);
CREATE INDEX IF NOT EXISTS idx_products_name ON products(name);

-- Trigger to update updated_at on products
CREATE TRIGGER IF NOT EXISTS update_products_timestamp 
AFTER UPDATE ON products
BEGIN
    UPDATE products SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- Trigger to update updated_at on inventory_items
CREATE TRIGGER IF NOT EXISTS update_inventory_items_timestamp 
AFTER UPDATE ON inventory_items
BEGIN
    UPDATE inventory_items SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- Trigger to update updated_at on reservations
CREATE TRIGGER IF NOT EXISTS update_reservations_timestamp 
AFTER UPDATE ON reservations
BEGIN
    UPDATE reservations SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;