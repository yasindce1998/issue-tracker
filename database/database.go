// Package database provides functionality for database operations and repository management.
// It supports both PostgreSQL and in-memory database implementations, handles connections,
// migrations, and exposes repositories for the application's domain entities.
package database

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/yasindce1998/issue-tracker/logger"
	"github.com/yasindce1998/issue-tracker/models"
	"github.com/yasindce1998/issue-tracker/pkg/svc/issuessvc"
	"github.com/yasindce1998/issue-tracker/pkg/svc/projectsvc"
	"github.com/yasindce1998/issue-tracker/pkg/svc/usersvc"
)

// Database type constants
const (
	PostgresDB = "postgres"
	MemDB      = "memdb"
)

var dbInstance *gorm.DB

// Repository encapsulates all data access repositories for the application.
// It provides access to users, issues, and projects repositories.
type Repository struct {
	UserRepo    usersvc.UserRepository
	IssuesRepo  issuessvc.IssuesRepository
	ProjectRepo projectsvc.ProjectRepository
}

// InitializeDatabase initializes the database connections and repositories.
func InitializeDatabase() (*Repository, error) {
	dbType, err := getEnv("DB_TYPE")
	if err != nil {
		logger.ZapLogger.Error("Failed to get DB_TYPE environment variable", zap.Error(err))
		return nil, fmt.Errorf("environment variable error: %w", err)
	}

	logger.ZapLogger.Info("Initializing database", zap.String("type", dbType))

	switch dbType {
	case PostgresDB:
		repos, err := initializePostgres()
		if err != nil {
			logger.ZapLogger.Error("Failed to initialize Postgres", zap.Error(err))
			return nil, err
		}
		logger.ZapLogger.Info("PostgreSQL database initialized successfully")
		return repos, nil
	case MemDB:
		repos, err := initializeMemDB()
		if err != nil {
			logger.ZapLogger.Error("Failed to initialize MemDB", zap.Error(err))
			return nil, err
		}
		logger.ZapLogger.Info("In-memory database initialized successfully")
		return repos, nil
	default:
		logger.ZapLogger.Error("Unsupported database type", zap.String("dbType", dbType))
		return nil, fmt.Errorf("unsupported DB_TYPE: %s", dbType)
	}
}

// initializePostgres sets up the PostgreSQL connection and repositories.
func initializePostgres() (*Repository, error) {
	dsn, err := buildPostgresDSN()
	if err != nil {
		return nil, err
	}

	// Configure connection pooling
	pgConfig := postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true,
	}

	gormConfig := &gorm.Config{
		PrepareStmt: true, // Cache prepared statements for better performance
		Logger:      gormlogger.Default.LogMode(gormlogger.Error),
	}

	db, err := gorm.Open(postgres.New(pgConfig), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Postgres: %w", err)
	}

	dbInstance = db

	// Set connection pool parameters
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get DB instance: %w", err)
	}

	// Set max open connections
	maxConn, err := strconv.Atoi(os.Getenv("DB_MAX_CONNECTIONS"))
	if err == nil && maxConn > 0 {
		sqlDB.SetMaxOpenConns(maxConn)
	} else {
		sqlDB.SetMaxOpenConns(25) // Default value
	}

	// Set max idle connections
	maxIdleConn, err := strconv.Atoi(os.Getenv("DB_MAX_IDLE_CONNECTIONS"))
	if err == nil && maxIdleConn > 0 {
		sqlDB.SetMaxIdleConns(maxIdleConn)
	} else {
		sqlDB.SetMaxIdleConns(10) // Default value
	}

	// Set connection max lifetime
	connMaxLifetime, err := strconv.Atoi(os.Getenv("DB_CONN_MAX_LIFETIME"))
	if err == nil && connMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(time.Duration(connMaxLifetime) * time.Minute)
	} else {
		sqlDB.SetConnMaxLifetime(30 * time.Minute) // Default value
	}

	// Run database migrations
	if err := migrateDatabase(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Initialize repositories
	repositories := &Repository{
		UserRepo:    usersvc.NewPostgresUserRepository(db),
		IssuesRepo:  issuessvc.NewPostgresIssuesRepository(db),
		ProjectRepo: projectsvc.NewPostgresProjectRepository(db),
	}

	return repositories, nil
}

// initializeMemDB sets up in-memory repositories.
func initializeMemDB() (*Repository, error) {
	userRepo, err := usersvc.NewMemDBUserRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MemDB UserRepository: %w", err)
	}

	issuesRepo, err := issuessvc.NewMemDBIssuesRepositoryWithoutClients()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MemDB IssuesRepository: %w", err)
	}

	projectRepo, err := projectsvc.NewMemDBProjectRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MemDB ProjectRepository: %w", err)
	}

	// Return a single struct encapsulating all repositories
	return &Repository{
		UserRepo:    userRepo,
		IssuesRepo:  issuesRepo,
		ProjectRepo: projectRepo,
	}, nil
}

// buildPostgresDSN builds the PostgreSQL Data Source Name (DSN) from environment variables.
func buildPostgresDSN() (string, error) {
	host, err := getEnv("POSTGRES_HOST")
	if err != nil {
		return "", err
	}

	port, err := getEnv("POSTGRES_PORT")
	if err != nil {
		return "", err
	}

	user, err := getEnv("POSTGRES_USER")
	if err != nil {
		return "", err
	}

	password, err := getEnv("POSTGRES_PASSWORD")
	if err != nil {
		return "", err
	}

	dbName, err := getEnv("POSTGRES_DB")
	if err != nil {
		return "", err
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbName)

	return dsn, nil
}

// migrateDatabase performs automatic migrations for the database schema.
func migrateDatabase(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.Issues{},
		&models.Project{},
	)
}

// CloseConnections closes any open database connections
func CloseConnections() error {
	if dbInstance == nil {
		return nil
	}

	sqlDB, err := dbInstance.DB()
	if err != nil {
		return fmt.Errorf("failed to get DB instance for closing: %w", err)
	}

	return sqlDB.Close()
}

// HealthCheck performs a health check on the database
func HealthCheck() error {
	if os.Getenv("DB_TYPE") != PostgresDB {
		return nil // In-memory DB is always healthy
	}

	if dbInstance == nil {
		return fmt.Errorf("database connection not initialized")
	}

	sqlDB, err := dbInstance.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

// getEnv retrieves an environment variable with optional enforcement.
func getEnv(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("missing required environment variable: %s", key)
	}
	return value, nil
}
