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

// Test the Users function with different environment variable settings
func TestUsers(t *testing.T) {
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
			originalValue := os.Getenv("SEED_USER_COUNT")
			defer func() {
				err := os.Setenv("SEED_USER_COUNT", originalValue)
				if err != nil {
					t.Logf("Failed to restore SEED_USER_COUNT environment variable: %v", err)
				}
			}()

			// Set environment variable for the test
			err := os.Setenv("SEED_USER_COUNT", tc.envValue)
			require.NoError(t, err, "Failed to set SEED_USER_COUNT environment variable")

			// Create mock repository using the generated mock
			mockRepo := mocks.NewMockUserRepository(ctrl)

			// Setup expectations - expect CreateUser to be called exactly expectedCount times
			mockRepo.EXPECT().
				CreateUser(gomock.Any()).
				Return(nil).
				Times(tc.expectedCount)

			// Call the function we're testing with the mock repository
			seed.Users(mockRepo)
		})
	}
}

// Test error handling in the Users function
func TestUsers_Error(t *testing.T) {
	// Initialize logger
	logger.ZapLogger, _ = zap.NewDevelopment()

	// Create controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Save original environment and restore after test
	originalValue := os.Getenv("SEED_USER_COUNT")
	defer func() {
		err := os.Setenv("SEED_USER_COUNT", originalValue)
		if err != nil {
			t.Logf("Failed to restore SEED_USER_COUNT environment variable: %v", err)
		}
	}()

	// Set environment variable for the test
	err := os.Setenv("SEED_USER_COUNT", "3")
	require.NoError(t, err, "Failed to set SEED_USER_COUNT environment variable")

	// Create mock repository using the generated mock
	mockRepo := mocks.NewMockUserRepository(ctrl)

	// Setup expectations
	// First two calls succeed, third one fails
	gomock.InOrder(
		mockRepo.EXPECT().CreateUser(gomock.Any()).Return(nil),
		mockRepo.EXPECT().CreateUser(gomock.Any()).Return(nil),
		mockRepo.EXPECT().CreateUser(gomock.Any()).Return(assert.AnError),
	)

	// Call the function we're testing with the mock repository
	seed.Users(mockRepo)
}

// Test that the function attempts to seed users only up to the specified count
func TestUsers_CountLimit(t *testing.T) {
	// Initialize logger
	logger.ZapLogger, _ = zap.NewDevelopment()

	// Create controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Save original environment and restore after test
	originalValue := os.Getenv("SEED_USER_COUNT")
	defer func() {
		err := os.Setenv("SEED_USER_COUNT", originalValue)
		if err != nil {
			t.Logf("Failed to restore SEED_USER_COUNT environment variable: %v", err)
		}
	}()

	// Set a very specific count
	err := os.Setenv("SEED_USER_COUNT", "7")
	require.NoError(t, err, "Failed to set SEED_USER_COUNT environment variable")

	// Create mock repository using the generated mock
	mockRepo := mocks.NewMockUserRepository(ctrl)

	// Setup expectations - should be called exactly 7 times
	mockRepo.EXPECT().
		CreateUser(gomock.Any()).
		Return(nil).
		Times(7)

	// Call the function we're testing with the mock repository
	seed.Users(mockRepo)
}
