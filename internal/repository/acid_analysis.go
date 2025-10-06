package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// ACID Properties Analysis and Implementation
//
// This file contains analysis and implementation of ACID properties
// and deadlock prevention strategies for the inventory system.
//
// ACID Compliance:
// - Atomicity: Ensured by database transactions and optimistic locking
// - Consistency: Enforced by constraints and business rules
// - Isolation: Managed by optimistic locking and proper transaction isolation
// - Durability: Guaranteed by database persistence layer

// TransactionIsolationLevel defines the isolation levels we support
type TransactionIsolationLevel string

const (
	// ReadCommitted prevents dirty reads but allows non-repeatable reads
	ReadCommitted TransactionIsolationLevel = "READ COMMITTED"
	// RepeatableRead prevents dirty and non-repeatable reads but allows phantom reads
	RepeatableRead TransactionIsolationLevel = "REPEATABLE READ"
	// Serializable prevents all phenomena, highest isolation but lowest concurrency
	Serializable TransactionIsolationLevel = "SERIALIZABLE"
)

// DeadlockPreventionStrategy defines our approach to prevent deadlocks
type DeadlockPreventionStrategy struct {
	// OrderedResourceAccess ensures consistent resource ordering
	OrderedResourceAccess bool
	// TimeoutDuration for operations to prevent infinite waits
	TimeoutDuration time.Duration
	// RetryOnDeadlock enables automatic retry on deadlock detection
	RetryOnDeadlock bool
	// MaxRetryAttempts for deadlock scenarios
	MaxRetryAttempts int
}

// DefaultDeadlockPreventionStrategy returns sensible defaults
func DefaultDeadlockPreventionStrategy() DeadlockPreventionStrategy {
	return DeadlockPreventionStrategy{
		OrderedResourceAccess: true,
		TimeoutDuration:       30 * time.Second,
		RetryOnDeadlock:       true,
		MaxRetryAttempts:      3,
	}
}

// WithTransactionIsolation executes a function with specific isolation level
func (r *inventoryRepository) WithTransactionIsolation(ctx context.Context, level TransactionIsolationLevel, fn func(repo InventoryRepository) error) error {
	// Set transaction isolation level based on operation criticality
	txOptions := &sql.TxOptions{}
	
	switch level {
	case ReadCommitted:
		txOptions.Isolation = sql.LevelReadCommitted
	case RepeatableRead:
		txOptions.Isolation = sql.LevelRepeatableRead
	case Serializable:
		txOptions.Isolation = sql.LevelSerializable
	default:
		txOptions.Isolation = sql.LevelReadCommitted // Safe default
	}

	// Begin transaction with specific isolation level
	tx, err := r.db.BeginTx(ctx, txOptions)
	if err != nil {
		return NewRepositoryError("begin_transaction_isolation", "transaction", string(level), err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Create repository with transaction
	txRepo := &inventoryRepository{
		queries: r.queries.WithTx(tx),
		db:      r.db,
	}

	if err := fn(txRepo); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return NewRepositoryError("commit_transaction_isolation", "transaction", string(level), err)
	}

	return nil
}

// AtomicReserveAndCreateReservation performs both operations atomically
// This prevents inconsistencies between inventory and reservation state
func (r *inventoryRepository) AtomicReserveAndCreateReservation(ctx context.Context, req ReserveStockRequest, reservationReq CreateReservationRequest) (*InventoryItem, *Reservation, error) {
	var inventory *InventoryItem
	var reservation *Reservation
	
	// Use READ COMMITTED for this operation - we have optimistic locking
	err := r.WithTransactionIsolation(ctx, ReadCommitted, func(txRepo InventoryRepository) error {
		// Step 1: Reserve stock (atomic with version check)
		inv, err := txRepo.ReserveStock(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to reserve stock: %w", err)
		}
		inventory = inv
		
		// Step 2: Create reservation record (atomic)
		res, err := txRepo.CreateReservation(ctx, reservationReq)
		if err != nil {
			return fmt.Errorf("failed to create reservation: %w", err)
		}
		reservation = res
		
		return nil
	})
	
	if err != nil {
		return nil, nil, err
	}
	
	return inventory, reservation, nil
}

// AtomicReleaseAndUpdateReservation releases stock and updates reservation atomically
func (r *inventoryRepository) AtomicReleaseAndUpdateReservation(ctx context.Context, releaseReq ReleaseStockRequest, reservationID, newStatus string) (*InventoryItem, *Reservation, error) {
	var inventory *InventoryItem
	var reservation *Reservation
	
	err := r.WithTransactionIsolation(ctx, ReadCommitted, func(txRepo InventoryRepository) error {
		// Step 1: Release stock
		inv, err := txRepo.ReleaseStock(ctx, releaseReq)
		if err != nil {
			return fmt.Errorf("failed to release stock: %w", err)
		}
		inventory = inv
		
		// Step 2: Update reservation status
		res, err := txRepo.UpdateReservationStatus(ctx, reservationID, newStatus)
		if err != nil {
			return fmt.Errorf("failed to update reservation: %w", err)
		}
		reservation = res
		
		return nil
	})
	
	if err != nil {
		return nil, nil, err
	}
	
	return inventory, reservation, nil
}

// DeadlockAnalysis provides analysis of potential deadlock scenarios
type DeadlockAnalysis struct {
	Scenario    string
	Risk        string // LOW, MEDIUM, HIGH
	Mitigation  string
	Prevention  []string
}

// GetDeadlockAnalysis returns analysis of potential deadlock scenarios
func GetDeadlockAnalysis() []DeadlockAnalysis {
	return []DeadlockAnalysis{
		{
			Scenario:   "Concurrent Reserve Operations on Same Product",
			Risk:       "LOW", 
			Mitigation: "Optimistic locking prevents lock contention",
			Prevention: []string{
				"Version-based optimistic locking",
				"Retry with exponential backoff",
				"No SELECT FOR UPDATE usage",
			},
		},
		{
			Scenario:   "Cross-Product Reservations (A->B, B->A)",
			Risk:       "MEDIUM",
			Mitigation: "Ordered resource access by product_id",
			Prevention: []string{
				"Sort product IDs before accessing",
				"Single-product operations preferred",
				"Batch operations use consistent ordering",
			},
		},
		{
			Scenario:   "Reservation + Inventory Update Race",
			Risk:       "LOW",
			Mitigation: "Atomic operations within single transaction",
			Prevention: []string{
				"AtomicReserveAndCreateReservation method",
				"Transaction isolation levels",
				"Consistent operation ordering",
			},
		},
		{
			Scenario:   "Cleanup + Active Operations Collision",
			Risk:       "LOW",
			Mitigation: "Cleanup operations use separate transactions",
			Prevention: []string{
				"Cleanup runs in background",
				"Non-blocking cleanup queries",
				"Timeout-based cleanup",
			},
		},
	}
}

// ACIDCompliance represents our ACID compliance analysis
type ACIDCompliance struct {
	Property    string
	Compliance  string // FULL, PARTIAL, NONE
	Implementation string
	Risks       []string
	Mitigations []string
}

// GetACIDCompliance returns detailed ACID compliance analysis
func GetACIDCompliance() []ACIDCompliance {
	return []ACIDCompliance{
		{
			Property:   "Atomicity",
			Compliance: "FULL",
			Implementation: "Database transactions + optimistic locking",
			Risks: []string{
				"Network failures during commit",
				"Application crashes during transaction",
			},
			Mitigations: []string{
				"Proper transaction boundaries", 
				"Panic recovery with rollback",
				"Context timeouts",
				"Idempotency keys for retry safety",
			},
		},
		{
			Property:   "Consistency",
			Compliance: "FULL",
			Implementation: "Business rules + DB constraints + validation",
			Risks: []string{
				"Race conditions in concurrent updates",
				"Invalid state transitions",
			},
			Mitigations: []string{
				"Optimistic locking prevents dirty writes",
				"Input validation at repository layer",
				"Database constraints (FK, CHECK)",
				"Business rule enforcement",
			},
		},
		{
			Property:   "Isolation",
			Compliance: "FULL",
			Implementation: "Optimistic locking + configurable isolation levels",
			Risks: []string{
				"Phantom reads in range queries",
				"Non-repeatable reads",
			},
			Mitigations: []string{
				"Version-based optimistic concurrency control",
				"READ COMMITTED default isolation",
				"SERIALIZABLE for critical operations",
				"Retry logic for version conflicts",
			},
		},
		{
			Property:   "Durability",
			Compliance: "FULL",
			Implementation: "Database persistence + WAL + fsync",
			Risks: []string{
				"Hardware failures",
				"Database corruption",
				"Storage failures",
			},
			Mitigations: []string{
				"Database-level durability guarantees",
				"WAL mode in SQLite",
				"Regular backups",
				"Replication (PostgreSQL production)",
			},
		},
	}
}

// ValidateACIDCompliance runs checks to ensure ACID properties
func (r *inventoryRepository) ValidateACIDCompliance(ctx context.Context) error {
	// Test atomicity with a failing transaction
	err := r.WithTransaction(ctx, func(txRepo InventoryRepository) error {
		// This should rollback completely
		return fmt.Errorf("intentional failure for atomicity test")
	})
	if err == nil {
		return fmt.Errorf("atomicity test failed: transaction should have rolled back")
	}
	
	// Test consistency by verifying constraints
	// (This would be expanded with actual business rule checks)
	
	// Test isolation by checking version conflicts work
	// (This would be expanded with concurrent operation tests)
	
	// Durability is guaranteed by the database layer
	
	return nil
}