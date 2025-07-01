package usersvc

import (
	"fmt"

	"github.com/yasindce1998/issue-tracker/logger"
	userPbv1 "github.com/yasindce1998/issue-tracker/pkg/pb/user/v1"
	"github.com/brianvoe/gofakeit/v7"
	"go.uber.org/zap"
)

// SeedUsers generates random user data and adds it to the repository
func SeedUsers(repo UserRepository, count int) error {
	logger.ZapLogger.Info("Seeding user data", zap.Int("count", count))

	for i := 0; i < count; i++ {
		// Create a new user with fake data
		user := &userPbv1.User{
			UserId:       gofakeit.UUID(),
			FirstName:    gofakeit.FirstName(),
			LastName:     gofakeit.LastName(),
			EmailAddress: gofakeit.Email(),
		}

		// Add to repository
		err := repo.CreateUser(user)
		if err != nil {
			logger.ZapLogger.Error("Failed to seed user",
				zap.String("email", user.EmailAddress),
				zap.Error(err))
			continue
		}

		logger.ZapLogger.Debug("Created seed user",
			zap.String("id", user.UserId),
			zap.String("name", fmt.Sprintf("%s %s", user.FirstName, user.LastName)))
	}

	logger.ZapLogger.Info("User data seeding completed")
	return nil
}

// CreateRandomUserRequest generates a random CreateUserRequest for testing
func CreateRandomUserRequest() *userPbv1.CreateUserRequest {
	return &userPbv1.CreateUserRequest{
		FirstName:    gofakeit.FirstName(),
		LastName:     gofakeit.LastName(),
		EmailAddress: gofakeit.Email(),
	}
}
