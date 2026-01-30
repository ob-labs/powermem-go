// Package usermemory provides user memory management with automatic profile extraction.
package usermemory

import (
	"context"
	"time"
)

// UserProfile represents a user profile extracted from conversations.
//
// A profile contains:
//   - ProfileContent: Unstructured text description of the user
//   - Topics: Structured key-value pairs of user characteristics
//
// Profiles are automatically extracted and updated when conversations are added.
type UserProfile struct {
	// ID is the unique identifier of the profile.
	ID int64 `json:"id"`

	// UserID identifies the user this profile belongs to.
	UserID string `json:"user_id"`

	// ProfileContent is the unstructured text description of the user.
	// Extracted automatically from conversations using LLM.
	ProfileContent string `json:"profile_content,omitempty"`

	// Topics contains structured user characteristics as key-value pairs.
	// Example: {"occupation": "software engineer", "interests": ["programming", "reading"]}
	Topics map[string]interface{} `json:"topics,omitempty"`

	// CreatedAt is when the profile was first created.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the profile was last updated.
	UpdatedAt time.Time `json:"updated_at"`
}

// UserProfileStore defines the interface for storing and managing user profiles.
//
// Implementations can use different storage backends (SQLite, OceanBase, PostgreSQL).
type UserProfileStore interface {
	// SaveProfile saves or updates a user profile.
	//
	// If a profile for the user already exists, it is updated.
	// Otherwise, a new profile is created.
	//
	// Parameters:
	//   - ctx: Context for cancellation
	//   - userID: User identifier
	//   - profileContent: Unstructured profile content (optional)
	//   - topics: Structured topics (optional)
	//
	// Returns the profile ID and any error.
	SaveProfile(ctx context.Context, userID string, profileContent *string, topics map[string]interface{}) (int64, error)

	// GetProfileByUserID retrieves a user profile by user ID.
	//
	// Parameters:
	//   - ctx: Context for cancellation
	//   - userID: User identifier
	//
	// Returns the UserProfile if found, or nil if not found.
	GetProfileByUserID(ctx context.Context, userID string) (*UserProfile, error)

	// GetProfiles retrieves a list of user profiles with optional filtering.
	//
	// Parameters:
	//   - ctx: Context for cancellation
	//   - opts: Filtering and pagination options
	//
	// Returns a list of matching user profiles.
	GetProfiles(ctx context.Context, opts *GetProfilesOptions) ([]*UserProfile, error)

	// DeleteProfile deletes a user profile by profile ID.
	//
	// Parameters:
	//   - ctx: Context for cancellation
	//   - profileID: Profile ID to delete
	//
	// Returns an error if deletion fails.
	DeleteProfile(ctx context.Context, profileID int64) error

	// Close closes the profile store and releases resources.
	//
	// Returns an error if closing fails.
	Close() error
}

// GetProfilesOptions contains options for querying user profiles.
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
