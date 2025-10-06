.PHONY: help build run clean test lint format mod dev createdb dropdb seed sqlc migrateup migratedown install-migrate
.DEFAULT_GOAL := help

# Build configuration
BINARY_NAME=inventory-server
BUILD_DIR=bin
SERVER_PATH=cmd/server

## help: Display available commands
help:
	@echo 'Available commands:'
	@echo '  make build         - Build the application'
	@echo '  make run           - Build and run the server'
	@echo '  make clean         - Clean build artifacts'
	@echo '  make test          - Run all tests with coverage'
	@echo '  make lint          - Run golangci-lint'
	@echo '  make format        - Format code with go fmt and goimports'
	@echo '  make mod           - Download and tidy dependencies'
	@echo '  make dev           - Setup development environment'
	@echo '  make createdb      - Create the database'
	@echo '  make dropdb        - Drop the database'
	@echo '  make seed          - Load sample data into database'
	@echo '  make sqlc          - Generate code from SQL'
	@echo '  make migrateup     - Run migrations up (apply schema)'
	@echo '  make migratedown   - Run migrations down (rollback)'
	@echo '  make install-migrate - Install migrate CLI tool'

## create db: Create the database and apply migrations
createdb:
	@echo "Creating SQLite database..."
	@rm -f inventory.db
	@touch inventory.db
	@echo "Applying migrations..."
	@sqlite3 inventory.db < db/migrations/000001_init_schema.up.sql
	@echo "✓ Database created: inventory.db"
	@echo "✓ Schema initialized successfully"

## dropdb: Drop the database
dropdb:
	@echo "Dropping database..."
	@rm -f inventory.db
	@echo "✓ Database dropped"

## seed: Load sample data into database
seed:
	@if [ ! -f inventory.db ]; then \
		echo "Error: Database doesn't exist. Run 'make createdb' first"; \
		exit 1; \
	fi
	@echo "Loading sample data..."
	@sqlite3 inventory.db < db/seed.sql
	@echo "✓ Sample data loaded successfully"
	@echo "✓ Products: 10, Inventory items: 10, Reservations: 5"

## build: Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) ./$(SERVER_PATH)
	@echo "✓ Build completed: $(BUILD_DIR)/$(BINARY_NAME)"

## run: Build and run the server
run: build
	@echo "Starting server..."
	@./$(BUILD_DIR)/$(BINARY_NAME)

## clean: Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@go clean -cache -modcache -testcache
	@rm -f coverage.out coverage.html
	@echo "✓ Clean completed"

## test: Run all tests with coverage
test:
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Tests completed - Coverage report: coverage.html"

## lint: Run golangci-lint
lint:
	@echo "Running linter..."
	@golangci-lint run --config .golangci.yml
	@echo "✓ Linting completed"

## format: Format code with go fmt and goimports
format:
	@echo "Formatting code..."
	@go fmt ./...
	@goimports -w .
	@echo "✓ Code formatted"

## mod: Download and tidy dependencies
mod:
	@echo "Managing dependencies..."
	@go mod download
	@go mod tidy
	@go mod verify
	@echo "✓ Dependencies updated"

## dev: Setup development environment
dev: mod createdb seed sqlc format build
	@echo "✓ Development environment ready"
	@echo "✓ Run 'make run' to start the server"

## sqlc: Generate Go code from SQL
sqlc:
	@echo "Generating SQLC code..."
	@sqlc generate
	@echo "✓ SQLC code generated"

## migrateup: Run database migrations up (apply schema)
migrateup:
	@if [ ! -f inventory.db ]; then \
		echo "Database doesn't exist. Creating..."; \
		touch inventory.db; \
	fi
	@echo "Applying migrations..."
	@sqlite3 inventory.db < db/migrations/000001_init_schema.up.sql
	@echo "✓ Migrations applied successfully"

## migratedown: Run database migrations down (rollback)
migratedown:
	@if [ ! -f inventory.db ]; then \
		echo "Error: Database doesn't exist"; \
		exit 1; \
	fi
	@echo "Rolling back migrations..."
	@sqlite3 inventory.db < db/migrations/000001_init_schema.down.sql
	@echo "✓ Migrations rolled back successfully"

## install-migrate: Install migrate CLI tool (optional - for production use)
install-migrate:
	@echo "Installing golang-migrate..."
	@go install -tags 'sqlite3' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@echo "✓ migrate installed successfully"
	@echo "Note: Current Makefile uses direct sqlite3 commands (simpler)"
