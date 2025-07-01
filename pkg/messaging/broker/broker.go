// Package broker defines the messaging interface for project updates.
// It provides a common interface that various messaging implementations can implement.
package broker

import (
	"context"

	projectPbv1 "github.com/yasindce1998/issue-tracker/pkg/pb/project/v1"
)

// MessageBroker defines methods for project updates messaging
type MessageBroker interface {
	// PublishUpdate sends a project update message
	PublishUpdate(ctx context.Context, projectID string, update *projectPbv1.ProjectUpdateResponse) error

	// Subscribe registers for updates on a specific project
	Subscribe(ctx context.Context, projectID string) (<-chan *projectPbv1.ProjectUpdateResponse, error)

	// Unsubscribe stops receiving updates for a project
	Unsubscribe(ctx context.Context, projectID string, ch <-chan *projectPbv1.ProjectUpdateResponse) error

	// Close releases resources
	Close() error
}
