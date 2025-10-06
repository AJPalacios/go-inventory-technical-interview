# Arquitectura Distribuida - Sistema de Inventario

## PROBLEMA A RESOLVER
- ❌ **Inconsistencias de stock** entre usuarios concurrentes
- ❌ **Alta latencia** en actualizaciones de inventario  
- ❌ **Race conditions** en reservas simultáneas
- ❌ **Sistema legacy monolítico** sin escalabilidad

## SOLUCIÓN PROPUESTA
✅ **Arquitectura distribuida** con implementación monolítica inicial  
✅ **Optimistic locking** para concurrencia sin bloqueos  
✅ **Idempotencia** para operaciones seguras  
✅ **Migration path** SQLite → PostgreSQL  

---

## 1. DIAGRAMA ASCII - 3 CAPAS ARQUITECTÓNICAS

```
┌───────────────────────────────────────────────────────────────────┐
│                          🌐 API LAYER                             │
│  ┌─────────────────┐ ┌─────────────────┐ ┌────────────────────────┐│
│  │   Gin Router    │ │   Middlewares   │ │      Handlers          ││
│  │                 │ │                 │ │                        ││
│  │ GET  /health    │ │ • Logger        │ │ • ReserveHandler       ││ 
│  │ POST /reserve   │ │ • Recovery      │ │ • ReleaseHandler       ││
│  │ POST /release   │ │ • CORS          │ │ • GetStockHandler      ││
│  │ GET  /:id       │ │ • Idempotency   │ │ • UpdateStockHandler   ││
│  │ PUT  /:id/stock │ │ • Rate Limiting │ │ • HealthHandler        ││
│  └─────────────────┘ └─────────────────┘ └────────────────────────┘│
└───────────────────────────────────────────────────────────────────┘
                                    │
                                    │ HTTP/JSON
                                    ▼
┌───────────────────────────────────────────────────────────────────┐
│                        🔧 SERVICE LAYER                           │
│  ┌─────────────────┐ ┌─────────────────┐ ┌────────────────────────┐│
│  │InventoryService │ │ReservationSvc   │ │   IdempotencyService   ││
│  │                 │ │                 │ │                        ││
│  │ • GetStock()    │ │ • Reserve()     │ │ • CheckKey()           ││
│  │ • UpdateStock() │ │ • Release()     │ │ • StoreResult()        ││
│  │ • Validate()    │ │ • Cleanup()     │ │ • CleanupExpired()     ││
│  │ • Retry Logic   │ │ • Timeout Mgmt  │ │ • Hash Generation      ││
│  └─────────────────┘ └─────────────────┘ └────────────────────────┘│
└───────────────────────────────────────────────────────────────────┘
                                    │
                                    │ Business Logic
                                    ▼
┌───────────────────────────────────────────────────────────────────┐
│                      🗄️  REPOSITORY LAYER                         │
│  ┌─────────────────┐ ┌─────────────────┐ ┌────────────────────────┐│
│  │  SQLC Queries   │ │  Transactions   │ │    Connection Pool     ││
│  │                 │ │                 │ │                        ││
│  │ • GetInventory  │ │ • BeginTx()     │ │ • Health Checks        ││
│  │ • UpdateWithVer │ │ • Commit()      │ │ • Connection Reuse     ││
│  │ • CreateReserv  │ │ • Rollback()    │ │ • Query Timeout        ││
│  │ • OptimisticUpd │ │ • Isolation     │ │ • Migration Support    ││
│  └─────────────────┘ └─────────────────┘ └────────────────────────┘│
└───────────────────────────────────────────────────────────────────┘
                                    │
                                    │ SQL Queries
                                    ▼
┌───────────────────────────────────────────────────────────────────┐
│                         💾 DATABASE                               │
│                                                                   │
│    SQLite (Desarrollo) ────────────────► PostgreSQL (Producción) │
│                                                                   │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ 📊 Tables & Indexes                                         │ │
│  │                                                             │ │
│  │ • products              (id, name, sku)                    │ │
│  │ • inventory_items       (id, product_id, stock, version)   │ │
│  │ • reservations          (id, product_id, qty, request_id)  │ │
│  │ • idempotency_keys      (key_hash, response, expires_at)   │ │
│  │                                                             │ │
│  │ 🔍 Critical Indexes:                                        │ │
│  │ • idx_inventory_product_id (UNIQUE)                        │ │
│  │ • idx_reservations_request_id                              │ │  
│  │ • idx_idempotency_expires                                  │ │
│  └─────────────────────────────────────────────────────────────┘ │
└───────────────────────────────────────────────────────────────────┘
```

---

## 2. CAP THEOREM DECISION: CONSISTENCIA > DISPONIBILIDAD

### 🎯 **DECISIÓN ARQUITECTÓNICA**

Para un sistema de inventario, elegimos **CONSISTENCIA** sobre **DISPONIBILIDAD**.

### 📊 **JUSTIFICACIÓN PARA INVENTARIO**

| Factor | Impacto Business | Decisión |
|--------|------------------|----------|
| **Overselling** | 🔴 Crítico - Pérdidas directas | **Consistencia** |
| **Stock Accuracy** | 🔴 Crítico - Confianza cliente | **Consistencia** |
| **Financial Loss** | 🔴 Alto - Ventas sin inventario | **Consistencia** |
| **User Experience** | 🟡 Medio - Latencia adicional | **Disponibilidad** |

### ⚖️ **TRADE-OFFS ANÁLISIS**

```
CONSISTENCIA (Elegido)              vs.    DISPONIBILIDAD (Rechazado)
┌─────────────────────────────┐           ┌──────────────────────────┐
│ ✅ BENEFICIOS               │           │ ❌ COSTOS               │
│ • No overselling           │           │ • Latencia +50-100ms    │
│ • Stock siempre correcto   │           │ • Posibles timeouts     │ 
│ • Integridad ACID          │           │ • Menor throughput      │
│ • Audit trail completo    │           │ • Single point failure  │
└─────────────────────────────┘           └──────────────────────────┘

DISPONIBILIDAD (Rechazado)          vs.    CONSISTENCIA (Elegido)
┌─────────────────────────────┐           ┌──────────────────────────┐
│ ❌ RIESGOS                  │           │ ✅ MITIGACIONES         │
│ • Eventual consistency     │           │ • Read replicas cache   │
│ • Race conditions          │           │ • Retry con backoff     │
│ • Data conflicts           │           │ • Circuit breaker       │
│ • Overselling crítico      │           │ • Graceful degradation  │
└─────────────────────────────┘           └──────────────────────────┘
```

### 🔄 **ESTRATEGIA HÍBRIDA**
- **Escrituras**: Fuerte consistencia (ACID transacciones)
- **Lecturas**: Eventual consistency (cache + replica)
- **Reservas**: Optimistic locking con retry
- **Fallback**: Modo degradado con alertas

---

## 3. CONCURRENCY: OPTIMISTIC LOCKING CON VERSION FIELD

### 🤔 **¿POR QUÉ OPTIMISTIC vs PESSIMISTIC?**

| Aspecto | Optimistic Locking ✅ | Pessimistic Locking ❌ |
|---------|----------------------|------------------------|
| **Throughput** | Alto - Sin bloqueos de lectura | Bajo - Bloquea operaciones |
| **Deadlocks** | Imposible - No hay locks | Propenso - Múltiples locks |
| **Scalability** | Excelente - Concurrencia alta | Limitada - Bottleneck |
| **Complexity** | Media - Retry en aplicación | Baja - DB maneja locks |
| **Performance** | Rápida en baja contención | Lenta con alta contención |

### 💡 **IMPLEMENTACIÓN CON SQLC**

#### **Query de Actualización Optimista**
```sql
-- name: UpdateInventoryOptimistic :one
UPDATE inventory_items 
SET 
    stock = stock - $1,
    reserved = reserved + $1,
    version = version + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE 
    product_id = $2 
    AND version = $3
    AND (stock - $1) >= 0  -- Prevenir stock negativo
RETURNING id, product_id, stock, reserved, version, updated_at;
```

#### **Query de Reserva con Version Check**
```sql
-- name: CreateReservationWithCheck :one
INSERT INTO reservations (
    id, product_id, quantity, request_id, expires_at, created_at
) VALUES (
    $1, $2, $3, $4, $5, CURRENT_TIMESTAMP
) 
RETURNING *;
```

#### **Flujo de Concurrencia**
```go
func (s *InventoryService) ReserveStock(ctx context.Context, req ReserveRequest) error {
    maxRetries := 3
    backoff := 100 * time.Millisecond
    
    for attempt := 0; attempt < maxRetries; attempt++ {
        // 1. Leer estado actual + version
        item, err := s.repo.GetInventoryByProduct(ctx, req.ProductID)
        if err != nil {
            return err
        }
        
        // 2. Validar disponibilidad
        if item.Stock < req.Quantity {
            return ErrInsufficientStock
        }
        
        // 3. Intentar update optimista
        updated, err := s.repo.UpdateInventoryOptimistic(ctx, 
            req.Quantity, req.ProductID, item.Version)
        
        if err == nil {
            // 4. Éxito - crear reserva
            return s.repo.CreateReservation(ctx, reservation)
        }
        
        if errors.Is(err, ErrVersionConflict) {
            // 5. Retry con exponential backoff
            time.Sleep(backoff * time.Duration(attempt+1))
            continue
        }
        
        return err // Error no recoverable
    }
    
    return ErrMaxRetriesExceeded
}
```

---

## 4. API ENDPOINTS SPECIFICATION

### 🔗 **CORE INVENTORY OPERATIONS**

#### **1. Reservar Stock**
```http
POST /api/v1/inventory/reserve
Content-Type: application/json
Idempotency-Key: uuid-v4

{
    "product_id": "550e8400-e29b-41d4-a716-446655440000",
    "quantity": 5,
    "request_id": "req_123456789",
    "timeout_seconds": 300,
    "reason": "order_checkout"
}
```

**Response 201 Created:**
```json
{
    "reservation_id": "res_abcd1234",
    "product_id": "550e8400-e29b-41d4-a716-446655440000",
    "quantity": 5,
    "status": "reserved",
    "expires_at": "2025-10-06T15:30:00Z",
    "created_at": "2025-10-06T15:25:00Z"
}
```

#### **2. Liberar Reserva**
```http
POST /api/v1/inventory/release
Content-Type: application/json
Idempotency-Key: uuid-v4

{
    "reservation_id": "res_abcd1234",
    "reason": "cancelled|purchased|timeout|expired"
}
```

**Response 200 OK:**
```json
{
    "reservation_id": "res_abcd1234",
    "status": "released", 
    "reason": "purchased",
    "released_at": "2025-10-06T15:28:00Z",
    "quantity_released": 5
}
```

#### **3. Consultar Stock**
```http
GET /api/v1/inventory/:id
```

**Response 200 OK:**
```json
{
    "product_id": "550e8400-e29b-41d4-a716-446655440000",
    "stock": 150,
    "reserved": 25,
    "available": 125,
    "version": 42,
    "last_updated": "2025-10-06T15:20:00Z",
    "reservations_count": 8
}
```

#### **4. Actualizar Stock**
```http
PUT /api/v1/inventory/:id/stock
Content-Type: application/json
Idempotency-Key: uuid-v4

{
    "stock": 200,
    "adjustment_type": "restock|adjustment|return|correction",
    "reason": "Weekly inventory restock",
    "reference": "PO-2025-001"
}
```

**Response 200 OK:**
```json
{
    "product_id": "550e8400-e29b-41d4-a716-446655440000",
    "previous_stock": 150,
    "new_stock": 200,
    "adjustment": +50,
    "version": 43,
    "updated_at": "2025-10-06T15:30:00Z"
}
```

#### **5. Health Check**
```http
GET /health
```

**Response 200 OK:**
```json
{
    "status": "healthy",
    "timestamp": "2025-10-06T15:30:00Z",
    "version": "1.0.0",
    "uptime": "2h30m15s",
    "checks": {
        "database": {
            "status": "connected",
            "latency": "2ms",
            "connections": "5/10"
        },
        "inventory": {
            "status": "operational",
            "active_reservations": 156,
            "cleanup_last_run": "2025-10-06T15:25:00Z"
        }
    }
}
```

---

## 5. DATABASE SCHEMA DESIGN

### 📋 **PRODUCTS TABLE**
```sql
CREATE TABLE products (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    sku TEXT UNIQUE NOT NULL,
    description TEXT,
    category TEXT,
    price_cents INTEGER,
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes para Products
CREATE INDEX idx_products_sku ON products(sku);
CREATE INDEX idx_products_active ON products(active) WHERE active = TRUE;
CREATE INDEX idx_products_category ON products(category);
```

### 📦 **INVENTORY_ITEMS TABLE (Con Version)**
```sql
CREATE TABLE inventory_items (
    id TEXT PRIMARY KEY,
    product_id TEXT NOT NULL REFERENCES products(id),
    stock INTEGER NOT NULL DEFAULT 0 CHECK (stock >= 0),
    reserved INTEGER NOT NULL DEFAULT 0 CHECK (reserved >= 0),
    version INTEGER NOT NULL DEFAULT 1,
    min_threshold INTEGER DEFAULT 10,
    max_capacity INTEGER DEFAULT 1000,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraint crítico: stock disponible >= reservado
    CONSTRAINT check_available_stock CHECK (stock >= reserved),
    
    -- Un producto = una entrada de inventario
    CONSTRAINT unique_product_inventory UNIQUE(product_id)
);

-- Indexes críticos para Inventory
CREATE UNIQUE INDEX idx_inventory_product_id ON inventory_items(product_id);
CREATE INDEX idx_inventory_version ON inventory_items(version);
CREATE INDEX idx_inventory_low_stock ON inventory_items(stock) 
    WHERE stock <= min_threshold;
CREATE INDEX idx_inventory_available ON inventory_items(stock - reserved) 
    WHERE (stock - reserved) > 0;
```

### 🎫 **RESERVATIONS TABLE (Con Request ID)**
```sql
CREATE TABLE reservations (
    id TEXT PRIMARY KEY,
    product_id TEXT NOT NULL REFERENCES products(id),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    status TEXT NOT NULL DEFAULT 'active' 
        CHECK (status IN ('active', 'released', 'expired', 'consumed')),
    request_id TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    released_at TIMESTAMP NULL,
    release_reason TEXT NULL CHECK (
        release_reason IN ('cancelled', 'purchased', 'timeout', 'expired', 'admin')
    ),
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes para Reservations
CREATE INDEX idx_reservations_product_id ON reservations(product_id);
CREATE INDEX idx_reservations_request_id ON reservations(request_id);
CREATE INDEX idx_reservations_active ON reservations(status, expires_at) 
    WHERE status = 'active';
CREATE INDEX idx_reservations_cleanup ON reservations(expires_at, status) 
    WHERE status = 'active' AND expires_at < CURRENT_TIMESTAMP;
```

### 🔑 **IDEMPOTENCY_KEYS TABLE**
```sql
CREATE TABLE idempotency_keys (
    key_hash TEXT PRIMARY KEY,
    endpoint TEXT NOT NULL,
    method TEXT NOT NULL,
    request_body_hash TEXT NOT NULL,
    response_status INTEGER NOT NULL,
    response_headers TEXT,
    response_body TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    
    -- TTL para cleanup automático
    CHECK (expires_at > created_at)
);

-- Indexes para Idempotency
CREATE INDEX idx_idempotency_expires ON idempotency_keys(expires_at);
CREATE INDEX idx_idempotency_endpoint ON idempotency_keys(endpoint, method);
CREATE INDEX idx_idempotency_cleanup ON idempotency_keys(expires_at) 
    WHERE expires_at < CURRENT_TIMESTAMP;
```

### 📊 **AUDIT_LOG TABLE (Opcional)**
```sql
CREATE TABLE audit_log (
    id TEXT PRIMARY KEY,
    table_name TEXT NOT NULL,
    record_id TEXT NOT NULL,
    operation TEXT NOT NULL CHECK (operation IN ('INSERT', 'UPDATE', 'DELETE')),
    old_values JSONB,
    new_values JSONB,
    user_id TEXT,
    request_id TEXT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Metadata para debugging
    client_ip TEXT,
    user_agent TEXT
);

-- Indexes para Audit
CREATE INDEX idx_audit_table_record ON audit_log(table_name, record_id);
CREATE INDEX idx_audit_timestamp ON audit_log(timestamp);
CREATE INDEX idx_audit_request ON audit_log(request_id);
```

---

## 6. ARQUITECTURA DISTRIBUIDA - PATTERNS

### 🏗️ **MICROSERVICES EVOLUTION PATH**

```
PHASE 1: Monolith Modular     PHASE 2: Service Extraction    PHASE 3: Full Distributed
┌─────────────────────────┐   ┌─────────────────────────┐   ┌──────────────────────┐
│     Single Binary       │   │    API Gateway          │   │   Service Mesh       │
│  ┌─────────────────────┐│   │  ┌─────────────────────┐│   │ ┌──────────────────┐ │
│  │   API Layer         ││   │  │   Load Balancer     ││   │ │  Inventory-API   │ │
│  ├─────────────────────┤│   │  └─────────────────────┘│   │ └──────────────────┘ │
│  │  Inventory Service  ││   │           │             │   │ ┌──────────────────┐ │
│  │  Reservation Svc    ││ ──► │  ┌─────────────────┐  │ ──► │ │ Reservation-Svc  │ │
│  │  Idempotency Svc    ││   │  │  Inventory-Svc  │  │   │ └──────────────────┘ │
│  ├─────────────────────┤│   │  │  (Extracted)    │  │   │ ┌──────────────────┐ │
│  │  Repository Layer   ││   │  └─────────────────┘  │   │ │  Notification    │ │
│  └─────────────────────┘│   │           │             │   │ │  Service         │ │
│                         │   │  ┌─────────────────┐    │   │ └──────────────────┘ │
│      SQLite/PostgreSQL  │   │  │   Shared DB     │    │   │                      │
└─────────────────────────┘   └─────────────────────────┘   └──────────────────────┘
```

### 🔄 **EVENT SOURCING (Futuro)**
```go
// Event Types
type InventoryEvent interface {
    EventType() string
    AggregateID() string
    Timestamp() time.Time
}

type StockReserved struct {
    ProductID   string    `json:"product_id"`
    Quantity    int       `json:"quantity"`
    RequestID   string    `json:"request_id"`
    Reservation string    `json:"reservation_id"`
    Timestamp   time.Time `json:"timestamp"`
}

type StockReleased struct {
    ProductID     string    `json:"product_id"`
    Quantity      int       `json:"quantity"`
    ReservationID string    `json:"reservation_id"`
    Reason        string    `json:"reason"`
    Timestamp     time.Time `json:"timestamp"`
}
```

### 🎭 **SAGA PATTERN para Transacciones Distribuidas**
```yaml
Reserve-Purchase-Saga:
  steps:
    1. Reserve-Stock:
        service: inventory-service
        action: reserve
        compensate: release-stock
    
    2. Process-Payment:
        service: payment-service  
        action: charge
        compensate: refund-payment
    
    3. Create-Order:
        service: order-service
        action: create
        compensate: cancel-order
    
    4. Confirm-Reservation:
        service: inventory-service
        action: consume-reservation
        compensate: restore-stock
```

---

## 7. MIGRATION STRATEGY: SQLite → PostgreSQL

### 📁 **PHASE 1: SQLite (Desarrollo)**
```go
// Database Connection
import (
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
)

func NewSQLiteDB(dataSourceName string) (*sql.DB, error) {
    db, err := sql.Open("sqlite3", dataSourceName+"?_foreign_keys=on&_journal_mode=WAL")
    if err != nil {
        return nil, err
    }
    
    // SQLite optimizations
    db.SetMaxOpenConns(1) // SQLite doesn't support concurrent writes
    db.SetMaxIdleConns(1)
    
    return db, nil
}
```

### 🐘 **PHASE 2: PostgreSQL (Producción)**
```go
// PostgreSQL Connection Pool
import (
    "database/sql"
    _ "github.com/lib/pq"
)

func NewPostgreSQLDB(dataSourceName string) (*sql.DB, error) {
    db, err := sql.Open("postgres", dataSourceName)
    if err != nil {
        return nil, err
    }
    
    // Connection pool configuration
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(5 * time.Minute)
    
    return db, nil
}
```

### 🔄 **SCHEMA COMPATIBILITY**
```sql
-- SQLite → PostgreSQL Mappings

-- UUIDs
SQLite:     TEXT PRIMARY KEY
PostgreSQL: UUID PRIMARY KEY DEFAULT gen_random_uuid()

-- Timestamps  
SQLite:     TIMESTAMP DEFAULT CURRENT_TIMESTAMP
PostgreSQL: TIMESTAMPTZ DEFAULT NOW()

-- JSON Data
SQLite:     TEXT (JSON as string)
PostgreSQL: JSONB (Native JSON with indexes)

-- Auto Increment
SQLite:     INTEGER PRIMARY KEY AUTOINCREMENT
PostgreSQL: SERIAL PRIMARY KEY

-- Boolean
SQLite:     INTEGER CHECK (value IN (0,1))  
PostgreSQL: BOOLEAN DEFAULT FALSE
```

### 🚀 **DATA MIGRATION SCRIPT**
```bash
#!/bin/bash
# migrate_sqlite_to_pg.sh

# 1. Export SQLite schema and data
sqlite3 inventory.db ".schema" > schema.sql
sqlite3 inventory.db ".dump" > data.sql

# 2. Transform SQLite SQL to PostgreSQL
sed -i 's/INTEGER PRIMARY KEY AUTOINCREMENT/SERIAL PRIMARY KEY/g' schema.sql
sed -i 's/CURRENT_TIMESTAMP/NOW()/g' schema.sql
sed -i 's/TEXT PRIMARY KEY/UUID PRIMARY KEY DEFAULT gen_random_uuid()/g' schema.sql

# 3. Create PostgreSQL database
createdb inventory_prod

# 4. Apply schema
psql -d inventory_prod -f schema.sql

# 5. Import data (with transformations)
psql -d inventory_prod -f data.sql

# 6. Update sequences
psql -d inventory_prod -c "
    SELECT setval(pg_get_serial_sequence('inventory_items', 'id'), 
                  COALESCE(MAX(id), 1)) 
    FROM inventory_items;
"

echo "Migration completed successfully!"
```

---

## 8. OBSERVABILIDAD Y MONITOREO

### 📊 **MÉTRICAS CLAVE DE NEGOCIO**
```go
// Business Metrics
var (
    InventoryReservations = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "inventory_reservations_total",
            Help: "Total number of inventory reservations",
        },
        []string{"product_id", "status", "reason"},
    )
    
    StockLevels = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "inventory_stock_current",
            Help: "Current stock levels by product",
        },
        []string{"product_id", "sku"},
    )
    
    ReservationDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "inventory_reservation_duration_seconds",
            Help: "Duration of inventory reservations",
            Buckets: []float64{1, 5, 10, 30, 60, 300, 600, 1800},
        },
        []string{"product_id", "outcome"},
    )
)
```

### 🔍 **LOGGING ESTRUCTURADO**
```go
// Structured Logging with Context
func (s *InventoryService) Reserve(ctx context.Context, req ReserveRequest) error {
    logger := s.logger.With(
        zap.String("operation", "inventory.reserve"),
        zap.String("product_id", req.ProductID),
        zap.String("request_id", req.RequestID),
        zap.Int("quantity", req.Quantity),
        zap.String("trace_id", trace.FromContext(ctx).TraceID()),
    )
    
    start := time.Now()
    logger.Info("reservation.attempt.started")
    
    defer func() {
        logger.Info("reservation.attempt.completed",
            zap.Duration("duration", time.Since(start)))
    }()
    
    // Business logic...
    if err := s.validateStock(ctx, req); err != nil {
        logger.Error("reservation.validation.failed",
            zap.Error(err),
            zap.String("reason", "insufficient_stock"))
        return err
    }
    
    logger.Info("reservation.success",
        zap.String("reservation_id", reservationID))
    return nil
}
```

### 🏥 **HEALTH CHECKS DETALLADOS**
```go
type HealthChecker struct {
    db         *sql.DB
    repository Repository
    cache      Cache
}

func (h *HealthChecker) Check(ctx context.Context) HealthStatus {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    
    checks := map[string]ComponentHealth{
        "database":   h.checkDatabase(ctx),
        "inventory":  h.checkInventory(ctx),
        "cache":      h.checkCache(ctx),
    }
    
    overall := "healthy"
    for _, check := range checks {
        if check.Status != "healthy" {
            overall = "degraded"
            break
        }
    }
    
    return HealthStatus{
        Status:    overall,
        Timestamp: time.Now(),
        Checks:    checks,
        Uptime:    time.Since(h.startTime),
    }
}

func (h *HealthChecker) checkInventory(ctx context.Context) ComponentHealth {
    // Test critical inventory operation
    _, err := h.repository.GetInventoryByProduct(ctx, "health-check-product")
    
    if err != nil {
        return ComponentHealth{
            Status:  "unhealthy",
            Message: err.Error(),
            Latency: 0,
        }
    }
    
    return ComponentHealth{
        Status:  "healthy", 
        Message: "inventory operations functional",
        Latency: time.Since(start),
    }
}
```

---

## 9. TESTING STRATEGY

### 🧪 **UNIT TESTS - Service Layer**
```go
func TestInventoryService_Reserve(t *testing.T) {
    tests := []struct {
        name           string
        setup          func(*mockRepository)
        request        ReserveRequest
        expectedError  error
        expectedResult *ReservationResult
    }{
        {
            name: "successful_reservation",
            setup: func(repo *mockRepository) {
                repo.On("GetInventoryByProduct", mock.Anything, "product-123").
                    Return(&InventoryItem{
                        ProductID: "product-123",
                        Stock:     100,
                        Version:   1,
                    }, nil)
                    
                repo.On("UpdateInventoryOptimistic", mock.Anything, 
                    5, "product-123", 1).
                    Return(&InventoryItem{
                        ProductID: "product-123", 
                        Stock:     95,
                        Reserved:  5,
                        Version:   2,
                    }, nil)
            },
            request: ReserveRequest{
                ProductID: "product-123",
                Quantity:  5,
                RequestID: "req-456",
            },
            expectedError: nil,
        },
        {
            name: "insufficient_stock",
            setup: func(repo *mockRepository) {
                repo.On("GetInventoryByProduct", mock.Anything, "product-123").
                    Return(&InventoryItem{
                        ProductID: "product-123",
                        Stock:     2, // Insufficient
                        Version:   1,
                    }, nil)
            },
            request: ReserveRequest{
                ProductID: "product-123", 
                Quantity:  5,
                RequestID: "req-456",
            },
            expectedError: ErrInsufficientStock,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            repo := &mockRepository{}
            tt.setup(repo)
            
            service := NewInventoryService(repo)
            result, err := service.Reserve(context.Background(), tt.request)
            
            assert.Equal(t, tt.expectedError, err)
            if tt.expectedResult != nil {
                assert.Equal(t, tt.expectedResult, result)
            }
            
            repo.AssertExpectations(t)
        })
    }
}
```

### 🔄 **CONCURRENCY TESTS**
```go
func TestInventoryService_ConcurrentReservations(t *testing.T) {
    // Setup real database for concurrency testing
    db := setupTestDB(t)
    defer db.Close()
    
    repo := NewRepository(db)
    service := NewInventoryService(repo)
    
    // Create product with stock
    productID := "concurrent-test-product"
    setupProduct(t, repo, productID, 100) // 100 units
    
    // Simulate 50 concurrent reservations of 2 units each
    // Only 50 should succeed (100/2 = 50)
    numGoroutines := 50
    reservationSize := 2
    
    var wg sync.WaitGroup
    results := make([]error, numGoroutines)
    
    for i := 0; i < numGoroutines; i++ {
        wg.Add(1)
        go func(index int) {
            defer wg.Done()
            
            req := ReserveRequest{
                ProductID: productID,
                Quantity:  reservationSize,
                RequestID: fmt.Sprintf("concurrent-req-%d", index),
            }
            
            _, err := service.Reserve(context.Background(), req)
            results[index] = err
        }(i)
    }
    
    wg.Wait()
    
    // Count successful reservations
    successCount := 0
    insufficientStockCount := 0
    
    for _, err := range results {
        if err == nil {
            successCount++
        } else if errors.Is(err, ErrInsufficientStock) {
            insufficientStockCount++
        } else {
            t.Errorf("Unexpected error: %v", err)
        }
    }
    
    // Verify exactly 50 succeeded and 0 failed with other errors
    expectedSuccess := 50
    expectedInsufficient := 0
    
    assert.Equal(t, expectedSuccess, successCount, 
        "Should have exactly 50 successful reservations")
    assert.Equal(t, expectedInsufficient, insufficientStockCount,
        "Should have 0 insufficient stock errors with proper optimistic locking")
        
    // Verify final stock state
    finalInventory, err := repo.GetInventoryByProduct(context.Background(), productID)
    require.NoError(t, err)
    
    expectedFinalStock := 0 // 100 - (50 * 2) = 0
    expectedFinalReserved := 100 // 50 * 2 = 100
    
    assert.Equal(t, expectedFinalStock, finalInventory.Stock)
    assert.Equal(t, expectedFinalReserved, finalInventory.Reserved)
}
```

### 🚀 **LOAD TESTING con Artillery**
```yaml
# load-test.yml
config:
  target: 'http://localhost:8080'
  phases:
    - duration: 60
      arrivalRate: 10
      name: "Warmup"
    - duration: 300  
      arrivalRate: 50
      name: "Load test"
    - duration: 120
      arrivalRate: 100
      name: "Spike test"
      
scenarios:
  - name: "Reserve and Release Flow"
    weight: 80
    flow:
      - post:
          url: "/api/v1/inventory/reserve"
          headers:
            Content-Type: "application/json"
            Idempotency-Key: "{{ $uuid }}"
          json:
            product_id: "load-test-product-{{ $randomInt(1, 10) }}"
            quantity: "{{ $randomInt(1, 5) }}"
            request_id: "load-{{ $uuid }}"
            timeout_seconds: 300
          capture:
            - json: "$.reservation_id"
              as: "reservationId"
      
      - think: 5
      
      - post:
          url: "/api/v1/inventory/release"
          headers:
            Content-Type: "application/json" 
            Idempotency-Key: "{{ $uuid }}"
          json:
            reservation_id: "{{ reservationId }}"
            reason: "load_test_completion"

  - name: "Stock Queries"
    weight: 20
    flow:
      - get:
          url: "/api/v1/inventory/load-test-product-{{ $randomInt(1, 10) }}"
```

---

## 10. DEPLOYMENT Y CONFIGURACIÓN

### ⚙️ **CONFIGURACIÓN POR AMBIENTE**
```go
// config/config.go
type Config struct {
    Server   ServerConfig   `mapstructure:"server"`
    Database DatabaseConfig `mapstructure:"database"`
    Cache    CacheConfig    `mapstructure:"cache"`
    Logging  LoggingConfig  `mapstructure:"logging"`
}

type ServerConfig struct {
    Port         int           `mapstructure:"port" default:"8080"`
    ReadTimeout  time.Duration `mapstructure:"read_timeout" default:"30s"`
    WriteTimeout time.Duration `mapstructure:"write_timeout" default:"30s"`
    IdleTimeout  time.Duration `mapstructure:"idle_timeout" default:"120s"`
}

type DatabaseConfig struct {
    Driver          string        `mapstructure:"driver" default:"sqlite3"`
    DSN             string        `mapstructure:"dsn" default:"./inventory.db"`
    MaxConnections  int           `mapstructure:"max_connections" default:"10"`
    MaxIdleConns    int           `mapstructure:"max_idle_conns" default:"2"`
    ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime" default:"5m"`
    MigrationsPath  string        `mapstructure:"migrations_path" default:"./db/migrations"`
}
```

### 🐳 **DOCKER MULTI-STAGE BUILD**
```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies for SQLite
RUN apk add --no-cache gcc musl-dev sqlite-dev

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary with optimizations
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-s -w -X main.version=$(git describe --tags --always)" \
    -o inventory ./main.go

# Runtime stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache ca-certificates sqlite tzdata

# Create non-root user
RUN addgroup -g 1001 -S inventory && \
    adduser -u 1001 -S inventory -G inventory

# Set working directory
WORKDIR /app

# Copy binary and config
COPY --from=builder /app/inventory .
COPY --from=builder /app/db/migrations ./db/migrations
COPY --from=builder /app/config ./config

# Set ownership
RUN chown -R inventory:inventory /app

# Switch to non-root user
USER inventory

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Expose port
EXPOSE 8080

# Run application
CMD ["./inventory"]
```

### 🚀 **DOCKER COMPOSE SETUP**
```yaml
# docker-compose.yml
version: '3.8'

services:
  inventory-api:
    build: .
    ports:
      - "8080:8080"
    environment:
      - ENV=production
      - DATABASE_DRIVER=postgres
      - DATABASE_DSN=postgres://inventory:password@postgres:5432/inventory?sslmode=disable
      - LOG_LEVEL=info
    depends_on:
      postgres:
        condition: service_healthy
    volumes:
      - ./config:/app/config:ro
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  postgres:
    image: postgres:15-alpine
    environment:
      - POSTGRES_DB=inventory
      - POSTGRES_USER=inventory
      - POSTGRES_PASSWORD=password
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./db/init.sql:/docker-entrypoint-initdb.d/01-init.sql
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U inventory -d inventory"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data
    ports:
      - "6379:6379" 
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 3s
      retries: 5

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'

volumes:
  postgres_data:
  redis_data:
  prometheus_data:
```

### 📊 **KUBERNETES MANIFESTS**
```yaml
# k8s/deployment.yml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: inventory-api
  labels:
    app: inventory-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: inventory-api
  template:
    metadata:
      labels:
        app: inventory-api
    spec:
      containers:
      - name: inventory-api
        image: inventory:latest
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_DSN
          valueFrom:
            secretKeyRef:
              name: inventory-secrets
              key: database-dsn
        resources:
          requests:
            memory: "64Mi"
            cpu: "50m"
          limits:
            memory: "128Mi" 
            cpu: "100m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5

---
apiVersion: v1
kind: Service
metadata:
  name: inventory-service
spec:
  selector:
    app: inventory-api
  ports:
  - port: 80
    targetPort: 8080
  type: ClusterIP

---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: inventory-ingress
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
  - hosts:
    - inventory-api.company.com
    secretName: inventory-tls
  rules:
  - host: inventory-api.company.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: inventory-service
            port:
              number: 80
```

---

## CONCLUSIÓN

Esta arquitectura distribuida proporciona:

✅ **Solución completa** para race conditions e inconsistencias  
✅ **Escalabilidad** desde monolito a microservicios  
✅ **Consistencia** de datos con optimistic locking  
✅ **Idempotencia** para operaciones seguras  
✅ **Observabilidad** completa con métricas y logs  
✅ **Migration path** claro SQLite → PostgreSQL  
✅ **Testing strategy** para concurrencia y carga  
✅ **Deployment ready** con Docker y Kubernetes  

### 🎯 **NEXT STEPS para Implementación**

1. **Day 1**: Implementar schema + queries SQLC
2. **Day 2**: Service layer + optimistic locking  
3. **Day 3**: API handlers + middleware
4. **Day 4**: Testing + concurrency validation
5. **Day 5**: Observability + deployment setup

**Stack confirmado**: Go + Gin + SQLite + SQLC + golang-migrate ✅