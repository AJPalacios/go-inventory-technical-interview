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
@productId = prod-001
```

### Productos de Prueba

La base de datos seed incluye estos productos:

- `prod-001` - Laptop
- `prod-002` - Mouse
- `prod-003` - Keyboard
- `prod-004` - Monitor
- `prod-005` - Headphones

## �� Ejemplos de Uso

### 1. Flujo Completo de Orden

```http
# 1. Reservar stock
POST {{baseUrl}}{{apiVersion}}/inventory/reserve
Content-Type: application/json

{
  "product_id": "prod-001",
  "quantity": 2,
  "request_id": "{{$guid}}",
  "timeout_seconds": 600,
  "reason": "Customer order"
}

# 2. Verificar stock
GET {{baseUrl}}{{apiVersion}}/inventory/stock/prod-001

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

{
  "product_id": "prod-001",
  "quantity": 1,
  "request_id": "idempotency-test-123",
  "timeout_seconds": 300,
  "reason": "Idempotency test"
}

# Segunda petición con mismo request_id (debe devolver misma respuesta)
POST {{baseUrl}}{{apiVersion}}/inventory/reserve
Content-Type: application/json

{
  "product_id": "prod-001",
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

{
  "product_id": "prod-001",
  "new_stock": 50,
  "adjustment_type": "restock",
  "reason": "New shipment",
  "reference": "PO-{{$timestamp}}",
  "request_id": "{{$guid}}"
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
