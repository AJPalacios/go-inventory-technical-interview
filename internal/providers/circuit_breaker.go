package providers

import (
	"fmt"
	"sync"
	"time"

	"github.com/AJPalacios/inventory/internal/domain"
)

// CircuitBreakerState represents the current state of the circuit breaker.
type CircuitBreakerState string

const (
	StateClosed   CircuitBreakerState = "closed"
	StateOpen     CircuitBreakerState = "open"
	StateHalfOpen CircuitBreakerState = "half-open"
)

// CircuitBreakerConfig holds configuration for the circuit breaker.
type CircuitBreakerConfig struct {
	MaxFailures      int           // Maximum failures before opening
	Timeout          time.Duration // Time to wait before trying again
	MaxRequests      int           // Maximum requests allowed in half-open state
	FailureThreshold float64       // Failure rate threshold (0.0 - 1.0)
}

// NewCircuitBreaker creates a new circuit breaker instance.
func NewCircuitBreaker(config CircuitBreakerConfig) domain.CircuitBreaker {
	return &circuitBreaker{
		maxFailures:      config.MaxFailures,
		timeout:          config.Timeout,
		maxRequests:      config.MaxRequests,
		failureThreshold: config.FailureThreshold,
		state:            StateClosed,
	}
}

// circuitBreaker implements the circuit breaker pattern for fault tolerance.
//
// This implementation provides automatic fault detection and recovery
// by monitoring operation success/failure rates and temporarily blocking
// requests when failure rates exceed thresholds.
type circuitBreaker struct {
	mu              sync.RWMutex
	state           CircuitBreakerState
	failures        int
	requests        int
	lastFailureTime time.Time
	lastStateChange time.Time

	// Configuration
	maxFailures      int
	timeout          time.Duration
	maxRequests      int
	failureThreshold float64

	// Half-open state tracking
	halfOpenRequests int
	halfOpenFailures int
}

// Execute runs an operation through the circuit breaker.
//
// The circuit breaker monitors the operation's success/failure and
// manages state transitions based on configured thresholds.
func (cb *circuitBreaker) Execute(operation func() (interface{}, error)) (interface{}, error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Check if we should transition states
	cb.checkStateTransition()

	// Handle different states
	switch cb.state {
	case StateOpen:
		return nil, fmt.Errorf("circuit breaker is open")
	case StateHalfOpen:
		if cb.halfOpenRequests >= cb.maxRequests {
			return nil, fmt.Errorf("circuit breaker half-open request limit exceeded")
		}
		cb.halfOpenRequests++
	case StateClosed:
		cb.requests++
	}

	// Execute the operation
	result, err := operation()

	// Record the result
	if err != nil {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}

	return result, err
}

// GetState returns the current state of the circuit breaker.
func (cb *circuitBreaker) GetState() string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return string(cb.state)
}

// Reset manually resets the circuit breaker to closed state.
func (cb *circuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failures = 0
	cb.requests = 0
	cb.halfOpenRequests = 0
	cb.halfOpenFailures = 0
	cb.lastStateChange = time.Now()
}

// recordFailure records a failed operation.
func (cb *circuitBreaker) recordFailure() {
	cb.failures++
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateClosed:
		if cb.shouldOpen() {
			cb.transitionToOpen()
		}
	case StateHalfOpen:
		cb.halfOpenFailures++
		cb.transitionToOpen()
	}
}

// recordSuccess records a successful operation.
func (cb *circuitBreaker) recordSuccess() {
	switch cb.state {
	case StateHalfOpen:
		// If we've had enough successful requests in half-open, close the circuit
		successRate := float64(cb.halfOpenRequests-cb.halfOpenFailures) / float64(cb.halfOpenRequests)
		if successRate >= (1.0 - cb.failureThreshold) {
			cb.transitionToClosed()
		}
	case StateClosed:
		// Reset failure count on success in closed state
		if cb.failures > 0 {
			cb.failures = max(0, cb.failures-1)
		}
	}
}

// shouldOpen determines if the circuit should open based on failure rate.
func (cb *circuitBreaker) shouldOpen() bool {
	if cb.requests < cb.maxFailures {
		return false // Not enough requests to make a decision
	}

	failureRate := float64(cb.failures) / float64(cb.requests)
	return failureRate >= cb.failureThreshold || cb.failures >= cb.maxFailures
}

// checkStateTransition checks if we should transition from open to half-open.
func (cb *circuitBreaker) checkStateTransition() {
	if cb.state == StateOpen {
		if time.Since(cb.lastStateChange) >= cb.timeout {
			cb.transitionToHalfOpen()
		}
	}
}

// transitionToOpen transitions the circuit breaker to open state.
func (cb *circuitBreaker) transitionToOpen() {
	cb.state = StateOpen
	cb.lastStateChange = time.Now()
}

// transitionToHalfOpen transitions the circuit breaker to half-open state.
func (cb *circuitBreaker) transitionToHalfOpen() {
	cb.state = StateHalfOpen
	cb.halfOpenRequests = 0
	cb.halfOpenFailures = 0
	cb.lastStateChange = time.Now()
}

// transitionToClosed transitions the circuit breaker to closed state.
func (cb *circuitBreaker) transitionToClosed() {
	cb.state = StateClosed
	cb.failures = 0
	cb.requests = 0
	cb.halfOpenRequests = 0
	cb.halfOpenFailures = 0
	cb.lastStateChange = time.Now()
}

// GetStats returns circuit breaker statistics.
func (cb *circuitBreaker) GetStats() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	var failureRate float64
	if cb.requests > 0 {
		failureRate = float64(cb.failures) / float64(cb.requests)
	}

	return map[string]interface{}{
		"state":              string(cb.state),
		"failures":           cb.failures,
		"requests":           cb.requests,
		"failure_rate":       failureRate,
		"half_open_requests": cb.halfOpenRequests,
		"half_open_failures": cb.halfOpenFailures,
		"last_failure_time":  cb.lastFailureTime,
		"last_state_change":  cb.lastStateChange,
	}
}

// Helper function for Go versions that don't have max built-in
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
