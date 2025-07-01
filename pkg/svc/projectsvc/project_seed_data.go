package projectsvc

import (
	"context"
	"crypto/rand"
	"log"
	"math/big"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"

	projectPbv1 "github.com/yasindce1998/issue-tracker/pkg/pb/project/v1"
)

// Project types and fields to create more realistic project data
var (
	projectTypes = []string{
		"Web Application",
		"Mobile App",
		"API Service",
		"Database Migration",
		"Infrastructure",
		"DevOps",
		"UI/UX Redesign",
		"Security Audit",
		"Performance Optimization",
		"Data Analytics",
	}
)

// GenerateRandomProjects creates a specified number of random projects
func GenerateRandomProjects(count int) []*projectPbv1.Project {
	projects := make([]*projectPbv1.Project, count)

	for i := 0; i < count; i++ {
		// Use crypto/rand to securely select a random item
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(projectTypes))))
		if err != nil {
			log.Printf("Error generating random number: %v, using default", err)
			n = big.NewInt(0)
		}
		projectType := projectTypes[n.Int64()]

		// Create a project with a descriptive name based on the type
		project := &projectPbv1.Project{
			ProjectId:   uuid.New().String(),
			Name:        projectType + " - " + gofakeit.ProductName(),
			Description: gofakeit.Paragraph(2, 4, 10, "\n"),
			IssueCount:  int32(15) * int32(gofakeit.Float32Range(0, 1)),
		}

		projects[i] = project
	}

	return projects
}

// SeedProjects inserts a set of random projects into the repository for testing
func SeedProjects(repository ProjectRepository, count int) error {
	log.Printf("Seeding %d projects into the database...", count)
	projects := GenerateRandomProjects(count)

	for _, project := range projects {
		err := repository.CreateProject(project)
		if err != nil {
			return err
		}
		log.Printf("Created project: %s (%s)", project.Name, project.ProjectId)

		// Optionally seed some issues for this project too
		if project.IssueCount > 0 {
			err = seedProjectIssues(repository, project.ProjectId, int(project.IssueCount))
			if err != nil {
				log.Printf("Warning: failed to seed all issues for project %s: %v", project.ProjectId, err)
			}
		}
	}

	log.Printf("Successfully seeded %d projects", count)
	return nil
}

// seedProjectIssues creates random issue relations for a project
func seedProjectIssues(repository ProjectRepository, projectID string, count int) error {
	for i := 0; i < count; i++ {
		issueID := uuid.New().String()
		err := repository.AddIssueToProject(projectID, issueID)
		if err != nil {
			return err
		}
	}
	return nil
}

// InitializeProjectSeedData initializes the database with seed data for projects
func InitializeProjectSeedData(ctx context.Context, projectService *ProjectService, count int) {
	if projectService == nil {
		log.Fatal("Project service is nil, cannot seed data")
		return
	}

	// Check if we already have projects
	resp, err := projectService.ListProjects(ctx, nil)
	if err == nil && len(resp.Projects) > 0 {
		log.Printf("Found %d existing projects, skipping seed data", len(resp.Projects))
		return
	}

	// Seed projects
	err = SeedProjects(projectService.repository, count)
	if err != nil {
		log.Printf("Error seeding projects: %v", err)
	}
}
