package issuessvc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/yasindce1998/issue-tracker/consts"
	"github.com/yasindce1998/issue-tracker/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	issuesPbv1 "github.com/yasindce1998/issue-tracker/pkg/pb/issues/v1"
	projectPbv1 "github.com/yasindce1998/issue-tracker/pkg/pb/project/v1"
	userPbv1 "github.com/yasindce1998/issue-tracker/pkg/pb/user/v1"
)

const (
	defaultPageSize = 10
	maxPageSize     = 100
)

// IssuesServiceServer is the main service structure for the Issues API
type IssuesServiceServer struct {
	issuesPbv1.UnimplementedIssuesServiceServer
	repository     IssuesRepository
	projectService projectPbv1.ProjectServiceClient
	userService    userPbv1.UserServiceClient
	projectFetcher *ProjectServiceClientFetcher
	userFetcher    *UserServiceClientFetcher
}

// ProjectServiceClientFetcher fetches project-related data
type ProjectServiceClientFetcher struct {
	client projectPbv1.ProjectServiceClient
}

// GetProjectDetails fetches project details using the project service
func (p *ProjectServiceClientFetcher) GetProjectDetails(ctx context.Context, projectID string) (*projectPbv1.Project, error) {
	resp, err := p.client.GetProject(ctx, &projectPbv1.GetProjectRequest{ProjectId: projectID})
	if err != nil {
		return nil, err
	}
	return resp.Project, nil
}

// UserServiceClientFetcher fetches user-related data
type UserServiceClientFetcher struct {
	client userPbv1.UserServiceClient
}

// GetUserDetails fetches user details using the user service
func (u *UserServiceClientFetcher) GetUserDetails(ctx context.Context, userID string) (*userPbv1.User, error) {
	resp, err := u.client.GetUser(ctx, &userPbv1.GetUserRequest{UserId: userID})
	if err != nil {
		return nil, err
	}
	return resp.User, nil
}

// NewIssuesService creates a new instance of the issues service
func NewIssuesService(repository IssuesRepository, projectServiceClient projectPbv1.ProjectServiceClient, userServiceClient userPbv1.UserServiceClient) *IssuesServiceServer {
	return &IssuesServiceServer{
		repository:     repository,
		projectService: projectServiceClient,
		userService:    userServiceClient,
		projectFetcher: &ProjectServiceClientFetcher{client: projectServiceClient},
		userFetcher:    &UserServiceClientFetcher{client: userServiceClient},
	}
}

// CreateIssue handles issue creation.
func (s *IssuesServiceServer) CreateIssue(ctx context.Context, req *issuesPbv1.CreateIssueRequest) (*issuesPbv1.CreateIssueResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	// Validate project existence
	if err := s.repository.ValidateProjectExists(ctx, req.ProjectId); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid project: %v", err)
	}

	// Validate assignee if provided
	if req.AssigneeId != nil && *req.AssigneeId != "" {
		if err := s.repository.ValidateUserExists(ctx, *req.AssigneeId); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid user: %v", err)
		}
	}

	// Determine issue status
	issueStatus := issuesPbv1.Status_NEW
	if req.AssigneeId != nil && *req.AssigneeId != "" {
		issueStatus = issuesPbv1.Status_ASSIGNED
	}

	// Create issue entity
	issue := &issuesPbv1.Issue{
		IssueId:     uuid.NewString(),
		Summary:     req.Summary,
		Description: req.GetDescription(),
		Type:        req.Type,
		Priority:    req.Priority,
		Status:      issueStatus,
		ProjectId:   req.ProjectId,
		CreateDate:  timestamppb.Now(),
		ModifyDate:  timestamppb.Now(),
	}

	// Assign assignee if provided
	if req.AssigneeId != nil {
		issue.AssigneeId = *req.AssigneeId
	}

	// Save issue
	if err := s.repository.CreateIssue(issue); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create issue: %v", err)
	}

	// Notify the ProjectService about the new issue, but don't fail if this fails
	projectErr := s.notifyProjectService(ctx, issue.ProjectId, issue.IssueId)
	if projectErr != nil {
		// Log the error but continue with issue creation
		logger.ZapLogger.Error("Failed to notify ProjectService about new issue",
			zap.String("issueId", issue.IssueId),
			zap.String("projectId", issue.ProjectId),
			zap.Error(projectErr))
	}

	// Return response
	return &issuesPbv1.CreateIssueResponse{Issue: issue}, nil
}

// GetIssue retrieves an issue by its ID.
func (s *IssuesServiceServer) GetIssue(ctx context.Context, req *issuesPbv1.GetIssueRequest) (*issuesPbv1.GetIssueResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	issue, err := s.repository.ReadIssue(req.IssueId)
	if err != nil {
		if errors.Is(err, consts.ErrNotFound) { // Ensure proper comparison
			return nil, status.Error(codes.NotFound, "issue not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get issue: %v", err) // Updated error message
	}

	resp := &issuesPbv1.GetIssueResponse{Issue: issue}

	// Optionally fetch and attach detailed project/user info
	if req.IncludeDetails {
		project, _ := s.projectFetcher.GetProjectDetails(ctx, issue.ProjectId)
		user, _ := s.userFetcher.GetUserDetails(ctx, issue.AssigneeId)

		if project != nil {
			resp.ProjectInfo = convertProjectToProjectInfo(project)
		}
		if user != nil {
			resp.UserInfo = convertUserToUserInfo(user)
		}
	}

	return resp, nil
}

// UpdateIssue modifies an existing issue.
//
//nolint:gocyclo,funlen
func (s *IssuesServiceServer) UpdateIssue(ctx context.Context, req *issuesPbv1.UpdateIssueRequest) (*issuesPbv1.UpdateIssueResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	issue, err := s.repository.ReadIssue(req.IssueId)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, status.Error(codes.NotFound, "issue not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to retrieve issue: %v", err)
	}

	// Basic field validations
	if req.Summary == "" || (req.Description != nil && *req.Description == "") ||
		req.Type == issuesPbv1.Type_TYPE_UNSPECIFIED ||
		req.Priority == issuesPbv1.Priority_PRIORITY_UNSPECIFIED ||
		req.Status == issuesPbv1.Status_STATUS_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "summary, description, type, priority, and status are required for update")
	}

	// Determine if assignee is being updated
	assigneeUpdated := req.AssigneeId != nil
	hasAssignee := assigneeUpdated && *req.AssigneeId != ""

	// Validate assignee ID if it's being updated
	if hasAssignee && *req.AssigneeId != issue.AssigneeId {
		if err := s.repository.ValidateUserExists(ctx, *req.AssigneeId); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid assignee: %v", err)
		}
	}

	// Enforce status based on assignee
	autoAdjustStatus := false
	requestedStatus := req.Status

	// If assignee is being removed, enforce status NEW
	if assigneeUpdated && !hasAssignee && issue.AssigneeId != "" {
		// Validate the status isn't ASSIGNED or IN_PROGRESS
		if req.Status == issuesPbv1.Status_ASSIGNED || req.Status == issuesPbv1.Status_IN_PROGRESS {
			return nil, status.Error(codes.InvalidArgument, "cannot set status to ASSIGNED or IN_PROGRESS when removing an assignee")
		}
	}

	// If assignee is being added (and there wasn't one before), enforce status ASSIGNED
	if hasAssignee && issue.AssigneeId == "" {
		req.Status = issuesPbv1.Status_ASSIGNED
		autoAdjustStatus = true
	}

	// Validate resolution if status is Resolved or Closed
	if (req.Status == issuesPbv1.Status_RESOLVED || req.Status == issuesPbv1.Status_CLOSED) &&
		req.Resolution == issuesPbv1.Resolution_RESOLUTION_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "resolution is required when status is Resolved or Closed")
	}

	// Validate assignee if status is Assigned or In Progress
	if (req.Status == issuesPbv1.Status_ASSIGNED || req.Status == issuesPbv1.Status_IN_PROGRESS) &&
		(assigneeUpdated && !hasAssignee || (!assigneeUpdated && issue.AssigneeId == "")) {
		return nil, status.Error(codes.InvalidArgument, "assignee is required when status is Assigned or In Progress")
	}

	// Validate status transition (skip if auto-adjusted)
	if !autoAdjustStatus {
		if err := s.repository.IsValidStatusTransition(issue.Status, req.Status); err != nil {
			return nil, err
		}
	}

	// Update issue fields
	issue.Summary = req.Summary
	issue.Description = req.GetDescription()
	issue.Type = req.Type
	issue.Priority = req.Priority
	issue.Status = req.Status
	issue.ModifyDate = timestamppb.Now()

	// Update assignee (if provided) or remove it (if explicitly set to empty)
	if assigneeUpdated {
		issue.AssigneeId = *req.AssigneeId
	} else if req.Status == issuesPbv1.Status_NEW {
		// Ensure NEW status has no assignee
		issue.AssigneeId = ""
	}

	// Only update resolution if specified
	if req.Resolution != issuesPbv1.Resolution_RESOLUTION_UNSPECIFIED {
		issue.Resolution = req.Resolution
	}

	if err := s.repository.UpdateIssue(issue); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update issue: %v", err)
	}

	// Create response with additional information
	responseMsg := fmt.Sprintf("Issue with id %s has been updated", issue.IssueId)
	if autoAdjustStatus {
		responseMsg += fmt.Sprintf(" (status automatically adjusted from %s to %s based on assignee)", requestedStatus, req.Status)
	}

	return &issuesPbv1.UpdateIssueResponse{
		Issue:   issue,
		Message: responseMsg,
	}, nil
}

// DeleteIssue removes an issue by its ID.
func (s *IssuesServiceServer) DeleteIssue(_ context.Context, req *issuesPbv1.DeleteIssueRequest) (*issuesPbv1.DeleteIssueResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	issue, err := s.repository.ReadIssue(req.IssueId)
	if err != nil {
		if errors.Is(err, status.Error(codes.NotFound, "issue not found")) {
			return nil, status.Error(codes.NotFound, "issue not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to retrieve issue: %v", err)
	}

	if err := s.repository.DeleteIssue(req.IssueId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete issue: %v", err)
	}

	return &issuesPbv1.DeleteIssueResponse{Issue: issue}, nil
}

// ListIssues retrieves paginated issues.
func (s *IssuesServiceServer) ListIssues(_ context.Context, req *issuesPbv1.ListIssuesRequest) (*issuesPbv1.ListIssuesResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	issues, nextPageToken, err := s.repository.ListIssues(req.PageToken, pageSize)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list issues: %v", err)
	}

	return &issuesPbv1.ListIssuesResponse{
		Issues:        issues,
		NextPageToken: nextPageToken,
	}, nil
}

// notifyProjectService notify the issue creation for the project
func (s *IssuesServiceServer) notifyProjectService(ctx context.Context, projectID, issueID string) error {
	// Add context timeout to prevent long-running requests
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Try to notify the project service
	_, err := s.projectService.UpdateProjectWithIssue(ctx, &projectPbv1.UpdateProjectWithIssueRequest{
		ProjectId: projectID,
		IssueId:   issueID,
	})

	return err
}

// Helper functions to convert between user/project and issue protobuf types
func convertProjectToProjectInfo(project *projectPbv1.Project) *issuesPbv1.ProjectInfo {
	return &issuesPbv1.ProjectInfo{
		ProjectId:   project.ProjectId,
		Name:        project.Name,
		Description: project.Description,
	}
}

func convertUserToUserInfo(user *userPbv1.User) *issuesPbv1.UserInfo {
	return &issuesPbv1.UserInfo{
		UserId:    user.UserId,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.EmailAddress,
	}
}
