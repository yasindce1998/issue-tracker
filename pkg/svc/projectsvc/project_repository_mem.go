// Package projectsvc provides services for managing projects and their relationships with issues.
// It includes implementations for both in-memory and database storage.
package projectsvc

import (
	"errors"

	projectPbv1 "github.com/yasindce1998/issue-tracker/pkg/pb/project/v1"
	"github.com/hashicorp/go-memdb"
)

// ProjectRepository defines repository methods required for project operations
type ProjectRepository interface {
	CreateProject(project *projectPbv1.Project) error
	ReadProject(projectID string) (*projectPbv1.Project, error)
	UpdateProject(project *projectPbv1.Project) error
	DeleteProject(projectID string) error
	ListProjects() ([]*projectPbv1.Project, error)
	AddIssueToProject(projectID string, issueID string) error
	RemoveIssueFromProject(projectID string, issueID string) error
}

// MemDBProjectRepository is an in-memory implementation of ProjectRepository
type MemDBProjectRepository struct {
	db *memdb.MemDB
}

// CreateProjectMemDBSchema defines the schema for the in-memory database
func CreateProjectMemDBSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"project": {
				Name: "project",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "ProjectId"},
					},
				},
			},
			"project_issue": {
				Name: "project_issue",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:   "id",
						Unique: true,
						Indexer: &memdb.CompoundIndex{
							Indexes: []memdb.Indexer{
								&memdb.StringFieldIndex{Field: "ProjectId"},
								&memdb.StringFieldIndex{Field: "IssueId"},
							},
						},
					},
					"project": {
						Name:    "project",
						Unique:  false,
						Indexer: &memdb.StringFieldIndex{Field: "ProjectId"},
					},
					"issue": {
						Name:    "issue",
						Unique:  false,
						Indexer: &memdb.StringFieldIndex{Field: "IssueId"},
					},
				},
			},
		},
	}
}

// NewMemDBProjectRepository initializes the repository
func NewMemDBProjectRepository() (*MemDBProjectRepository, error) {
	db, err := memdb.NewMemDB(CreateProjectMemDBSchema())
	if err != nil {
		return nil, err
	}
	return &MemDBProjectRepository{
		db: db,
	}, nil
}

// ProjectIssueRelation stores the relationship between projects and issues
type ProjectIssueRelation struct {
	ProjectID string
	IssueID   string
}

// CreateProject adds a new project to the repository
func (r *MemDBProjectRepository) CreateProject(project *projectPbv1.Project) error {
	txn := r.db.Txn(true)
	defer txn.Commit()
	return txn.Insert("project", project)
}

// ReadProject retrieves a project by its ID
func (r *MemDBProjectRepository) ReadProject(projectID string) (*projectPbv1.Project, error) {
	txn := r.db.Txn(false)
	defer txn.Abort()

	raw, err := txn.First("project", "id", projectID)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, errors.New("project not found")
	}
	return raw.(*projectPbv1.Project), nil
}

// UpdateProject updates an existing project
func (r *MemDBProjectRepository) UpdateProject(project *projectPbv1.Project) error {
	txn := r.db.Txn(true)
	defer txn.Commit()
	return txn.Insert("project", project)
}

// DeleteProject removes a project from the repository
func (r *MemDBProjectRepository) DeleteProject(projectID string) error {
	txn := r.db.Txn(true)
	defer txn.Commit()

	// First check if project exists
	raw, err := txn.First("project", "id", projectID)
	if err != nil {
		return err
	}
	if raw == nil {
		return errors.New("project not found")
	}

	// Delete the project
	if err := txn.Delete("project", raw); err != nil {
		return err
	}

	// Delete all project-issue relationships for this project
	issueIt, err := txn.Get("project_issue", "project", projectID)
	if err != nil {
		return err
	}
	for relation := issueIt.Next(); relation != nil; relation = issueIt.Next() {
		if err := txn.Delete("project_issue", relation); err != nil {
			return err
		}
	}

	return nil
}

// ListProjects retrieves all projects from the repository
func (r *MemDBProjectRepository) ListProjects() ([]*projectPbv1.Project, error) {
	txn := r.db.Txn(false)
	defer txn.Abort()

	it, err := txn.Get("project", "id")
	if err != nil {
		return nil, err
	}

	var projects []*projectPbv1.Project
	for obj := it.Next(); obj != nil; obj = it.Next() {
		projects = append(projects, obj.(*projectPbv1.Project))
	}

	return projects, nil
}

// AddIssueToProject associates an issue with a project
func (r *MemDBProjectRepository) AddIssueToProject(projectID string, issueID string) error {
	txn := r.db.Txn(true)
	defer txn.Commit()

	// Check if project exists
	projectRaw, err := txn.First("project", "id", projectID)
	if err != nil {
		return err
	}
	if projectRaw == nil {
		return errors.New("project not found")
	}
	project := projectRaw.(*projectPbv1.Project)

	// Check if relation already exists
	relationRaw, err := txn.First("project_issue", "id", projectID, issueID)
	if err != nil {
		return err
	}
	if relationRaw != nil {
		return errors.New("issue already added to project")
	}

	// Add the relation
	relation := &ProjectIssueRelation{
		ProjectID: projectID,
		IssueID:   issueID,
	}
	if err := txn.Insert("project_issue", relation); err != nil {
		return err
	}

	// Update issue count in project
	project.IssueCount++
	return txn.Insert("project", project)
}

// RemoveIssueFromProject removes an association between an issue and a project
func (r *MemDBProjectRepository) RemoveIssueFromProject(projectID string, issueID string) error {
	txn := r.db.Txn(true)
	defer txn.Commit()

	// Check if project exists
	projectRaw, err := txn.First("project", "id", projectID)
	if err != nil {
		return err
	}
	if projectRaw == nil {
		return errors.New("project not found")
	}
	project := projectRaw.(*projectPbv1.Project)

	// Check if relation exists
	relationRaw, err := txn.First("project_issue", "id", projectID, issueID)
	if err != nil {
		return err
	}
	if relationRaw == nil {
		return errors.New("issue not found in project")
	}

	// Remove the relation
	if err := txn.Delete("project_issue", relationRaw); err != nil {
		return err
	}

	// Update issue count in project
	if project.IssueCount > 0 {
		project.IssueCount--
	}
	return txn.Insert("project", project)
}
