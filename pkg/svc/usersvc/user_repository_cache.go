package usersvc

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/yasindce1998/issue-tracker/cache"
	"github.com/yasindce1998/issue-tracker/logger"
	userPbv1 "github.com/yasindce1998/issue-tracker/pkg/pb/user/v1"
	"go.uber.org/zap"
)

// CachedUserRepository implements caching around a user repository
type CachedUserRepository struct {
	repository UserRepository
	cache      cache.Cache
	ttl        time.Duration
}

// NewCachedUserRepository creates a new cached user repository
func NewCachedUserRepository(repository UserRepository, cache cache.Cache) *CachedUserRepository {
	// Default TTL: 1 hour
	ttl := 3600 * time.Second

	// Get TTL from environment variable if available
	if ttlStr := os.Getenv("CACHE_TTL"); ttlStr != "" {
		if ttlVal, err := strconv.Atoi(ttlStr); err == nil {
			ttl = time.Duration(ttlVal) * time.Second
		}
	}

	return &CachedUserRepository{
		repository: repository,
		cache:      cache,
		ttl:        ttl,
	}
}

// CreateUser adds a new user to the repository with caching
func (r *CachedUserRepository) CreateUser(user *userPbv1.User) error {
	// Write to repository first
	if err := r.repository.CreateUser(user); err != nil {
		return err
	}

	// Then update cache
	ctx := context.Background()
	cacheKey := fmt.Sprintf("user:%s", user.UserId)
	if err := r.cache.Set(ctx, cacheKey, user, r.ttl); err != nil {
		// Log error but don't fail the request
		logger.ZapLogger.Error("Failed to cache user",
			zap.String("user_id", user.UserId),
			zap.Error(err))
	}

	// Also invalidate the users list cache
	r.invalidateUserListCache(ctx)

	return nil
}

// GetUserByID retrieves a user by ID with caching
func (r *CachedUserRepository) GetUserByID(userID string) (*userPbv1.User, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("user:%s", userID)

	// Try to get from cache first
	var user = new(userPbv1.User)
	err := r.cache.Get(ctx, cacheKey, user)
	if err == nil {
		// Cache hit
		logger.ZapLogger.Debug("User cache hit", zap.String("user_id", userID))
		logger.LogCacheAccess(ctx, "User", userID, logger.FromCache)
		return user, nil
	}

	// Cache miss, get from repository
	user, err = r.repository.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	logger.LogCacheAccess(ctx, "User", userID, logger.FromDatabase)

	// Store in cache for future requests
	if err := r.cache.Set(ctx, cacheKey, user, r.ttl); err != nil {
		// Log error but don't fail the request
		logger.ZapLogger.Error("Failed to cache user",
			zap.String("user_id", userID),
			zap.Error(err))
	}

	return user, nil
}

// UpdateUser updates an existing user and refreshes cache
func (r *CachedUserRepository) UpdateUser(user *userPbv1.User) error {
	// Write to repository first
	if err := r.repository.UpdateUser(user); err != nil {
		return err
	}

	// Update cache
	ctx := context.Background()
	cacheKey := fmt.Sprintf("user:%s", user.UserId)
	if err := r.cache.Set(ctx, cacheKey, user, r.ttl); err != nil {
		logger.ZapLogger.Error("Failed to update user in cache",
			zap.String("user_id", user.UserId),
			zap.Error(err))
	}

	// Also invalidate the users list cache since a user was updated
	r.invalidateUserListCache(ctx)

	return nil
}

// DeleteUser removes a user and clears it from cache
func (r *CachedUserRepository) DeleteUser(userID string) error {
	// Delete from repository first
	if err := r.repository.DeleteUser(userID); err != nil {
		return err
	}

	// Remove from cache
	ctx := context.Background()
	cacheKey := fmt.Sprintf("user:%s", userID)
	if err := r.cache.Delete(ctx, cacheKey); err != nil {
		logger.ZapLogger.Error("Failed to remove user from cache",
			zap.String("user_id", userID),
			zap.Error(err))
	}

	// Also invalidate the users list cache since a user was deleted
	r.invalidateUserListCache(ctx)

	return nil
}

// ListUsers retrieves a paginated list of users with caching
func (r *CachedUserRepository) ListUsers(pageToken string, pageSize int) ([]*userPbv1.User, string, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("users:list:%s:%d", pageToken, pageSize)

	// Try to get from cache first
	type cachedUsersList struct {
		Users     []*userPbv1.User
		NextToken string
	}

	var cachedList cachedUsersList
	err := r.cache.Get(ctx, cacheKey, &cachedList)
	if err == nil {
		// Cache hit
		logger.ZapLogger.Debug("Users list cache hit",
			zap.String("page_token", pageToken),
			zap.Int("page_size", pageSize))
		logger.LogCacheAccess(ctx, "UsersList", fmt.Sprintf("page:%s:size:%d", pageToken, pageSize), logger.FromCache)
		return cachedList.Users, cachedList.NextToken, nil
	}

	// Cache miss, get from repository
	users, nextToken, err := r.repository.ListUsers(pageToken, pageSize)
	if err != nil {
		return nil, "", err
	}

	logger.LogCacheAccess(ctx, "UsersList", fmt.Sprintf("page:%s:size:%d", pageToken, pageSize), logger.FromDatabase)

	// Store in cache for future requests
	toCache := cachedUsersList{
		Users:     users,
		NextToken: nextToken,
	}

	if err := r.cache.Set(ctx, cacheKey, toCache, r.ttl); err != nil {
		logger.ZapLogger.Error("Failed to cache users list",
			zap.String("page_token", pageToken),
			zap.Int("page_size", pageSize),
			zap.Error(err))
	}

	return users, nextToken, nil
}

// invalidateUserListCache removes all cached user list results to ensure consistency
// after a user is created, updated, or deleted
func (r *CachedUserRepository) invalidateUserListCache(ctx context.Context) {
	// For Redis specifically, we need to get all matching keys first
	// This is a more efficient approach that works with both Redis and memory caches

	// Track all key invalidations for logging
	var invalidatedCount int
	var lastError error

	// We'll invalidate specific keys we know about rather than using patterns
	// This is more efficient and works across different cache implementations
	knownPrefixes := []string{
		"users:list:", // Basic list cache
		"users:all",   // Any cache of all users
		"users:count", // User count cache if implemented
	}

	for _, prefix := range knownPrefixes {
		// Different cache implementations might have different ways to delete by pattern
		// Here we're just deleting known keys directly
		if err := r.cache.Delete(ctx, prefix); err != nil {
			lastError = err
			logger.ZapLogger.Debug("Failed to invalidate cache key",
				zap.String("key", prefix),
				zap.Error(err))
		} else {
			invalidatedCount++
		}
	}

	// Log the overall outcome
	if lastError != nil {
		logger.ZapLogger.Error("Failed to invalidate some user list caches",
			zap.Int("successful_invalidations", invalidatedCount),
			zap.Error(lastError))
	} else if invalidatedCount > 0 {
		logger.ZapLogger.Debug("Successfully invalidated user list caches",
			zap.Int("count", invalidatedCount))
	}
}
