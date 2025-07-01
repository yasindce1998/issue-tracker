// Package consts provides constants used throughout the application
package consts

import (
	"errors"
)

// Error definitions for repository operations
var (
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrUserNotFound       = errors.New("user not found")
	ErrDatabaseError      = errors.New("database error")
	ErrNotFound           = errors.New("not found")

	// Issues related error constants
	ErrIssueNotFound           = errors.New("issue not found")
	ErrProjectNotFound         = errors.New("project not found")
	ErrIssueAlreadyExists      = errors.New("issue already exists")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrInvalidIssueType        = errors.New("invalid issue type")
	ErrInvalidIssuePriority    = errors.New("invalid issue priority")
	ErrInvalidIssueStatus      = errors.New("invalid issue status")
	ErrInvalidIssueResolution  = errors.New("invalid issue resolution")

	ErrNoSubscription = errors.New("no subscription found for project")
	ErrPublishFailed  = errors.New("failed to publish update")
)
