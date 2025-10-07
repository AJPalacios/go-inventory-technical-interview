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
git clone <repository-url>
cd inventory
go mod tidy

# 2. Create and setup database
make createdb

# 3. Start the server
make server
```

**Server starts at**: `http://localhost:8080`

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
  -d '{
    "product_id": "e08e3e7e-9126-49e4-9caf-63885a07bd78",
    "quantity": 2,
    "reason": "quickstart_test"
  }'
```

### Get Stock Info
```bash
curl http://localhost:8080/api/v1/inventory/e08e3e7e-9126-49e4-9caf-63885a07bd78
```

### Release Reservation
```bash
# Use reservation_id from reserve response
curl -X POST http://localhost:8080/api/v1/inventory/release \
  -H "Content-Type: application/json" \
  -d '{
    "reservation_id": "<reservation_id_from_reserve>",
    "reason": "quickstart_test_complete"
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
# Open API documentation (when implemented)
curl http://localhost:8080/docs
```

---

## 🏗️ Development Commands

```bash
# Database operations
make createdb       # Create database with sample data
make dropdb         # Drop database
make migrateup      # Apply migrations
make migratedown    # Rollback migrations

# Development
make server         # Start development server
make build          # Build binary
make clean          # Clean build artifacts

# Testing
make test           # Run all tests
make test-coverage  # Run tests with coverage
make test-race      # Run tests with race detection

# Code quality
make fmt            # Format code
make lint           # Run linter
make vet            # Run go vet
```

---

## 📊 Sample Data

The system comes with pre-loaded sample data:

| Product | UUID | Initial Stock |
|---------|------|---------------|
| **Premium Headphones** | `e08e3e7e-9126-49e4-9caf-63885a07bd78` | 150 units |
| **Wireless Mouse** | `f19f4f8f-a237-5a55-b468-74996b18ce89` | 75 units |
| **Mechanical Keyboard** | `c27c5c9c-b348-6b66-c579-85aa7c29df9a` | 50 units |
| **USB-C Cable** | `d38d6d0d-c459-7c77-d68a-96bb8d3aef0b` | 200 units |
| **Laptop Stand** | `a15a1a1a-d56a-8d88-e79b-a7cc9e4bf1c` | 25 units |

---

## 🔍 Troubleshooting

### Server won't start
```bash
# Check if port 8080 is in use
lsof -i :8080

# Use different port
SERVER_PORT=8081 make server
```

### Database issues
```bash
# Reset database
make dropdb
make createdb

# Check database file
ls -la inventory.db
```

### Build issues
```bash
# Clean and rebuild
make clean
go mod tidy
make build
```

---

## 📚 Next Steps

- **API Documentation**: See `docs/API_SPECIFICATION.md`
- **Architecture Guide**: See `ARCHITECTURE.md`
- **Production Deployment**: See `docs/DEPLOYMENT_GUIDE.md`
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