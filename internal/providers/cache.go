package providers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/AJPalacios/inventory/internal/domain"
)

// CacheProviderType defines available cache provider types.
type CacheProviderType string

const (
	CacheProviderMemory    CacheProviderType = "memory"
	CacheProviderRedis     CacheProviderType = "redis"
	CacheProviderMemcached CacheProviderType = "memcached"
)

// CacheConfig holds configuration for cache provider.
type CacheConfig struct {
	Provider CacheProviderType
	Address  string
	Password string
	DB       int
	MaxSize  int
}

// NewCacheProvider creates a cache provider based on configuration.
func NewCacheProvider(config CacheConfig) domain.CacheProvider {
	switch config.Provider {
	case CacheProviderMemory:
		return NewMemoryCacheProvider(config)
	case CacheProviderRedis:
		// return NewRedisCacheProvider(config)
		fmt.Printf("Redis provider not implemented, falling back to memory\n")
		return NewMemoryCacheProvider(config)
	case CacheProviderMemcached:
		// return NewMemcachedCacheProvider(config)
		fmt.Printf("Memcached provider not implemented, falling back to memory\n")
		return NewMemoryCacheProvider(config)
	default:
		fmt.Printf("Unknown cache provider %s, falling back to memory\n", config.Provider)
		return NewMemoryCacheProvider(config)
	}
}

// memoryCacheProvider provides in-memory caching.
type memoryCacheProvider struct {
	cache   map[string]cacheEntry
	mu      sync.RWMutex
	maxSize int
}

// cacheEntry represents a cached value with expiration.
type cacheEntry struct {
	Value     []byte
	ExpiresAt time.Time
}

// NewMemoryCacheProvider creates an in-memory cache provider.
func NewMemoryCacheProvider(config CacheConfig) domain.CacheProvider {
	provider := &memoryCacheProvider{
		cache:   make(map[string]cacheEntry),
		maxSize: config.MaxSize,
	}

	// Start cleanup goroutine
	go provider.cleanupExpired()

	return provider
}

// Get retrieves a value from the cache.
func (c *memoryCacheProvider) Get(ctx context.Context, key string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.cache[key]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	if time.Now().After(entry.ExpiresAt) {
		return nil, fmt.Errorf("key expired: %s", key)
	}

	return entry.Value, nil
}

// Set stores a value in the cache with TTL.
func (c *memoryCacheProvider) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check size limit
	if len(c.cache) >= c.maxSize {
		c.evictOldest()
	}

	c.cache[key] = cacheEntry{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
	}

	return nil
}

// Delete removes a value from the cache.
func (c *memoryCacheProvider) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cache, key)
	return nil
}

// Exists checks if a key exists in the cache.
func (c *memoryCacheProvider) Exists(ctx context.Context, key string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.cache[key]
	if !exists {
		return false, nil
	}

	if time.Now().After(entry.ExpiresAt) {
		return false, nil
	}

	return true, nil
}

// evictOldest removes the oldest entry.
func (c *memoryCacheProvider) evictOldest() {
	if len(c.cache) == 0 {
		return
	}

	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.cache {
		if oldestKey == "" || entry.ExpiresAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.ExpiresAt
		}
	}

	if oldestKey != "" {
		delete(c.cache, oldestKey)
	}
}

// cleanupExpired removes expired entries periodically.
func (c *memoryCacheProvider) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, entry := range c.cache {
			if now.After(entry.ExpiresAt) {
				delete(c.cache, key)
			}
		}
		c.mu.Unlock()
	}
}
