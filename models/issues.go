// Package models contains the data structures and types representing the domain entities
// used throughout the application. These structures are also used for database mapping.
package models

import (
	"time"

	"gorm.io/gorm"
)

// Issues represents the database schema for the Issue entity
type Issues struct {
	IssueID     string         `gorm:"type:uuid;primaryKey"` // Unique identifier for the issue
	Summary     string         `gorm:"size:100;not null"`    // Short summary of the issue
	Description string         `gorm:"size:500"`             // Detailed description of the issue
	Status      string         `gorm:"size:50;not null"`     // Status of the issue (e.g., NEW, ASSIGNED)
	Resolution  string         `gorm:"size:50"`              // Resolution status (e.g., FIXED, INVALID)
	Type        string         `gorm:"size:50;not null"`     // Type of the issue (e.g., BUG, FEATURE)
	Priority    string         `gorm:"size:50;not null"`     // Priority level (e.g., CRITICAL, MINOR)
	ProjectID   string         `gorm:"type:uuid;not null"`   // Associated project ID
	AssigneeID  *string        `gorm:"type:uuid"`            // ID of the assigned user (nullable)
	CreateDate  time.Time      `gorm:"autoCreateTime"`       // Timestamp when the issue was created
	ModifyDate  time.Time      `gorm:"autoUpdateTime"`       // Timestamp when the issue was last modified
	DeletedAt   gorm.DeletedAt `gorm:"index"`                // Soft delete field
}
