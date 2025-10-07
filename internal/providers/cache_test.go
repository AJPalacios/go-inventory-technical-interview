package providers

import (
"context"
"testing"
"time"

"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/require"
)

func TestNewMemoryCacheProvider(t *testing.T) {
	config := CacheConfig{
		Provider: CacheProviderMemory,
		MaxSize:  100,
	}
	cache := NewMemoryCacheProvider(config)
	assert.NotNil(t, cache)
}

func TestMemoryCacheProvider_SetAndGet(t *testing.T) {
	config := CacheConfig{
		Provider: CacheProviderMemory,
		MaxSize:  10,
	}
	cache := NewMemoryCacheProvider(config)
	ctx := context.Background()

	// Test setting and getting a value
	value := []byte("test_value")
	err := cache.Set(ctx, "key1", value, time.Hour)
	require.NoError(t, err)

	retrieved, err := cache.Get(ctx, "key1")
	require.NoError(t, err)
	assert.Equal(t, value, retrieved)
}

func TestMemoryCacheProvider_GetNonExistent(t *testing.T) {
	config := CacheConfig{
		Provider: CacheProviderMemory,
		MaxSize:  10,
	}
	cache := NewMemoryCacheProvider(config)
	ctx := context.Background()

	value, err := cache.Get(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Nil(t, value)
	assert.Contains(t, err.Error(), "key not found")
}

func TestMemoryCacheProvider_SetWithExpiration(t *testing.T) {
	config := CacheConfig{
		Provider: CacheProviderMemory,
		MaxSize:  10,
	}
	cache := NewMemoryCacheProvider(config)
	ctx := context.Background()

	// Set a value with short expiration
	value := []byte("test_value")
	err := cache.Set(ctx, "key1", value, 100*time.Millisecond)
	require.NoError(t, err)

	// Value should exist immediately
	retrieved, err := cache.Get(ctx, "key1")
	require.NoError(t, err)
	assert.Equal(t, value, retrieved)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Value should be expired
	retrieved, err = cache.Get(ctx, "key1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key expired")
}

func TestMemoryCacheProvider_Delete(t *testing.T) {
	config := CacheConfig{
		Provider: CacheProviderMemory,
		MaxSize:  10,
	}
	cache := NewMemoryCacheProvider(config)
	ctx := context.Background()

	// Set a value
	value := []byte("test_value")
	err := cache.Set(ctx, "key1", value, time.Hour)
	require.NoError(t, err)

	// Confirm it exists
	exists, err := cache.Exists(ctx, "key1")
	require.NoError(t, err)
	assert.True(t, exists)

	// Delete it
	err = cache.Delete(ctx, "key1")
	require.NoError(t, err)

	// Confirm it's gone
exists, err = cache.Exists(ctx, "key1")
require.NoError(t, err)
assert.False(t, exists)
}

func TestMemoryCacheProvider_Exists(t *testing.T) {
config := CacheConfig{
Provider: CacheProviderMemory,
MaxSize:  10,
}
cache := NewMemoryCacheProvider(config)
ctx := context.Background()

// Key should not exist initially
exists, err := cache.Exists(ctx, "key1")
require.NoError(t, err)
assert.False(t, exists)

// Set a value
value := []byte("test_value")
err = cache.Set(ctx, "key1", value, time.Hour)
require.NoError(t, err)

// Key should exist now
exists, err = cache.Exists(ctx, "key1")
require.NoError(t, err)
assert.True(t, exists)
}

func TestMemoryCacheProvider_MaxSize(t *testing.T) {
config := CacheConfig{
Provider: CacheProviderMemory,
MaxSize:  2, // Small max size
}
cache := NewMemoryCacheProvider(config)
ctx := context.Background()

// Fill the cache to capacity
err := cache.Set(ctx, "key1", []byte("value1"), time.Hour)
require.NoError(t, err)
err = cache.Set(ctx, "key2", []byte("value2"), time.Hour)
require.NoError(t, err)

// Both keys should exist
exists1, err := cache.Exists(ctx, "key1")
require.NoError(t, err)
assert.True(t, exists1)
exists2, err := cache.Exists(ctx, "key2")
require.NoError(t, err)
assert.True(t, exists2)

// Add one more - should evict the oldest
err = cache.Set(ctx, "key3", []byte("value3"), time.Hour)
require.NoError(t, err)

// key1 should be evicted, key2 and key3 should exist
exists1, err = cache.Exists(ctx, "key1")
require.NoError(t, err)
assert.False(t, exists1)
exists2, err = cache.Exists(ctx, "key2")
require.NoError(t, err)
assert.True(t, exists2)
exists3, err := cache.Exists(ctx, "key3")
require.NoError(t, err)
assert.True(t, exists3)
}

func TestMemoryCacheProvider_UpdateExistingKey(t *testing.T) {
config := CacheConfig{
Provider: CacheProviderMemory,
MaxSize:  10,
}
cache := NewMemoryCacheProvider(config)
ctx := context.Background()

// Set initial value
err := cache.Set(ctx, "key1", []byte("value1"), time.Hour)
require.NoError(t, err)

// Update the value
err = cache.Set(ctx, "key1", []byte("updated_value"), time.Hour)
require.NoError(t, err)

// Should get the updated value
value, err := cache.Get(ctx, "key1")
require.NoError(t, err)
assert.Equal(t, []byte("updated_value"), value)
}

func TestNewCacheProvider(t *testing.T) {
tests := []struct {
name       string
config     CacheConfig
shouldFail bool
}{
{
name: "valid memory provider",
config: CacheConfig{
Provider: CacheProviderMemory,
MaxSize:  100,
},
shouldFail: false,
},
{
name: "redis provider falls back to memory",
config: CacheConfig{
Provider: CacheProviderRedis,
MaxSize:  100,
},
shouldFail: false,
},
{
name: "memcached provider falls back to memory",
config: CacheConfig{
Provider: CacheProviderMemcached,
MaxSize:  100,
},
shouldFail: false,
},
{
name: "unknown provider defaults to memory",
config: CacheConfig{
Provider: "unknown",
MaxSize:  100,
},
shouldFail: false,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
provider := NewCacheProvider(tt.config)
assert.NotNil(t, provider)
})
}
}
