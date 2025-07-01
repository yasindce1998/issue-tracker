// Package usersvc provides services for managing users in the issue tracking system.
// It includes implementations for both in-memory and database storage of user information.
package usersvc

import (
	"github.com/yasindce1998/issue-tracker/consts"
	userPbv1 "github.com/yasindce1998/issue-tracker/pkg/pb/user/v1"
	"github.com/hashicorp/go-memdb"
)

// UserRepository defines the interface for database operations
type UserRepository interface {
	CreateUser(user *userPbv1.User) error
	GetUserByID(userID string) (*userPbv1.User, error)
	UpdateUser(user *userPbv1.User) error
	DeleteUser(userID string) error
	ListUsers(pageToken string, pageSize int) ([]*userPbv1.User, string, error)
}

// MemDBUserRepository implements UserRepository using Hashicorp MemDB
type MemDBUserRepository struct {
	db *memdb.MemDB
}

// CreateMemDBSchema defines the schema for the in-memory database
func CreateMemDBSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"user": {
				Name: "user",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id", // Primary index on UserId
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "UserId"},
					},
					"email": {
						Name:    "email", // Secondary index on EmailAddress
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "EmailAddress"},
					},
				},
			},
		},
	}
}

// NewMemDBUserRepository initializes the repository with a MemDB instance
func NewMemDBUserRepository() (*MemDBUserRepository, error) {
	db, err := memdb.NewMemDB(CreateMemDBSchema())
	if err != nil {
		return nil, err
	}
	return &MemDBUserRepository{db: db}, nil
}

// CreateUser adds a new user to the repository
func (r *MemDBUserRepository) CreateUser(user *userPbv1.User) error {
	txn := r.db.Txn(true)
	defer txn.Commit()

	// Ensure email uniqueness
	raw, err := txn.First("user", "email", user.EmailAddress)
	if err != nil {
		return err
	}
	if raw != nil {
		return consts.ErrEmailAlreadyExists
	}

	// Insert the user into the database
	return txn.Insert("user", user)
}

// GetUserByID retrieves a user by their ID
func (r *MemDBUserRepository) GetUserByID(userID string) (*userPbv1.User, error) {
	txn := r.db.Txn(false)
	defer txn.Abort()

	raw, err := txn.First("user", "id", userID)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, consts.ErrUserNotFound
	}
	return raw.(*userPbv1.User), nil
}

// UpdateUser updates an existing user
func (r *MemDBUserRepository) UpdateUser(user *userPbv1.User) error {
	txn := r.db.Txn(true)
	defer txn.Commit()

	// Check whether the user exists
	raw, err := txn.First("user", "id", user.UserId)
	if err != nil {
		return err
	}
	if raw == nil {
		return consts.ErrUserNotFound
	}

	existingUser := raw.(*userPbv1.User)
	if existingUser.EmailAddress != user.EmailAddress {
		// Check if email is already in use by another user
		emailCheck, err := txn.First("user", "email", user.EmailAddress)
		if err != nil {
			return err
		}
		if emailCheck != nil {
			return consts.ErrEmailAlreadyExists
		}
	}

	// Replace the user record in the database
	if err := txn.Delete("user", existingUser); err != nil {
		return err
	}
	return txn.Insert("user", user)
}

// DeleteUser removes a user from the repository
func (r *MemDBUserRepository) DeleteUser(userID string) error {
	txn := r.db.Txn(true)
	defer txn.Commit()

	raw, err := txn.First("user", "id", userID)
	if err != nil {
		return err
	}
	if raw == nil {
		return consts.ErrUserNotFound
	}
	return txn.Delete("user", raw)
}

// ListUsers retrieves a paginated list of users
func (r *MemDBUserRepository) ListUsers(pageToken string, pageSize int) ([]*userPbv1.User, string, error) {
	txn := r.db.Txn(false)
	defer txn.Abort()

	it, err := txn.Get("user", "id")
	if err != nil {
		return nil, "", err
	}

	var users []*userPbv1.User
	for obj := it.Next(); obj != nil; obj = it.Next() {
		users = append(users, obj.(*userPbv1.User))
	}

	// Perform pagination using the helper
	paginatedUsers, nextPageToken := paginateUsers(users, pageSize, pageToken)
	return paginatedUsers, nextPageToken, nil
}

// Pagination Helper
func paginateUsers(users []*userPbv1.User, pageSize int, pageToken string) ([]*userPbv1.User, string) {
	startIndex := 0
	if pageToken != "" {
		for i, user := range users {
			if user.UserId == pageToken {
				startIndex = i + 1
				break
			}
		}
	}

	endIndex := startIndex + pageSize
	if endIndex > len(users) {
		endIndex = len(users)
	}

	var nextPageToken string
	if endIndex < len(users) {
		nextPageToken = users[endIndex-1].UserId
	}

	return users[startIndex:endIndex], nextPageToken
}
