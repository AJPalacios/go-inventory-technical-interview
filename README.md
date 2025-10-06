# Inventory Management System

Sistema de gestión de inventario con Go, Gin y SQLite.

## 🚀 Inicio Rápido

### 1. Configuración Inicial

```bash
# Clonar el repositorio
git clone <repo-url>
cd inventory

# Instalar dependencias
go mod tidy
```

### 2. Configurar Variables de Entorno

Edita el archivo `app.env`:

```env
SERVER_PORT=8080
DATABASE_PATH=./inventory.db
APP_ENVIRONMENT=development
APP_LOG_LEVEL=debug
```

### 3. Crear la Base de Datos

```bash
# Crear la base de datos y aplicar migraciones
make createdb
```

### 4. Ejecutar el Servidor

```bash
make server
```

El servidor estará disponible en: `http://localhost:8080`

## 📋 Comandos Make Disponibles

### Base de Datos

```bash
# Crear base de datos con schema inicial
make createdb

# Eliminar la base de datos
make dropdb

# Aplicar migraciones (si la DB ya existe)
make migrateup

# Revertir migraciones (rollback)
make migratedown
```

### Desarrollo

```bash
# Ejecutar el servidor
make server

# Ejecutar tests
make test

# Generar código desde SQL queries con sqlc
make sqlc
```

### Utilidades

```bash
# Ver todos los comandos disponibles
make help

# Instalar herramientas de desarrollo (opcional)
make install-migrate
```

## 🗄️ Estructura de la Base de Datos

El sistema incluye 4 tablas principales:

1. **products** - Catálogo de productos
2. **inventory_items** - Control de inventario con optimistic locking
3. **reservations** - Sistema de reservas con estados
4. **idempotency_keys** - Soporte para operaciones idempotentes

## 📁 Estructura del Proyecto

```
inventory/
├── api/                    # API handlers y rutas HTTP
│   ├── server.go          # Configuración del servidor
│   └── health.go          # Endpoints de health check
├── db/                    # Base de datos
│   ├── migrations/        # Migraciones SQL
│   └── query/            # Queries SQL para sqlc
├── util/                  # Utilidades
│   ├── config.go         # Configuración con Viper
│   └── logger.go         # Logger con Zap
├── main.go               # Punto de entrada
├── app.env              # Variables de configuración
├── Makefile            # Comandos de automatización
└── sqlc.yaml          # Configuración de sqlc
```

## 🔧 Migraciones

### ¿Qué son las migraciones?

Las migraciones son archivos SQL versionados que permiten:
- ✅ Aplicar cambios al schema de la DB de forma controlada
- ✅ Revertir cambios si algo sale mal
- ✅ Mantener un historial de cambios
- ✅ Sincronizar el schema entre entornos

### Archivos de Migración

Cada migración tiene 2 archivos:

- `000001_init_schema.up.sql` - Aplica los cambios (CREATE TABLE, etc.)
- `000001_init_schema.down.sql` - Revierte los cambios (DROP TABLE, etc.)

### Uso de Migraciones

#### Crear nueva base de datos desde cero:
```bash
make createdb
```

#### Si la base de datos ya existe:
```bash
# Aplicar migraciones
make migrateup

# Revertir migraciones (cuidado: borra datos!)
make migratedown

# Rehacer migraciones (down + up)
make migratedown
make migrateup
```

#### Recrear la base de datos desde cero:
```bash
make dropdb
make createdb
```

### Verificar el Estado de la DB

```bash
# Ver las tablas
sqlite3 inventory.db ".tables"

# Ver el schema de una tabla
sqlite3 inventory.db ".schema products"

# Ver todos los schemas
sqlite3 inventory.db ".schema"

# Ejecutar consultas
sqlite3 inventory.db "SELECT * FROM products;"
```

## 🔌 API Endpoints

### Health Check
```bash
GET /health
```

### API v1
```bash
GET /api/v1/ping
```

## 🧪 Testing

```bash
# Ejecutar todos los tests
make test

# Ejecutar tests de un paquete específico
go test -v ./util
go test -v ./api
```

## 📝 Notas

- El proyecto usa **SQLite3** para simplicidad
- Las migraciones se aplican directamente con `sqlite3` (no requiere herramientas adicionales)
- Para proyectos más grandes, considera usar `golang-migrate` (ver `make install-migrate`)

## 🛠️ Desarrollo

### Agregar nuevos endpoints

1. Añade tus handlers en `api/`
2. Registra las rutas en `api/server.go`

### Agregar queries SQL

1. Crea archivos `.sql` en `db/query/`
2. Ejecuta `make sqlc` para generar código Go
3. Usa el código generado en tus handlers

## 📦 Dependencias Principales

- **Gin** - Framework web
- **Zap** - Logger estructurado
- **Viper** - Configuración
- **SQLite3** - Base de datos
- **sqlc** - Generador de código SQL type-safe

## 📄 Licencia

[Tu licencia aquí]
