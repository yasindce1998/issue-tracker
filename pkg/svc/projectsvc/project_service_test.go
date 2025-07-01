package projectsvc_test

import (
	"context"
	"errors"
	"testing"

	"github.com/yasindce1998/issue-tracker/logger"
	"github.com/yasindce1998/issue-tracker/mocks"
	projectPbv1 "github.com/yasindce1998/issue-tracker/pkg/pb/project/v1"
	"github.com/yasindce1998/issue-tracker/pkg/svc/projectsvc"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestCreateProject(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name        string
		req         *projectPbv1.CreateProjectRequest
		mockSetup   func(mockRepo *mocks.MockProjectRepository)
		expectedErr codes.Code
		checkResp   func(t *testing.T, resp *projectPbv1.CreateProjectResponse)
	}{
		{
			name: "Successful project creation",
			req: &projectPbv1.CreateProjectRequest{
				Name:        "Test Project",
				Description: "This is a test project",
			},
			mockSetup: func(mockRepo *mocks.MockProjectRepository) {
				// Verify the project has proper fields before saving
				mockRepo.EXPECT().CreateProject(gomock.Any()).DoAndReturn(
					func(project *projectPbv1.Project) error {
						// Verify project fields
						if project.Name != "Test Project" ||
							project.Description != "This is a test project" ||
							project.IssueCount != 0 {
							return errors.New("project fields don't match expected values")
						}
						return nil
					})
			},
			expectedErr: codes.OK,
			checkResp: func(t *testing.T, resp *projectPbv1.CreateProjectResponse) {
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Project.ProjectId)
				assert.Equal(t, "Test Project", resp.Project.Name)
				assert.Equal(t, "This is a test project", resp.Project.Description)
				assert.Equal(t, int32(0), resp.Project.IssueCount)
			},
		},
		{
			name: "Repository error",
			req: &projectPbv1.CreateProjectRequest{
				Name:        "Failed Project",
				Description: "This project will fail",
			},
			mockSetup: func(mockRepo *mocks.MockProjectRepository) {
				mockRepo.EXPECT().CreateProject(gomock.Any()).Return(errors.New("database error"))
			},
			expectedErr: codes.Internal,
			checkResp: func(t *testing.T, resp *projectPbv1.CreateProjectResponse) {
				assert.Nil(t, resp)
			},
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock controller
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Create mock repository
			mockRepo := mocks.NewMockProjectRepository(ctrl)

			// Configure mock behavior
			tc.mockSetup(mockRepo)

			// Create the service with mock repository
			service, _ := projectsvc.NewProjectService(mockRepo)

			// Call the method
			resp, err := service.CreateProject(context.Background(), tc.req)

			// Check error if expected
			if tc.expectedErr != codes.OK {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tc.expectedErr, st.Code())
			} else {
				assert.NoError(t, err)
			}

			// Verify response
			tc.checkResp(t, resp)
		})
	}
}

func TestGetProject(t *testing.T) {
	// Create test project
	testProject := &projectPbv1.Project{
		ProjectId:   "test-project-id",
		Name:        "Test Project",
		Description: "Test Description",
		IssueCount:  2,
	}

	testCases := []struct {
		name        string
		req         *projectPbv1.GetProjectRequest
		mockSetup   func(mockRepo *mocks.MockProjectRepository)
		expectedErr codes.Code
		checkResp   func(t *testing.T, resp *projectPbv1.GetProjectResponse)
	}{
		{
			name: "Successful get project",
			req: &projectPbv1.GetProjectRequest{
				ProjectId: "test-project-id",
			},
			mockSetup: func(mockRepo *mocks.MockProjectRepository) {
				mockRepo.EXPECT().ReadProject("test-project-id").Return(testProject, nil)
			},
			expectedErr: codes.OK,
			checkResp: func(t *testing.T, resp *projectPbv1.GetProjectResponse) {
				assert.NotNil(t, resp)
				assert.Equal(t, testProject, resp.Project)
			},
		},
		{
			name: "Project not found",
			req: &projectPbv1.GetProjectRequest{
				ProjectId: "non-existent-id",
			},
			mockSetup: func(mockRepo *mocks.MockProjectRepository) {
				mockRepo.EXPECT().ReadProject("non-existent-id").Return(nil, errors.New("project not found"))
			},
			expectedErr: codes.NotFound,
			checkResp: func(t *testing.T, resp *projectPbv1.GetProjectResponse) {
				assert.Nil(t, resp)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock controller
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Create mock repository
			mockRepo := mocks.NewMockProjectRepository(ctrl)

			// Configure mock behavior
			tc.mockSetup(mockRepo)

			// Create the service with mock repository
			service, _ := projectsvc.NewProjectService(mockRepo)

			// Call the method
			resp, err := service.GetProject(context.Background(), tc.req)

			// Check error if expected
			if tc.expectedErr != codes.OK {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tc.expectedErr, st.Code())
			} else {
				assert.NoError(t, err)
			}

			// Verify response
			tc.checkResp(t, resp)
		})
	}
}

func TestUpdateProject(t *testing.T) {
	testCases := []struct {
		name        string
		req         *projectPbv1.UpdateProjectRequest
		mockSetup   func(mockRepo *mocks.MockProjectRepository)
		expectedErr codes.Code
		checkResp   func(t *testing.T, resp *projectPbv1.UpdateProjectResponse)
	}{
		{
			name: "Successful update",
			req: &projectPbv1.UpdateProjectRequest{
				ProjectId:   "test-project-id",
				Name:        "Updated Project",
				Description: "Updated Description",
			},
			mockSetup: func(mockRepo *mocks.MockProjectRepository) {
				// Return existing project when ReadProject is called
				mockRepo.EXPECT().ReadProject("test-project-id").Return(&projectPbv1.Project{
					ProjectId:   "test-project-id",
					Name:        "Old Project",
					Description: "Old Description",
					IssueCount:  1,
				}, nil)

				// Verify the update has correct fields
				mockRepo.EXPECT().UpdateProject(gomock.Any()).DoAndReturn(
					func(project *projectPbv1.Project) error {
						// Verify project fields were updated correctly
						if project.ProjectId != "test-project-id" ||
							project.Name != "Updated Project" ||
							project.Description != "Updated Description" {
							return errors.New("project fields weren't updated correctly")
						}
						return nil
					})
			},
			expectedErr: codes.OK,
			checkResp: func(t *testing.T, resp *projectPbv1.UpdateProjectResponse) {
				assert.NotNil(t, resp)
				assert.Equal(t, "test-project-id", resp.Project.ProjectId)
				assert.Equal(t, "Updated Project", resp.Project.Name)
				assert.Equal(t, "Updated Description", resp.Project.Description)
			},
		},
		{
			name: "Project not found",
			req: &projectPbv1.UpdateProjectRequest{
				ProjectId:   "non-existent-id",
				Name:        "Updated Project",
				Description: "Updated Description",
			},
			mockSetup: func(mockRepo *mocks.MockProjectRepository) {
				mockRepo.EXPECT().ReadProject("non-existent-id").Return(nil, errors.New("project not found"))
			},
			expectedErr: codes.NotFound,
			checkResp: func(t *testing.T, resp *projectPbv1.UpdateProjectResponse) {
				assert.Nil(t, resp)
			},
		},
		{
			name: "Update fails",
			req: &projectPbv1.UpdateProjectRequest{
				ProjectId:   "test-project-id",
				Name:        "Updated Project",
				Description: "Updated Description",
			},
			mockSetup: func(mockRepo *mocks.MockProjectRepository) {
				// Project exists
				mockRepo.EXPECT().ReadProject("test-project-id").Return(&projectPbv1.Project{
					ProjectId:   "test-project-id",
					Name:        "Old Project",
					Description: "Old Description",
				}, nil)

				// Update fails
				mockRepo.EXPECT().UpdateProject(gomock.Any()).Return(errors.New("update failed"))
			},
			expectedErr: codes.Internal,
			checkResp: func(t *testing.T, resp *projectPbv1.UpdateProjectResponse) {
				assert.Nil(t, resp)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock controller
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Create mock repository
			mockRepo := mocks.NewMockProjectRepository(ctrl)

			// Configure mock behavior
			tc.mockSetup(mockRepo)

			// Create the service with mock repository
			service, _ := projectsvc.NewProjectService(mockRepo)

			// Call the method
			resp, err := service.UpdateProject(context.Background(), tc.req)

			// Check error if expected
			if tc.expectedErr != codes.OK {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tc.expectedErr, st.Code())
			} else {
				assert.NoError(t, err)
			}

			// Verify response
			tc.checkResp(t, resp)
		})
	}
}

func TestDeleteProject(t *testing.T) {
	testCases := []struct {
		name        string
		req         *projectPbv1.DeleteProjectRequest
		mockSetup   func(mockRepo *mocks.MockProjectRepository)
		expectedErr codes.Code
	}{
		{
			name: "Successful delete",
			req: &projectPbv1.DeleteProjectRequest{
				ProjectId: "existing-project-id",
			},
			mockSetup: func(mockRepo *mocks.MockProjectRepository) {
				mockRepo.EXPECT().DeleteProject("existing-project-id").Return(nil)
			},
			expectedErr: codes.OK,
		},
		{
			name: "Project not found",
			req: &projectPbv1.DeleteProjectRequest{
				ProjectId: "non-existent-id",
			},
			mockSetup: func(mockRepo *mocks.MockProjectRepository) {
				mockRepo.EXPECT().DeleteProject("non-existent-id").Return(errors.New("project not found"))
			},
			expectedErr: codes.NotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock controller
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Create mock repository
			mockRepo := mocks.NewMockProjectRepository(ctrl)

			// Configure mock behavior
			tc.mockSetup(mockRepo)

			// Create the service with mock repository
			service, _ := projectsvc.NewProjectService(mockRepo)

			// Call the method
			resp, err := service.DeleteProject(context.Background(), tc.req)

			// Check error if expected
			if tc.expectedErr != codes.OK {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tc.expectedErr, st.Code())
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp) // Empty but not nil
			}
		})
	}
}

func TestListProjects(t *testing.T) {
	// Sample projects list
	sampleProjects := []*projectPbv1.Project{
		{
			ProjectId:   "project-1",
			Name:        "Project One",
			Description: "First test project",
			IssueCount:  2,
		},
		{
			ProjectId:   "project-2",
			Name:        "Project Two",
			Description: "Second test project",
			IssueCount:  0,
		},
	}

	testCases := []struct {
		name        string
		mockSetup   func(mockRepo *mocks.MockProjectRepository)
		expectedErr codes.Code
		checkResp   func(t *testing.T, resp *projectPbv1.ListProjectsResponse)
	}{
		{
			name: "Successful list projects",
			mockSetup: func(mockRepo *mocks.MockProjectRepository) {
				mockRepo.EXPECT().ListProjects().Return(sampleProjects, nil)
			},
			expectedErr: codes.OK,
			checkResp: func(t *testing.T, resp *projectPbv1.ListProjectsResponse) {
				assert.NotNil(t, resp)
				assert.Equal(t, 2, len(resp.Projects))
				assert.Equal(t, "project-1", resp.Projects[0].ProjectId)
				assert.Equal(t, "Project One", resp.Projects[0].Name)
				assert.Equal(t, "project-2", resp.Projects[1].ProjectId)
				assert.Equal(t, "Project Two", resp.Projects[1].Name)
			},
		},
		{
			name: "Empty projects list",
			mockSetup: func(mockRepo *mocks.MockProjectRepository) {
				mockRepo.EXPECT().ListProjects().Return([]*projectPbv1.Project{}, nil)
			},
			expectedErr: codes.OK,
			checkResp: func(t *testing.T, resp *projectPbv1.ListProjectsResponse) {
				assert.NotNil(t, resp)
				assert.Equal(t, 0, len(resp.Projects))
			},
		},
		{
			name: "Repository error",
			mockSetup: func(mockRepo *mocks.MockProjectRepository) {
				mockRepo.EXPECT().ListProjects().Return(nil, errors.New("database error"))
			},
			expectedErr: codes.Internal,
			checkResp: func(t *testing.T, resp *projectPbv1.ListProjectsResponse) {
				assert.Nil(t, resp)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock controller
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Create mock repository
			mockRepo := mocks.NewMockProjectRepository(ctrl)

			// Configure mock behavior
			tc.mockSetup(mockRepo)

			// Create the service with mock repository
			service, _ := projectsvc.NewProjectService(mockRepo)

			// Call the method
			resp, err := service.ListProjects(context.Background(), &emptypb.Empty{})

			// Check error if expected
			if tc.expectedErr != codes.OK {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tc.expectedErr, st.Code())
			} else {
				assert.NoError(t, err)
			}

			// Verify response
			tc.checkResp(t, resp)
		})
	}
}

func TestUpdateProjectWithIssue(t *testing.T) {
	logger.ZapLogger, _ = zap.NewDevelopment()
	testCases := []struct {
		name        string
		req         *projectPbv1.UpdateProjectWithIssueRequest
		mockSetup   func(mockRepo *mocks.MockProjectRepository)
		expectedErr codes.Code
		checkResp   func(t *testing.T, resp *projectPbv1.UpdateProjectWithIssueResponse)
	}{
		{
			name: "Successfully add issue to project",
			req: &projectPbv1.UpdateProjectWithIssueRequest{
				ProjectId: "project-1",
				IssueId:   "issue-1",
			},
			mockSetup: func(mockRepo *mocks.MockProjectRepository) {
				// First check if project exists
				mockRepo.EXPECT().ReadProject("project-1").Return(&projectPbv1.Project{
					ProjectId:   "project-1",
					Name:        "Test Project",
					Description: "Test Description",
				}, nil)
				mockRepo.EXPECT().AddIssueToProject("project-1", "issue-1").Return(nil)
			},
			expectedErr: codes.OK,
			checkResp: func(t *testing.T, resp *projectPbv1.UpdateProjectWithIssueResponse) {
				assert.NotNil(t, resp)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock controller
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Create mock repository
			mockRepo := mocks.NewMockProjectRepository(ctrl)

			// Configure mock behavior
			tc.mockSetup(mockRepo)

			// Create the service with mock repository
			service, _ := projectsvc.NewProjectService(mockRepo)

			// Call the method
			resp, err := service.UpdateProjectWithIssue(context.Background(), tc.req)

			// Check error if expected
			if tc.expectedErr != codes.OK {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tc.expectedErr, st.Code())
			} else {
				assert.NoError(t, err)
			}

			// Verify response
			tc.checkResp(t, resp)
		})
	}
}
