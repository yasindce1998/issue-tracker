// Package issuessvc provides services for managing issues tracking and operations
package issuessvc

import (
	"context"
	"errors"

	issuesPbv1 "github.com/yasindce1998/issue-tracker/pkg/pb/issues/v1"
	projectPbv1 "github.com/yasindce1998/issue-tracker/pkg/pb/project/v1"
	userPbv1 "github.com/yasindce1998/issue-tracker/pkg/pb/user/v1"
	"github.com/hashicorp/go-memdb"
)

// IssuesRepository defines repository methods required for issue operations
type IssuesRepository interface {
	CreateIssue(issue *issuesPbv1.Issue) error
	ReadIssue(issueID string) (*issuesPbv1.Issue, error)
	UpdateIssue(issue *issuesPbv1.Issue) error
	DeleteIssue(issueID string) error
	ListIssues(pageToken string, pageSize int) ([]*issuesPbv1.Issue, string, error)
	ValidateProjectExists(ctx context.Context, projectID string) error
	ValidateUserExists(ctx context.Context, userID string) error
	IsValidStatusTransition(currentStatus, newStatus issuesPbv1.Status) error
}

// MemDBIssuesRepository is an in-memory implementation of IssuesStore
type MemDBIssuesRepository struct {
	db            *memdb.MemDB
	projectClient projectPbv1.ProjectServiceClient
	userClient    userPbv1.UserServiceClient
}

// CreateIssuesMemDBSchema defines the schema for the in-memory database
func CreateIssuesMemDBSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"issue": {
				Name: "issue",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "IssueId"},
					},
				},
			},
		},
	}
}

// SetClients configures the repository with project and user service clients
// after initialization for cross-service validation
func (r *MemDBIssuesRepository) SetClients(projectClient projectPbv1.ProjectServiceClient, userClient userPbv1.UserServiceClient) {
	r.projectClient = projectClient
	r.userClient = userClient
}

// NewMemDBIssuesRepositoryWithoutClients creates a new repository without clients
func NewMemDBIssuesRepositoryWithoutClients() (*MemDBIssuesRepository, error) {
	db, err := memdb.NewMemDB(CreateIssuesMemDBSchema())
	if err != nil {
		return nil, err
	}

	return &MemDBIssuesRepository{
		db: db,
	}, nil
}

// CreateIssue adds a new issue to the repository
func (r *MemDBIssuesRepository) CreateIssue(issue *issuesPbv1.Issue) error {
	txn := r.db.Txn(true)
	defer txn.Commit()
	return txn.Insert("issue", issue)
}

// ReadIssue retrieves an issue by its ID
func (r *MemDBIssuesRepository) ReadIssue(issueID string) (*issuesPbv1.Issue, error) {
	txn := r.db.Txn(false)
	defer txn.Abort()

	raw, err := txn.First("issue", "id", issueID)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, errors.New("issue not found")
	}
	return raw.(*issuesPbv1.Issue), nil
}

// UpdateIssue updates an existing issue in the repository
func (r *MemDBIssuesRepository) UpdateIssue(issue *issuesPbv1.Issue) error {
	txn := r.db.Txn(true)
	defer txn.Commit()
	return txn.Insert("issue", issue)
}

// DeleteIssue removes an issue from the repository
func (r *MemDBIssuesRepository) DeleteIssue(issueID string) error {
	txn := r.db.Txn(true)
	defer txn.Commit()

	raw, err := txn.First("issue", "id", issueID)
	if err != nil {
		return err
	}
	if raw == nil {
		return errors.New("issue not found")
	}

	return txn.Delete("issue", raw)
}

// ListIssues retrieves a paginated list of issues
func (r *MemDBIssuesRepository) ListIssues(pageToken string, pageSize int) ([]*issuesPbv1.Issue, string, error) {
	txn := r.db.Txn(false)
	defer txn.Abort()

	it, err := txn.Get("issue", "id")
	if err != nil {
		return nil, "", err
	}

	var issues []*issuesPbv1.Issue
	for obj := it.Next(); obj != nil; obj = it.Next() {
		issues = append(issues, obj.(*issuesPbv1.Issue))
	}

	issuesPage, nextPageToken := paginateIssues(issues, pageSize, pageToken)
	return issuesPage, nextPageToken, nil
}

// ValidateProjectExists checks if a project with the given ID exists
func (r *MemDBIssuesRepository) ValidateProjectExists(ctx context.Context, projectID string) error {
	// Use the ProjectServiceClient to validate if the project ID exists
	_, err := r.projectClient.GetProject(ctx, &projectPbv1.GetProjectRequest{ProjectId: projectID})
	if err != nil {
		return errors.New("project ID does not exist or could not be validated")
	}
	return nil
}

// ValidateUserExists checks if a user with the given ID exists
func (r *MemDBIssuesRepository) ValidateUserExists(ctx context.Context, userID string) error {
	// Use the UserServiceClient to validate if the user ID exists
	_, err := r.userClient.GetUser(ctx, &userPbv1.GetUserRequest{UserId: userID})
	if err != nil {
		return errors.New("user ID does not exist or could not be validated")
	}
	return nil
}

// IsValidStatusTransition validates whether a status transition is allowed
func (r *MemDBIssuesRepository) IsValidStatusTransition(currentStatus, newStatus issuesPbv1.Status) error {
	validTransitions := map[issuesPbv1.Status][]issuesPbv1.Status{
		issuesPbv1.Status_NEW:         {issuesPbv1.Status_ASSIGNED},
		issuesPbv1.Status_ASSIGNED:    {issuesPbv1.Status_IN_PROGRESS, issuesPbv1.Status_RESOLVED},
		issuesPbv1.Status_IN_PROGRESS: {issuesPbv1.Status_RESOLVED, issuesPbv1.Status_CLOSED},
		issuesPbv1.Status_RESOLVED:    {issuesPbv1.Status_CLOSED},
		issuesPbv1.Status_CLOSED:      {}, // No transitions allowed
	}

	allowed, exists := validTransitions[currentStatus]
	if !exists {
		return errors.New("invalid current status")
	}

	if currentStatus == newStatus {
		return nil
	}

	for _, valid := range allowed {
		if valid == newStatus {
			return nil
		}
	}

	return errors.New("invalid status transition")
}

// Pagination Helper
func paginateIssues(issues []*issuesPbv1.Issue, pageSize int, pageToken string) ([]*issuesPbv1.Issue, string) {
	startIndex := 0
	if pageToken != "" {
		for i, issue := range issues {
			if issue.IssueId == pageToken {
				startIndex = i + 1
				break
			}
		}
	}

	endIndex := startIndex + pageSize
	if endIndex > len(issues) {
		endIndex = len(issues)
	}

	var nextPageToken string
	if endIndex < len(issues) {
		nextPageToken = issues[endIndex-1].IssueId
	}

	return issues[startIndex:endIndex], nextPageToken
}
