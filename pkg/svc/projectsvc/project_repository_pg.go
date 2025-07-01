package projectsvc

import (
	"errors"

	"github.com/yasindce1998/issue-tracker/consts"
	"github.com/yasindce1998/issue-tracker/logger"
	"github.com/yasindce1998/issue-tracker/models"
	projectPbv1 "github.com/yasindce1998/issue-tracker/pkg/pb/project/v1"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// PostgresProjectRepository implements ProjectRepository using GORM for PostgreSQL
type PostgresProjectRepository struct {
	db *gorm.DB
}

// NewPostgresProjectRepository initializes the repository with a GORM DB instance
func NewPostgresProjectRepository(db *gorm.DB) *PostgresProjectRepository {
	return &PostgresProjectRepository{db: db}
}

// CreateProject adds a new project to the database
func (r *PostgresProjectRepository) CreateProject(project *projectPbv1.Project) error {
	// Convert protobuf project to model
	dbProject := &models.Project{
		ProjectID:   project.ProjectId,
		Name:        project.Name,
		Description: project.Description,
		IssueCount:  project.IssueCount,
	}

	// Save to database
	return r.db.Create(dbProject).Error
}

// ReadProject retrieves a project by its ID
func (r *PostgresProjectRepository) ReadProject(projectID string) (*projectPbv1.Project, error) {
	var dbProject models.Project
	if err := r.db.First(&dbProject, "project_id = ?", projectID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, consts.ErrProjectNotFound
		}
		return nil, err
	}

	// Convert model to protobuf project
	return &projectPbv1.Project{
		ProjectId:   dbProject.ProjectID,
		Name:        dbProject.Name,
		Description: dbProject.Description,
		IssueCount:  dbProject.IssueCount,
	}, nil
}

// UpdateProject updates an existing project
func (r *PostgresProjectRepository) UpdateProject(project *projectPbv1.Project) error {
	// Check if the project exists first
	var existingProject models.Project
	if err := r.db.First(&existingProject, "project_id = ?", project.ProjectId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return consts.ErrProjectNotFound
		}
		return err
	}

	// Update the project
	updates := map[string]interface{}{
		"name":        project.Name,
		"description": project.Description,
		"issue_count": project.IssueCount,
	}

	return r.db.Model(&models.Project{}).Where("project_id = ?", project.ProjectId).Updates(updates).Error
}

// DeleteProject removes a project from the database
func (r *PostgresProjectRepository) DeleteProject(projectID string) error {
	result := r.db.Delete(&models.Project{}, "project_id = ?", projectID)
	if result.Error != nil {
		return result.Error
	}

	// Check if any rows were affected
	if result.RowsAffected == 0 {
		return consts.ErrProjectNotFound
	}

	return nil
}

// ListProjects retrieves all projects
func (r *PostgresProjectRepository) ListProjects() ([]*projectPbv1.Project, error) {
	var dbProjects []models.Project
	if err := r.db.Find(&dbProjects).Error; err != nil {
		return nil, err
	}

	// Convert DB models to protobuf projects
	projects := make([]*projectPbv1.Project, len(dbProjects))
	for i, dbProject := range dbProjects {
		projects[i] = &projectPbv1.Project{
			ProjectId:   dbProject.ProjectID,
			Name:        dbProject.Name,
			Description: dbProject.Description,
			IssueCount:  dbProject.IssueCount,
		}
	}

	return projects, nil
}

// AddIssueToProject associates an issue with a project
func (r *PostgresProjectRepository) AddIssueToProject(projectID string, issueID string) error {
	logger.ZapLogger.Debug("AddIssueToProject called",
		zap.String("project_id", projectID),
		zap.String("issue_id", issueID))

	// Check if project exists first
	var project models.Project
	if err := r.db.First(&project, "project_id = ?", projectID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return consts.ErrProjectNotFound
		}
		return err
	}

	// Check if the issue exists
	var issue models.Issues
	if err := r.db.First(&issue, "issue_id = ?", issueID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return consts.ErrIssueNotFound
		}
		return err
	}

	// Check if issue already belongs to this project
	if issue.ProjectID == projectID {
		return nil
	}

	// Use a transaction with pessimistic locking for both operations
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Lock the project row for update to prevent concurrent modifications
		var lockedProject models.Project
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&lockedProject, "project_id = ?", projectID).Error; err != nil {
			return err
		}

		// Update the issue's project_id field
		if err := tx.Model(&issue).Update("project_id", projectID).Error; err != nil {
			return err
		}

		// Directly increment issue count with SQL to avoid race conditions
		if err := tx.Model(&models.Project{}).
			Where("project_id = ?", projectID).
			UpdateColumn("issue_count", gorm.Expr("issue_count + ?", 1)).Error; err != nil {
			return err
		}

		logger.ZapLogger.Debug("Project issue count incremented",
			zap.String("project_id", projectID),
			zap.String("issue_id", issueID),
			zap.Int32("new_count", lockedProject.IssueCount+1))

		return nil
	})
}

// RemoveIssueFromProject removes an association between an issue and a project
func (r *PostgresProjectRepository) RemoveIssueFromProject(projectID string, issueID string) error {
	// Check if project exists
	var project models.Project
	if err := r.db.First(&project, "project_id = ?", projectID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return consts.ErrProjectNotFound
		}
		return err
	}

	// Check if issue exists and belongs to project (this would be better with a join table)
	var issue models.Issues
	if err := r.db.First(&issue, "issue_id = ? AND project_id = ?", issueID, projectID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return consts.ErrIssueNotFound
		}
		return err
	}

	// Decrement issue count
	if project.IssueCount > 0 {
		project.IssueCount--
	}

	// Update project
	return r.db.Model(&project).Update("issue_count", project.IssueCount).Error
}
