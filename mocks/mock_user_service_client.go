// Code generated by MockGen. DO NOT EDIT.
// Source: c:/Users/mohammed_yasin/Downloads/issue-tracker-mohammed_yasin-in-mem/pkg/pb/user/v1/user_grpc.pb.go
//
// Generated by this command:
//
//	mockgen -destination=c:/Users/mohammed_yasin/Downloads/issue-tracker-mohammed_yasin-in-mem/mocks/mock_user_service_client.go -package=mocks -source=c:/Users/mohammed_yasin/Downloads/issue-tracker-mohammed_yasin-in-mem/pkg/pb/user/v1/user_grpc.pb.go UserServiceClient
//

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	userv1 "github.com/yasindce1998/issue-tracker/pkg/pb/user/v1"
	gomock "go.uber.org/mock/gomock"
	grpc "google.golang.org/grpc"
)

// MockUserServiceClient is a mock of UserServiceClient interface.
type MockUserServiceClient struct {
	ctrl     *gomock.Controller
	recorder *MockUserServiceClientMockRecorder
	isgomock struct{}
}

// MockUserServiceClientMockRecorder is the mock recorder for MockUserServiceClient.
type MockUserServiceClientMockRecorder struct {
	mock *MockUserServiceClient
}

// NewMockUserServiceClient creates a new mock instance.
func NewMockUserServiceClient(ctrl *gomock.Controller) *MockUserServiceClient {
	mock := &MockUserServiceClient{ctrl: ctrl}
	mock.recorder = &MockUserServiceClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUserServiceClient) EXPECT() *MockUserServiceClientMockRecorder {
	return m.recorder
}

// CreateUser mocks base method.
func (m *MockUserServiceClient) CreateUser(ctx context.Context, in *userv1.CreateUserRequest, opts ...grpc.CallOption) (*userv1.CreateUserResponse, error) {
	m.ctrl.T.Helper()
	varargs := []any{ctx, in}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "CreateUser", varargs...)
	ret0, _ := ret[0].(*userv1.CreateUserResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateUser indicates an expected call of CreateUser.
func (mr *MockUserServiceClientMockRecorder) CreateUser(ctx, in any, opts ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{ctx, in}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateUser", reflect.TypeOf((*MockUserServiceClient)(nil).CreateUser), varargs...)
}

// DeleteUser mocks base method.
func (m *MockUserServiceClient) DeleteUser(ctx context.Context, in *userv1.DeleteUserRequest, opts ...grpc.CallOption) (*userv1.DeleteUserResponse, error) {
	m.ctrl.T.Helper()
	varargs := []any{ctx, in}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "DeleteUser", varargs...)
	ret0, _ := ret[0].(*userv1.DeleteUserResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DeleteUser indicates an expected call of DeleteUser.
func (mr *MockUserServiceClientMockRecorder) DeleteUser(ctx, in any, opts ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{ctx, in}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteUser", reflect.TypeOf((*MockUserServiceClient)(nil).DeleteUser), varargs...)
}

// GetUser mocks base method.
func (m *MockUserServiceClient) GetUser(ctx context.Context, in *userv1.GetUserRequest, opts ...grpc.CallOption) (*userv1.GetUserResponse, error) {
	m.ctrl.T.Helper()
	varargs := []any{ctx, in}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetUser", varargs...)
	ret0, _ := ret[0].(*userv1.GetUserResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUser indicates an expected call of GetUser.
func (mr *MockUserServiceClientMockRecorder) GetUser(ctx, in any, opts ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{ctx, in}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUser", reflect.TypeOf((*MockUserServiceClient)(nil).GetUser), varargs...)
}

// ListUsers mocks base method.
func (m *MockUserServiceClient) ListUsers(ctx context.Context, in *userv1.ListUsersRequest, opts ...grpc.CallOption) (*userv1.ListUsersResponse, error) {
	m.ctrl.T.Helper()
	varargs := []any{ctx, in}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ListUsers", varargs...)
	ret0, _ := ret[0].(*userv1.ListUsersResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListUsers indicates an expected call of ListUsers.
func (mr *MockUserServiceClientMockRecorder) ListUsers(ctx, in any, opts ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{ctx, in}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListUsers", reflect.TypeOf((*MockUserServiceClient)(nil).ListUsers), varargs...)
}

// UpdateUser mocks base method.
func (m *MockUserServiceClient) UpdateUser(ctx context.Context, in *userv1.UpdateUserRequest, opts ...grpc.CallOption) (*userv1.UpdateUserResponse, error) {
	m.ctrl.T.Helper()
	varargs := []any{ctx, in}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "UpdateUser", varargs...)
	ret0, _ := ret[0].(*userv1.UpdateUserResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateUser indicates an expected call of UpdateUser.
func (mr *MockUserServiceClientMockRecorder) UpdateUser(ctx, in any, opts ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{ctx, in}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateUser", reflect.TypeOf((*MockUserServiceClient)(nil).UpdateUser), varargs...)
}

// MockUserServiceServer is a mock of UserServiceServer interface.
type MockUserServiceServer struct {
	ctrl     *gomock.Controller
	recorder *MockUserServiceServerMockRecorder
	isgomock struct{}
}

// MockUserServiceServerMockRecorder is the mock recorder for MockUserServiceServer.
type MockUserServiceServerMockRecorder struct {
	mock *MockUserServiceServer
}

// NewMockUserServiceServer creates a new mock instance.
func NewMockUserServiceServer(ctrl *gomock.Controller) *MockUserServiceServer {
	mock := &MockUserServiceServer{ctrl: ctrl}
	mock.recorder = &MockUserServiceServerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUserServiceServer) EXPECT() *MockUserServiceServerMockRecorder {
	return m.recorder
}

// CreateUser mocks base method.
func (m *MockUserServiceServer) CreateUser(arg0 context.Context, arg1 *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateUser", arg0, arg1)
	ret0, _ := ret[0].(*userv1.CreateUserResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateUser indicates an expected call of CreateUser.
func (mr *MockUserServiceServerMockRecorder) CreateUser(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateUser", reflect.TypeOf((*MockUserServiceServer)(nil).CreateUser), arg0, arg1)
}

// DeleteUser mocks base method.
func (m *MockUserServiceServer) DeleteUser(arg0 context.Context, arg1 *userv1.DeleteUserRequest) (*userv1.DeleteUserResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteUser", arg0, arg1)
	ret0, _ := ret[0].(*userv1.DeleteUserResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DeleteUser indicates an expected call of DeleteUser.
func (mr *MockUserServiceServerMockRecorder) DeleteUser(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteUser", reflect.TypeOf((*MockUserServiceServer)(nil).DeleteUser), arg0, arg1)
}

// GetUser mocks base method.
func (m *MockUserServiceServer) GetUser(arg0 context.Context, arg1 *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUser", arg0, arg1)
	ret0, _ := ret[0].(*userv1.GetUserResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUser indicates an expected call of GetUser.
func (mr *MockUserServiceServerMockRecorder) GetUser(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUser", reflect.TypeOf((*MockUserServiceServer)(nil).GetUser), arg0, arg1)
}

// ListUsers mocks base method.
func (m *MockUserServiceServer) ListUsers(arg0 context.Context, arg1 *userv1.ListUsersRequest) (*userv1.ListUsersResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListUsers", arg0, arg1)
	ret0, _ := ret[0].(*userv1.ListUsersResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListUsers indicates an expected call of ListUsers.
func (mr *MockUserServiceServerMockRecorder) ListUsers(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListUsers", reflect.TypeOf((*MockUserServiceServer)(nil).ListUsers), arg0, arg1)
}

// UpdateUser mocks base method.
func (m *MockUserServiceServer) UpdateUser(arg0 context.Context, arg1 *userv1.UpdateUserRequest) (*userv1.UpdateUserResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateUser", arg0, arg1)
	ret0, _ := ret[0].(*userv1.UpdateUserResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateUser indicates an expected call of UpdateUser.
func (mr *MockUserServiceServerMockRecorder) UpdateUser(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateUser", reflect.TypeOf((*MockUserServiceServer)(nil).UpdateUser), arg0, arg1)
}

// mustEmbedUnimplementedUserServiceServer mocks base method.
func (m *MockUserServiceServer) mustEmbedUnimplementedUserServiceServer() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "mustEmbedUnimplementedUserServiceServer")
}

// mustEmbedUnimplementedUserServiceServer indicates an expected call of mustEmbedUnimplementedUserServiceServer.
func (mr *MockUserServiceServerMockRecorder) mustEmbedUnimplementedUserServiceServer() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "mustEmbedUnimplementedUserServiceServer", reflect.TypeOf((*MockUserServiceServer)(nil).mustEmbedUnimplementedUserServiceServer))
}

// MockUnsafeUserServiceServer is a mock of UnsafeUserServiceServer interface.
type MockUnsafeUserServiceServer struct {
	ctrl     *gomock.Controller
	recorder *MockUnsafeUserServiceServerMockRecorder
	isgomock struct{}
}

// MockUnsafeUserServiceServerMockRecorder is the mock recorder for MockUnsafeUserServiceServer.
type MockUnsafeUserServiceServerMockRecorder struct {
	mock *MockUnsafeUserServiceServer
}

// NewMockUnsafeUserServiceServer creates a new mock instance.
func NewMockUnsafeUserServiceServer(ctrl *gomock.Controller) *MockUnsafeUserServiceServer {
	mock := &MockUnsafeUserServiceServer{ctrl: ctrl}
	mock.recorder = &MockUnsafeUserServiceServerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUnsafeUserServiceServer) EXPECT() *MockUnsafeUserServiceServerMockRecorder {
	return m.recorder
}

// mustEmbedUnimplementedUserServiceServer mocks base method.
func (m *MockUnsafeUserServiceServer) mustEmbedUnimplementedUserServiceServer() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "mustEmbedUnimplementedUserServiceServer")
}

// mustEmbedUnimplementedUserServiceServer indicates an expected call of mustEmbedUnimplementedUserServiceServer.
func (mr *MockUnsafeUserServiceServerMockRecorder) mustEmbedUnimplementedUserServiceServer() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "mustEmbedUnimplementedUserServiceServer", reflect.TypeOf((*MockUnsafeUserServiceServer)(nil).mustEmbedUnimplementedUserServiceServer))
}
