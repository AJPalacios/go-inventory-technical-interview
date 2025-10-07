# 📡 API Specification - Inventory Management System

**Complete RESTful API Documentation with Examples**

---

## 🌐 Base Information

```
Base URL: https://api.inventory.company.com
API Version: v1
Content-Type: application/json
Authentication: API Key (X-API-Key header)
```

### 🔧 **Global Headers**

```http
Content-Type: application/json
Accept: application/json
X-API-Key: your-api-key-here
Idempotency-Key: uuid-v4  # For POST/PUT operations
X-Request-ID: uuid-v4     # Optional correlation ID
```

---

## 📋 **Core Endpoints**

### 1️⃣ **Reserve Stock**

**Reserve inventory for a pending transaction**

```http
POST /api/v1/inventory/reserve
```

#### **Request Body**
```json
{
  "product_id": "550e8400-e29b-41d4-a716-446655440000",
  "quantity": 5,
  "timeout_seconds": 300,
  "reason": "order_checkout",
  "client_id": "user_12345",
  "metadata": {
    "order_id": "ORD-2025-001",
    "session_id": "sess_abcd1234"
  }
}
```

#### **Validation Rules**
- `product_id`: **required**, must be valid UUID
- `quantity`: **required**, integer 1-100,000
- `timeout_seconds`: **optional**, integer 60-86,400 (default: 300)
- `reason`: **optional**, string max 500 chars
- `client_id`: **optional**, string 1-100 chars
- `metadata`: **optional**, JSON object

#### **Success Response - 201 Created**
```json
{
  "success": true,
  "data": {
    "reservation_id": "res_abcd1234efgh5678",
    "product_id": "550e8400-e29b-41d4-a716-446655440000",
    "quantity": 5,
    "status": "active",
    "expires_at": "2025-10-06T15:30:00Z",
    "created_at": "2025-10-06T15:25:00Z",
    "reason": "order_checkout",
    "client_id": "user_12345"
  },
  "meta": {
    "request_id": "123e4567-e89b-12d3-a456-426614174000",
    "timestamp": "2025-10-06T15:25:00Z",
    "version": "v1",
    "processing_time_ms": 15
  }
}
```

#### **Error Responses**

**400 Bad Request - Validation Error**
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Request validation failed",
    "details": "field 'quantity' must be between 1 and 100000",
    "field_errors": {
      "quantity": "must be between 1 and 100000",
      "product_id": "must be a valid UUID"
    }
  },
  "meta": {
    "request_id": "123e4567-e89b-12d3-a456-426614174000",
    "timestamp": "2025-10-06T15:25:00Z",
    "version": "v1"
  }
}
```

**409 Conflict - Insufficient Stock**
```json
{
  "success": false,
  "error": {
    "code": "INSUFFICIENT_STOCK",
    "message": "Insufficient stock available",
    "details": "requested 5 units, only 2 available",
    "context": {
      "product_id": "550e8400-e29b-41d4-a716-446655440000",
      "requested_quantity": 5,
      "available_stock": 2,
      "reserved_stock": 8
    }
  },
  "meta": {
    "request_id": "123e4567-e89b-12d3-a456-426614174000",
    "timestamp": "2025-10-06T15:25:00Z",
    "version": "v1"
  }
}
```

---

### 2️⃣ **Release Stock**

**Release a previously created reservation**

```http
POST /api/v1/inventory/release
```

#### **Request Body**
```json
{
  "reservation_id": "res_abcd1234efgh5678",
  "reason": "order_cancelled",
  "metadata": {
    "cancellation_reason": "customer_request",
    "refund_id": "REF-2025-001"
  }
}
```

#### **Validation Rules**
- `reservation_id`: **required**, valid UUID
- `reason`: **required**, string 3-500 chars, enum: `cancelled|purchased|timeout|expired|admin`
- `metadata`: **optional**, JSON object

#### **Success Response - 200 OK**
```json
{
  "success": true,
  "data": {
    "reservation_id": "res_abcd1234efgh5678",
    "product_id": "550e8400-e29b-41d4-a716-446655440000",
    "quantity_released": 5,
    "status": "released",
    "reason": "order_cancelled",
    "released_at": "2025-10-06T15:28:00Z",
    "original_expiry": "2025-10-06T15:30:00Z"
  },
  "meta": {
    "request_id": "123e4567-e89b-12d3-a456-426614174000",
    "timestamp": "2025-10-06T15:28:00Z",
    "version": "v1",
    "processing_time_ms": 8
  }
}
```

---

### 3️⃣ **Get Stock Information**

**Retrieve current stock levels and reservation information**

```http
GET /api/v1/inventory/{product_id}
```

#### **Path Parameters**
- `product_id`: **required**, valid UUID

#### **Query Parameters**
- `include_reservations`: **optional**, boolean (default: false)
- `include_history`: **optional**, boolean (default: false)

#### **Success Response - 200 OK**
```json
{
  "success": true,
  "data": {
    "product_id": "550e8400-e29b-41d4-a716-446655440000",
    "product_info": {
      "name": "Premium Headphones",
      "sku": "HDX-2025-PRO",
      "category": "electronics"
    },
    "stock_info": {
      "total_stock": 100,
      "available_stock": 85,
      "reserved_stock": 15,
      "version": 42,
      "last_updated": "2025-10-06T15:20:00Z"
    },
    "reservations_summary": {
      "active_count": 8,
      "total_reserved_quantity": 15,
      "oldest_reservation": "2025-10-06T14:30:00Z",
      "next_expiry": "2025-10-06T15:35:00Z"
    }
  },
  "meta": {
    "request_id": "123e4567-e89b-12d3-a456-426614174000",
    "timestamp": "2025-10-06T15:30:00Z",
    "version": "v1",
    "cache_status": "hit",
    "processing_time_ms": 3
  }
}
```

#### **With Reservations Detail**
```http
GET /api/v1/inventory/{product_id}?include_reservations=true
```

```json
{
  "success": true,
  "data": {
    "product_id": "550e8400-e29b-41d4-a716-446655440000",
    "stock_info": {
      "total_stock": 100,
      "available_stock": 85,
      "reserved_stock": 15,
      "version": 42
    },
    "active_reservations": [
      {
        "reservation_id": "res_active_001",
        "quantity": 5,
        "client_id": "user_12345",
        "expires_at": "2025-10-06T15:35:00Z",
        "created_at": "2025-10-06T15:30:00Z",
        "reason": "order_checkout"
      },
      {
        "reservation_id": "res_active_002", 
        "quantity": 10,
        "client_id": "user_67890",
        "expires_at": "2025-10-06T15:40:00Z",
        "created_at": "2025-10-06T15:35:00Z",
        "reason": "cart_hold"
      }
    ]
  },
  "meta": {
    "request_id": "123e4567-e89b-12d3-a456-426614174000",
    "timestamp": "2025-10-06T15:30:00Z",
    "version": "v1",
    "processing_time_ms": 12
  }
}
```

---

### 4️⃣ **Update Stock Levels**

**Administrative endpoint to adjust inventory levels**

```http
PUT /api/v1/inventory/{product_id}/stock
```

#### **Request Body**
```json
{
  "adjustment_type": "SET",
  "new_stock": 200,
  "reason": "Weekly inventory restock",
  "reference": "PO-2025-001",
  "metadata": {
    "supplier_id": "SUP-ACME-001",
    "delivery_date": "2025-10-06",
    "batch_number": "BATCH-2025-Q4-001"
  }
}
```

#### **Validation Rules**
- `adjustment_type`: **required**, enum: `SET|INCREMENT|DECREMENT`
- `new_stock`: **required**, integer 0-100,000
- `reason`: **required**, string 3-500 chars
- `reference`: **optional**, string max 100 chars
- `metadata`: **optional**, JSON object

#### **Success Response - 200 OK**
```json
{
  "success": true,
  "data": {
    "product_id": "550e8400-e29b-41d4-a716-446655440000",
    "stock_change": {
      "previous_stock": 150,
      "new_stock": 200,
      "adjustment": +50,
      "adjustment_type": "SET"
    },
    "current_state": {
      "total_stock": 200,
      "available_stock": 185,
      "reserved_stock": 15,
      "version": 43
    },
    "audit_info": {
      "reason": "Weekly inventory restock",
      "reference": "PO-2025-001",
      "updated_by": "admin_user",
      "updated_at": "2025-10-06T15:30:00Z"
    }
  },
  "meta": {
    "request_id": "123e4567-e89b-12d3-a456-426614174000",
    "timestamp": "2025-10-06T15:30:00Z",
    "version": "v1",
    "processing_time_ms": 25
  }
}
```

---

### 5️⃣ **Batch Reserve Stock**

**Reserve multiple products in a single atomic operation**

```http
POST /api/v1/inventory/batch/reserve
```

#### **Request Body**
```json
{
  "requests": [
    {
      "product_id": "550e8400-e29b-41d4-a716-446655440000",
      "quantity": 2,
      "reason": "batch_order_item_1"
    },
    {
      "product_id": "550e8400-e29b-41d4-a716-446655440001",
      "quantity": 1,
      "reason": "batch_order_item_2"
    },
    {
      "product_id": "550e8400-e29b-41d4-a716-446655440002",
      "quantity": 3,
      "reason": "batch_order_item_3"
    }
  ],
  "batch_timeout_seconds": 300,
  "client_id": "user_12345",
  "metadata": {
    "order_id": "ORD-2025-BATCH-001",
    "checkout_session": "sess_batch_xyz789"
  }
}
```

#### **Validation Rules**
- `requests`: **required**, array 1-100 items
- Each request follows individual reserve validation
- `batch_timeout_seconds`: **optional**, integer 60-86,400
- `client_id`: **optional**, string 1-100 chars

#### **Success Response - 201 Created**
```json
{
  "success": true,
  "data": {
    "batch_id": "batch_xyz789abc123",
    "total_items": 3,
    "successful_reservations": 3,
    "failed_reservations": 0,
    "reservations": [
      {
        "product_id": "550e8400-e29b-41d4-a716-446655440000",
        "reservation_id": "res_batch_001",
        "quantity": 2,
        "status": "active",
        "expires_at": "2025-10-06T15:35:00Z"
      },
      {
        "product_id": "550e8400-e29b-41d4-a716-446655440001",
        "reservation_id": "res_batch_002",
        "quantity": 1,
        "status": "active",
        "expires_at": "2025-10-06T15:35:00Z"
      },
      {
        "product_id": "550e8400-e29b-41d4-a716-446655440002",
        "reservation_id": "res_batch_003",
        "quantity": 3,
        "status": "active",
        "expires_at": "2025-10-06T15:35:00Z"
      }
    ],
    "batch_expires_at": "2025-10-06T15:35:00Z",
    "created_at": "2025-10-06T15:30:00Z"
  },
  "meta": {
    "request_id": "123e4567-e89b-12d3-a456-426614174000",
    "timestamp": "2025-10-06T15:30:00Z",
    "version": "v1",
    "processing_time_ms": 45
  }
}
```

---

### 6️⃣ **Health Check**

**System health and operational status**

```http
GET /health
```

#### **Success Response - 200 OK**
```json
{
  "status": "healthy",
  "timestamp": "2025-10-06T15:30:00Z",
  "version": "1.2.0",
  "uptime": "2h30m15s",
  "environment": "production",
  "checks": {
    "database": {
      "status": "healthy",
      "latency_ms": 2,
      "connections": {
        "active": 5,
        "idle": 3,
        "max": 25
      },
      "last_check": "2025-10-06T15:30:00Z"
    },
    "cache": {
      "status": "healthy",
      "latency_ms": 1,
      "connections": 2,
      "memory_usage": "45%",
      "last_check": "2025-10-06T15:30:00Z"
    },
    "inventory_service": {
      "status": "operational",
      "active_reservations": 156,
      "cleanup_last_run": "2025-10-06T15:25:00Z",
      "avg_response_time_ms": 3.2,
      "last_check": "2025-10-06T15:30:00Z"
    }
  },
  "metrics": {
    "requests_per_second": 125.4,
    "error_rate_percentage": 0.01,
    "avg_response_time_ms": 4.8,
    "version_conflicts_per_minute": 2.1
  }
}
```

#### **Degraded Response - 200 OK**
```json
{
  "status": "degraded",
  "timestamp": "2025-10-06T15:30:00Z",
  "version": "1.2.0",
  "uptime": "2h30m15s",
  "environment": "production",
  "issues": [
    {
      "component": "cache",
      "status": "degraded",
      "message": "High memory usage detected",
      "severity": "warning",
      "impact": "Increased response times for read operations"
    }
  ],
  "checks": {
    "database": {
      "status": "healthy",
      "latency_ms": 2
    },
    "cache": {
      "status": "degraded",
      "latency_ms": 15,
      "memory_usage": "89%",
      "warning": "Memory usage above 85% threshold"
    }
  }
}
```

---

## 🔧 **Error Handling**

### **Standard Error Response Format**

All API errors follow this consistent structure:

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": "Additional context about the error",
    "field_errors": {
      "field_name": "field-specific error message"
    },
    "context": {
      "operation": "reserve_stock",
      "entity": "inventory",
      "entity_id": "550e8400-e29b-41d4-a716-446655440000"
    },
    "retry_after_seconds": 5,
    "documentation_url": "https://docs.inventory.company.com/errors/INSUFFICIENT_STOCK"
  },
  "meta": {
    "request_id": "123e4567-e89b-12d3-a456-426614174000",
    "timestamp": "2025-10-06T15:30:00Z",
    "version": "v1"
  }
}
```

### **HTTP Status Codes**

| Status Code | Description | When Used |
|-------------|-------------|----------|
| **200** | OK | Successful GET, PUT operations |
| **201** | Created | Successful POST operations (reservations) |
| **400** | Bad Request | Validation errors, malformed requests |
| **401** | Unauthorized | Missing or invalid API key |
| **403** | Forbidden | Insufficient permissions |
| **404** | Not Found | Product not found |
| **409** | Conflict | Business rule violation (insufficient stock) |
| **422** | Unprocessable Entity | Semantic validation errors |
| **429** | Too Many Requests | Rate limiting exceeded |
| **500** | Internal Server Error | Unexpected server errors |
| **503** | Service Unavailable | System maintenance, overload |

### **Error Codes**

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `VALIDATION_ERROR` | 400 | Input validation failed |
| `PRODUCT_NOT_FOUND` | 404 | Product does not exist |
| `RESERVATION_NOT_FOUND` | 404 | Reservation does not exist |
| `INSUFFICIENT_STOCK` | 409 | Not enough stock available |
| `RESERVATION_EXPIRED` | 409 | Reservation has expired |
| `VERSION_CONFLICT` | 409 | Optimistic locking conflict |
| `DUPLICATE_REQUEST` | 409 | Idempotency key already used |
| `RATE_LIMIT_EXCEEDED` | 429 | Too many requests |
| `SYSTEM_OVERLOAD` | 503 | System at capacity |
| `INTERNAL_ERROR` | 500 | Unexpected system error |

---

## 🔐 **Authentication & Security**

### **API Key Authentication**

```http
X-API-Key: your-api-key-here
```

### **Idempotency**

For safe retry operations, include an idempotency key:

```http
Idempotency-Key: 550e8400-e29b-41d4-a716-446655440000
```

- **Required for**: POST, PUT operations
- **Format**: UUID v4
- **Behavior**: Duplicate requests return cached response
- **TTL**: 24 hours

### **Rate Limiting**

```
Rate Limits:
- Authenticated: 1000 requests/minute
- Unauthenticated: 100 requests/minute
- Batch operations: 100 requests/minute
```

**Rate Limit Headers**:
```http
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1704067200
Retry-After: 60
```

---

## 📊 **Performance & Monitoring**

### **Response Headers**

```http
X-Response-Time: 15ms
X-Request-ID: 123e4567-e89b-12d3-a456-426614174000
X-Version: v1
X-Cache-Status: hit|miss
X-Service-Version: 1.2.0
```

### **Performance Metrics**

| Operation | Avg Latency | p95 Latency | Throughput |
|-----------|-------------|-------------|------------|
| **Reserve Stock** | 5ms | 15ms | 10,000 ops/sec |
| **Release Stock** | 3ms | 8ms | 12,000 ops/sec |
| **Get Stock** | 2ms | 5ms | 50,000 ops/sec |
| **Update Stock** | 8ms | 20ms | 5,000 ops/sec |
| **Batch Reserve** | 15ms | 35ms | 2,000 ops/sec |

---

## 🧪 **Testing & Examples**

### **cURL Examples**

```bash
# Reserve stock
curl -X POST "https://api.inventory.company.com/api/v1/inventory/reserve" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -H "Idempotency-Key: $(uuidgen)" \
  -d '{
    "product_id": "550e8400-e29b-41d4-a716-446655440000",
    "quantity": 5,
    "reason": "order_checkout"
  }'

# Get stock info
curl -X GET "https://api.inventory.company.com/api/v1/inventory/550e8400-e29b-41d4-a716-446655440000" \
  -H "X-API-Key: your-api-key"

# Health check
curl -X GET "https://api.inventory.company.com/health"
```

### **SDKs & Libraries**

- **Go**: `go get github.com/company/inventory-go-sdk`
- **JavaScript**: `npm install @company/inventory-sdk`
- **Python**: `pip install inventory-sdk`
- **Java**: Maven/Gradle artifacts available

---

## 📚 **Additional Resources**

- **OpenAPI Spec**: `/api/v1/docs/openapi.json`
- **Swagger UI**: `/api/v1/docs`
- **Postman Collection**: Available in repository
- **Test Suite**: Complete HTTP test files in `test-api/` directory

---

*This API specification provides a complete reference for integrating with the distributed inventory management system.*