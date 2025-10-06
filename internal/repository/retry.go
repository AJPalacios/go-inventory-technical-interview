package repository

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"time"
)

// RetryConfig defines retry behavior configuration
type RetryConfig struct {
	MaxRetries   int           // Maximum number of retry attempts
	BaseDelay    time.Duration // Base delay between retries
	MaxDelay     time.Duration // Maximum delay cap
	JitterFactor float64       // Jitter factor (0.0 to 1.0)
	Multiplier   float64       // Backoff multiplier
}

// DefaultRetryConfig returns sensible defaults for inventory operations
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:   5,
		BaseDelay:    50 * time.Millisecond,
		MaxDelay:     2 * time.Second,
		JitterFactor: 0.1,
		Multiplier:   2.0,
	}
}

// RetryableFunc defines a function that can be retried
type RetryableFunc func() error

// RetryWithBackoff executes a function with exponential backoff retry logic
func RetryWithBackoff(ctx context.Context, config RetryConfig, fn RetryableFunc) error {
	var lastErr error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// Execute the function
		err := fn()
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if error is retryable
		if !IsRetryable(err) {
			return err // Non-retryable error, fail immediately
		}

		// If this was the last attempt, don't wait
		if attempt == config.MaxRetries {
			break
		}

		// Calculate delay with exponential backoff
		delay := calculateDelay(config, attempt)

		// Wait for the delay or until context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			continue
		}
	}

	// All retries exhausted - try to extract context from last error
	var op, entity, id string = "retry_exhausted", "unknown", "unknown"
	var repoErr *RepositoryError
	if errors.As(lastErr, &repoErr) {
		op = repoErr.Op
		entity = repoErr.Entity
		id = repoErr.ID
	}
	return NewMaxRetriesError(op, entity, id, config.MaxRetries+1, lastErr)
}

// calculateDelay computes the delay for a given attempt with jitter
func calculateDelay(config RetryConfig, attempt int) time.Duration {
	// Exponential backoff: baseDelay * multiplier^attempt
	delay := time.Duration(float64(config.BaseDelay) * math.Pow(config.Multiplier, float64(attempt)))

	// Cap at maxDelay
	if delay > config.MaxDelay {
		delay = config.MaxDelay
	}

	// Add jitter to prevent thundering herd
	if config.JitterFactor > 0 {
		jitter := time.Duration(rand.Float64() * float64(delay) * config.JitterFactor)
		delay += jitter
	}

	return delay
}

// RetryableOperation wraps a repository operation with retry logic
type RetryableOperation struct {
	config RetryConfig
	name   string
}

// NewRetryableOperation creates a new retryable operation
func NewRetryableOperation(name string, config RetryConfig) *RetryableOperation {
	return &RetryableOperation{
		config: config,
		name:   name,
	}
}

// Execute runs the operation with retry logic
func (r *RetryableOperation) Execute(ctx context.Context, fn RetryableFunc) error {
	return RetryWithBackoff(ctx, r.config, fn)
}

// Common retry operations for inventory
var (
	// StandardRetry for normal inventory operations
	StandardRetry = NewRetryableOperation("standard", DefaultRetryConfig())

	// AggressiveRetry for critical operations that need more attempts
	AggressiveRetry = NewRetryableOperation("aggressive", RetryConfig{
		MaxRetries:   10,
		BaseDelay:    25 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		JitterFactor: 0.15,
		Multiplier:   1.8,
	})

	// ConservativeRetry for operations that should fail fast
	ConservativeRetry = NewRetryableOperation("conservative", RetryConfig{
		MaxRetries:   2,
		BaseDelay:    100 * time.Millisecond,
		MaxDelay:     500 * time.Millisecond,
		JitterFactor: 0.05,
		Multiplier:   2.0,
	})
)
