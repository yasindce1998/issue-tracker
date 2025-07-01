package projectsvc

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/yasindce1998/issue-tracker/cache"
	"github.com/yasindce1998/issue-tracker/logger"
	projectPbv1 "github.com/yasindce1998/issue-tracker/pkg/pb/project/v1"
	"go.uber.org/zap"
)

// CachedProjectRepository implements caching around a project repository
type CachedProjectRepository struct {
	repository ProjectRepository
	cache      cache.Cache
	ttl        time.Duration
}

// NewCachedProjectRepository creates a new cached project repository
func NewCachedProjectRepository(repository ProjectRepository, cache cache.Cache) *CachedProjectRepository {
	// Default TTL: 1 hour
	ttl := 3600 * time.Second

	// Get TTL from environment variable if available
	if ttlStr := os.Getenv("CACHE_TTL"); ttlStr != "" {
		if ttlVal, err := strconv.Atoi(ttlStr); err == nil {
			ttl = time.Duration(ttlVal) * time.Second
		}
	}

	return &CachedProjectRepository{
		repository: repository,
		cache:      cache,
		ttl:        ttl,
	}
}

// CreateProject adds a new project to the repository with caching
func (r *CachedProjectRepository) CreateProject(project *projectPbv1.Project) error {
	// Write to repository first
	if err := r.repository.CreateProject(project); err != nil {
		return err
	}

	// Then update cache
	ctx := context.Background()
	cacheKey := fmt.Sprintf("project:%s", project.ProjectId)
	if err := r.cache.Set(ctx, cacheKey, project, r.ttl); err != nil {
		// Log error but don't fail the request
		logger.ZapLogger.Error("Failed to cache project",
			zap.String("project_id", project.ProjectId),
			zap.Error(err))
	}

	return nil
}

// ReadProject retrieves a project by ID with caching
func (r *CachedProjectRepository) ReadProject(projectID string) (*projectPbv1.Project, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("project:%s", projectID)

	// Try to get from cache first
	var project = new(projectPbv1.Project)
	err := r.cache.Get(ctx, cacheKey, project)
	if err == nil {
		// Cache hit
		logger.ZapLogger.Debug("Project cache hit", zap.String("project_id", projectID))
		logger.LogCacheAccess(ctx, "Project", projectID, logger.FromCache)
		return project, nil
	}

	// Cache miss, get from repository
	project, err = r.repository.ReadProject(projectID)
	if err != nil {
		return nil, err
	}

	logger.LogCacheAccess(ctx, "Project", projectID, logger.FromDatabase)

	// Store in cache for future requests
	if err := r.cache.Set(ctx, cacheKey, project, r.ttl); err != nil {
		// Log error but don't fail the request
		logger.ZapLogger.Error("Failed to cache project",
			zap.String("project_id", projectID),
			zap.Error(err))
	}

	return project, nil
}

// UpdateProject updates an existing project and refreshes cache
func (r *CachedProjectRepository) UpdateProject(project *projectPbv1.Project) error {
	// Write to repository first
	if err := r.repository.UpdateProject(project); err != nil {
		return err
	}

	// Update cache
	ctx := context.Background()
	cacheKey := fmt.Sprintf("project:%s", project.ProjectId)
	if err := r.cache.Set(ctx, cacheKey, project, r.ttl); err != nil {
		logger.ZapLogger.Error("Failed to update project in cache",
			zap.String("project_id", project.ProjectId),
			zap.Error(err))
	}

	return nil
}

// DeleteProject removes a project and clears it from cache
func (r *CachedProjectRepository) DeleteProject(projectID string) error {
	// Delete from repository first
	if err := r.repository.DeleteProject(projectID); err != nil {
		return err
	}

	// Remove from cache
	ctx := context.Background()
	cacheKey := fmt.Sprintf("project:%s", projectID)
	if err := r.cache.Delete(ctx, cacheKey); err != nil {
		logger.ZapLogger.Error("Failed to remove project from cache",
			zap.String("project_id", projectID),
			zap.Error(err))
	}

	return nil
}

// ListProjects retrieves all projects with caching
func (r *CachedProjectRepository) ListProjects() ([]*projectPbv1.Project, error) {
	ctx := context.Background()
	cacheKey := "projects:all"

	// Try to get from cache first
	var projects []*projectPbv1.Project
	err := r.cache.Get(ctx, cacheKey, &projects)
	if err == nil {
		// Cache hit
		logger.ZapLogger.Debug("Projects list cache hit")
		logger.LogCacheAccess(ctx, "ProjectsList", "all", logger.FromCache)
		return projects, nil
	}

	// Cache miss, get from repository
	projects, err = r.repository.ListProjects()
	if err != nil {
		return nil, err
	}

	logger.LogCacheAccess(ctx, "ProjectsList", "all", logger.FromDatabase)

	// Store in cache for future requests
	if err := r.cache.Set(ctx, cacheKey, projects, r.ttl); err != nil {
		logger.ZapLogger.Error("Failed to cache projects list", zap.Error(err))
	}

	return projects, nil
}

// AddIssueToProject associates an issue with a project and updates cache
func (r *CachedProjectRepository) AddIssueToProject(projectID string, issueID string) error {
	// Update in repository first
	if err := r.repository.AddIssueToProject(projectID, issueID); err != nil {
		return err
	}

	// Invalidate project cache since issue count changed
	ctx := context.Background()
	projectCacheKey := fmt.Sprintf("project:%s", projectID)
	if err := r.cache.Delete(ctx, projectCacheKey); err != nil {
		logger.ZapLogger.Error("Failed to invalidate project cache after adding issue",
			zap.String("project_id", projectID),
			zap.String("issue_id", issueID),
			zap.Error(err))
	}

	// Also invalidate projects list cache
	if err := r.cache.Delete(ctx, "projects:all"); err != nil {
		logger.ZapLogger.Error("Failed to invalidate projects list cache", zap.Error(err))
	}

	return nil
}

// RemoveIssueFromProject removes an association between an issue and a project
func (r *CachedProjectRepository) RemoveIssueFromProject(projectID string, issueID string) error {
	// Update in repository first
	if err := r.repository.RemoveIssueFromProject(projectID, issueID); err != nil {
		return err
	}

	// Invalidate project cache since issue count changed
	ctx := context.Background()
	projectCacheKey := fmt.Sprintf("project:%s", projectID)
	if err := r.cache.Delete(ctx, projectCacheKey); err != nil {
		logger.ZapLogger.Error("Failed to invalidate project cache after removing issue",
			zap.String("project_id", projectID),
			zap.String("issue_id", issueID),
			zap.Error(err))
	}

	// Also invalidate projects list cache
	if err := r.cache.Delete(ctx, "projects:all"); err != nil {
		logger.ZapLogger.Error("Failed to invalidate projects list cache", zap.Error(err))
	}

	return nil
}
