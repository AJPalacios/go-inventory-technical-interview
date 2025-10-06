.PHONY: help server test sqlc migrateup migratedown createdb dropdb install-migrate seed

## help: Display available commands
help:
	@echo 'Available commands:'
	@echo '  make createdb      - Create the database'
	@echo '  make dropdb        - Drop the database'
	@echo '  make seed          - Load sample data into database'
	@echo '  make server        - Build and run the server'
	@echo '  make test          - Run tests'
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

## server: Build and run the server
server:
	go run main.go

## test: Run all tests
test:
	go test -v ./...

## sqlc: Generate Go code from SQL
sqlc:
	sqlc generate

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

.DEFAULT_GOAL := help
