# 🚀 Quick Start Guide - Inventory Management System

**Get up and running in 5 minutes**

---

## ⚡ Prerequisites

```bash
# Required tools
go version      # Go 1.21+
make --version  # Build automation
curl --version  # API testing
```

---

## 🏃‍♂️ 1-Minute Setup

```bash
# 1. Clone and setup dependencies
cd inventory
make mod

# 2. Setup complete development environment
make dev

# 3. Start the server
make run
```

**Server starts at**: `http://localhost:8080`

> 💡 **Pro tip**: `make dev` sets up everything automatically - dependencies, database, schema, sample data, code generation, and builds the binary!

---

## ✅ Verify Installation

```bash
# Health check
curl http://localhost:8080/health

# Expected response:
{
  "status": "healthy",
  "timestamp": "2025-10-06T15:30:00Z",
  "version": "1.0.0",
  "checks": {
    "database": {
      "status": "healthy",
      "latency_ms": 2
    }
  }
}
```

---

## 🧪 Test Core Functionality

### Reserve Stock
```bash
curl -X POST http://localhost:8080/api/v1/inventory/reserve \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: quickstart-$(date +%s)" \
  -d '{
    "product_id": "e08e3e7e-9126-49e4-9caf-63885a07bd78",
    "quantity": 2,
    "request_id": "quickstart-test-'$(date +%s)'",
    "reason": "quickstart_test",
    "timeout_seconds": 600
  }'
```

### Get Stock Info
```bash
curl http://localhost:8080/api/v1/inventory/stock/e08e3e7e-9126-49e4-9caf-63885a07bd78
```

### Release Reservation
```bash
# Use reservation_id from reserve response above
curl -X POST http://localhost:8080/api/v1/inventory/release \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: quickstart-release-$(date +%s)" \
  -d '{
    "reservation_id": "<reservation_id_from_reserve>",
    "reason": "purchased",
    "request_id": "quickstart-release-'$(date +%s)'"
  }'
```

---

## 📱 Interactive Testing

### Using VS Code REST Client

1. Install **REST Client** extension in VS Code
2. Open `test-api/inventory-api.http`
3. Click "Send Request" on any HTTP request

### Using the Web Interface

```bash
# Interactive Swagger UI
open http://localhost:8080/docs

# OpenAPI specification
curl http://localhost:8080/openapi.json

# Versioned API docs
curl http://localhost:8080/api/v1/docs
```

---

## 🏗️ Development Commands

```bash
# Database operations
make createdb       # Create database and apply migrations
make dropdb         # Drop the database
make seed          # Load sample data into database
make migrateup     # Run migrations up (apply schema)
make migratedown   # Run migrations down (rollback)

# Development
make run           # Build and run the server
make build         # Build the application
make clean         # Clean build artifacts
make dev           # Setup complete development environment

# Testing
make test          # Run all tests with coverage

# Code quality
make format        # Format code with go fmt and goimports
make lint          # Run golangci-lint

# Dependencies
make mod           # Download and tidy dependencies
make sqlc          # Generate code from SQL
```

---

## 📊 Sample Data

The system comes with pre-loaded sample data (loaded with `make seed`):

| Product | UUID | Available/Reserved Stock |
|---------|------|-------------------------|
| **Laptop HP Pavilion 15** | `2d70d1dc-cd3a-4f40-afb0-52e16445bf36` | 25/5 units |
| **Keychron K2 Keyboard** | `e08e3e7e-9126-49e4-9caf-63885a07bd78` | 80/15 units |
| **Logitech MX Master 3** | `2da3b8d3-69f1-46e6-a068-874532d5126a` | 150/10 units |
| **Dell 27" 4K Monitor** | `fc39adf6-784c-43f3-bb0d-9ed79613dd21` | 30/0 units |
| **Samsung 1TB SSD** | `f7d85ff3-6dbf-4ee8-bd61-54453610e441` | 100/0 units |

---

## 🔍 Troubleshooting

### Server won't start
```bash
# Check if port 8080 is in use
lsof -i :8080

# Use different port (set in app.env)
echo "SERVER_PORT=8081" >> app.env
make run
```

### Database issues
```bash
# Reset database completely
make dropdb
make createdb
make seed

# Check database file
ls -la inventory.db
```

### Build issues
```bash
# Clean and rebuild everything
make clean
make mod
make build
```

---

## 📚 Next Steps

- **Interactive API Docs**: Visit http://localhost:8080/docs
- **Architecture Guide**: See `/ARCHITECTURE.md`
- **API Testing Suite**: Explore `test-api/` directory
- **Complete Test Matrix**: See `test-api/TEST_MATRIX.md`
- **Production Setup**: See `docs/DEPLOYMENT_GUIDE.md`
- **Full Test Suite**: Run `make test` for comprehensive validation

---

## 🎯 Quick Performance Test

```bash
# Run concurrent operations test
go test ./internal/repository/ -run "Concurrent" -v

# Expected output:
=== RUN   TestVersionConflictHandling
--- PASS: TestVersionConflictHandling (1.62s)  # 1000+ concurrent ops
PASS
```

**🎉 Success!** Your inventory system is ready for development and testing.

---

*For detailed documentation and advanced features, see the complete documentation in the `docs/` directory.*