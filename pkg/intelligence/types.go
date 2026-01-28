// Package intelligence provides intelligent memory management features.
package intelligence

import "time"

// Memory represents a memory in the intelligence package.
//
// This type is used to avoid circular dependencies between the intelligence
// package and the core package. It mirrors the core.Memory structure but
// is defined locally to prevent import cycles.
type Memory struct {
	// ID is the unique identifier of the memory.
	ID int64

	// UserID is the identifier of the user who owns this memory.
	UserID string

	// AgentID is the identifier of the agent associated with this memory.
	AgentID string

	// Content is the text content of the memory.
	Content string

	// Embedding is the vector embedding of the memory content.
	Embedding []float64

	// Metadata contains additional metadata about the memory.
	Metadata map[string]interface{}

	// CreatedAt is when the memory was created.
	CreatedAt time.Time

	// UpdatedAt is when the memory was last updated.
	UpdatedAt time.Time

	// RetentionStrength is the current retention strength (0.0-1.0).
	// Higher values indicate stronger retention.
	RetentionStrength float64

	// LastAccessedAt is when the memory was last accessed (nil if never accessed).
	LastAccessedAt *time.Time

	// Score is the similarity score from search operations.
	Score float64
}
