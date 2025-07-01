package issuessvc

import (
	"context"
	"errors"

	"github.com/yasindce1998/issue-tracker/consts"
	"github.com/yasindce1998/issue-tracker/models"
	issuesPbv1 "github.com/yasindce1998/issue-tracker/pkg/pb/issues/v1"
	"gorm.io/gorm"
)

// PostgresIssuesRepository implements IssuesRepository using GORM for PostgreSQL
type PostgresIssuesRepository struct {
	db *gorm.DB
}

// NewPostgresIssuesRepository initializes the repository with a GORM DB instance
func NewPostgresIssuesRepository(db *gorm.DB) *PostgresIssuesRepository {
	return &PostgresIssuesRepository{db: db}
}

// CreateIssue adds a new issue to the database
func (r *PostgresIssuesRepository) CreateIssue(issue *issuesPbv1.Issue) error {
	// Convert protobuf issue to model
	dbIssue := &models.Issues{
		IssueID:     issue.IssueId,
		Summary:     issue.Summary,
		Description: issue.Description,
		Status:      issue.Status.String(),
		Resolution:  issue.Resolution.String(),
		Type:        issue.Type.String(),
		Priority:    issue.Priority.String(),
		ProjectID:   issue.ProjectId,
		AssigneeID:  &issue.AssigneeId,
	}

	// Save to database
	return r.db.Create(dbIssue).Error
}

// ReadIssue retrieves an issue by its ID
func (r *PostgresIssuesRepository) ReadIssue(issueID string) (*issuesPbv1.Issue, error) {
	var dbIssue models.Issues
	if err := r.db.First(&dbIssue, "issue_id = ?", issueID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, consts.ErrIssueNotFound
		}
		return nil, err
	}

	// Convert model to protobuf issue
	var assigneeID string
	if dbIssue.AssigneeID != nil {
		assigneeID = *dbIssue.AssigneeID
	}

	// Parse enum values
	var status issuesPbv1.Status
	var resolution issuesPbv1.Resolution
	var issueType issuesPbv1.Type
	var priority issuesPbv1.Priority

	// Convert string status to enum (this would need proper validation)
	statusValue := issuesPbv1.Status_value[dbIssue.Status]
	status = issuesPbv1.Status(statusValue)

	resolutionValue := issuesPbv1.Resolution_value[dbIssue.Resolution]
	resolution = issuesPbv1.Resolution(resolutionValue)

	typeValue := issuesPbv1.Type_value[dbIssue.Type]
	issueType = issuesPbv1.Type(typeValue)

	priorityValue := issuesPbv1.Priority_value[dbIssue.Priority]
	priority = issuesPbv1.Priority(priorityValue)

	return &issuesPbv1.Issue{
		IssueId:     dbIssue.IssueID,
		Summary:     dbIssue.Summary,
		Description: dbIssue.Description,
		Status:      status,
		Resolution:  resolution,
		Type:        issueType,
		Priority:    priority,
		ProjectId:   dbIssue.ProjectID,
		AssigneeId:  assigneeID,
	}, nil
}

// UpdateIssue updates an existing issue
func (r *PostgresIssuesRepository) UpdateIssue(issue *issuesPbv1.Issue) error {
	// Check if the issue exists first
	var existingIssue models.Issues
	if err := r.db.First(&existingIssue, "issue_id = ?", issue.IssueId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return consts.ErrIssueNotFound
		}
		return err
	}

	// Update the issue
	updates := map[string]interface{}{
		"summary":     issue.Summary,
		"description": issue.Description,
		"status":      issue.Status.String(),
		"resolution":  issue.Resolution.String(),
		"type":        issue.Type.String(),
		"priority":    issue.Priority.String(),
		"project_id":  issue.ProjectId,
		"assignee_id": &issue.AssigneeId,
	}

	return r.db.Model(&models.Issues{}).Where("issue_id = ?", issue.IssueId).Updates(updates).Error
}

// DeleteIssue removes an issue from the database
func (r *PostgresIssuesRepository) DeleteIssue(issueID string) error {
	result := r.db.Delete(&models.Issues{}, "issue_id = ?", issueID)
	if result.Error != nil {
		return result.Error
	}

	// Check if any rows were affected
	if result.RowsAffected == 0 {
		return consts.ErrIssueNotFound
	}

	return nil
}

// ListIssues retrieves a paginated list of issues
func (r *PostgresIssuesRepository) ListIssues(pageToken string, pageSize int) ([]*issuesPbv1.Issue, string, error) {
	var dbIssues []models.Issues
	query := r.db.Limit(pageSize)

	// If we have a page token, use it as an offset
	if pageToken != "" {
		query = query.Where("issue_id > ?", pageToken)
	}

	if err := query.Order("issue_id").Find(&dbIssues).Error; err != nil {
		return nil, "", err
	}

	// Convert DB models to protobuf issues
	issues := make([]*issuesPbv1.Issue, len(dbIssues))
	for i, dbIssue := range dbIssues {
		var assigneeID string
		if dbIssue.AssigneeID != nil {
			assigneeID = *dbIssue.AssigneeID
		}

		// Parse enum values
		statusValue := issuesPbv1.Status_value[dbIssue.Status]
		resolutionValue := issuesPbv1.Resolution_value[dbIssue.Resolution]
		typeValue := issuesPbv1.Type_value[dbIssue.Type]
		priorityValue := issuesPbv1.Priority_value[dbIssue.Priority]

		issues[i] = &issuesPbv1.Issue{
			IssueId:     dbIssue.IssueID,
			Summary:     dbIssue.Summary,
			Description: dbIssue.Description,
			Status:      issuesPbv1.Status(statusValue),
			Resolution:  issuesPbv1.Resolution(resolutionValue),
			Type:        issuesPbv1.Type(typeValue),
			Priority:    issuesPbv1.Priority(priorityValue),
			ProjectId:   dbIssue.ProjectID,
			AssigneeId:  assigneeID,
		}
	}

	// Calculate the next page token
	var nextPageToken string
	if len(issues) == pageSize {
		nextPageToken = issues[len(issues)-1].IssueId
	}

	return issues, nextPageToken, nil
}

// ValidateProjectExists checks if a project with the given ID exists
func (r *PostgresIssuesRepository) ValidateProjectExists(_ context.Context, projectID string) error {
	var count int64
	if err := r.db.Model(&models.Project{}).Where("project_id = ?", projectID).Count(&count).Error; err != nil {
		return err
	}

	if count == 0 {
		return consts.ErrProjectNotFound
	}

	return nil
}

// ValidateUserExists checks if a user with the given ID exists
func (r *PostgresIssuesRepository) ValidateUserExists(_ context.Context, userID string) error {
	var count int64
	if err := r.db.Model(&models.User{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return err
	}

	if count == 0 {
		return consts.ErrUserNotFound
	}

	return nil
}

// IsValidStatusTransition validates whether a status transition is allowed
func (r *PostgresIssuesRepository) IsValidStatusTransition(currentStatus, newStatus issuesPbv1.Status) error {
	// Define valid transitions - same as in MemDB implementation
	validTransitions := map[issuesPbv1.Status][]issuesPbv1.Status{
		issuesPbv1.Status_NEW:         {issuesPbv1.Status_ASSIGNED},
		issuesPbv1.Status_ASSIGNED:    {issuesPbv1.Status_IN_PROGRESS, issuesPbv1.Status_RESOLVED},
		issuesPbv1.Status_IN_PROGRESS: {issuesPbv1.Status_RESOLVED, issuesPbv1.Status_CLOSED},
		issuesPbv1.Status_RESOLVED:    {issuesPbv1.Status_CLOSED},
		issuesPbv1.Status_CLOSED:      {}, // No transitions allowed
	}

	// Check if current status exists in our map
	allowed, exists := validTransitions[currentStatus]
	if !exists {
		return errors.New("invalid current status")
	}

	// If status is not changing, it's always valid
	if currentStatus == newStatus {
		return nil
	}

	// Check if new status is in the allowed transitions
	for _, valid := range allowed {
		if valid == newStatus {
			return nil
		}
	}

	// If we get here, the transition is not allowed
	return errors.New("invalid status transition")
}
