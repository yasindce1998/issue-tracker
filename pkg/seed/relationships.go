package seed

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/yasindce1998/issue-tracker/logger"
	issuesPbv1 "github.com/yasindce1998/issue-tracker/pkg/pb/issues/v1"
	projectPbv1 "github.com/yasindce1998/issue-tracker/pkg/pb/project/v1"
	userPbv1 "github.com/yasindce1998/issue-tracker/pkg/pb/user/v1"
	"github.com/yasindce1998/issue-tracker/pkg/svc/issuessvc"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	issueTypes = []issuesPbv1.Type{
		issuesPbv1.Type_BUG,
		issuesPbv1.Type_FEATURE,
		issuesPbv1.Type_COSMETIC,
		issuesPbv1.Type_PERFORMANCE,
	}

	priorities = []issuesPbv1.Priority{
		issuesPbv1.Priority_MINOR,
		issuesPbv1.Priority_IMPORTANT,
		issuesPbv1.Priority_MAJOR,
		issuesPbv1.Priority_CRITICAL,
	}
)

// RelationshipsIfEnabled creates relationships between users, projects, and issues if enabled
func RelationshipsIfEnabled(
	userService userPbv1.UserServiceServer,
	projectService projectPbv1.ProjectServiceServer,
	issuesRepository *issuessvc.MemDBIssuesRepository,
) {
	// Check if relationships should be seeded
	if os.Getenv("SEED_RELATIONSHIPS") == "true" {
		logger.ZapLogger.Info("Seeding entity relationships")

		if err := Relationships(userService, projectService, issuesRepository); err != nil {
			logger.ZapLogger.Error("Failed to seed relationships", zap.Error(err))
		} else {
			logger.ZapLogger.Info("Successfully seeded entity relationships")
		}
	} else {
		logger.ZapLogger.Info("Skipping relationship seeding (SEED_RELATIONSHIPS != true)")
	}
}

// Relationships creates relationships between users, projects and issues
func Relationships(
	userService userPbv1.UserServiceServer,
	projectService projectPbv1.ProjectServiceServer,
	issuesRepository *issuessvc.MemDBIssuesRepository,
) error {
	ctx := context.Background()

	// Wait a moment for services to be fully ready
	time.Sleep(500 * time.Millisecond)

	// Get all users
	usersList, err := userService.ListUsers(ctx, &userPbv1.ListUsersRequest{})
	if err != nil {
		return fmt.Errorf("failed to list users for creating relationships: %w", err)
	}

	// Get all projects
	projectsList, err := projectService.ListProjects(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("failed to list projects for creating relationships: %w", err)
	}

	if len(usersList.Users) == 0 || len(projectsList.Projects) == 0 {
		return fmt.Errorf("no users or projects available for seeding relationships")
	}

	// Create some realistic issues and assign users
	logger.ZapLogger.Info("Creating issues with user assignments")

	// Create issues for each project
	for _, project := range projectsList.Projects {
		if err := createIssuesForProject(project, usersList.Users, issuesRepository); err != nil {
			logger.ZapLogger.Warn("Error creating issues for project",
				zap.String("project_id", project.ProjectId),
				zap.Error(err))
			continue
		}
	}

	logger.ZapLogger.Info("Finished creating relationships between entities")
	return nil
}

// createIssuesForProject creates 1-5 issues for a specific project
func createIssuesForProject(
	project *projectPbv1.Project,
	users []*userPbv1.User,
	issuesRepository *issuessvc.MemDBIssuesRepository,
) error {
	// Create 1-5 issues per project
	maxNum := big.NewInt(5)
	randomNum, err := rand.Int(rand.Reader, maxNum)
	if err != nil {
		return fmt.Errorf("failed to generate random number: %w", err)
	}
	numIssues := int(randomNum.Int64()) + 1

	for i := 0; i < numIssues; i++ {
		// Generate a random issue type
		typeIndex, err := randomInt(len(issueTypes))
		if err != nil {
			return fmt.Errorf("failed to generate random issue type: %w", err)
		}
		issueType := issueTypes[typeIndex]

		// Generate a random priority
		priorityIndex, err := randomInt(len(priorities))
		if err != nil {
			return fmt.Errorf("failed to generate random priority: %w", err)
		}
		priority := priorities[priorityIndex]

		// Generate summary and description
		summary := generateIssueSummary(issueType, project.Name)
		shortDesc := fmt.Sprintf("Issue for %s", project.Name)

		// Get an assignee if applicable
		assigneeID := selectRandomAssignee(users)

		// Create and save the issue
		if err := createAndSaveIssue(project, issueType, priority, summary, shortDesc, assigneeID, issuesRepository); err != nil {
			logger.ZapLogger.Error("Failed to create issue",
				zap.String("project", project.ProjectId),
				zap.Error(err))
			continue
		}
	}

	return nil
}

// generateIssueSummary creates an appropriate summary based on issue type
func generateIssueSummary(issueType issuesPbv1.Type, projectName string) string {
	switch issueType {
	case issuesPbv1.Type_TYPE_UNSPECIFIED:
		return fmt.Sprintf("General issue in %s", projectName)
	case issuesPbv1.Type_BUG:
		bugTerms := []string{"crash", "error", "exception", "bug", "problem"}
		termIndex, _ := randomInt(len(bugTerms))
		return fmt.Sprintf("Fix %s in %s", bugTerms[termIndex], projectName)
	case issuesPbv1.Type_FEATURE:
		featureTerms := []string{"login", "search", "export", "report", "dashboard"}
		termIndex, _ := randomInt(len(featureTerms))
		return fmt.Sprintf("Add %s feature to %s", featureTerms[termIndex], projectName)
	case issuesPbv1.Type_COSMETIC:
		return fmt.Sprintf("Improve visual appearance of %s", projectName)
	case issuesPbv1.Type_PERFORMANCE:
		return fmt.Sprintf("Optimize performance in %s", projectName)
	default:
		improveTerms := []string{"performance", "UI", "UX", "security", "accessibility"}
		termIndex, _ := randomInt(len(improveTerms))
		return fmt.Sprintf("Improve %s for %s", improveTerms[termIndex], projectName)
	}
}

// selectRandomAssignee returns a random user ID or empty string (70% chance of having an assignee)
func selectRandomAssignee(users []*userPbv1.User) string {
	if len(users) == 0 {
		return ""
	}

	hasAssigneeRand, _ := randomInt(100)
	if hasAssigneeRand < 70 { // 70% chance of having an assignee
		userIndex, _ := randomInt(len(users))
		return users[userIndex].UserId
	}
	return ""
}

// createAndSaveIssue creates an issue and adds it to the repository
func createAndSaveIssue(
	project *projectPbv1.Project,
	issueType issuesPbv1.Type,
	priority issuesPbv1.Priority,
	summary string,
	description string,
	assigneeID string,
	issuesRepository *issuessvc.MemDBIssuesRepository,
) error {
	// Create the issue directly
	issueID := uuid.New().String()
	currentTime := time.Now()
	timestamp := timestamppb.New(currentTime)

	issue := &issuesPbv1.Issue{
		IssueId:     issueID,
		Summary:     summary,
		Description: description,
		Type:        issueType,
		Priority:    priority,
		Status:      issuesPbv1.Status_ASSIGNED,
		ProjectId:   project.ProjectId,
		AssigneeId:  assigneeID,
		CreateDate:  timestamp,
		ModifyDate:  timestamp,
	}

	// Insert directly into repository
	if err := issuesRepository.CreateIssue(issue); err != nil {
		return err
	}

	if err := issuesRepository.UpdateIssue(issue); err != nil {
		logger.ZapLogger.Info("Attempting alternative approach for project-issue relationship")
		logger.ZapLogger.Info("Created issue but couldn't link to project",
			zap.String("project", project.ProjectId),
			zap.String("issue", issueID),
			zap.String("summary", summary))
		return err
	}

	logger.ZapLogger.Debug("Created issue and added to project",
		zap.String("project", project.ProjectId),
		zap.String("issue", issueID),
		zap.String("summary", summary))

	return nil
}

// randomInt generates a secure random integer between 0 and max-1
func randomInt(maxInput int) (int, error) {
	if maxInput <= 0 {
		return 0, fmt.Errorf("max must be positive")
	}

	bigMax := big.NewInt(int64(maxInput))
	n, err := rand.Int(rand.Reader, bigMax)
	if err != nil {
		return 0, err
	}
	return int(n.Int64()), nil
}
