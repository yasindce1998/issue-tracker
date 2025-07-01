package models

import "gorm.io/gorm"

// User schema reflecting the protobuf message
type User struct {
	UserID       string         `gorm:"type:uuid;primaryKey"`     // Unique identifier for the user
	FirstName    string         `gorm:"size:50;not null"`         // First name of the user
	LastName     string         `gorm:"size:50;not null"`         // Last name of the user
	EmailAddress string         `gorm:"size:255;unique;not null"` // Email address of the user
	DeletedAt    gorm.DeletedAt `gorm:"index"`                    // Soft delete field
}
