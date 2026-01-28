// Package core provides the main PowerMem client and memory management functionality.
package core

import (
	"errors"
	"fmt"
)

// Predefined errors for common failure scenarios.
var (
	// ErrNotFound indicates that a requested memory was not found.
	ErrNotFound = errors.New("memory not found")

	// ErrInvalidConfig indicates that the provided configuration is invalid.
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrConnectionFailed indicates that a connection to the storage backend failed.
	ErrConnectionFailed = errors.New("connection failed")

	// ErrEmbeddingFailed indicates that embedding generation failed.
	ErrEmbeddingFailed = errors.New("embedding generation failed")

	// ErrDuplicateMemory indicates that a duplicate memory was detected.
	ErrDuplicateMemory = errors.New("duplicate memory detected")

	// ErrInvalidInput indicates that the provided input is invalid.
	ErrInvalidInput = errors.New("invalid input")

	// ErrStorageOperation indicates that a storage operation failed.
	ErrStorageOperation = errors.New("storage operation failed")

	// ErrLLMOperation indicates that an LLM operation failed.
	ErrLLMOperation = errors.New("llm operation failed")
)

// MemoryError wraps errors with operation context.
//
// It provides additional context about which operation failed,
// making error messages more informative for debugging.
//
// Example:
//
//	err := &MemoryError{
//	    Op:  "Add",
//	    Err: ErrEmbeddingFailed,
//	}
//	// Error() returns: "powermem: Add: embedding generation failed"
type MemoryError struct {
	// Op is the name of the operation that failed.
	Op string

	// Err is the underlying error.
	Err error
}

// Error returns a formatted error message.
//
// The format is: "powermem: <Op>: <Err>"
func (e *MemoryError) Error() string {
	return fmt.Sprintf("powermem: %s: %v", e.Op, e.Err)
}

// Unwrap returns the underlying error for error unwrapping.
//
// This allows using errors.Is() and errors.As() with MemoryError.
func (e *MemoryError) Unwrap() error {
	return e.Err
}

// NewMemoryError creates a new MemoryError wrapping the given error.
//
// If err is nil, returns nil. This allows safe error wrapping:
//
//	if err != nil {
//	    return NewMemoryError("Add", err)
//	}
//
// Parameters:
//   - op: Name of the operation (e.g., "Add", "Search", "Update")
//   - err: The underlying error to wrap
//
// Returns a MemoryError, or nil if err is nil.
func NewMemoryError(op string, err error) error {
	if err == nil {
		return nil
	}
	return &MemoryError{
		Op:  op,
		Err: err,
	}
}
