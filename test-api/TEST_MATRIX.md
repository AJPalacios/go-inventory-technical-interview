# 🧪 Matriz de Pruebas - Inventory API

## 📋 Índice

- [Casos de Prueba por Endpoint](#casos-de-prueba-por-endpoint)
- [Matriz de Validaciones](#matriz-de-validaciones)
- [Escenarios de Negocio](#escenarios-de-negocio)
- [Pruebas de Rendimiento](#pruebas-de-rendimiento)
- [Casos Edge](#casos-edge)
- [Checklist de Testing](#checklist-de-testing)

---

## 🎯 Casos de Prueba por Endpoint

### 1. GET `/health`

| Test ID | Escenario | Input | Expected Output | Status |
|---------|-----------|--------|-----------------|--------|
| H001 | Health check básico | `GET /health` | `200 OK` con status healthy | ✅ |
| H002 | Health check con headers | `GET /health` + custom headers | `200 OK` ignorando headers extra | ✅ |
| H003 | Health check método incorrecto | `POST /health` | `405 Method Not Allowed` | ✅ |

---

### 2. GET `/api/v1/inventory/stock/:productId`

| Test ID | Escenario | Product ID | Expected Output | Status |
|---------|-----------|------------|-----------------|--------|
| S001 | Consultar stock válido | `e08e3e7e-9126-49e4-9caf-63885a07bd78` | `200 OK` con datos de stock | ✅ |
| S002 | Producto inexistente | `00000000-0000-0000-0000-000000000000` | `404 Not Found` | ✅ |
| S003 | UUID inválido | `invalid-uuid` | `400 Bad Request` | ✅ |
| S004 | UUID malformado | `e08e3e7e-xxxx-41d4-a716-446655440000` | `400 Bad Request` | ✅ |
| S005 | Sin productId | `/api/v1/inventory/stock/` | `404 Not Found` | ✅ |
| S006 | ProductId vacío | `/api/v1/inventory/stock/""` | `400 Bad Request` | ✅ |

---

### 3. POST `/api/v1/inventory/reserve`

| Test ID | Escenario | Input | Expected Output | Status |
|---------|-----------|--------|-----------------|--------|
| R001 | Reserva básica válida | Payload completo válido | `201 Created` con reservation_id | ✅ |
| R002 | Sin X-Request-ID header | Payload válido sin header | `400 Bad Request` | ✅ |
| R003 | Product ID inexistente | Product ID no existe | `404 Not Found` | ✅ |
| R004 | Stock insuficiente | Quantity > available_stock | `409 Conflict` | ✅ |
| R005 | Idempotencia - mismo request_id | Mismo request_id 2 veces | Segunda: misma respuesta | ✅ |
| R006 | Timeout personalizado | timeout_seconds = 600 | `201 Created` con timeout correcto | ✅ |
| R007 | Con metadata | Payload + metadata object | `201 Created` con metadata guardada | ✅ |
| R008 | Quantity = 0 | quantity: 0 | `400 Bad Request` | ✅ |
| R009 | Quantity negativa | quantity: -1 | `400 Bad Request` | ✅ |
| R010 | Quantity excesiva | quantity: 100001 | `400 Bad Request` | ✅ |
| R011 | Timeout muy bajo | timeout_seconds: 59 | `400 Bad Request` | ✅ |
| R012 | Timeout muy alto | timeout_seconds: 3601 | `400 Bad Request` | ✅ |
| R013 | Product ID inválido | UUID malformado | `400 Bad Request` | ✅ |
| R014 | Request ID inválido | request_id vacío | `400 Bad Request` | ✅ |
| R015 | JSON malformado | JSON syntax error | `400 Bad Request` | ✅ |
| R016 | Content-Type incorrecto | text/plain | `400 Bad Request` | ✅ |
| R017 | Campos faltantes | Sin product_id | `400 Bad Request` | ✅ |
| R018 | Reason muy largo | reason > 500 chars | `400 Bad Request` | ✅ |
| R019 | Client ID muy largo | client_id > 100 chars | `400 Bad Request` | ✅ |
| R020 | Metadata muy grande | metadata con muchos campos | `201 Created` o `400` según límite | ⚠️ |

---

### 4. POST `/api/v1/inventory/batch/reserve`

| Test ID | Escenario | Input | Expected Output | Status |
|---------|-----------|--------|-----------------|--------|
| BR001 | Batch válido múltiple | Array de 3 reservas válidas | `201 Created` con array de resultados | ✅ |
| BR002 | Batch con un error | 2 válidas + 1 inválida | `207 Multi-Status` con resultados mixtos | ✅ |
| BR003 | Batch vacío | requests: [] | `400 Bad Request` | ✅ |
| BR004 | Batch muy grande | requests: [100+ items] | `400 Bad Request` | ✅ |
| BR005 | Productos duplicados | Mismo product_id múltiples veces | `400 Bad Request` o procesamiento | ⚠️ |
| BR006 | Request IDs duplicados | Mismo request_id en batch | `400 Bad Request` | ✅ |
| BR007 | Sin requests array | Payload sin "requests" | `400 Bad Request` | ✅ |

---

### 5. POST `/api/v1/inventory/release`

| Test ID | Escenario | Input | Expected Output | Status |
|---------|-----------|--------|-----------------|--------|
| RL001 | Release válido - cancelled | reservation_id + reason: cancelled | `200 OK` | ✅ |
| RL002 | Release válido - purchased | reservation_id + reason: purchased | `200 OK` | ✅ |
| RL003 | Release válido - timeout | reservation_id + reason: timeout | `200 OK` | ✅ |
| RL004 | Release válido - expired | reservation_id + reason: expired | `200 OK` | ✅ |
| RL005 | Release válido - admin | reservation_id + reason: admin | `200 OK` | ✅ |
| RL006 | Reservation inexistente | UUID válido no existe | `404 Not Found` | ✅ |
| RL007 | Reservation ya liberada | ID de reserva ya procesada | `409 Conflict` | ✅ |
| RL008 | UUID inválido | reservation_id malformado | `400 Bad Request` | ✅ |
| RL009 | Reason inválido | reason: "invalid_reason" | `400 Bad Request` | ✅ |
| RL010 | Reason muy corto | reason: "ab" | `400 Bad Request` | ✅ |
| RL011 | Reason muy largo | reason > 500 chars | `400 Bad Request` | ✅ |
| RL012 | Sin campos requeridos | Sin reservation_id o reason | `400 Bad Request` | ✅ |
| RL013 | Idempotencia release | Mismo request_id múltiples veces | Segunda: misma respuesta | ✅ |

---

### 6. PUT `/api/v1/inventory/stock`

| Test ID | Escenario | Input | Expected Output | Status |
|---------|-----------|--------|-----------------|--------|
| U001 | Update válido - restock | adjustment_type: restock | `200 OK` | ✅ |
| U002 | Update válido - adjustment | adjustment_type: adjustment | `200 OK` | ✅ |
| U003 | Update válido - return | adjustment_type: return | `200 OK` | ✅ |
| U004 | Update válido - correction | adjustment_type: correction | `200 OK` | ✅ |
| U005 | Stock negativo | new_stock: -1 | `400 Bad Request` | ✅ |
| U006 | Stock excesivo | new_stock: 100001 | `400 Bad Request` | ✅ |
| U007 | Adjustment type inválido | adjustment_type: "invalid" | `400 Bad Request` | ✅ |
| U008 | Product inexistente | UUID válido no existe | `404 Not Found` | ✅ |
| U009 | UUID inválido | product_id malformado | `400 Bad Request` | ✅ |
| U010 | Reason muy corto | reason: "ab" | `400 Bad Request` | ✅ |
| U011 | Reason muy largo | reason > 500 chars | `400 Bad Request` | ✅ |
| U012 | Con reference | reference: "PO-12345" | `200 OK` con reference | ✅ |
| U013 | Con metadata | metadata object | `200 OK` con metadata | ✅ |
| U014 | Campos faltantes | Sin campos requeridos | `400 Bad Request` | ✅ |
| U015 | Idempotencia update | Mismo request_id múltiples veces | Segunda: misma respuesta | ✅ |

---

## 🔍 Matriz de Validaciones

### Validaciones de UUID

| Campo | Formato Válido | Formatos Inválidos | Error Expected |
|-------|----------------|-------------------|----------------|
| product_id | `e08e3e7e-9126-49e4-9caf-63885a07bd78` | `invalid-uuid`, `123`, `""`, `null` | `400 Bad Request` |
| reservation_id | `11171f8d-a6a4-42d3-9ab1-c6d3d829c83e` | `not-uuid`, `12345`, missing hyphens | `400 Bad Request` |
| request_id | Cualquier string no vacío | `""`, `null`, missing | `400 Bad Request` |

### Validaciones de Rangos

| Campo | Rango Válido | Valores Inválidos | Error Expected |
|-------|--------------|------------------|----------------|
| quantity | 1 - 100000 | 0, -1, 100001, "abc" | `400 Bad Request` |
| new_stock | 0 - 100000 | -1, 100001, "abc" | `400 Bad Request` |
| timeout_seconds | 60 - 3600 | 59, 3601, 0, -1 | `400 Bad Request` |

### Validaciones de Strings

| Campo | Longitud | Valores Especiales | Patrones |
|-------|----------|------------------|----------|
| reason | 3-500 chars | Requerido | Cualquier string |
| client_id | 1-100 chars | Opcional | Cualquier string |
| reference | 0-255 chars | Opcional | Cualquier string |

### Validaciones Enum

| Campo | Valores Válidos | Valores Inválidos |
|-------|----------------|------------------|
| reason (release) | `cancelled`, `purchased`, `timeout`, `expired`, `admin` | `invalid`, `other`, `""` |
| adjustment_type | `restock`, `adjustment`, `return`, `correction` | `invalid`, `other`, `""` |

---

## 📊 Escenarios de Negocio Completos

### 🛒 Flujo A: E-commerce Order (Orden Completa)

| Paso | Acción | Endpoint | Input | Expected Output | Validación Post-Test |
|------|--------|----------|-------|-----------------|---------------------|
| A1 | **Consultar disponibilidad** | `GET /api/v1/inventory/stock/{id}` | Laptop HP (`2d70d1dc-cd3a-4f40-afb0-52e16445bf36`) | `200 OK` + stock info | `available_stock >= 2` |
| A2 | **Agregar al carrito** | `POST /api/v1/inventory/reserve` | quantity: 2, timeout: 1800, reason: "e-commerce cart" | `201 Created` + reservation_id | Stock reservado: `reserved_stock += 2` |
| A3 | **Procesar pago** | `GET /api/v1/inventory/stock/{id}` | Verificar reserva activa | Stock reflejado | `available_stock -= 2` |
| A4 | **Confirmar venta** | `POST /api/v1/inventory/release` | reason: "purchased", reservation_id | `200 OK` | Stock definitivo vendido |
| A5 | **Verificar transacción** | `GET /api/v1/inventory/stock/{id}` | Estado final del stock | Consistencia total | `total_stock` unchanged, `available_stock` reduced |

### 🛍️ Flujo B: Cart Abandonment (Carrito Abandonado)

| Paso | Acción | Endpoint | Input | Expected Output | Validación Post-Test |
|------|--------|----------|-------|-----------------|---------------------|
| B1 | **Agregar al carrito** | `POST /api/v1/inventory/reserve` | Sony Headphones + timeout: 900 | `201 Created` | Reserva temporal activa |
| B2 | **Simular navegación** | `GET /api/v1/inventory/stock/{id}` | Verificar stock reservado | Stock temporalmente reducido | `reserved_stock` incrementado |
| B3 | **Cliente abandona** | `POST /api/v1/inventory/release` | reason: "cancelled" o "timeout" | `200 OK` | Stock regresa a disponible |
| B4 | **Verificar liberación** | `GET /api/v1/inventory/stock/{id}` | Stock restaurado | `available_stock` restaurado | Inventory consistency restored |

### 📦 Flujo C: Warehouse Operations (Operaciones de Almacén)

| Paso | Acción | Endpoint | Input | Expected Output | Validación Post-Test |
|------|--------|----------|-------|-----------------|---------------------|
| C1 | **Auditoría inicial** | `GET /api/v1/inventory/stock/{id}` | Samsung SSD check | Current stock levels | Baseline establecido |
| C2 | **Recibir mercancía** | `PUT /api/v1/inventory/stock` | adjustment_type: "restock", +50 units | `200 OK` | `available_stock += 50` |
| C3 | **Corrección manual** | `PUT /api/v1/inventory/stock` | adjustment_type: "adjustment", reason: "audit correction" | `200 OK` | Discrepancias corregidas |
| C4 | **Procesar devolución** | `PUT /api/v1/inventory/stock` | adjustment_type: "return", +3 units | `200 OK` | Stock de devoluciones |
| C5 | **Auditoría final** | `GET /api/v1/inventory/stock/{id}` | Verificar totales | Consistency check | Todos los movimientos reflejados |

### 🏢 Flujo D: B2B Bulk Order (Orden Corporativa)

| Paso | Acción | Endpoint | Input | Expected Output | Validación Post-Test |
|------|--------|----------|-------|-----------------|---------------------|
| D1 | **Verificar disponibilidad** | Multiple `GET /stock/{id}` | Lista de productos (5 diferentes) | Stock de cada producto | Sufficient inventory check |
| D2 | **Reserva en lote** | `POST /api/v1/inventory/batch/reserve` | Array de 5 productos | `201 Created` con resultados | Batch atomic operation |
| D3 | **Proceso de aprobación** | `GET /api/v1/inventory/stock/{id}` (x5) | Verificar todas las reservas | Stock reservado correctamente | Business approval simulation |
| D4 | **Confirmar orden** | `POST /api/v1/inventory/release` (x5) | reason: "purchased" para todas | All `200 OK` | Bulk order completion |
| D5 | **Reporte final** | Multiple `GET /stock/{id}` | Estado final de todos | Inventory updated | Complete business transaction |

### 🔧 Flujo E: Admin Inventory Management (Gestión Administrativa)

| Paso | Acción | Endpoint | Input | Expected Output | Validación Post-Test |
|------|--------|----------|-------|-----------------|---------------------|
| E1 | **Revisar reservas activas** | `GET /api/v1/inventory/stock/{id}` | Corsair RAM check | Reserved stock levels | Active reservations identified |
| E2 | **Liberar reserva administrativa** | `POST /api/v1/inventory/release` | reason: "admin", reservation_id | `200 OK` | Admin override successful |
| E3 | **Ajuste de inventario** | `PUT /api/v1/inventory/stock` | adjustment_type: "correction" | `200 OK` | Manual inventory correction |
| E4 | **Auditoría de consistencia** | `GET /api/v1/inventory/stock/{id}` | Final state verification | Consistent totals | Data integrity confirmed |

### 🔄 Flujo F: Idempotency & Error Recovery

| Paso | Acción | Endpoint | Input | Expected Output | Validación Post-Test |
|------|--------|----------|-------|-----------------|---------------------|
| F1 | **Primera reserva** | `POST /api/v1/inventory/reserve` | request_id: "idempotent-test-123" | `201 Created` | Reservation created |
| F2 | **Repetir reserva** | `POST /api/v1/inventory/reserve` | Same request_id | Same response (cached) | No duplicate reservation |
| F3 | **Error recovery** | `POST /api/v1/inventory/reserve` | Insufficient stock | `409 Conflict` | Proper error handling |
| F4 | **Retry con nuevo ID** | `POST /api/v1/inventory/reserve` | New request_id, sufficient stock | `201 Created` | Recovery successful |

---

### 📋 Checklist de Flujos Completos

#### ✅ Pre-requisitos
- [ ] Servidor corriendo (`make run`)
- [ ] Base de datos con seed data
- [ ] VSCode con REST Client extension
- [ ] Files: `complete-workflows.http` ready

#### 🎯 Flujos de Negocio Core
- [ ] **Flujo A (E-commerce)**: Orden completa exitosa
- [ ] **Flujo B (Abandonment)**: Carrito abandonado y stock liberado  
- [ ] **Flujo C (Warehouse)**: Operaciones de almacén completas
- [ ] **Flujo D (B2B)**: Orden corporativa en lote
- [ ] **Flujo E (Admin)**: Gestión administrativa

#### 🔧 Flujos Técnicos
- [ ] **Flujo F (Idempotency)**: Manejo de duplicados
- [ ] **Error Handling**: Casos de error y recovery
- [ ] **Concurrency**: Múltiples usuarios simultáneos
- [ ] **Performance**: Tiempo de respuesta bajo carga

#### 📊 Validaciones de Consistencia
- [ ] Stock totals siempre cuadran
- [ ] Reserved + Available = Total Stock
- [ ] No overselling bajo ninguna circunstancia
- [ ] Idempotency funciona correctamente
- [ ] Error responses son consistentes

### 🎯 Success Criteria

| Flujo | Success Rate | Response Time | Data Consistency |
|-------|-------------|---------------|------------------|
| A (E-commerce) | 100% | < 200ms avg | ✅ Perfect |
| B (Abandonment) | 100% | < 100ms avg | ✅ Perfect |
| C (Warehouse) | 100% | < 300ms avg | ✅ Perfect |
| D (B2B Bulk) | 100% | < 500ms avg | ✅ Perfect |
| E (Admin) | 100% | < 150ms avg | ✅ Perfect |
| F (Technical) | 100% | < 100ms avg | ✅ Perfect |

### 🔗 Quick Links

- **Run Workflows**: Open `test-api/complete-workflows.http` in VSCode
- **Validation Tests**: Open `test-api/validation-errors.http`
- **API Documentation**: http://localhost:8080/docs
- **Health Check**: http://localhost:8080/health

---

## ⚡ Pruebas de Rendimiento

### Carga Básica

| Test | Requests/sec | Duration | Success Rate | Avg Response Time |
|------|-------------|----------|--------------|-------------------|
| Health Check | 1000 rps | 60s | > 99.9% | < 10ms |
| Get Stock | 500 rps | 60s | > 99.5% | < 50ms |
| Reserve Stock | 100 rps | 60s | > 99% | < 200ms |
| Batch Reserve | 50 rps | 60s | > 95% | < 500ms |

### Carga de Estrés

| Escenario | Users | Duration | Ramp-up | Success Criteria |
|-----------|-------|----------|---------|------------------|
| Peak Traffic | 1000 concurrent | 300s | 60s | > 95% success, < 2s p95 |
| Sustained Load | 500 concurrent | 1800s | 120s | > 99% success, < 1s p95 |
| Burst Load | 2000 concurrent | 60s | 10s | > 90% success, no errors |

---

## 🚨 Casos Edge

### Concurrencia

| Test ID | Escenario | Setup | Expected Behavior |
|---------|-----------|-------|-------------------|
| E001 | Reservas simultáneas | 10 requests para mismo producto al mismo tiempo | Solo las que caben en stock |
| E002 | Stock insuficiente | Múltiples reservas > stock disponible | Error 409 para excedentes |
| E003 | Liberación simultánea | Múltiples releases de misma reserva | Solo primera exitosa |
| E004 | Update durante reserva | Update stock mientras hay reservas activas | Bloqueo optimista |

### Límites del Sistema

| Test ID | Escenario | Input | Expected |
|---------|-----------|-------|----------|
| E005 | Payload muy grande | JSON > 1MB | 413 Request Entity Too Large |
| E006 | Demasiados headers | 100+ HTTP headers | Request procesado o 431 |
| E007 | URL muy larga | URL > 8KB | 414 URI Too Long |
| E008 | Conexiones simultáneas | 1000+ concurrent connections | Rate limiting |

### Casos Extremos de Datos

| Test ID | Escenario | Input | Expected |
|---------|-----------|-------|----------|
| E009 | Unicode en strings | Reason con emojis/caracteres especiales | Procesado correctamente |
| E010 | Números muy grandes | quantity: Number.MAX_SAFE_INTEGER | Validation error |
| E011 | Strings muy largos | reason: 10000 caracteres | 400 Bad Request |
| E012 | Metadata complejo | Nested objects en metadata | Procesado o rechazado |

---

## ✅ Checklist de Testing

### Pre-Testing

- [ ] Servidor corriendo en puerto 8080
- [ ] Base de datos seed ejecutada
- [ ] Logs habilitados para debugging
- [ ] REST Client extension instalada
- [ ] Variables de entorno configuradas

### Testing Básico

- [ ] Todos los endpoints responden 200/201 con datos válidos
- [ ] Validaciones funcionan correctamente (400 errors)
- [ ] Casos de not found funcionan (404 errors)
- [ ] Headers requeridos son validados
- [ ] JSON malformado es rechazado

### Testing Avanzado

- [ ] Idempotencia funciona correctamente
- [ ] Timeouts de reserva son respetados
- [ ] Stock consistency mantenida
- [ ] Batch operations funcionan
- [ ] Metadata es persistida correctamente

### Testing de Integración

- [ ] Flujos completos funcionan end-to-end
- [ ] Base de datos refleja cambios correctos
- [ ] Logs son generados apropiadamente
- [ ] Métricas son registradas
- [ ] Error handling es consistente

### Testing de Rendimiento

- [ ] Response times dentro de SLA
- [ ] Sistema maneja carga esperada
- [ ] Memory/CPU usage aceptable
- [ ] No memory leaks
- [ ] Graceful degradation bajo carga

---

## 📝 Formato de Reporte de Bugs

```markdown
### Bug Report: [TÍTULO_CORTO]

**Test ID:** [ej: R001]
**Severity:** [Critical/High/Medium/Low]
**Environment:** [local/staging/prod]

**Steps to Reproduce:**
1. Step 1
2. Step 2
3. Step 3

**Expected Result:**
[Descripción del resultado esperado]

**Actual Result:**
[Descripción del resultado actual]

**Request:**
```http
POST /api/v1/inventory/reserve
Content-Type: application/json
X-Request-ID: test-123

{
  "product_id": "...",
  "quantity": 1
}
```

**Response:**
```json
{
  "error": "...",
  "message": "..."
}
```

**Additional Notes:**
[Cualquier información adicional]
```

---

## 📊 Métricas de Éxito

| Métrica | Target | Actual | Status |
|---------|--------|---------|---------|
| Test Coverage | > 95% | ___ % | ⚠️ |
| Success Rate | > 99% | ___ % | ⚠️ |
| P95 Response Time | < 200ms | ___ ms | ⚠️ |
| Error Rate | < 1% | ___ % | ⚠️ |
| Bug Density | < 1 bug/endpoint | ___ | ⚠️ |

---

**Última actualización:** `2025-10-06`  
**Versión API:** `v1`  
**Total Tests:** `XX passed, YY failed, ZZ skipped`