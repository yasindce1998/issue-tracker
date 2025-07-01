package server_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/yasindce1998/issue-tracker/logger"
	"github.com/yasindce1998/issue-tracker/pkg/server"
)

func TestLoggingMiddleware(t *testing.T) {
	// Setup
	logger.ZapLogger, _ = zap.NewDevelopment()

	// Create a test handler that we'll wrap with the middleware
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("test response"))
		require.NoError(t, err, "Failed to write response")
	})

	// Create a test handler that returns an error status
	errorHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, err := w.Write([]byte("not found"))
		require.NoError(t, err, "Failed to write response")
	})

	// Create test cases
	testCases := []struct {
		name           string
		handler        http.Handler
		method         string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Successful request",
			handler:        testHandler,
			method:         "GET",
			path:           "/test",
			expectedStatus: http.StatusOK,
			expectedBody:   "test response",
		},
		{
			name:           "Error request",
			handler:        errorHandler,
			method:         "POST",
			path:           "/notfound",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Wrap the handler with our middleware - fixed to use server package
			wrappedHandler := server.LoggingMiddleware(tc.handler)

			// Create request and response recorder
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()

			// Call the handler
			wrappedHandler.ServeHTTP(rr, req)

			// Assert the status code
			assert.Equal(t, tc.expectedStatus, rr.Code)

			// Assert the response body
			body, err := io.ReadAll(rr.Body)
			require.NoError(t, err, "Failed to read response body")
			assert.Equal(t, tc.expectedBody, string(body))
		})
	}
}

func TestLoggingInterceptor(t *testing.T) {
	// Setup
	logger.ZapLogger, _ = zap.NewDevelopment()

	// Create test request
	req := "test request"
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	// Mock handler
	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return "response", nil
	}

	// Call the interceptor - fixed to use server package
	resp, err := server.LoggingInterceptor(context.Background(), req, info, handler)

	// Assert results
	assert.NoError(t, err)
	assert.Equal(t, "response", resp)

	// Create an error handler
	errorHandler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return nil, assert.AnError
	}

	// Call with error handler - fixed to use server package
	resp, err = server.LoggingInterceptor(context.Background(), req, info, errorHandler)

	// Assert results
	assert.Error(t, err)
	assert.Nil(t, resp)
}
