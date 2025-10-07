# API Testing with HTTP Files

Este directorio contiene archivos `.http` para probar los endpoints de la API de inventario usando la extensión **REST Client** de VSCode.

## 📋 Prerequisitos

1. **VSCode REST Client Extension**
   ```
   Nombre: REST Client
   ID: humao.rest-client
   Enlace: https://marketplace.visualstudio.com/items\?itemName\=humao.rest-client
   ```

2. **Servidor corriendo**
   ```bash
   make run
   # O directamente:
   go run cmd/server/main.go
   ```

## 📁 Archivos Disponibles

### `inventory-api.http`
Archivo completo con todos los endpoints de la API:
- ✅ Health check básico
- ✅ Operaciones de stock (consulta, actualización)
- ✅ Reservas de stock (simple, batch, con timeout)
- ✅ Liberación de reservas
- ✅ Pruebas de idempotencia
- ✅ Pruebas de manejo de errores
- ✅ Workflows completos (orden, carrito abandonado, restock)

### `validation-errors.http`
Suite completa de pruebas de validación:
- ✅ Todos los errores de validación posibles por endpoint
- ✅ Campos requeridos faltantes
- ✅ Formatos inválidos (UUID, rangos, etc.)
- ✅ Valores fuera de rango (min/max)
- ✅ Validaciones de tipo `oneof`
- ✅ Validación de arrays (dive)
- ✅ Ejemplos de múltiples errores simultáneos
- ✅ Formato de respuesta de error esperado

## 🚀 Cómo Usar

### Método 1: VSCode REST Client

1. **Instalar la extensión:**
   - Abrir VSCode
   - Ir a Extensions (Cmd+Shift+X / Ctrl+Shift+X)
   - Buscar "REST Client"
   - Instalar "REST Client" by Huachao Mao

2. **Abrir archivo HTTP:**
   - Abrir `test-api/inventory-api.http`
   - Verás que aparece "Send Request" sobre cada petición

3. **Ejecutar requests:**
   - Click en "Send Request" sobre la petición deseada
   - O usar atajo: `Cmd+Alt+R` (Mac) / `Ctrl+Alt+R` (Windows/Linux)
   - La respuesta aparecerá en un panel lateral

4. **Variables dinámicas:**
   - `{{$guid}}` - Genera un UUID único
   - `{{$timestamp}}` - Timestamp actual
   - `{{$randomInt}}` - Número aleatorio
   - Variables personalizadas al inicio del archivo

### Método 2: cURL (Alternativo)

Si prefieres usar la terminal:

```bash
# Health check
curl -X GET http://localhost:8080/health

# Get stock
curl -X GET http://localhost:8080/api/v1/inventory/stock/prod-001

# Reserve stock
curl -X POST http://localhost:8080/api/v1/inventory/reserve \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: req-$(uuidgen)" \
  -d '{
    "product_id": "prod-001",
    "quantity": 2,
    "request_id": "req-'$(uuidgen)'",
    "timeout_seconds": 300,
    "reason": "Test order"
  }'
```

## 📊 Endpoints Disponibles

| Método | Endpoint | Descripción |
|--------|----------|-------------|
| `GET` | `/health` | Health check básico |
| `GET` | `/api/v1/inventory/stock/:productId` | Consultar stock de producto |
| `POST` | `/api/v1/inventory/reserve` | Reservar stock |
| `POST` | `/api/v1/inventory/batch/reserve` | Reservar múltiples productos |
| `POST` | `/api/v1/inventory/release` | Liberar reserva |
| `PUT` | `/api/v1/inventory/stock` | Actualizar stock |

## 🔧 Configuración

### Variables de Entorno

Puedes modificar las variables al inicio de `inventory-api.http`:

```http
@baseUrl = http://localhost:8080
@apiVersion = /api/v1
@productId = e08e3e7e-9126-49e4-9caf-63885a07bd78  # Teclado Keychron K2
@requestId = {{$guid}}
@reservationId = # Usar ID real de respuesta de reserva
```

### Productos de Prueba (Datos Reales del Seed)

La base de datos seed incluye estos productos con sus UUIDs reales:

| Product ID (UUID) | Nombre | Stock Disponible | Stock Reservado |
|-------------------|--------|------------------|------------------|
| `2d70d1dc-cd3a-4f40-afb0-52e16445bf36` | Laptop HP Pavilion 15 | 25 | 5 |
| `2da3b8d3-69f1-46e6-a068-874532d5126a` | Mouse Logitech MX Master 3 | 150 | 10 |
| `e08e3e7e-9126-49e4-9caf-63885a07bd78` | Teclado Mecánico Keychron K2 | 80 | 15 |
| `fc39adf6-784c-43f3-bb0d-9ed79613dd21` | Monitor Dell 27" 4K | 30 | 0 |
| `cf43ddc3-c4da-4a98-b011-67b33223fae1` | Webcam Logitech C920 | 45 | 5 |
| `47569eb2-fe19-43cb-929d-aedfd59dc199` | Audífonos Sony WH-1000XM4 | 60 | 10 |
| `f7d85ff3-6dbf-4ee8-bd61-54453610e441` | SSD Samsung 1TB | 100 | 0 |
| `834004f0-f683-4e96-ae6b-bb6673869d24` | RAM Corsair 16GB DDR4 | 200 | 20 |
| `cbb6a942-8687-4dd0-85ba-82f102f25ce1` | Hub USB-C Anker | 75 | 5 |
| `00907a59-5b4b-4432-8c49-e8bca4683799` | Cable HDMI 4K 2m | 300 | 0 |

**Producto Principal para Tests:** `e08e3e7e-9126-49e4-9caf-63885a07bd78` (Teclado Keychron K2)

### Reservas Existentes

| Reservation ID | Request ID | Producto | Cantidad | Status | Expira |
|----------------|------------|----------|-----------|--------|---------|
| `11171f8d-a6a4-42d3-9ab1-c6d3d829c83e` | `req-001` | Laptop HP | 2 | confirmed | +7 días |
| `1e30b07b-03cc-4b8d-9324-e69a316f0d5e` | `req-002` | Mouse Logitech | 5 | confirmed | +5 días |
| `b97bdd7a-29bb-498a-8dcc-a523ea4cedd0` | `req-003` | Teclado Keychron | 3 | pending | +2 días |
| `846c4180-de70-410f-bc4b-4d287b123f2f` | `req-004` | Audífonos Sony | 10 | confirmed | +10 días |
| `ef9a3dc8-860f-4374-ab3c-4317f9f30d8c` | `req-005` | RAM Corsair | 15 | pending | +3 días |

## �� Ejemplos de Uso

### 1. Flujo Completo de Orden

```http
# 1. Reservar stock
POST {{baseUrl}}{{apiVersion}}/inventory/reserve
Content-Type: application/json
X-Request-ID: req-order-{{$guid}}

{
  "product_id": "e08e3e7e-9126-49e4-9caf-63885a07bd78",
  "quantity": 2,
  "request_id": "{{$guid}}",
  "timeout_seconds": 600,
  "reason": "Customer order",
  "metadata": {
    "order_id": "ORD-{{$timestamp}}",
    "customer_id": "CUST-12345"
  }
}

# 2. Verificar stock
GET {{baseUrl}}{{apiVersion}}/inventory/stock/e08e3e7e-9126-49e4-9caf-63885a07bd78

# 3. Completar orden (copiar reservation_id de respuesta anterior)
POST {{baseUrl}}{{apiVersion}}/inventory/release
Content-Type: application/json

{
  "reservation_id": "USAR_ID_DE_RESPUESTA_ANTERIOR",
  "reason": "purchased",
  "request_id": "{{$guid}}"
}
```

### 2. Test de Idempotencia

```http
# Primera petición
POST {{baseUrl}}{{apiVersion}}/inventory/reserve
Content-Type: application/json
X-Request-ID: req-idempotency-1

{
  "product_id": "e08e3e7e-9126-49e4-9caf-63885a07bd78",
  "quantity": 1,
  "request_id": "idempotency-test-123",
  "timeout_seconds": 300,
  "reason": "Idempotency test"
}

# Segunda petición con mismo request_id (debe devolver misma respuesta)
POST {{baseUrl}}{{apiVersion}}/inventory/reserve
Content-Type: application/json
X-Request-ID: req-idempotency-2

{
  "product_id": "e08e3e7e-9126-49e4-9caf-63885a07bd78",
  "quantity": 1,
  "request_id": "idempotency-test-123",
  "timeout_seconds": 300,
  "reason": "Idempotency test duplicate"
}
```

### 3. Actualizar Stock

```http
# Agregar inventario
PUT {{baseUrl}}{{apiVersion}}/inventory/stock
Content-Type: application/json
X-Request-ID: req-restock-{{$guid}}

{
  "product_id": "e08e3e7e-9126-49e4-9caf-63885a07bd78",
  "new_stock": 50,
  "adjustment_type": "restock",
  "reason": "New shipment received",
  "reference": "PO-{{$timestamp}}",
  "request_id": "{{$guid}}",
  "metadata": {
    "supplier": "Keychron Inc",
    "warehouse": "WH-001",
    "batch": "KB-{{$timestamp}}"
  }
}
```

## 🐛 Debugging

### Ver logs del servidor

```bash
# Terminal donde corre el servidor
make run

# Los logs aparecerán en formato JSON:
{"level":"INFO","message":"HTTP request completed","status":200,...}
```

### Verificar base de datos

```bash
# Conectar a SQLite
sqlite3 inventory.db

# Ver productos
SELECT * FROM products;

# Ver inventario
SELECT * FROM inventory_items;

# Ver reservas
SELECT * FROM reservations;
```

### Headers útiles

```http
X-Request-ID: custom-request-id-123  # Para tracking
Content-Type: application/json       # Requerido para POST/PUT
```

## 📚 Recursos

- [REST Client Documentation](https://github.com/Huachao/vscode-restclient)
- [HTTP File Syntax](https://github.com/Huachao/vscode-restclient#usage)
- [Variables](https://github.com/Huachao/vscode-restclient#variables)
- [Environments](https://github.com/Huachao/vscode-restclient#environments)

## 🤝 Contribuir

Para agregar nuevos tests:

1. Editar `inventory-api.http`
2. Agregar sección con `###` como separador
3. Documentar con comentarios `#`
4. Incluir ejemplo de payload
5. Actualizar este README si es necesario

---

**Nota:** Asegúrate de que el servidor esté corriendo en `http://localhost:8080` antes de ejecutar los tests.
