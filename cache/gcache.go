package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bluele/gcache"
)

// MemoryCache implements the Cache interface using gcache
type MemoryCache struct {
	cache gcache.Cache
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache(size int) *MemoryCache {
	cache := gcache.New(size).
		LRU().
		Build()

	return &MemoryCache{
		cache: cache,
	}
}

// Set stores a value in the memory cache with expiration
func (m *MemoryCache) Set(_ context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return m.cache.SetWithExpire(key, data, expiration)
}

// Get retrieves a value from the memory cache
func (m *MemoryCache) Get(_ context.Context, key string, dest interface{}) error {
	value, err := m.cache.Get(key)
	if err != nil {
		return err
	}

	data, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("invalid cached data type")
	}

	return json.Unmarshal(data, dest)
}

// Delete removes a key from the memory cache
func (m *MemoryCache) Delete(_ context.Context, keys ...string) error {
	for _, key := range keys {
		m.cache.Remove(key)
	}
	return nil
}

// Exists checks if a key exists in the memory cache
func (m *MemoryCache) Exists(_ context.Context, key string) (bool, error) {
	return m.cache.Has(key), nil
}

// Close is a no-op for memory cache
func (m *MemoryCache) Close() error {
	return nil
}
