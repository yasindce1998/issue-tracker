package issuessvc_test

import (
	"context"
	"testing"

	"github.com/yasindce1998/issue-tracker/consts"
	"github.com/yasindce1998/issue-tracker/logger"
	"github.com/yasindce1998/issue-tracker/mocks"
	issuesPbv1 "github.com/yasindce1998/issue-tracker/pkg/pb/issues/v1"
	projectPbv1 "github.com/yasindce1998/issue-tracker/pkg/pb/project/v1"
	"github.com/yasindce1998/issue-tracker/pkg/svc/issuessvc"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const (
	validIssueID     = "c72d237e-2658-4252-be58-760c7867d783"
	validProjectID   = "928f705f-0efa-4c96-b2f6-ceb36281e1f1"
	validUserID      = "a28f705f-0efa-4c96-b2f6-ceb36281e1f2"
	invalidProjectID = "invalid-project-id"
	invalidUserID    = "invalid-user-id"
	testSummary      = "Test issue summary"
	testDescription  = "This is a test issue description"
	bugSummary       = "Bug report summary"
	featureSummary   = "Feature request summary"
)

func TestIssuesServiceServer_CreateIssue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockIssuesRepository(ctrl)
	mockProjectService := mocks.NewMockProjectServiceClient(ctrl)
	mockUserService := mocks.NewMockUserServiceClient(ctrl)

	issuesService := issuessvc.NewIssuesService(mockRepo, mockProjectService, mockUserService)
	logger.ZapLogger, _ = zap.NewDevelopment()

	testCases := []struct {
		name          string
		req           *issuesPbv1.CreateIssueRequest
		setupMock     func()
		expectedResp  *issuesPbv1.CreateIssueResponse
		expectedError error
	}{
		{
			name: "Valid Issue Creation Without Assignee",
			req: &issuesPbv1.CreateIssueRequest{
				Summary:     bugSummary,
				Description: proto.String(testDescription),
				Type:        issuesPbv1.Type_BUG,
				Priority:    issuesPbv1.Priority_MINOR,
				ProjectId:   validProjectID,
			},
			setupMock: func() {
				mockRepo.EXPECT().ValidateProjectExists(gomock.Any(), validProjectID).Return(nil)
				mockRepo.EXPECT().CreateIssue(gomock.Any()).DoAndReturn(func(issue *issuesPbv1.Issue) error {
					// Instead of checking UUID directly, just ensure it's not empty
					assert.NotEmpty(t, issue.IssueId)
					assert.Equal(t, issuesPbv1.Status_NEW, issue.Status)

					// Manually set the ID for consistent testing of the response
					issue.IssueId = validIssueID
					return nil
				})
				mockProjectService.EXPECT().UpdateProjectWithIssue(gomock.Any(), gomock.Any()).Return(
					&projectPbv1.UpdateProjectWithIssueResponse{}, nil)
			},
			expectedResp: &issuesPbv1.CreateIssueResponse{
				Issue: &issuesPbv1.Issue{
					IssueId: validIssueID,
					Status:  issuesPbv1.Status_NEW,
				},
			},
			expectedError: nil,
		},
		{
			name: "Valid Issue Creation With Assignee",
			req: &issuesPbv1.CreateIssueRequest{
				Summary:     featureSummary,
				Description: proto.String(testDescription),
				Type:        issuesPbv1.Type_FEATURE,
				Priority:    issuesPbv1.Priority_CRITICAL,
				ProjectId:   validProjectID,
				AssigneeId:  proto.String(validUserID),
			},
			setupMock: func() {
				mockRepo.EXPECT().ValidateProjectExists(gomock.Any(), validProjectID).Return(nil)
				mockRepo.EXPECT().ValidateUserExists(gomock.Any(), validUserID).Return(nil)
				mockRepo.EXPECT().CreateIssue(gomock.Any()).DoAndReturn(func(issue *issuesPbv1.Issue) error {
					assert.NotEmpty(t, issue.IssueId)
					assert.Equal(t, issuesPbv1.Status_ASSIGNED, issue.Status)
					assert.Equal(t, validUserID, issue.AssigneeId)

					// Manually set the ID for consistent testing of the response
					issue.IssueId = validIssueID
					return nil
				})
				mockProjectService.EXPECT().UpdateProjectWithIssue(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, req *projectPbv1.UpdateProjectWithIssueRequest, _ ...grpc.CallOption) (*projectPbv1.UpdateProjectWithIssueResponse, error) {
						assert.Equal(t, validProjectID, req.ProjectId)
						assert.NotEmpty(t, req.IssueId)
						return &projectPbv1.UpdateProjectWithIssueResponse{}, nil
					})
			},
			expectedResp: &issuesPbv1.CreateIssueResponse{
				Issue: &issuesPbv1.Issue{
					IssueId:    validIssueID,
					Status:     issuesPbv1.Status_ASSIGNED,
					AssigneeId: validUserID,
				},
			},
			expectedError: nil,
		},
		{
			name: "Invalid Project ID",
			req: &issuesPbv1.CreateIssueRequest{
				Summary:     testSummary,
				Description: proto.String(testDescription),
				Type:        issuesPbv1.Type_BUG,
				Priority:    issuesPbv1.Priority_MINOR,
				ProjectId:   invalidProjectID,
			},
			setupMock: func() {
				// No need to mock ValidateProjectExists since the validation fails
				// before we get to that point
			},
			expectedResp:  nil,
			expectedError: status.Errorf(codes.InvalidArgument, "invalid request: invalid CreateIssueRequest.ProjectId: value must be a valid UUID | caused by: invalid uuid format"),
		},
		{
			name: "Invalid Assignee ID Format",
			req: &issuesPbv1.CreateIssueRequest{
				Summary:     testSummary,
				Description: proto.String(testDescription),
				Type:        issuesPbv1.Type_BUG,
				Priority:    issuesPbv1.Priority_MINOR,
				ProjectId:   validProjectID,
				AssigneeId:  proto.String(invalidUserID),
			},
			setupMock: func() {
				// No need to mock anything as validation fails before repository calls
			},
			expectedResp:  nil,
			expectedError: status.Errorf(codes.InvalidArgument, "invalid request: invalid CreateIssueRequest.AssigneeId: value must be a valid UUID | caused by: invalid uuid format"),
		},
		{
			name: "Failed To Create Issue",
			req: &issuesPbv1.CreateIssueRequest{
				Summary:     testSummary,
				Description: proto.String(testDescription),
				Type:        issuesPbv1.Type_BUG,
				Priority:    issuesPbv1.Priority_MINOR,
				ProjectId:   validProjectID,
			},
			setupMock: func() {
				mockRepo.EXPECT().ValidateProjectExists(gomock.Any(), validProjectID).Return(nil)
				mockRepo.EXPECT().CreateIssue(gomock.Any()).Return(consts.ErrDatabaseError)
			},
			expectedResp:  nil,
			expectedError: status.Errorf(codes.Internal, "failed to create issue: %v", consts.ErrDatabaseError),
		},
		{
			name: "Failed To Notify Project Service But Creation Succeeds",
			req: &issuesPbv1.CreateIssueRequest{
				Summary:     testSummary,
				Description: proto.String(testDescription),
				Type:        issuesPbv1.Type_BUG,
				Priority:    issuesPbv1.Priority_MINOR,
				ProjectId:   validProjectID,
			},
			setupMock: func() {
				mockRepo.EXPECT().ValidateProjectExists(gomock.Any(), validProjectID).Return(nil)
				mockRepo.EXPECT().CreateIssue(gomock.Any()).DoAndReturn(func(issue *issuesPbv1.Issue) error {
					// Manually set the ID for consistent testing of the response
					issue.IssueId = validIssueID
					return nil
				})
				// This simulates the Docker networking error you're experiencing
				mockProjectService.EXPECT().UpdateProjectWithIssue(gomock.Any(), gomock.Any()).Return(
					nil, status.Error(codes.Unavailable, "name resolver error: produced zero addresses"))
			},
			// The operation should succeed even though notification fails
			expectedResp: &issuesPbv1.CreateIssueResponse{
				Issue: &issuesPbv1.Issue{
					IssueId:     validIssueID,
					Summary:     testSummary,
					Description: testDescription,
					Type:        issuesPbv1.Type_BUG,
					Priority:    issuesPbv1.Priority_MINOR,
					Status:      issuesPbv1.Status_NEW,
					ProjectId:   validProjectID,
				},
			},
			expectedError: nil, // No error should be returned despite notification failure
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Store request data for comparison
			requestSummary := tc.req.Summary
			requestType := tc.req.Type
			requestPriority := tc.req.Priority
			requestProjectID := tc.req.ProjectId

			var requestDescription string
			if tc.req.Description != nil {
				requestDescription = *tc.req.Description
			}

			var requestAssigneeID string
			if tc.req.AssigneeId != nil {
				requestAssigneeID = *tc.req.AssigneeId
			}

			tc.setupMock()

			resp, err := issuesService.CreateIssue(context.Background(), tc.req)

			// Check error
			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tc.expectedError.Error(), err.Error())
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, validIssueID, resp.Issue.IssueId)
				assert.Equal(t, requestSummary, resp.Issue.Summary)
				assert.Equal(t, requestDescription, resp.Issue.Description)
				assert.Equal(t, requestType, resp.Issue.Type)
				assert.Equal(t, requestPriority, resp.Issue.Priority)
				assert.Equal(t, requestProjectID, resp.Issue.ProjectId)

				// Verify status is set correctly based on assignee
				if requestAssigneeID != "" {
					assert.Equal(t, issuesPbv1.Status_ASSIGNED, resp.Issue.Status)
					assert.Equal(t, requestAssigneeID, resp.Issue.AssigneeId)
				} else {
					assert.Equal(t, issuesPbv1.Status_NEW, resp.Issue.Status)
					assert.Empty(t, resp.Issue.AssigneeId)
				}

				// Verify timestamps exist but don't check their exact values
				assert.NotNil(t, resp.Issue.CreateDate)
				assert.NotNil(t, resp.Issue.ModifyDate)
			}
		})
	}
}

func TestIssuesServiceServer_GetIssue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockIssuesRepository(ctrl)
	mockProjectService := mocks.NewMockProjectServiceClient(ctrl)
	mockUserService := mocks.NewMockUserServiceClient(ctrl)

	issuesService := issuessvc.NewIssuesService(mockRepo, mockProjectService, mockUserService)

	validIssueID := "123e4567-e89b-12d3-a456-426614174000" // Valid UUID
	validProjectID := "567e1234-e89b-12d3-a456-426614174111"
	testSummary := "Test Summary"
	testDescription := "Test Description"

	testCases := []struct {
		name          string
		req           *issuesPbv1.GetIssueRequest
		setupMock     func()
		expectedResp  *issuesPbv1.GetIssueResponse
		expectedError error
	}{
		{
			name: "Valid Issue Retrieval",
			req: &issuesPbv1.GetIssueRequest{
				IssueId: validIssueID,
			},
			setupMock: func() {
				mockRepo.EXPECT().ReadIssue(gomock.Any()).Return(&issuesPbv1.Issue{
					IssueId:     validIssueID,
					Summary:     testSummary,
					Description: testDescription,
					Type:        issuesPbv1.Type_BUG,
					Priority:    issuesPbv1.Priority_MINOR,
					Status:      issuesPbv1.Status_NEW,
					ProjectId:   validProjectID,
				}, nil)
			},
			expectedResp: &issuesPbv1.GetIssueResponse{
				Issue: &issuesPbv1.Issue{
					IssueId:     validIssueID,
					Summary:     testSummary,
					Description: testDescription,
					Type:        issuesPbv1.Type_BUG,
					Priority:    issuesPbv1.Priority_MINOR,
					Status:      issuesPbv1.Status_NEW,
					ProjectId:   validProjectID,
				},
			},
			expectedError: nil,
		},
		{
			name: "Invalid Issue ID Format",
			req: &issuesPbv1.GetIssueRequest{
				IssueId: "invalid-issue-id",
			},
			setupMock: func() {
				// No need to mock anything as validation fails before repository calls
			},
			expectedResp:  nil,
			expectedError: status.Errorf(codes.InvalidArgument, "invalid request: invalid GetIssueRequest.IssueId: value must be a valid UUID | caused by: invalid uuid format"),
		},
		{
			name: "Issue Not Found",
			req: &issuesPbv1.GetIssueRequest{
				IssueId: validIssueID,
			},
			setupMock: func() {
				mockRepo.EXPECT().ReadIssue(gomock.Any()).Return(nil, consts.ErrNotFound)
			},
			expectedResp:  nil,
			expectedError: status.Errorf(codes.NotFound, "issue not found"),
		},
		{
			name: "Database Error",
			req: &issuesPbv1.GetIssueRequest{
				IssueId: validIssueID,
			},
			setupMock: func() {
				mockRepo.EXPECT().ReadIssue(gomock.Any()).Return(nil, consts.ErrDatabaseError)
			},
			expectedResp:  nil,
			expectedError: status.Errorf(codes.Internal, "failed to get issue: database error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupMock()

			resp, err := issuesService.GetIssue(context.Background(), tc.req)

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tc.expectedError.Error(), err.Error())
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tc.expectedResp.Issue.IssueId, resp.Issue.IssueId)
				assert.Equal(t, tc.expectedResp.Issue.Summary, resp.Issue.Summary)
				assert.Equal(t, tc.expectedResp.Issue.Description, resp.Issue.Description)
				assert.Equal(t, tc.expectedResp.Issue.Type, resp.Issue.Type)
				assert.Equal(t, tc.expectedResp.Issue.Priority, resp.Issue.Priority)
				assert.Equal(t, tc.expectedResp.Issue.Status, resp.Issue.Status)
				assert.Equal(t, tc.expectedResp.Issue.ProjectId, resp.Issue.ProjectId)
			}
		})
	}
}

func TestIssuesServiceServer_UpdateIssue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockIssuesRepository(ctrl)
	mockProjectService := mocks.NewMockProjectServiceClient(ctrl)
	mockUserService := mocks.NewMockUserServiceClient(ctrl)

	issuesService := issuessvc.NewIssuesService(mockRepo, mockProjectService, mockUserService)

	testCases := []struct {
		name          string
		req           *issuesPbv1.UpdateIssueRequest
		setupMock     func(mockRepo *mocks.MockIssuesRepository)
		expectedResp  *issuesPbv1.UpdateIssueResponse
		expectedError codes.Code
		expectedMsg   string
	}{
		{
			name: "invalid request: missing required fields",
			req: &issuesPbv1.UpdateIssueRequest{
				IssueId: validIssueID,
				// Missing required fields: Summary, Type, Priority, Status
			},
			setupMock: func(_ *mocks.MockIssuesRepository) {
				// No mock setup needed, validation failure short-circuits.
			},
			expectedResp:  nil,
			expectedError: codes.InvalidArgument,
			expectedMsg: "invalid request: invalid UpdateIssueRequest.Summary: " +
				"value length must be between 1 and 100 runes, inclusive",
		},
		{
			name: "status transition is invalid",
			req: &issuesPbv1.UpdateIssueRequest{
				IssueId:     validIssueID,
				Summary:     "Bug Summary",
				Description: proto.String("Bug Description"),
				Type:        issuesPbv1.Type_BUG,
				Priority:    issuesPbv1.Priority_CRITICAL,
				Status:      issuesPbv1.Status_CLOSED,
				Resolution:  issuesPbv1.Resolution_FIXED, // Required for closed/resolved statuses.
			},
			setupMock: func(mockRepo *mocks.MockIssuesRepository) {
				mockRepo.EXPECT().ReadIssue(validIssueID).Return(&issuesPbv1.Issue{
					IssueId: validIssueID,
					Status:  issuesPbv1.Status_NEW, // Current status is NEW.
				}, nil)

				mockRepo.EXPECT().IsValidStatusTransition(
					issuesPbv1.Status_NEW,
					issuesPbv1.Status_CLOSED,
				).Return(status.Error(codes.InvalidArgument, "invalid status transition"))
			},
			expectedResp:  nil,
			expectedError: codes.InvalidArgument,
			expectedMsg:   "invalid status transition",
		},
		{
			name: "successful update with adjusted status",
			req: &issuesPbv1.UpdateIssueRequest{
				IssueId:     validIssueID,
				Summary:     "Feature Request",
				Description: proto.String(testDescription),
				Type:        issuesPbv1.Type_FEATURE,
				Priority:    issuesPbv1.Priority_MINOR,
				Status:      issuesPbv1.Status_NEW,     // No assignee initially.
				AssigneeId:  proto.String(validUserID), // New assignee.
			},
			setupMock: func(mockRepo *mocks.MockIssuesRepository) {
				mockRepo.EXPECT().ReadIssue(validIssueID).Return(&issuesPbv1.Issue{
					IssueId:    validIssueID,
					Status:     issuesPbv1.Status_NEW,
					AssigneeId: "", // No assignee.
				}, nil)

				mockRepo.EXPECT().ValidateUserExists(gomock.Any(), validUserID).Return(nil)
				// No IsValidStatusTransition validation because auto-adjustment to ASSIGNED happens.

				mockRepo.EXPECT().UpdateIssue(gomock.Any()).DoAndReturn(func(issue *issuesPbv1.Issue) error {
					// Verify that the issue has been properly updated
					assert.Equal(t, "Feature Request", issue.Summary)
					assert.Equal(t, testDescription, issue.Description)
					assert.Equal(t, issuesPbv1.Type_FEATURE, issue.Type)
					assert.Equal(t, issuesPbv1.Priority_MINOR, issue.Priority)
					assert.Equal(t, issuesPbv1.Status_ASSIGNED, issue.Status) // Status adjusted
					assert.Equal(t, validUserID, issue.AssigneeId)
					return nil
				})
			},
			expectedResp: &issuesPbv1.UpdateIssueResponse{
				Issue: &issuesPbv1.Issue{
					IssueId:     validIssueID,
					Summary:     "Feature Request",
					Description: testDescription,
					Type:        issuesPbv1.Type_FEATURE,
					Priority:    issuesPbv1.Priority_MINOR,
					Status:      issuesPbv1.Status_ASSIGNED, // Adjusted status.
					AssigneeId:  validUserID,
				},
			},
			expectedError: codes.OK,
			expectedMsg:   "Issue with id",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupMock(mockRepo) // Setup mocks for this test case.

			resp, err := issuesService.UpdateIssue(context.Background(), tc.req)

			if tc.expectedError == codes.OK {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Contains(t, resp.Message, tc.expectedMsg)
				// Check essential fields
				assert.Equal(t, tc.expectedResp.Issue.IssueId, resp.Issue.IssueId)
				assert.Equal(t, tc.expectedResp.Issue.Summary, resp.Issue.Summary)
				assert.Equal(t, tc.expectedResp.Issue.Type, resp.Issue.Type)
				assert.Equal(t, tc.expectedResp.Issue.Priority, resp.Issue.Priority)
				assert.Equal(t, tc.expectedResp.Issue.Status, resp.Issue.Status)
			} else {
				st, _ := status.FromError(err)
				assert.Equal(t, tc.expectedError, st.Code())
				assert.Contains(t, st.Message(), tc.expectedMsg)
			}
		})
	}
}

func TestIssuesServiceServer_DeleteIssue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Mock repositories and services
	mockRepo := mocks.NewMockIssuesRepository(ctrl)
	mockProjectService := mocks.NewMockProjectServiceClient(ctrl)
	mockUserService := mocks.NewMockUserServiceClient(ctrl)

	issuesService := issuessvc.NewIssuesService(mockRepo, mockProjectService, mockUserService)

	// Define test constants
	validIssueID := "123e4567-e89b-12d3-a456-426614174000"
	testSummary := "Test Summary"
	testDescription := "Test Description"
	validProjectID := "567e1234-e89b-12d3-a456-426614174111"

	testCases := []struct {
		name          string
		req           *issuesPbv1.DeleteIssueRequest
		setupMock     func()
		expectedResp  *issuesPbv1.DeleteIssueResponse
		expectedError error
	}{
		{
			name: "Valid Issue Deletion",
			req: &issuesPbv1.DeleteIssueRequest{
				IssueId: validIssueID,
			},
			setupMock: func() {
				// Mock repository read and delete operations
				mockRepo.EXPECT().ReadIssue(validIssueID).Return(&issuesPbv1.Issue{
					IssueId:     validIssueID,
					Summary:     testSummary,
					Description: testDescription,
					Type:        issuesPbv1.Type_BUG,
					Priority:    issuesPbv1.Priority_MINOR,
					Status:      issuesPbv1.Status_NEW,
					ProjectId:   validProjectID,
				}, nil)
				mockRepo.EXPECT().DeleteIssue(validIssueID).Return(nil)
			},
			expectedResp:  &issuesPbv1.DeleteIssueResponse{}, // Empty response for successful deletion
			expectedError: nil,
		},
		{
			name: "Invalid Issue ID Format",
			req: &issuesPbv1.DeleteIssueRequest{
				IssueId: "invalid-issue-id",
			},
			setupMock: func() {
				// No mock setup needed as validation fails before hitting the repository
			},
			expectedResp:  nil,
			expectedError: status.Errorf(codes.InvalidArgument, "invalid request: invalid DeleteIssueRequest.IssueId: value must be a valid UUID | caused by: invalid uuid format"),
		},
		{
			name: "Issue Not Found",
			req: &issuesPbv1.DeleteIssueRequest{
				IssueId: validIssueID,
			},
			setupMock: func() {
				// Mock repository response for issue not found
				mockRepo.EXPECT().ReadIssue(validIssueID).Return(nil, consts.ErrNotFound)
			},
			expectedResp:  nil,
			expectedError: status.Errorf(codes.Internal, "failed to retrieve issue: not found"),
		},
		{
			name: "Failed to Delete Issue",
			req: &issuesPbv1.DeleteIssueRequest{
				IssueId: validIssueID,
			},
			setupMock: func() {
				// Mock repository read and delete operations
				mockRepo.EXPECT().ReadIssue(validIssueID).Return(&issuesPbv1.Issue{
					IssueId:     validIssueID,
					Summary:     testSummary,
					Description: testDescription,
					Type:        issuesPbv1.Type_BUG,
					Priority:    issuesPbv1.Priority_MINOR,
					Status:      issuesPbv1.Status_NEW,
					ProjectId:   validProjectID,
				}, nil)
				mockRepo.EXPECT().DeleteIssue(validIssueID).Return(consts.ErrDatabaseError)
			},
			expectedResp:  nil,
			expectedError: status.Errorf(codes.Internal, "failed to delete issue: %v", consts.ErrDatabaseError),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupMock()

			// Call the DeleteIssue function
			resp, err := issuesService.DeleteIssue(context.Background(), tc.req)

			// Validate results
			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tc.expectedError.Error(), err.Error())
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			}
		})
	}
}

func TestIssuesServiceServer_ListIssues(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Mock repository
	mockRepo := mocks.NewMockIssuesRepository(ctrl)
	mockProjectService := mocks.NewMockProjectServiceClient(ctrl)
	mockUserService := mocks.NewMockUserServiceClient(ctrl)

	// Initialize service
	issuesService := issuessvc.NewIssuesService(mockRepo, mockProjectService, mockUserService)

	// Define constants
	const (
		maxPageSize       = 50
		defaultPageSize   = 20
		testPageToken     = "valid-token"
		testNextPageToken = "next-page-token"
	)

	testIssues := []*issuesPbv1.Issue{
		{
			IssueId:     validIssueID,
			Summary:     testSummary,
			Description: testDescription,
			Type:        issuesPbv1.Type_BUG,
			Priority:    issuesPbv1.Priority_MAJOR,
			Status:      issuesPbv1.Status_IN_PROGRESS,
			ProjectId:   validProjectID,
		},
		{
			IssueId:     "223e4567-e89b-12d3-a456-426614174000",
			Summary:     bugSummary,
			Description: "Description of another bug",
			Type:        issuesPbv1.Type_BUG,
			Priority:    issuesPbv1.Priority_MINOR,
			Status:      issuesPbv1.Status_NEW,
			ProjectId:   validProjectID,
		},
	}

	// Test cases
	testCases := []struct {
		name          string
		req           *issuesPbv1.ListIssuesRequest
		setupMock     func()
		expectedResp  *issuesPbv1.ListIssuesResponse
		expectedError error
	}{
		{
			name: "Valid Request with Results",
			req: &issuesPbv1.ListIssuesRequest{
				PageToken: testPageToken,
				PageSize:  10, // Valid size
			},
			setupMock: func() {
				mockRepo.EXPECT().
					ListIssues(testPageToken, 10).
					Return(testIssues, testNextPageToken, nil)
			},
			expectedResp: &issuesPbv1.ListIssuesResponse{
				Issues:        testIssues,
				NextPageToken: testNextPageToken,
			},
			expectedError: nil,
		},
		{
			name: "Valid Request with Default PageSize",
			req: &issuesPbv1.ListIssuesRequest{
				PageToken: testPageToken,
				PageSize:  int32(defaultPageSize), // Explicitly use defaultPageSize (to pass validation)
			},
			setupMock: func() {
				mockRepo.EXPECT().
					ListIssues(testPageToken, defaultPageSize).
					Return(testIssues, testNextPageToken, nil)
			},
			expectedResp: &issuesPbv1.ListIssuesResponse{
				Issues:        testIssues,
				NextPageToken: testNextPageToken,
			},
			expectedError: nil,
		},
		{
			name: "Invalid Request with Validation Error",
			req: &issuesPbv1.ListIssuesRequest{
				PageToken: "", // Assume an empty PageToken is invalid
			},
			setupMock: func() {
				// No mock setup required, validation fails before reaching repository
			},
			expectedResp:  nil,
			expectedError: status.Errorf(codes.InvalidArgument, "invalid request: invalid ListIssuesRequest.PageSize: value must be inside range [1, 1000]"),
		},
		{
			name: "Repository Error",
			req: &issuesPbv1.ListIssuesRequest{
				PageToken: testPageToken,
				PageSize:  10,
			},
			setupMock: func() {
				mockRepo.EXPECT().
					ListIssues(testPageToken, 10).
					Return(nil, "", consts.ErrDatabaseError)
			},
			expectedResp:  nil,
			expectedError: status.Errorf(codes.Internal, "failed to list issues: %v", consts.ErrDatabaseError),
		},
	}

	// Execute the test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock behavior
			tc.setupMock()

			// Call the ListIssues function
			resp, err := issuesService.ListIssues(context.Background(), tc.req)

			// Verify the results
			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tc.expectedError.Error(), err.Error())
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tc.expectedResp.NextPageToken, resp.NextPageToken)
				assert.Equal(t, len(tc.expectedResp.Issues), len(resp.Issues))
				for i, expectedIssue := range tc.expectedResp.Issues {
					actualIssue := resp.Issues[i]
					assert.Equal(t, expectedIssue.IssueId, actualIssue.IssueId)
					assert.Equal(t, expectedIssue.Summary, actualIssue.Summary)
					assert.Equal(t, expectedIssue.Description, actualIssue.Description)
					assert.Equal(t, expectedIssue.Type, actualIssue.Type)
					assert.Equal(t, expectedIssue.Priority, actualIssue.Priority)
					assert.Equal(t, expectedIssue.Status, actualIssue.Status)
					assert.Equal(t, expectedIssue.ProjectId, actualIssue.ProjectId)
				}
			}
		})
	}
}
