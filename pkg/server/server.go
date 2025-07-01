// Package server provides http and gRPC related operations
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/yasindce1998/issue-tracker/cache"
	"github.com/yasindce1998/issue-tracker/database"
	"github.com/yasindce1998/issue-tracker/logger"
	"github.com/yasindce1998/issue-tracker/pkg/config"
	issuesPbv1 "github.com/yasindce1998/issue-tracker/pkg/pb/issues/v1"
	projectPbv1 "github.com/yasindce1998/issue-tracker/pkg/pb/project/v1"
	userPbv1 "github.com/yasindce1998/issue-tracker/pkg/pb/user/v1"
	"github.com/yasindce1998/issue-tracker/pkg/seed"
	"github.com/yasindce1998/issue-tracker/pkg/svc/issuessvc"
	"github.com/yasindce1998/issue-tracker/pkg/svc/projectsvc"
	"github.com/yasindce1998/issue-tracker/pkg/svc/usersvc"
	"github.com/google/uuid"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

// GRPCServer encapsulates the gRPC server and its services
type GRPCServer struct {
	server         *grpc.Server
	userService    userPbv1.UserServiceServer
	issuesService  issuesPbv1.IssuesServiceServer
	projectService projectPbv1.ProjectServiceServer
	httpPort       string
}

// Application represents the main application structure
type Application struct {
	GRPCServer *GRPCServer
	GRPCPort   string
	HTTPPort   string
}

// HealthResponse is the response structure for health checks
type HealthResponse struct {
	Status              string `json:"status"`
	DbStatus            string `json:"db_status"`
	DbType              string `json:"db_type"`
	CacheStatus         string `json:"cache_status"`
	CacheType           string `json:"cache_type"`
	AppName             string `json:"app_name"`
	CommunicationMethod string `json:"communication_method"`
}

// NewApplication creates and initializes a new application instance
func NewApplication() (*Application, error) {
	app := &Application{}

	// Load environment variables
	if err := config.LoadEnv(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Initialize logger
	if err := logger.InitializeLogger("debug"); err != nil {
		return nil, err
	}

	logger.ZapLogger.Info("Starting Issue Tracker Service")

	// Get required environment variables
	app.GRPCPort = os.Getenv("GRPC_PORT")
	app.HTTPPort = os.Getenv("HTTP_PORT")
	if app.GRPCPort == "" || app.HTTPPort == "" {
		log.Fatal("GRPC_PORT or HTTP_PORT is not set in environment variables")
	}

	// Create gRPC clients
	projectClient, userClient, err := createClients()
	if err != nil {
		logger.ZapLogger.Fatal("Failed to create gRPC clients", zap.Error(err))
	}

	// Initialize repositories using the database package
	repos, err := database.InitializeDatabase()
	if err != nil {
		logger.ZapLogger.Fatal("Failed to initialize database", zap.Error(err))
	}

	// Initialize cache
	cacheInstance := cache.NewCache()
	logger.ZapLogger.Info("Cache initialized",
		zap.String("type", os.Getenv("CACHE_TYPE")))

	// Wrap repositories with cache
	cachedUserRepo := usersvc.NewCachedUserRepository(repos.UserRepo, cacheInstance)
	cachedIssuesRepo := issuessvc.NewCachedIssuesRepository(repos.IssuesRepo, cacheInstance)
	cachedProjectRepo := projectsvc.NewCachedProjectRepository(repos.ProjectRepo, cacheInstance)

	// Initialize services first - they need to exist before seeding relationships
	userService := usersvc.NewUserService(cachedUserRepo)
	issuesService := issuessvc.NewIssuesService(cachedIssuesRepo, projectClient, userClient)
	projectService, err := projectsvc.NewProjectService(cachedProjectRepo)
	if err != nil {
		logger.ZapLogger.Fatal("Failed to initialize project service", zap.Error(err))
	}

	// Handle data seeding
	// Note: We only seed data if using memDB, skip for postgres
	seed.Data(
		repos.UserRepo,
		repos.ProjectRepo,
		repos.IssuesRepo,
		userService,
		projectService,
		projectClient,
		userClient,
	)

	// Configure gRPC Server
	app.GRPCServer = NewGRPCServer(userService, issuesService, projectService)

	return app, nil
}

// NewGRPCServer creates a new GRPCServer with the provided services
func NewGRPCServer(
	userService userPbv1.UserServiceServer,
	issuesService issuesPbv1.IssuesServiceServer,
	projectService projectPbv1.ProjectServiceServer,
) *GRPCServer {
	// Add server interceptors for logging
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(LoggingInterceptor),
	}
	server := grpc.NewServer(opts...)

	// Register services
	userPbv1.RegisterUserServiceServer(server, userService)
	issuesPbv1.RegisterIssuesServiceServer(server, issuesService)
	projectPbv1.RegisterProjectServiceServer(server, projectService)

	// Enable reflection for tools like grpcurl
	reflection.Register(server)

	return &GRPCServer{
		server:         server,
		userService:    userService,
		issuesService:  issuesService,
		projectService: projectService,
	}
}

// loggingInterceptor logs information about gRPC method calls
// Define a custom type for context keys
type contextKey string

// LoggingInterceptor is a gRPC interceptor that logs method calls with trace IDs and timing information.
// It adds a trace ID to the context and tracks cache statistics for each request.
func LoggingInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()

	// Generate or extract trace ID
	traceID := uuid.New().String()
	ctx = context.WithValue(ctx, contextKey("trace_id"), traceID)

	// Add cache stats tracking
	ctx = logger.WithCacheStats(ctx)

	// Log method entry
	logger.ZapLogger.Info("gRPC method called",
		zap.String("trace_id", traceID),
		zap.String("method", info.FullMethod),
		zap.Any("request", req),
	)

	// Call the handler
	resp, err := handler(ctx, req)

	// Log method exit
	duration := time.Since(start)
	if err != nil {
		logger.ZapLogger.Error("gRPC method failed",
			zap.String("trace_id", traceID),
			zap.String("method", info.FullMethod),
			zap.Duration("duration", duration),
			zap.Error(err),
		)
	} else {
		logger.ZapLogger.Info("gRPC method completed",
			zap.String("trace_id", traceID),
			zap.String("method", info.FullMethod),
			zap.Duration("duration", duration),
		)
	}

	return resp, err
}

func (s *GRPCServer) startHTTPGateway(grpcPort string, httpPort string) error {
	ctx := context.Background()
	// Use a WithLogEntry wrapper for the mux
	mux := runtime.NewServeMux()

	// Register health check endpoint
	healthHandler := http.HandlerFunc(HealthHandler)

	// Wrap the mux with logging middleware
	wrappedHandler := LoggingMiddleware(mux)

	// Create a handler that routes to health check or gRPC-gateway
	combinedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			healthHandler.ServeHTTP(w, r)
			return
		}
		wrappedHandler.ServeHTTP(w, r)
	})

	// Configure gRPC dial options
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	// Register UserService HTTP gateway
	if err := userPbv1.RegisterUserServiceHandlerFromEndpoint(ctx, mux, grpcPort, opts); err != nil {
		return fmt.Errorf("failed to register UserService handler: %w", err)
	}

	// Register IssuesService HTTP gateway
	if err := issuesPbv1.RegisterIssuesServiceHandlerFromEndpoint(ctx, mux, grpcPort, opts); err != nil {
		return fmt.Errorf("failed to register IssuesService handler: %w", err)
	}

	// Register ProjectService HTTP gateway
	if err := projectPbv1.RegisterProjectServiceHandlerFromEndpoint(ctx, mux, grpcPort, opts); err != nil {
		return fmt.Errorf("failed to register ProjectService handler: %w", err)
	}

	// Create a server with proper timeouts
	httpAddr := httpPort
	server := &http.Server{
		Addr:         httpAddr,
		Handler:      combinedHandler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Println("HTTP server started on " + httpAddr)
	return server.ListenAndServe()
}

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response recorder to capture the status code
		recorder := &statusRecorder{
			ResponseWriter: w,
			Status:         http.StatusOK,
		}

		// Generate trace ID
		traceID := uuid.New().String()
		ctx := context.WithValue(r.Context(), contextKey("trace_id"), traceID)

		// Add cache stats tracking
		ctx = logger.WithCacheStats(ctx)

		// Log request
		logger.ZapLogger.Info("HTTP request received",
			zap.String("trace_id", traceID),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("remote_addr", r.RemoteAddr),
		)

		// Call the handler
		next.ServeHTTP(recorder, r.WithContext(ctx))

		// Log response
		duration := time.Since(start)
		logger.ZapLogger.Info("HTTP request completed",
			zap.String("trace_id", traceID),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", recorder.Status),
			zap.Duration("duration", duration),
		)
	})
}

// statusRecorder wraps http.ResponseWriter to capture the status code
type statusRecorder struct {
	http.ResponseWriter
	Status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.Status = status
	r.ResponseWriter.WriteHeader(status)
}

// Start runs the application with graceful shutdown handling
func (app *Application) Start() error {
	// Ensure ports have colons for proper listening format
	grpcPort := app.GRPCPort
	httpPort := app.HTTPPort

	if !strings.HasPrefix(grpcPort, ":") && !strings.Contains(grpcPort, ":") {
		grpcPort = ":" + grpcPort
	}

	if !strings.HasPrefix(httpPort, ":") && !strings.Contains(httpPort, ":") {
		httpPort = ":" + httpPort
	}
	logger.ZapLogger.Info("Starting application",
		zap.String("grpc_port", grpcPort),
		zap.String("http_port", httpPort))

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create error channel for server errors
	errChan := make(chan error, 2)

	// Start grpc server in goroutine
	go func() {
		logger.ZapLogger.Info("Starting gRPC and HTTP servers")
		if err := app.GRPCServer.Start(grpcPort, httpPort); err != nil {
			logger.ZapLogger.Error("Server error", zap.Error(err))
			errChan <- err
		}
	}()

	// Wait for termination signal or error
	select {
	case sig := <-sigChan:
		logger.ZapLogger.Info("Received shutdown signal", zap.String("signal", sig.String()))

		// Create context with timeout for graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Call shutdown
		return app.Shutdown(ctx)

	case err := <-errChan:
		logger.ZapLogger.Error("Server error, initiating shutdown", zap.Error(err))

		// Create context with timeout for graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Call shutdown but return the original server error
		shutdownErr := app.Shutdown(ctx)
		if shutdownErr != nil {
			logger.ZapLogger.Error("Error during shutdown", zap.Error(shutdownErr))
			// We still want to return the original error that caused the shutdown
		}

		return fmt.Errorf("server error: %w", err)
	}
}

// Shutdown performs cleanup before application exit
func (app *Application) Shutdown(ctx context.Context) error {
	logger.ZapLogger.Info("Shutting down application...")

	// Use the context for timeout operations
	var shutdownErr error

	// Create a channel to signal completion
	done := make(chan struct{})

	go func() {
		// Close project service messaging resources
		if projectService, ok := app.GRPCServer.projectService.(*projectsvc.ProjectService); ok {
			if err := projectService.Close(); err != nil {
				logger.ZapLogger.Error("Error closing project service", zap.Error(err))
				shutdownErr = err
			}
		}
		// Close gRPC server
		if err := app.GRPCServer.Stop(); err != nil {
			logger.ZapLogger.Error("Error shutting down gRPC server", zap.Error(err))
			shutdownErr = err
		}

		// Close cache connections
		if os.Getenv("CACHE_TYPE") == "redis" {
			// Since we don't have direct access to the cache instance here,
			// we'll close it indirectly through a global function
			if err := cache.CloseConnections(); err != nil {
				logger.ZapLogger.Error("Error closing cache connections", zap.Error(err))
				shutdownErr = err
			}
		}

		// Close database connections if PostgreSQL
		if os.Getenv("DB_TYPE") == "postgres" {
			if err := database.CloseConnections(); err != nil {
				logger.ZapLogger.Error("Error closing database connections", zap.Error(err))
				shutdownErr = err
			}
		}

		close(done)
	}()

	// Wait for shutdown to complete or context to be done
	select {
	case <-done:
		return shutdownErr
	case <-ctx.Done():
		return fmt.Errorf("shutdown timed out: %w", ctx.Err())
	}
}

// Start function starts the grpc and http server with given port
func (s *GRPCServer) Start(grpcPort string, httpPort string) error {
	s.httpPort = httpPort
	address := fmt.Sprintf("0.0.0.0%s", grpcPort)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen on gRPC port %s: %w", grpcPort, err)
	}

	// Start HTTP Gateway in a goroutine
	go func() {
		if err := s.startHTTPGateway(grpcPort, httpPort); err != nil {
			log.Fatalf("Failed to start HTTP Gateway: %v", err)
		}
	}()

	log.Println("gRPC server started on " + grpcPort)
	return s.server.Serve(listener)
}

// Stop gracefully stops the gRPC server
func (s *GRPCServer) Stop() error {
	s.server.GracefulStop()
	return nil
}

// createClients sets up the gRPC clients for Project and User services.
func createClients() (projectPbv1.ProjectServiceClient, userPbv1.UserServiceClient, error) {
	// For in-memory mode, we might not need actual clients initially
	if os.Getenv("DB_TYPE") == "memdb" && os.Getenv("USE_LOCAL_CLIENTS") == "true" {
		return nil, nil, nil
	}
	grpcHost := os.Getenv("GRPC_HOST")
	grpcPort := os.Getenv("GRPC_PORT")

	addr := fmt.Sprintf("%s:%s", grpcHost, grpcPort)
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	projectClient := projectPbv1.NewProjectServiceClient(conn)
	userClient := userPbv1.NewUserServiceClient(conn)

	return projectClient, userClient, nil
}

// HealthHandler handles health check requests
func HealthHandler(w http.ResponseWriter, _ *http.Request) {
	status := "ok"
	dbStatus := "ok"
	cacheStatus := "ok"
	httpStatus := http.StatusOK

	// Check database health
	if err := database.HealthCheck(); err != nil {
		dbStatus = "error: " + err.Error()
		status = "error"
		httpStatus = http.StatusServiceUnavailable
	}

	// Check cache health
	if err := cache.HealthCheck(); err != nil {
		cacheStatus = "error: " + err.Error()
		status = "error" // Update overall status
		httpStatus = http.StatusServiceUnavailable
	}

	response := HealthResponse{
		Status:              status,
		DbStatus:            dbStatus,
		DbType:              os.Getenv("DB_TYPE"),
		CacheStatus:         cacheStatus,
		CacheType:           os.Getenv("CACHE_TYPE"),
		AppName:             "Issue Tracker",
		CommunicationMethod: getCommMethod(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus) // Set appropriate HTTP status code

	// Check for encoding errors
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.ZapLogger.Error("Failed to encode health check response", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Log health check results
	logger.ZapLogger.Debug("Health check performed",
		zap.String("status", status),
		zap.String("db_status", dbStatus),
		zap.String("cache_status", cacheStatus))
}

func getCommMethod() string {
	method := os.Getenv("COMMUNICATION_METHOD")
	if method == "" {
		return "stream"
	}
	return method
}
