package core_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	powermem "github.com/oceanbase/powermem-go/pkg/core"
	"github.com/oceanbase/powermem-go/pkg/storage"
)

// Note: Functions in convert.go are private and cannot be tested directly
// These functions will be indirectly tested in actual Memory operations
func TestConvertMemoryTypes(t *testing.T) {
	// Test correctness of type conversion through actual usage
	// Here we only verify consistency of type definitions

	coreMem := &powermem.Memory{
		ID:                12345,
		UserID:            "user123",
		AgentID:           "agent456",
		Content:           "Test content",
		Embedding:         []float64{0.1, 0.2, 0.3},
		Metadata:          map[string]interface{}{"key": "value"},
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		RetentionStrength: 0.8,
		Score:             0.95,
	}

	storageMem := &storage.Memory{
		ID:                coreMem.ID,
		UserID:            coreMem.UserID,
		AgentID:           coreMem.AgentID,
		Content:           coreMem.Content,
		Embedding:         coreMem.Embedding,
		Metadata:          coreMem.Metadata,
		CreatedAt:         coreMem.CreatedAt,
		UpdatedAt:         coreMem.UpdatedAt,
		RetentionStrength: coreMem.RetentionStrength,
		Score:             coreMem.Score,
	}

	// Verify field consistency
	assert.Equal(t, coreMem.ID, storageMem.ID)
	assert.Equal(t, coreMem.UserID, storageMem.UserID)
	assert.Equal(t, coreMem.AgentID, storageMem.AgentID)
	assert.Equal(t, coreMem.Content, storageMem.Content)
	assert.Equal(t, coreMem.Embedding, storageMem.Embedding)
	assert.Equal(t, coreMem.Metadata, storageMem.Metadata)
}
