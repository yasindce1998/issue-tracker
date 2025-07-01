package usersvc

import (
	"errors"
	"fmt"
	"strings"

	"github.com/yasindce1998/issue-tracker/consts"
	"github.com/yasindce1998/issue-tracker/models"
	userPbv1 "github.com/yasindce1998/issue-tracker/pkg/pb/user/v1"
	"gorm.io/gorm"
)

// PostgresUserRepository implements UserRepository using GORM for PostgreSQL
type PostgresUserRepository struct {
	db *gorm.DB
}

// NewPostgresUserRepository initializes the repository with a GORM DB instance
func NewPostgresUserRepository(db *gorm.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

// CreateUser adds a new user to the database
func (r *PostgresUserRepository) CreateUser(user *userPbv1.User) error {
	// Convert protobuf user to model
	dbUser := &models.User{
		UserID:       user.UserId,
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		EmailAddress: user.EmailAddress,
	}

	// Try to create the user
	result := r.db.Create(dbUser)
	if result.Error != nil {
		// Check for common errors
		if strings.Contains(result.Error.Error(), "unique constraint") &&
			strings.Contains(result.Error.Error(), "email_address") {
			return consts.ErrEmailAlreadyExists
		}
		return fmt.Errorf("%w: %s", consts.ErrDatabaseError, result.Error.Error())
	}

	return nil
}

// GetUserByID retrieves a user by their ID
func (r *PostgresUserRepository) GetUserByID(userID string) (*userPbv1.User, error) {
	var dbUser models.User

	if err := r.db.Where("user_id = ?", userID).First(&dbUser).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, consts.ErrUserNotFound
		}
		return nil, fmt.Errorf("%w: %s", consts.ErrDatabaseError, err.Error())
	}

	// Convert database model to protobuf
	return &userPbv1.User{
		UserId:       dbUser.UserID,
		FirstName:    dbUser.FirstName,
		LastName:     dbUser.LastName,
		EmailAddress: dbUser.EmailAddress,
	}, nil
}

// UpdateUser updates an existing user
func (r *PostgresUserRepository) UpdateUser(user *userPbv1.User) error {
	// Create a map for update values (excluding UserID)
	updates := map[string]interface{}{
		"first_name":    user.FirstName,
		"last_name":     user.LastName,
		"email_address": user.EmailAddress,
	}

	// Update user where UserID matches
	result := r.db.Model(&models.User{}).Where("user_id = ?", user.UserId).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("%w: %s", consts.ErrDatabaseError, result.Error.Error())
	}

	// Check if user was found
	if result.RowsAffected == 0 {
		return consts.ErrUserNotFound
	}

	return nil
}

// DeleteUser removes a user from the database
func (r *PostgresUserRepository) DeleteUser(userID string) error {
	result := r.db.Delete(&models.User{}, "user_id = ?", userID)
	if result.Error != nil {
		return fmt.Errorf("%w: %s", consts.ErrDatabaseError, result.Error.Error())
	}

	// Check if user was actually found and deleted
	if result.RowsAffected == 0 {
		return consts.ErrUserNotFound
	}

	return nil
}

// ListUsers retrieves a paginated list of users
func (r *PostgresUserRepository) ListUsers(pageToken string, pageSize int) ([]*userPbv1.User, string, error) {
	var dbUsers []models.User

	query := r.db.Model(&models.User{}).Limit(pageSize)
	if pageToken != "" {
		query = query.Where("user_id > ?", pageToken)
	}

	if err := query.Order("user_id").Find(&dbUsers).Error; err != nil {
		return nil, "", fmt.Errorf("%w: %s", consts.ErrDatabaseError, err.Error())
	}

	// Convert database models to protobuf responses
	users := make([]*userPbv1.User, len(dbUsers))
	for i, dbUser := range dbUsers {
		users[i] = &userPbv1.User{
			UserId:       dbUser.UserID,
			FirstName:    dbUser.FirstName,
			LastName:     dbUser.LastName,
			EmailAddress: dbUser.EmailAddress,
		}
	}

	var nextPageToken string
	if len(users) > 0 && len(users) == pageSize {
		nextPageToken = users[len(users)-1].UserId
	}

	return users, nextPageToken, nil
}
