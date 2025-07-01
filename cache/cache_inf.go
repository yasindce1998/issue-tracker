package cache

import (
	"context"
	"time"
)

// Cache defines the interface for caching operations
type Cache interface {
	// Set stores a value in the cache with the given key and expiration
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error

	// Get retrieves a value from the cache with the given key
	Get(ctx context.Context, key string, dest interface{}) error

	// Delete removes a key from the cache
	Delete(ctx context.Context, keys ...string) error

	// Exists checks if a key exists in the cache
	Exists(ctx context.Context, key string) (bool, error)

	// Close closes the cache connection if needed
	Close() error
}
