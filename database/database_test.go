package database_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yasindce1998/issue-tracker/database"
	"github.com/yasindce1998/issue-tracker/logger"
	"github.com/yasindce1998/issue-tracker/pkg/svc/issuessvc"
	"github.com/yasindce1998/issue-tracker/pkg/svc/projectsvc"
	"github.com/yasindce1998/issue-tracker/pkg/svc/usersvc"
	"go.uber.org/zap"
)

func TestInitializeDatabase_Postgres(t *testing.T) {
	// Setup: Set environment variables for postgres
	originalDBType := os.Getenv("DB_TYPE")
	defer func() {
		err := os.Setenv("DB_TYPE", originalDBType)
		if err != nil {
			t.Logf("Failed to restore DB_TYPE environment variable: %v", err)
		}
	}()

	require.NoError(t, os.Setenv("DB_TYPE", "postgres"))
	require.NoError(t, os.Setenv("DB_HOST", "localhost"))
	require.NoError(t, os.Setenv("DB_PORT", "5432"))
	require.NoError(t, os.Setenv("DB_USER", "test_user"))
	require.NoError(t, os.Setenv("DB_PASSWORD", "test_password"))
	require.NoError(t, os.Setenv("DB_NAME", "test_db"))
	require.NoError(t, os.Setenv("DB_SSL_MODE", "disable"))

	// This test requires an actual postgres instance to connect to
	// For a true unit test, we would need to mock the gorm.DB connection
	// However, for demonstration, we'll skip this test if no postgres is available
	t.Skip("Skipping postgres test as it requires a real database instance")

	// The actual test
	repo, err := database.InitializeDatabase()

	assert.NoError(t, err)
	assert.NotNil(t, repo)
	assert.NotNil(t, repo.UserRepo)
	assert.NotNil(t, repo.IssuesRepo)
	assert.NotNil(t, repo.ProjectRepo)
}

func TestInitializeDatabase_MemDB(t *testing.T) {
	// Setup: Set environment variable for in-memory database
	logger.ZapLogger, _ = zap.NewDevelopment()
	originalDBType := os.Getenv("DB_TYPE")
	defer func() {
		err := os.Setenv("DB_TYPE", originalDBType)
		if err != nil {
			t.Logf("Failed to restore DB_TYPE environment variable: %v", err)
		}
	}()

	require.NoError(t, os.Setenv("DB_TYPE", "memdb"))

	// The actual test
	repo, err := database.InitializeDatabase()

	assert.NoError(t, err)
	assert.NotNil(t, repo)
	assert.NotNil(t, repo.UserRepo)
	assert.NotNil(t, repo.IssuesRepo)
	assert.NotNil(t, repo.ProjectRepo)

	// Verify we got the correct implementation types
	_, ok := repo.UserRepo.(*usersvc.MemDBUserRepository)
	assert.True(t, ok, "Expected UserRepo to be MemDBUserRepository")

	_, ok = repo.IssuesRepo.(*issuessvc.MemDBIssuesRepository)
	assert.True(t, ok, "Expected IssuesRepo to be MemDBIssuesRepository")

	_, ok = repo.ProjectRepo.(*projectsvc.MemDBProjectRepository)
	assert.True(t, ok, "Expected ProjectRepo to be MemDBProjectRepository")
}

func TestInitializeDatabase_UnsupportedType(t *testing.T) {
	// Setup: Set environment variable to an unsupported DB type
	logger.ZapLogger, _ = zap.NewDevelopment()
	originalDBType := os.Getenv("DB_TYPE")
	defer func() {
		err := os.Setenv("DB_TYPE", originalDBType)
		if err != nil {
			t.Logf("Failed to restore DB_TYPE environment variable: %v", err)
		}
	}()

	require.NoError(t, os.Setenv("DB_TYPE", "unsupported"))

	// The actual test
	repo, err := database.InitializeDatabase()

	assert.Error(t, err)
	assert.Nil(t, repo)
	assert.Contains(t, err.Error(), "unsupported DB_TYPE")
}

func TestHealthCheck(t *testing.T) {
	logger.ZapLogger, _ = zap.NewDevelopment()
	// For in-memory DB, HealthCheck should always return nil
	originalDBType := os.Getenv("DB_TYPE")
	defer func() {
		err := os.Setenv("DB_TYPE", originalDBType)
		if err != nil {
			t.Logf("Failed to restore DB_TYPE environment variable: %v", err)
		}
	}()

	require.NoError(t, os.Setenv("DB_TYPE", "memdb"))

	err := database.HealthCheck()
	assert.NoError(t, err)
}
