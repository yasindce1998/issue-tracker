// Package cache provides caching implementations to optimize data access.
//
// This package offers a flexible caching system with multiple backend options:
// - Redis-based caching for distributed applications
// - In-memory caching for single-instance deployments
//
// The cache implementation can be selected at runtime via environment variables,
// allowing for easy switching between caching strategies without code changes.
package cache

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Type represents the type of cache to use
type Type string

const (
	// Redis indicates Redis should be used for caching
	Redis Type = "redis"
	// Memory indicates in-memory cache should be used
	Memory Type = "memory"
)

// Global cache instance for direct access when needed
var globalCacheInstance Cache

// NewCache creates a new cache based on environment configuration
func NewCache() Cache {
	cacheType := Type(getEnv("CACHE_TYPE", string(Redis)))

	var instance Cache
	switch cacheType {
	case Redis:
		instance = NewRedisClient(
			getEnv("REDIS_ADDR", "localhost:6379"),
			getEnv("REDIS_PASSWORD", ""),
			getEnvAsInt("REDIS_DB", 0),
		)
	case Memory:
		instance = NewMemoryCache(
			getEnvAsInt("MEMORY_CACHE_SIZE", 100),
		)
	default:
		// Default to Redis
		instance = NewRedisClient(
			getEnv("REDIS_ADDR", "localhost:6379"),
			getEnv("REDIS_PASSWORD", ""),
			getEnvAsInt("REDIS_DB", 0),
		)
	}

	// Store in global variable for access during shutdown
	globalCacheInstance = instance

	return instance
}

// CloseConnections closes any open cache connections
func CloseConnections() error {
	if globalCacheInstance != nil {
		return globalCacheInstance.Close()
	}
	return nil
}

// HealthCheck verifies that the cache is working properly by performing
// a simple set and get operation. Returns an error if the cache is not functioning.
func HealthCheck() error {
	if globalCacheInstance == nil {
		return nil // No cache initialized yet
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Try a simple ping operation
	healthKey := "health:check"
	healthValue := "ok"

	// Set a value
	err := globalCacheInstance.Set(ctx, healthKey, healthValue, 10*time.Second)
	if err != nil {
		return err
	}

	// Try to get it back
	var value string
	err = globalCacheInstance.Get(ctx, healthKey, &value)
	if err != nil {
		return err
	}

	// Verify the value
	if value != healthValue {
		return fmt.Errorf("cache health check failed: expected %s, got %s", healthValue, value)
	}

	// Clean up
	_ = globalCacheInstance.Delete(ctx, healthKey)

	return nil
}

// Helper function to get environment variable with default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// Helper function to get environment variable as integer
func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
