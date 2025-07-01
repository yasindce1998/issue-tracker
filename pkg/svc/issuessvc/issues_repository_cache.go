package issuessvc

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/yasindce1998/issue-tracker/cache"
	"github.com/yasindce1998/issue-tracker/logger"
	issuesPbv1 "github.com/yasindce1998/issue-tracker/pkg/pb/issues/v1"
	"go.uber.org/zap"
)

// CachedIssuesRepository implements caching around an issues repository
type CachedIssuesRepository struct {
	repository IssuesRepository
	cache      cache.Cache
	ttl        time.Duration
}

// NewCachedIssuesRepository creates a new cached issues repository
func NewCachedIssuesRepository(repository IssuesRepository, cache cache.Cache) *CachedIssuesRepository {
	// Default TTL: 1 hour
	ttl := 3600 * time.Second

	// Get TTL from environment variable if available
	if ttlStr := os.Getenv("CACHE_TTL"); ttlStr != "" {
		if ttlVal, err := strconv.Atoi(ttlStr); err == nil {
			ttl = time.Duration(ttlVal) * time.Second
		}
	}

	return &CachedIssuesRepository{
		repository: repository,
		cache:      cache,
		ttl:        ttl,
	}
}

// CreateIssue adds a new issue to the repository with caching
func (r *CachedIssuesRepository) CreateIssue(issue *issuesPbv1.Issue) error {
	// Write to repository first
	if err := r.repository.CreateIssue(issue); err != nil {
		return err
	}

	// Then update cache
	ctx := context.Background()
	cacheKey := fmt.Sprintf("issue:%s", issue.IssueId)
	if err := r.cache.Set(ctx, cacheKey, issue, r.ttl); err != nil {
		// Log error but don't fail the request
		logger.ZapLogger.Error("Failed to cache issue",
			zap.String("issue_id", issue.IssueId),
			zap.Error(err))
	}

	// Also invalidate the issues list cache
	r.invalidateIssueListCache(ctx)

	return nil
}

// ReadIssue retrieves an issue by ID with caching
func (r *CachedIssuesRepository) ReadIssue(issueID string) (*issuesPbv1.Issue, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("issue:%s", issueID)

	// Try to get from cache first
	var issue = new(issuesPbv1.Issue)
	err := r.cache.Get(ctx, cacheKey, issue)
	if err == nil {
		// Cache hit
		logger.ZapLogger.Debug("Issue cache hit", zap.String("issue_id", issueID))
		logger.LogCacheAccess(ctx, "Issue", issueID, logger.FromCache)
		return issue, nil
	}

	// Cache miss, get from repository
	issue, err = r.repository.ReadIssue(issueID)
	if err != nil {
		return nil, err
	}

	logger.LogCacheAccess(ctx, "Issue", issueID, logger.FromDatabase)

	// Store in cache for future requests
	if err := r.cache.Set(ctx, cacheKey, issue, r.ttl); err != nil {
		// Log error but don't fail the request
		logger.ZapLogger.Error("Failed to cache issue",
			zap.String("issue_id", issueID),
			zap.Error(err))
	}

	return issue, nil
}

// UpdateIssue updates an existing issue and refreshes cache
func (r *CachedIssuesRepository) UpdateIssue(issue *issuesPbv1.Issue) error {
	// Write to repository first
	if err := r.repository.UpdateIssue(issue); err != nil {
		return err
	}

	// Update cache
	ctx := context.Background()
	cacheKey := fmt.Sprintf("issue:%s", issue.IssueId)
	if err := r.cache.Set(ctx, cacheKey, issue, r.ttl); err != nil {
		logger.ZapLogger.Error("Failed to update issue in cache",
			zap.String("issue_id", issue.IssueId),
			zap.Error(err))
	}

	// Also invalidate the issues list cache since an issue was updated
	r.invalidateIssueListCache(ctx)

	return nil
}

// DeleteIssue removes an issue and clears it from cache
func (r *CachedIssuesRepository) DeleteIssue(issueID string) error {
	// Delete from repository first
	if err := r.repository.DeleteIssue(issueID); err != nil {
		return err
	}

	// Remove from cache
	ctx := context.Background()
	cacheKey := fmt.Sprintf("issue:%s", issueID)
	if err := r.cache.Delete(ctx, cacheKey); err != nil {
		logger.ZapLogger.Error("Failed to remove issue from cache",
			zap.String("issue_id", issueID),
			zap.Error(err))
	}

	// Also invalidate the issues list cache since an issue was deleted
	r.invalidateIssueListCache(ctx)

	return nil
}

// ListIssues retrieves a paginated list of issues with caching
func (r *CachedIssuesRepository) ListIssues(pageToken string, pageSize int) ([]*issuesPbv1.Issue, string, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("issues:list:%s:%d", pageToken, pageSize)

	// Try to get from cache first
	type cachedIssuesList struct {
		Issues    []*issuesPbv1.Issue
		NextToken string
	}

	var cachedList cachedIssuesList
	err := r.cache.Get(ctx, cacheKey, &cachedList)
	if err == nil {
		// Cache hit
		logger.ZapLogger.Debug("Issues list cache hit",
			zap.String("page_token", pageToken),
			zap.Int("page_size", pageSize))
		logger.LogCacheAccess(ctx, "IssuesList", fmt.Sprintf("page:%s:size:%d", pageToken, pageSize), logger.FromCache)
		return cachedList.Issues, cachedList.NextToken, nil
	}

	// Cache miss, get from repository
	issues, nextToken, err := r.repository.ListIssues(pageToken, pageSize)
	if err != nil {
		return nil, "", err
	}

	logger.LogCacheAccess(ctx, "IssuesList", fmt.Sprintf("page:%s:size:%d", pageToken, pageSize), logger.FromDatabase)

	// Store in cache for future requests
	toCache := cachedIssuesList{
		Issues:    issues,
		NextToken: nextToken,
	}

	if err := r.cache.Set(ctx, cacheKey, toCache, r.ttl); err != nil {
		logger.ZapLogger.Error("Failed to cache issues list",
			zap.String("page_token", pageToken),
			zap.Int("page_size", pageSize),
			zap.Error(err))
	}

	return issues, nextToken, nil
}

// ValidateProjectExists checks if a project exists
func (r *CachedIssuesRepository) ValidateProjectExists(ctx context.Context, projectID string) error {
	return r.repository.ValidateProjectExists(ctx, projectID)
}

// ValidateUserExists checks if a user exists
func (r *CachedIssuesRepository) ValidateUserExists(ctx context.Context, userID string) error {
	return r.repository.ValidateUserExists(ctx, userID)
}

// IsValidStatusTransition checks if a status transition is valid
func (r *CachedIssuesRepository) IsValidStatusTransition(currentStatus, newStatus issuesPbv1.Status) error {
	return r.repository.IsValidStatusTransition(currentStatus, newStatus)
}

// invalidateIssueListCache removes all cached issue list results to ensure consistency
// after an issue is created, updated, or deleted
func (r *CachedIssuesRepository) invalidateIssueListCache(ctx context.Context) {
	// Track all key invalidations for logging
	var invalidatedCount int
	var lastError error

	// We'll invalidate specific keys we know about rather than using patterns
	// This is more efficient and works across different cache implementations
	knownPrefixes := []string{
		"issues:list:", // Basic list cache
		"issues:all",   // Any cache of all issues
		"issues:count", // Issue count cache if implemented
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
		logger.ZapLogger.Error("Failed to invalidate some issue list caches",
			zap.Int("successful_invalidations", invalidatedCount),
			zap.Error(lastError))
	} else if invalidatedCount > 0 {
		logger.ZapLogger.Debug("Successfully invalidated issue list caches",
			zap.Int("count", invalidatedCount))
	}
}
