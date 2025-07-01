package seed

import (
	"os"
	"strconv"

	"github.com/yasindce1998/issue-tracker/logger"
	"github.com/yasindce1998/issue-tracker/pkg/svc/projectsvc"
	"go.uber.org/zap"
)

// Projects seeds the project repository with sample data
func Projects(projectRepository projectsvc.ProjectRepository) {
	projectSeedCount := 5 // Default number of fake projects to create

	// Get custom seed count if specified
	if seedCountEnv := os.Getenv("SEED_PROJECT_COUNT"); seedCountEnv != "" {
		if count, err := strconv.Atoi(seedCountEnv); err == nil {
			projectSeedCount = count
		}
	}

	if err := projectsvc.SeedProjects(projectRepository, projectSeedCount); err != nil {
		logger.ZapLogger.Error("Failed to seed project data", zap.Error(err))
		// Continue anyway - seeding failure shouldn't stop the application
	} else {
		logger.ZapLogger.Info("Successfully seeded project data", zap.Int("count", projectSeedCount))
	}
}
