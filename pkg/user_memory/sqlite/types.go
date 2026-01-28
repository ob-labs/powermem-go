// Package sqlite provides SQLite implementation for user profile storage.
//
// This package implements the UserProfileStore interface using SQLite as the backend.
// It is defined in a separate package to avoid circular dependencies.
package sqlite

import "time"

// UserProfile represents a user profile stored in SQLite.
//
// This type is defined in the sqlite package to avoid circular dependencies
// with the usermemory package. It mirrors the usermemory.UserProfile structure.
type UserProfile struct {
	// ID is the unique identifier of the profile.
	ID int64 `json:"id"`

	// UserID identifies the user this profile belongs to.
	UserID string `json:"user_id"`

	// ProfileContent is the unstructured text description of the user.
	ProfileContent string `json:"profile_content,omitempty"`

	// Topics contains structured user characteristics as key-value pairs.
	Topics map[string]interface{} `json:"topics,omitempty"`

	// CreatedAt is when the profile was first created.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the profile was last updated.
	UpdatedAt time.Time `json:"updated_at"`
}

// GetProfilesOptions contains options for querying user profiles.
//
// This type is defined in the sqlite package to avoid circular dependencies.
type GetProfilesOptions struct {
	// UserID filters profiles by user ID.
	UserID string

	// MainTopic filters profiles by main topic (for structured topics).
	MainTopic []string

	// SubTopic filters profiles by sub-topic (for structured topics).
	SubTopic []string

	// TopicValue filters profiles by topic value (for structured topics).
	TopicValue []string

	// Limit sets the maximum number of results to return.
	Limit int

	// Offset sets the number of results to skip (for pagination).
	Offset int
}
