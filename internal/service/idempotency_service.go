package service

import (
	"context"
	"sync"
	"time"

	"github.com/AJPalacios/inventory/internal/domain"
)

// idempotencyService provides in-memory idempotency management.
//
// This service stores request results to ensure idempotent operations
// across multiple requests with the same request ID.
type idempotencyService struct {
	cache      map[string]idempotencyEntry
	mu         sync.RWMutex
	maxSize    int
	defaultTTL time.Duration
}

// idempotencyEntry represents a cached idempotency result.
type idempotencyEntry struct {
	Result    interface{}
	ExpiresAt time.Time
	CreatedAt time.Time
}

// NewIdempotencyService creates a new idempotency service instance.
func NewIdempotencyService(config IdempotencyServiceConfig) domain.IdempotencyService {
	service := &idempotencyService{
		cache:      make(map[string]idempotencyEntry),
		maxSize:    config.MaxSize,
		defaultTTL: config.DefaultTTL,
	}

	// Start cleanup goroutine
	go service.cleanupExpired()

	return service
}

// CheckIdempotency checks if a request ID has been processed before.
//
// Returns the cached result if found and not expired, along with a boolean
// indicating whether the result was found.
func (s *idempotencyService) CheckIdempotency(ctx context.Context, requestID string) (interface{}, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, exists := s.cache[requestID]
	if !exists {
		return nil, false, nil
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		// Entry is expired, but we'll clean it up later
		return nil, false, nil
	}

	return entry.Result, true, nil
}

// StoreResult stores a result for idempotency with the specified TTL.
//
// If TTL is 0, uses the default TTL. Results are stored in memory
// with automatic cleanup of expired entries.
func (s *idempotencyService) StoreResult(ctx context.Context, requestID string, result interface{}, ttl time.Duration) error {
	if ttl == 0 {
		ttl = s.defaultTTL
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check cache size limit
	if len(s.cache) >= s.maxSize {
		// Remove oldest entries (simple FIFO eviction)
		s.evictOldest()
	}

	s.cache[requestID] = idempotencyEntry{
		Result:    result,
		ExpiresAt: time.Now().Add(ttl),
		CreatedAt: time.Now(),
	}

	return nil
}

// CleanupExpired removes expired entries from the cache.
//
// Returns the number of entries that were cleaned up.
func (s *idempotencyService) CleanupExpired(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	cleanedCount := 0

	for requestID, entry := range s.cache {
		if now.After(entry.ExpiresAt) {
			delete(s.cache, requestID)
			cleanedCount++
		}
	}

	return cleanedCount, nil
}

// evictOldest removes the oldest entries to make room for new ones.
func (s *idempotencyService) evictOldest() {
	if len(s.cache) == 0 {
		return
	}

	// Find oldest entry
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range s.cache {
		if oldestKey == "" || entry.CreatedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.CreatedAt
		}
	}

	if oldestKey != "" {
		delete(s.cache, oldestKey)
	}
}

// cleanupExpired runs periodic cleanup of expired entries.
func (s *idempotencyService) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute) // Cleanup every 5 minutes
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		s.CleanupExpired(ctx)
		cancel()
	}
}

// GetStats returns statistics about the idempotency cache.
func (s *idempotencyService) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"cache_size":  len(s.cache),
		"max_size":    s.maxSize,
		"default_ttl": s.defaultTTL.String(),
	}
}
