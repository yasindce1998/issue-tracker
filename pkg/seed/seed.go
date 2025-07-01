// Package seed provides functions for seeding test data
package seed

import (
	"os"

	"github.com/yasindce1998/issue-tracker/logger"
	projectPbv1 "github.com/yasindce1998/issue-tracker/pkg/pb/project/v1"
	userPbv1 "github.com/yasindce1998/issue-tracker/pkg/pb/user/v1"
	"github.com/yasindce1998/issue-tracker/pkg/svc/issuessvc"
	"github.com/yasindce1998/issue-tracker/pkg/svc/projectsvc"
	"github.com/yasindce1998/issue-tracker/pkg/svc/usersvc"
)

// Data seeds all test data if environment conditions are met
func Data(
	userRepo usersvc.UserRepository,
	projectRepo projectsvc.ProjectRepository,
	issuesRepo issuessvc.IssuesRepository,
	userService userPbv1.UserServiceServer,
	projectService projectPbv1.ProjectServiceServer,
	projectClient projectPbv1.ProjectServiceClient,
	userClient userPbv1.UserServiceClient,
) {
	// Only seed data for memDB and non-production environments
	if os.Getenv("DB_TYPE") != "memdb" || os.Getenv("ENVIRONMENT") == "production" {
		logger.ZapLogger.Info("Skipping data seeding (not memdb or in production)")
		return
	}

	// Type assertions to get the concrete types
	memdbUserRepo, userOk := userRepo.(*usersvc.MemDBUserRepository)
	memdbProjectRepo, projectOk := projectRepo.(*projectsvc.MemDBProjectRepository)
	memdbIssuesRepo, issuesOk := issuesRepo.(*issuessvc.MemDBIssuesRepository)

	if !userOk || !projectOk || !issuesOk {
		logger.ZapLogger.Info("Repositories are not MemDB implementations - skipping seeding")
		return
	}

	// Now we have the concrete types, proceed with seeding
	if projectClient != nil && userClient != nil {
		memdbIssuesRepo.SetClients(projectClient, userClient)
	}

	Users(memdbUserRepo)
	Projects(memdbProjectRepo)
	RelationshipsIfEnabled(userService, projectService, memdbIssuesRepo)
}
