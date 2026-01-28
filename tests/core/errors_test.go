package core_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	powermem "github.com/oceanbase/powermem-go/pkg/core"
)

func TestErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "ErrNotFound",
			err:      powermem.ErrNotFound,
			expected: "memory not found",
		},
		{
			name:     "ErrInvalidConfig",
			err:      powermem.ErrInvalidConfig,
			expected: "invalid configuration",
		},
		{
			name:     "ErrConnectionFailed",
			err:      powermem.ErrConnectionFailed,
			expected: "connection failed",
		},
		{
			name:     "ErrEmbeddingFailed",
			err:      powermem.ErrEmbeddingFailed,
			expected: "embedding generation failed",
		},
		{
			name:     "ErrDuplicateMemory",
			err:      powermem.ErrDuplicateMemory,
			expected: "duplicate memory detected",
		},
		{
			name:     "ErrLLMOperation",
			err:      powermem.ErrLLMOperation,
			expected: "llm operation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestMemoryError(t *testing.T) {
	originalErr := errors.New("original error")
	memErr := powermem.NewMemoryError("test_operation", originalErr)

	assert.Error(t, memErr)
	assert.Contains(t, memErr.Error(), "test_operation")
	assert.Contains(t, memErr.Error(), "original error")

	// Verify MemoryError structure
	var target *powermem.MemoryError
	if errors.As(memErr, &target) {
		assert.Equal(t, "test_operation", target.Op)
		assert.Equal(t, originalErr, target.Err)
	}
}

func TestMemoryErrorUnwrap(t *testing.T) {
	originalErr := errors.New("original error")
	memErr := powermem.NewMemoryError("test_operation", originalErr)

	unwrapped := errors.Unwrap(memErr)
	assert.Equal(t, originalErr, unwrapped)
}

func TestIsMemoryError(t *testing.T) {
	originalErr := errors.New("original error")
	memErr := powermem.NewMemoryError("test_operation", originalErr)

	var target *powermem.MemoryError
	assert.True(t, errors.As(memErr, &target))
	assert.Equal(t, "test_operation", target.Op)
}
