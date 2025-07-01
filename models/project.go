package models

import "gorm.io/gorm"

// Project represents the database schema for the Project entity
type Project struct {
	ProjectID   string         `gorm:"type:uuid;primaryKey"` // Unique identifier for the project
	Name        string         `gorm:"size:100;not null"`    // Name of the project
	Description string         `gorm:"size:1000"`            // Detailed description of the project
	IssueCount  int32          `gorm:"default:0"`            // Number of issues associated with the project
	DeletedAt   gorm.DeletedAt `gorm:"index"`                // Soft delete field
}
