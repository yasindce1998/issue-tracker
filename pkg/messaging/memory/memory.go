// Package memory implements an in-memory message broker for project updates.
// It provides a simple implementation for development and testing purposes.
package memory

import (
	"context"
	"sync"

	"github.com/yasindce1998/issue-tracker/pkg/messaging/broker"
	projectPbv1 "github.com/yasindce1998/issue-tracker/pkg/pb/project/v1"
)

// InMemoryBroker implements MessageBroker using in-memory channels
type InMemoryBroker struct {
	subscribers map[string]map[chan<- *projectPbv1.ProjectUpdateResponse]struct{}
	mu          sync.RWMutex
}

// NewInMemoryBroker creates a new in-memory message broker
func NewInMemoryBroker() broker.MessageBroker {
	return &InMemoryBroker{
		subscribers: make(map[string]map[chan<- *projectPbv1.ProjectUpdateResponse]struct{}),
	}
}

// PublishUpdate sends a project update to all subscribers
func (b *InMemoryBroker) PublishUpdate(ctx context.Context, projectID string, update *projectPbv1.ProjectUpdateResponse) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if channels, ok := b.subscribers[projectID]; ok {
		for ch := range channels {
			select {
			case ch <- update:
				// Message sent successfully
			case <-ctx.Done():
				// Context canceled, stop processing
				return ctx.Err()
			default:
				// Skip if channel is full (non-blocking)
			}
		}
	}
	return nil
}

// Subscribe registers for project updates
func (b *InMemoryBroker) Subscribe(_ context.Context, projectID string) (<-chan *projectPbv1.ProjectUpdateResponse, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan *projectPbv1.ProjectUpdateResponse, 10)

	if _, ok := b.subscribers[projectID]; !ok {
		b.subscribers[projectID] = make(map[chan<- *projectPbv1.ProjectUpdateResponse]struct{})
	}

	b.subscribers[projectID][ch] = struct{}{}
	return ch, nil
}

// Unsubscribe stops receiving updates for a project
func (b *InMemoryBroker) Unsubscribe(_ context.Context, projectID string, _ <-chan *projectPbv1.ProjectUpdateResponse) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// For this simplified implementation, we'll remove all subscriptions for the project
	delete(b.subscribers, projectID)

	return nil
}

// Close releases resources
func (b *InMemoryBroker) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Close all channels
	for _, channels := range b.subscribers {
		for ch := range channels {
			close(ch)
		}
	}

	b.subscribers = make(map[string]map[chan<- *projectPbv1.ProjectUpdateResponse]struct{})
	return nil
}
