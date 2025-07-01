package seed

import (
	"os"
	"strconv"

	"github.com/yasindce1998/issue-tracker/logger"
	"github.com/yasindce1998/issue-tracker/pkg/svc/usersvc"
	"go.uber.org/zap"
)

// Users seeds the user repository with sample data
func Users(userRepository usersvc.UserRepository) {
	userSeedCount := 5 // Default number of fake users to create

	// Get custom seed count if specified
	if seedCountEnv := os.Getenv("SEED_USER_COUNT"); seedCountEnv != "" {
		if count, err := strconv.Atoi(seedCountEnv); err == nil {
			userSeedCount = count
		}
	}

	if err := usersvc.SeedUsers(userRepository, userSeedCount); err != nil {
		logger.ZapLogger.Error("Failed to seed user data", zap.Error(err))
		// Continue anyway - seeding failure shouldn't stop the application
	} else {
		logger.ZapLogger.Info("Successfully seeded user data", zap.Int("count", userSeedCount))
	}
}
