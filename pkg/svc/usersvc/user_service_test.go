package usersvc_test

import (
	"context"
	"testing"

	"github.com/yasindce1998/issue-tracker/consts"
	"github.com/yasindce1998/issue-tracker/mocks"
	userPbv1 "github.com/yasindce1998/issue-tracker/pkg/pb/user/v1"
	"github.com/yasindce1998/issue-tracker/pkg/svc/usersvc"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	validUUID    = "928f705f-0efa-4c96-b2f6-ceb36281e1f1"
	nonExistUUID = "e8289e6f-efc2-4c94-8dcf-0650f7693890"
)

func TestUserServiceServer_CreateUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepository(ctrl)
	userService := usersvc.NewUserService(mockRepo)

	err := gofakeit.Seed(42)
	if err != nil {
		t.Fatalf("Failed to seed random generator: %v", err)
	}

	testCases := []struct {
		name          string
		req           *userPbv1.CreateUserRequest
		setupMock     func()
		expectedResp  *userPbv1.CreateUserResponse
		expectedError error
	}{
		{
			name: "Valid User Creation",
			req: &userPbv1.CreateUserRequest{
				FirstName:    gofakeit.FirstName(),
				LastName:     gofakeit.LastName(),
				EmailAddress: gofakeit.Email(),
			},
			setupMock: func() {
				mockRepo.EXPECT().CreateUser(gomock.Any()).DoAndReturn(func(user *userPbv1.User) error {
					user.UserId = validUUID
					return nil
				})
			},
			expectedResp: &userPbv1.CreateUserResponse{
				User: &userPbv1.User{
					UserId: validUUID,
					// We'll compare these fields directly in the test assertion
				},
			},
			expectedError: nil,
		},
		{
			name: "Email Already Exists",
			req: &userPbv1.CreateUserRequest{
				FirstName:    gofakeit.FirstName(),
				LastName:     gofakeit.LastName(),
				EmailAddress: gofakeit.Email(),
			},
			setupMock: func() {
				mockRepo.EXPECT().CreateUser(gomock.Any()).Return(consts.ErrEmailAlreadyExists)
			},
			expectedResp:  nil,
			expectedError: status.Error(codes.AlreadyExists, "email already exists"),
		},
		{
			name: "Internal Error During User Creation",
			req: &userPbv1.CreateUserRequest{
				FirstName:    gofakeit.FirstName(),
				LastName:     gofakeit.LastName(),
				EmailAddress: gofakeit.Email(),
			},
			setupMock: func() {
				mockRepo.EXPECT().CreateUser(gomock.Any()).Return(consts.ErrDatabaseError)
			},
			expectedResp:  nil,
			expectedError: status.Error(codes.Internal, "failed to create user"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Store the request data to compare with response
			requestFirstName := tc.req.FirstName
			requestLastName := tc.req.LastName
			requestEmail := tc.req.EmailAddress

			tc.setupMock()

			resp, err := userService.CreateUser(context.Background(), tc.req)

			// If expected response exists, validate its fields
			if tc.expectedResp != nil {
				assert.NotNil(t, resp)
				assert.Equal(t, validUUID, resp.User.UserId)
				assert.Equal(t, requestFirstName, resp.User.FirstName)
				assert.Equal(t, requestLastName, resp.User.LastName)
				assert.Equal(t, requestEmail, resp.User.EmailAddress)
			} else {
				assert.Nil(t, resp)
			}

			// Validate the error
			if tc.expectedError != nil {
				assert.EqualError(t, err, tc.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUserServiceServer_GetUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepository(ctrl)
	userService := usersvc.NewUserService(mockRepo)

	testCases := []struct {
		name          string
		req           *userPbv1.GetUserRequest
		setupMock     func()
		expectedResp  *userPbv1.GetUserResponse
		expectedError error
	}{
		{
			name: "Valid Request and Existing User",
			req: &userPbv1.GetUserRequest{
				UserId: validUUID,
			},
			setupMock: func() {
				mockRepo.EXPECT().GetUserByID(validUUID).Return(&userPbv1.User{
					UserId:       validUUID,
					FirstName:    "John",
					LastName:     "Doe",
					EmailAddress: "john.doe@example.com",
				}, nil)
			},
			expectedResp: &userPbv1.GetUserResponse{
				User: &userPbv1.User{
					UserId:       validUUID,
					FirstName:    "John",
					LastName:     "Doe",
					EmailAddress: "john.doe@example.com",
				},
			},
			expectedError: nil,
		},
		{
			name: "User Not Found",
			req: &userPbv1.GetUserRequest{
				UserId: nonExistUUID,
			},
			setupMock: func() {
				mockRepo.EXPECT().GetUserByID(nonExistUUID).Return(nil, consts.ErrUserNotFound)
			},
			expectedResp:  nil,
			expectedError: status.Error(codes.NotFound, "user not found"),
		},
		{
			name: "Internal Error from Repository",
			req: &userPbv1.GetUserRequest{
				UserId: validUUID,
			},
			setupMock: func() {
				mockRepo.EXPECT().GetUserByID(validUUID).Return(nil, consts.ErrDatabaseError)
			},
			expectedResp:  nil,
			expectedError: status.Error(codes.Internal, "failed to retrieve user"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup the mock behavior
			tc.setupMock()

			// Call the method to test
			resp, err := userService.GetUser(context.Background(), tc.req)

			// Assertions
			if tc.expectedResp != nil {
				assert.NotNil(t, resp)
				validateUserResponse(t, tc.expectedResp.User, resp.User)
			} else {
				assert.Nil(t, resp)
			}

			validateError(t, tc.expectedError, err)
		})
	}
}

func TestUserServiceServer_UpdateUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepository(ctrl)
	userService := usersvc.NewUserService(mockRepo)

	testCases := []struct {
		name          string
		req           *userPbv1.UpdateUserRequest
		setupMock     func()
		expectedResp  *userPbv1.UpdateUserResponse
		expectedError error
	}{
		{
			name: "Valid Update",
			req: &userPbv1.UpdateUserRequest{
				UserId:       validUUID,
				FirstName:    "UpdatedFirstName",
				LastName:     "UpdatedLastName",
				EmailAddress: "updated.email@example.com",
			},
			setupMock: func() {
				mockRepo.EXPECT().UpdateUser(&userPbv1.User{
					UserId:       validUUID,
					FirstName:    "UpdatedFirstName",
					LastName:     "UpdatedLastName",
					EmailAddress: "updated.email@example.com",
				}).Return(nil)
			},
			expectedResp: &userPbv1.UpdateUserResponse{
				User: &userPbv1.User{
					UserId:       validUUID,
					FirstName:    "UpdatedFirstName",
					LastName:     "UpdatedLastName",
					EmailAddress: "updated.email@example.com",
				},
			},
			expectedError: nil,
		},
		{
			name: "User Not Found",
			req: &userPbv1.UpdateUserRequest{
				UserId:       nonExistUUID,
				FirstName:    "UpdatedFirstName",
				LastName:     "UpdatedLastName",
				EmailAddress: "updated.email@example.com",
			},
			setupMock: func() {
				mockRepo.EXPECT().UpdateUser(&userPbv1.User{
					UserId:       nonExistUUID,
					FirstName:    "UpdatedFirstName",
					LastName:     "UpdatedLastName",
					EmailAddress: "updated.email@example.com",
				}).Return(consts.ErrUserNotFound)
			},
			expectedResp:  nil,
			expectedError: status.Error(codes.NotFound, "user not found"),
		},
		{
			name: "Email Already Exists",
			req: &userPbv1.UpdateUserRequest{
				UserId:       validUUID,
				FirstName:    "UpdatedFirstName",
				LastName:     "UpdatedLastName",
				EmailAddress: "existing.email@example.com",
			},
			setupMock: func() {
				mockRepo.EXPECT().UpdateUser(&userPbv1.User{
					UserId:       validUUID,
					FirstName:    "UpdatedFirstName",
					LastName:     "UpdatedLastName",
					EmailAddress: "existing.email@example.com",
				}).Return(consts.ErrEmailAlreadyExists)
			},
			expectedResp:  nil,
			expectedError: status.Error(codes.AlreadyExists, "email already exists"),
		},
		{
			name: "Internal Error",
			req: &userPbv1.UpdateUserRequest{
				UserId:       validUUID,
				FirstName:    "UpdatedFirstName",
				LastName:     "UpdatedLastName",
				EmailAddress: "updated.email@example.com",
			},
			setupMock: func() {
				mockRepo.EXPECT().UpdateUser(&userPbv1.User{
					UserId:       validUUID,
					FirstName:    "UpdatedFirstName",
					LastName:     "UpdatedLastName",
					EmailAddress: "updated.email@example.com",
				}).Return(consts.ErrDatabaseError)
			},
			expectedResp:  nil,
			expectedError: status.Error(codes.Internal, "failed to update user"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock expectations
			tc.setupMock()

			// Call the method under test
			resp, err := userService.UpdateUser(context.Background(), tc.req)

			// Validate the response
			if tc.expectedResp != nil {
				assert.NotNil(t, resp)
				validateUserResponse(t, tc.expectedResp.User, resp.User)
			} else {
				assert.Nil(t, resp)
			}

			validateError(t, tc.expectedError, err)
		})
	}
}

func TestUserServiceServer_DeleteUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepository(ctrl)
	userService := usersvc.NewUserService(mockRepo)

	testCases := []struct {
		name          string
		req           *userPbv1.DeleteUserRequest
		setupMock     func()
		expectedResp  *userPbv1.DeleteUserResponse
		expectedError error
	}{
		{
			name: "Valid Deletion",
			req: &userPbv1.DeleteUserRequest{
				UserId: validUUID,
			},
			setupMock: func() {
				// Mock the repository to successfully delete the user
				mockRepo.EXPECT().DeleteUser(validUUID).Return(nil)
			},
			expectedResp:  &userPbv1.DeleteUserResponse{},
			expectedError: nil,
		},
		{
			name: "User Not Found",
			req: &userPbv1.DeleteUserRequest{
				UserId: nonExistUUID,
			},
			setupMock: func() {
				// Mock the repository to return ErrUserNotFound
				mockRepo.EXPECT().DeleteUser(nonExistUUID).Return(consts.ErrUserNotFound)
			},
			expectedResp:  nil,
			expectedError: status.Error(codes.NotFound, "user not found"),
		},
		{
			name: "Internal Error During Deletion",
			req: &userPbv1.DeleteUserRequest{
				UserId: validUUID,
			},
			setupMock: func() {
				// Mock the repository to return a generic internal error
				mockRepo.EXPECT().DeleteUser(validUUID).Return(consts.ErrDatabaseError)
			},
			expectedResp:  nil,
			expectedError: status.Error(codes.Internal, "failed to delete user"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up the mock behavior
			tc.setupMock()

			// Call the method being tested
			resp, err := userService.DeleteUser(context.Background(), tc.req)

			// Validate the response
			if tc.expectedResp != nil {
				assert.NotNil(t, resp)
				assert.Equal(t, tc.expectedResp, resp)
			} else {
				assert.Nil(t, resp)
			}

			// Validate the error
			if tc.expectedError != nil {
				assert.EqualError(t, err, tc.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUserServiceServer_ListUsers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepository(ctrl)
	userService := usersvc.NewUserService(mockRepo)

	validUsers := []*userPbv1.User{
		{
			UserId:       "user-1",
			FirstName:    "John",
			LastName:     "Doe",
			EmailAddress: "john.doe@example.com",
		},
		{
			UserId:       "user-2",
			FirstName:    "Jane",
			LastName:     "Smith",
			EmailAddress: "jane.smith@example.com",
		},
	}

	testCases := []struct {
		name          string
		req           *userPbv1.ListUsersRequest
		setupMock     func()
		expectedResp  *userPbv1.ListUsersResponse
		expectedError error
	}{
		{
			name: "Valid Request - List Users",
			req: &userPbv1.ListUsersRequest{
				PageSize:  2,
				PageToken: "",
			},
			setupMock: func() {
				mockRepo.EXPECT().ListUsers("", 2).Return(validUsers, "next-token", nil)
			},
			expectedResp: &userPbv1.ListUsersResponse{
				Users:         validUsers,
				NextPageToken: "next-token",
			},
			expectedError: nil,
		},
		{
			name: "Default Page Size",
			req: &userPbv1.ListUsersRequest{
				PageSize:  0, // The method should default this to 10
				PageToken: "",
			},
			setupMock: func() {
				mockRepo.EXPECT().ListUsers("", 10).Return(validUsers, "next-token", nil)
			},
			expectedResp: &userPbv1.ListUsersResponse{
				Users:         validUsers,
				NextPageToken: "next-token",
			},
			expectedError: nil,
		},
		{
			name: "Empty Response - No Users",
			req: &userPbv1.ListUsersRequest{
				PageSize:  10,
				PageToken: "",
			},
			setupMock: func() {
				mockRepo.EXPECT().ListUsers("", 10).Return([]*userPbv1.User{}, "", nil)
			},
			expectedResp: &userPbv1.ListUsersResponse{
				Users:         []*userPbv1.User{}, // Empty list
				NextPageToken: "",
			},
			expectedError: nil,
		},
		{
			name: "Pagination Handling",
			req: &userPbv1.ListUsersRequest{
				PageSize:  2,
				PageToken: "user-2",
			},
			setupMock: func() {
				mockRepo.EXPECT().ListUsers("user-2", 2).Return(validUsers, "next-token-2", nil)
			},
			expectedResp: &userPbv1.ListUsersResponse{
				Users:         validUsers,
				NextPageToken: "next-token-2",
			},
			expectedError: nil,
		},
		{
			name: "Internal Error",
			req: &userPbv1.ListUsersRequest{
				PageSize:  10,
				PageToken: "",
			},
			setupMock: func() {
				mockRepo.EXPECT().ListUsers("", 10).Return(nil, "", consts.ErrDatabaseError)
			},
			expectedResp:  nil,
			expectedError: status.Error(codes.Internal, "failed to list users"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup the mock behavior
			tc.setupMock()

			// Call the ListUsers method
			resp, err := userService.ListUsers(context.Background(), tc.req)

			// Validate the response
			if tc.expectedResp != nil {
				assert.NotNil(t, resp)
				assert.Equal(t, tc.expectedResp.Users, resp.Users)
				assert.Equal(t, tc.expectedResp.NextPageToken, resp.NextPageToken)
			} else {
				assert.Nil(t, resp)
			}

			// Validate the error
			if tc.expectedError != nil {
				assert.EqualError(t, err, tc.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Helper function to validate a user response
func validateUserResponse(t *testing.T, expected *userPbv1.User, actual *userPbv1.User) {
	if expected != nil {
		assert.NotNil(t, actual)
		assert.Equal(t, expected.UserId, actual.UserId)
		assert.Equal(t, expected.FirstName, actual.FirstName)
		assert.Equal(t, expected.LastName, actual.LastName)
		assert.Equal(t, expected.EmailAddress, actual.EmailAddress)
	} else {
		assert.Nil(t, actual)
	}
}

func validateError(t *testing.T, expectedErr, actualErr error) {
	if expectedErr != nil {
		assert.EqualError(t, actualErr, expectedErr.Error())
	} else {
		assert.NoError(t, actualErr)
	}
}
