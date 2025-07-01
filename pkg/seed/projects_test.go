package seed_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/yasindce1998/issue-tracker/logger"
	"github.com/yasindce1998/issue-tracker/mocks"
	"github.com/yasindce1998/issue-tracker/pkg/seed"
)

// Test the Projects function with different environment variable settings
func TestProjects(t *testing.T) {
	// Initialize logger
	logger.ZapLogger, _ = zap.NewDevelopment()

	testCases := []struct {
		name          string
		envValue      string
		expectedCount int
	}{
		{
			name:          "Default seed count",
			envValue:      "", // No environment variable set
			expectedCount: 5,  // Default should be 5
		},
		{
			name:          "Custom seed count",
			envValue:      "10",
			expectedCount: 10,
		},
		{
			name:          "Invalid seed count",
			envValue:      "not-a-number",
			expectedCount: 5, // Should fall back to default
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a new controller for each test
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Save original environment and restore after test
			originalValue := os.Getenv("SEED_PROJECT_COUNT")
			defer func() {
				err := os.Setenv("SEED_PROJECT_COUNT", originalValue)
				if err != nil {
					t.Logf("Failed to restore SEED_PROJECT_COUNT environment variable: %v", err)
				}
			}()

			// Set environment variable for the test
			err := os.Setenv("SEED_PROJECT_COUNT", tc.envValue)
			require.NoError(t, err, "Failed to set SEED_PROJECT_COUNT environment variable")

			// Create mock repository using the generated mock
			mockRepo := mocks.NewMockProjectRepository(ctrl)

			// Setup expectations - expect CreateProject to be called exactly expectedCount times
			mockRepo.EXPECT().
				CreateProject(gomock.Any()).
				Return(nil).
				Times(tc.expectedCount)

			// The SeedProjects function might call seedProjectIssues, which uses AddIssueToProject
			mockRepo.EXPECT().
				AddIssueToProject(gomock.Any(), gomock.Any()).
				Return(nil).
				AnyTimes()

			// Call the function we're testing with the mock repository
			seed.Projects(mockRepo)
		})
	}
}

// Test error handling in the Projects function
func TestProjects_Error(t *testing.T) {
	// Initialize logger
	logger.ZapLogger, _ = zap.NewDevelopment()

	// Create controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Save original environment and restore after test
	originalValue := os.Getenv("SEED_PROJECT_COUNT")
	defer func() {
		err := os.Setenv("SEED_PROJECT_COUNT", originalValue)
		if err != nil {
			t.Logf("Failed to restore SEED_PROJECT_COUNT environment variable: %v", err)
		}
	}()

	// Set environment variable for the test
	err := os.Setenv("SEED_PROJECT_COUNT", "3")
	require.NoError(t, err, "Failed to set SEED_PROJECT_COUNT environment variable")

	// Create mock repository using the generated mock
	mockRepo := mocks.NewMockProjectRepository(ctrl)

	// Setup expectations
	// First two calls succeed, third one fails
	gomock.InOrder(
		mockRepo.EXPECT().CreateProject(gomock.Any()).Return(nil),
		mockRepo.EXPECT().CreateProject(gomock.Any()).Return(nil),
		mockRepo.EXPECT().CreateProject(gomock.Any()).Return(assert.AnError),
	)

	// The seedProjectIssues function may be called for successful projects
	mockRepo.EXPECT().AddIssueToProject(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	// Call the function we're testing with the mock repository
	seed.Projects(mockRepo)
	// Note: No assertions needed here as we're testing that the function doesn't panic
	// when SeedProjects returns an error
}

// Test that the function attempts to seed projects only up to the specified count
func TestProjects_CountLimit(t *testing.T) {
	// Initialize logger
	logger.ZapLogger, _ = zap.NewDevelopment()

	// Create controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Save original environment and restore after test
	originalValue := os.Getenv("SEED_PROJECT_COUNT")
	defer func() {
		err := os.Setenv("SEED_PROJECT_COUNT", originalValue)
		if err != nil {
			t.Logf("Failed to restore SEED_PROJECT_COUNT environment variable: %v", err)
		}
	}()

	// Set a very specific count
	err := os.Setenv("SEED_PROJECT_COUNT", "7")
	require.NoError(t, err, "Failed to set SEED_PROJECT_COUNT environment variable")

	// Create mock repository using the generated mock
	mockRepo := mocks.NewMockProjectRepository(ctrl)

	// Setup expectations - should be called exactly 7 times
	mockRepo.EXPECT().
		CreateProject(gomock.Any()).
		Return(nil).
		Times(7)

	// Allow seedProjectIssues calls
	mockRepo.EXPECT().AddIssueToProject(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	// Call the function we're testing with the mock repository
	seed.Projects(mockRepo)
}

// Test handling of AddIssueToProject errors
func TestProjects_AddIssueError(t *testing.T) {
	// Initialize logger
	logger.ZapLogger, _ = zap.NewDevelopment()

	// Create controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Save original environment and restore after test
	originalValue := os.Getenv("SEED_PROJECT_COUNT")
	defer func() {
		err := os.Setenv("SEED_PROJECT_COUNT", originalValue)
		if err != nil {
			t.Logf("Failed to restore SEED_PROJECT_COUNT environment variable: %v", err)
		}
	}()

	// Set environment variable for the test
	err := os.Setenv("SEED_PROJECT_COUNT", "1")
	require.NoError(t, err, "Failed to set SEED_PROJECT_COUNT environment variable")

	// Create mock repository using the generated mock
	mockRepo := mocks.NewMockProjectRepository(ctrl)

	// Setup expectations - project creation succeeds
	mockRepo.EXPECT().CreateProject(gomock.Any()).Return(nil)

	// But adding issues fails
	mockRepo.EXPECT().
		AddIssueToProject(gomock.Any(), gomock.Any()).
		Return(assert.AnError).
		AnyTimes()

	// Call the function we're testing with the mock repository
	// The function should handle the AddIssueToProject error and continue
	seed.Projects(mockRepo)
	// We're testing that the function doesn't panic when AddIssueToProject returns an error
}
