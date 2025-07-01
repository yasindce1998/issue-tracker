package usersvc

import (
	"context"
	"errors"

	"github.com/yasindce1998/issue-tracker/consts"
	userPbv1 "github.com/yasindce1998/issue-tracker/pkg/pb/user/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UserService serves as the application/gRPC service interface
type UserService struct {
	userPbv1.UnimplementedUserServiceServer
	repository UserRepository
}

// NewUserService initializes the service with a repository
func NewUserService(repository UserRepository) *UserService {
	return &UserService{repository: repository}
}

// CreateUser creates a new user
func (s *UserService) CreateUser(_ context.Context, req *userPbv1.CreateUserRequest) (*userPbv1.CreateUserResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	user := &userPbv1.User{
		UserId:       uuid.NewString(),
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		EmailAddress: req.EmailAddress,
	}

	if err := s.repository.CreateUser(user); err != nil {
		if errors.Is(err, consts.ErrEmailAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, "email already exists")
		}
		return nil, status.Error(codes.Internal, "failed to create user")
	}

	return &userPbv1.CreateUserResponse{User: user}, nil
}

// GetUser retrieves a user by ID
func (s *UserService) GetUser(_ context.Context, req *userPbv1.GetUserRequest) (*userPbv1.GetUserResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	user, err := s.repository.GetUserByID(req.UserId)
	if err != nil {
		if errors.Is(err, consts.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, "failed to retrieve user")
	}

	return &userPbv1.GetUserResponse{User: user}, nil
}

// UpdateUser updates an existing user
func (s *UserService) UpdateUser(_ context.Context, req *userPbv1.UpdateUserRequest) (*userPbv1.UpdateUserResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	user := &userPbv1.User{
		UserId:       req.UserId,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		EmailAddress: req.EmailAddress,
	}

	if err := s.repository.UpdateUser(user); err != nil {
		if errors.Is(err, consts.ErrEmailAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, "email already exists")
		}
		if errors.Is(err, consts.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, "failed to update user")
	}

	return &userPbv1.UpdateUserResponse{User: user}, nil
}

// DeleteUser removes a user
func (s *UserService) DeleteUser(_ context.Context, req *userPbv1.DeleteUserRequest) (*userPbv1.DeleteUserResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	err := s.repository.DeleteUser(req.UserId)
	if err != nil {
		if errors.Is(err, consts.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, "failed to delete user")
	}

	return &userPbv1.DeleteUserResponse{}, nil
}

// ListUsers retrieves a paginated list of users
func (s *UserService) ListUsers(_ context.Context, req *userPbv1.ListUsersRequest) (*userPbv1.ListUsersResponse, error) {
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 10
	}

	users, nextPageToken, err := s.repository.ListUsers(req.PageToken, pageSize)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list users")
	}

	return &userPbv1.ListUsersResponse{
		Users:         users,
		NextPageToken: nextPageToken,
	}, nil
}
