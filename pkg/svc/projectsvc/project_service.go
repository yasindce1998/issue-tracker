package projectsvc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/yasindce1998/issue-tracker/logger"
	"github.com/yasindce1998/issue-tracker/pkg/messaging"
	"github.com/yasindce1998/issue-tracker/pkg/messaging/broker"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	projectPbv1 "github.com/yasindce1998/issue-tracker/pkg/pb/project/v1"
)

// Constants for communication methods
const (
	commMethodKafka = "kafka"
)

// ProjectService implements the ProjectServiceServer interface
type ProjectService struct {
	projectPbv1.UnimplementedProjectServiceServer
	repository    ProjectRepository
	messageBroker broker.MessageBroker
	subscribers   map[string][]chan *projectPbv1.ProjectUpdateResponse
	subscribersMu sync.RWMutex
}

// NewProjectService creates a new ProjectService with dependency injection
func NewProjectService(repository ProjectRepository) (*ProjectService, error) {
	// Get the message broker from the factory
	mb, err := messaging.NewMessageBroker()
	if err != nil {
		return nil, fmt.Errorf("failed to create message broker: %w", err)
	}
	return &ProjectService{
		repository:    repository,
		messageBroker: mb,
		subscribers:   make(map[string][]chan *projectPbv1.ProjectUpdateResponse),
	}, nil
}

// CreateProject creates a new project
func (s *ProjectService) CreateProject(_ context.Context, req *projectPbv1.CreateProjectRequest) (*projectPbv1.CreateProjectResponse, error) {
	// Generate a new UUID for the project
	projectID := uuid.New().String()

	// Create a new project
	project := &projectPbv1.Project{
		ProjectId:   projectID,
		Name:        req.Name,
		Description: req.Description,
		IssueCount:  0,
	}

	// Store the project in the repository
	err := s.repository.CreateProject(project)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create project: %v", err)
	}

	return &projectPbv1.CreateProjectResponse{
		Project: project,
	}, nil
}

// GetProject retrieves a project by ID
func (s *ProjectService) GetProject(_ context.Context, req *projectPbv1.GetProjectRequest) (*projectPbv1.GetProjectResponse, error) {
	// Retrieve the project from the repository
	project, err := s.repository.ReadProject(req.ProjectId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "project not found: %v", err)
	}

	return &projectPbv1.GetProjectResponse{
		Project: project,
	}, nil
}

// UpdateProject updates an existing project
func (s *ProjectService) UpdateProject(_ context.Context, req *projectPbv1.UpdateProjectRequest) (*projectPbv1.UpdateProjectResponse, error) {
	// First check if the project exists
	existingProject, err := s.repository.ReadProject(req.ProjectId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "project not found: %v", err)
	}

	// Update the project fields
	existingProject.Name = req.Name
	existingProject.Description = req.Description

	// Save the updated project
	err = s.repository.UpdateProject(existingProject)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update project: %v", err)
	}

	return &projectPbv1.UpdateProjectResponse{
		Project: existingProject,
	}, nil
}

// DeleteProject deletes a project by ID
func (s *ProjectService) DeleteProject(_ context.Context, req *projectPbv1.DeleteProjectRequest) (*emptypb.Empty, error) {
	// Delete the project
	err := s.repository.DeleteProject(req.ProjectId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "failed to delete project: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// ListProjects lists all projects
func (s *ProjectService) ListProjects(_ context.Context, _ *emptypb.Empty) (*projectPbv1.ListProjectsResponse, error) {
	// Retrieve all projects
	projects, err := s.repository.ListProjects()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list projects: %v", err)
	}

	return &projectPbv1.ListProjectsResponse{
		Projects: projects,
	}, nil
}

// UpdateProjectWithIssue adds an issue to a project
func (s *ProjectService) UpdateProjectWithIssue(_ context.Context, req *projectPbv1.UpdateProjectWithIssueRequest) (*projectPbv1.UpdateProjectWithIssueResponse, error) {
	// Add the issue to the project
	err := s.repository.AddIssueToProject(req.ProjectId, req.IssueId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update project with issue: %v", err)
	}

	// Get the updated project
	project, err := s.repository.ReadProject(req.ProjectId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get updated project: %v", err)
	}

	// Debug log to verify issue count
	logger.ZapLogger.Info("Project issue count after update",
		zap.String("project_id", req.ProjectId),
		zap.Int32("issue_count", project.IssueCount))

	// Notify subscribers about the update
	s.notifySubscribers(req.ProjectId, &projectPbv1.ProjectUpdateResponse{
		ProjectId:  req.ProjectId,
		IssueCount: project.IssueCount,
		Message:    fmt.Sprintf("Issue %s added to project %s", req.IssueId, req.ProjectId),
	})

	return &projectPbv1.UpdateProjectWithIssueResponse{
		ProjectId:  req.ProjectId,
		IssueCount: project.IssueCount,
		Message:    fmt.Sprintf("Issue %s added to project %s", req.IssueId, req.ProjectId),
	}, nil
}

// StreamProjectUpdates handles streaming project updates
func (s *ProjectService) StreamProjectUpdates(stream projectPbv1.ProjectService_StreamProjectUpdatesServer) error {
	var subscribedProjectID string
	var updateCh <-chan *projectPbv1.ProjectUpdateResponse

	ctx := stream.Context()

	// Create in-memory channel if not using Kafka
	var inMemoryCh chan *projectPbv1.ProjectUpdateResponse
	if os.Getenv("COMMUNICATION_METHOD") != commMethodKafka {
		inMemoryCh = make(chan *projectPbv1.ProjectUpdateResponse, 10)
		updateCh = inMemoryCh
		defer close(inMemoryCh)
	}

	// Process incoming messages in a separate goroutine
	errCh := make(chan error, 1)
	go func() {
		for {
			req, err := stream.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					// Client closed the stream gracefully
					errCh <- nil
				} else {
					// Some other error
					logger.ZapLogger.Warn("Error receiving from stream",
						zap.Error(err))
					errCh <- err
				}
				return
			}

			switch req.Action {
			case "subscribe":
				// If already subscribed, clean up first
				if subscribedProjectID != "" {
					if os.Getenv("COMMUNICATION_METHOD") == commMethodKafka {
						if updateCh != nil {
							_ = s.messageBroker.Unsubscribe(ctx, subscribedProjectID, updateCh)
						}
					} else {
						s.removeSubscriber(subscribedProjectID, inMemoryCh)
					}
				}

				// Subscribe to project updates
				subscribedProjectID = req.ProjectId

				if os.Getenv("COMMUNICATION_METHOD") == commMethodKafka {
					// Subscribe via Kafka
					kCh, err := s.messageBroker.Subscribe(ctx, subscribedProjectID)
					if err != nil {
						logger.ZapLogger.Error("Failed to subscribe via Kafka",
							zap.String("project_id", subscribedProjectID),
							zap.Error(err))
						return
					}
					updateCh = kCh
				} else {
					// Use in-memory subscription
					s.addSubscriber(subscribedProjectID, inMemoryCh)
				}

				logger.ZapLogger.Info("Client subscribed to project",
					zap.String("project_id", subscribedProjectID),
					zap.String("method", getCommMethod()))

			case "update":
				if req.ProjectId != subscribedProjectID {
					// Can't update a project you're not subscribed to
					continue
				}

				// Just notify about the update
				project, err := s.repository.ReadProject(req.ProjectId)
				if err == nil {
					s.notifySubscribers(req.ProjectId, &projectPbv1.ProjectUpdateResponse{
						ProjectId:  req.ProjectId,
						IssueCount: project.IssueCount,
						Message:    fmt.Sprintf("Project %s updated", req.ProjectId),
					})
				}
			}
		}
	}()

	// Send updates to the client
	if updateCh != nil {
		return s.handleProjectUpdates(ctx, stream, updateCh, errCh, subscribedProjectID, inMemoryCh)
	}

	return nil
}

// handleProjectUpdates processes updates from the update channel and sends them to the client
func (s *ProjectService) handleProjectUpdates(
	ctx context.Context,
	stream projectPbv1.ProjectService_StreamProjectUpdatesServer,
	updateCh <-chan *projectPbv1.ProjectUpdateResponse,
	errCh <-chan error,
	subscribedProjectID string,
	inMemoryCh chan *projectPbv1.ProjectUpdateResponse,
) error {
	for {
		select {
		case update, ok := <-updateCh:
			if !ok {
				// Channel closed
				return nil
			}
			if err := stream.Send(update); err != nil {
				logger.ZapLogger.Error("Error sending to stream", zap.Error(err))

				// Clean up subscription
				if os.Getenv("COMMUNICATION_METHOD") == commMethodKafka {
					_ = s.messageBroker.Unsubscribe(ctx, subscribedProjectID, updateCh)
				} else {
					s.removeSubscriber(subscribedProjectID, inMemoryCh)
				}

				return err
			}
		case err := <-errCh:
			// Handle errors from the receiving goroutine
			return err
		case <-ctx.Done():
			// Context canceled (client disconnected, timeout, etc.)
			return ctx.Err()
		}
	}
}

// Helper function to get communication method
func getCommMethod() string {
	method := os.Getenv("COMMUNICATION_METHOD")
	if method == "" {
		return "stream"
	}
	return method
}

// Helper methods for managing subscribers
func (s *ProjectService) addSubscriber(projectID string, ch chan *projectPbv1.ProjectUpdateResponse) {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()
	s.subscribers[projectID] = append(s.subscribers[projectID], ch)
}

func (s *ProjectService) removeSubscriber(projectID string, ch chan *projectPbv1.ProjectUpdateResponse) {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()
	channels := s.subscribers[projectID]
	for i, c := range channels {
		if c == ch {
			// Remove this channel
			s.subscribers[projectID] = append(channels[:i], channels[i+1:]...)
			break
		}
	}
}

func (s *ProjectService) notifySubscribers(projectID string, update *projectPbv1.ProjectUpdateResponse) {
	// If we're using Kafka, publish to Kafka
	if os.Getenv("COMMUNICATION_METHOD") == commMethodKafka {
		err := s.messageBroker.PublishUpdate(context.Background(), projectID, update)
		if err != nil {
			logger.ZapLogger.Error("Failed to publish update to Kafka",
				zap.String("project_id", projectID),
				zap.Error(err))
		}
		return
	}

	// Otherwise use the existing in-memory notification
	s.subscribersMu.RLock()
	defer s.subscribersMu.RUnlock()

	projectSubs, ok := s.subscribers[projectID]
	if !ok {
		logger.ZapLogger.Debug("No subscribers for project", zap.String("project_id", projectID))
		return
	}

	logger.ZapLogger.Info("Broadcasting update to subscribers",
		zap.String("project_id", projectID),
		zap.Int("subscriber_count", len(projectSubs)),
		zap.Int("issue_count", int(update.IssueCount)))

	for _, ch := range projectSubs {
		select {
		case ch <- update:
			logger.ZapLogger.Debug("Notification sent to subscriber", zap.String("project_id", projectID))
		default:
			logger.ZapLogger.Warn("Subscriber channel blocked, notification skipped", zap.String("project_id", projectID))
		}
	}
}

// Close releases resources used by the project service
func (s *ProjectService) Close() error {
	if s.messageBroker != nil {
		return s.messageBroker.Close()
	}
	return nil
}
