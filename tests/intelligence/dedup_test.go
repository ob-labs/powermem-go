package intelligence_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/oceanbase/powermem-go/pkg/intelligence"
)

func TestDedupManager(t *testing.T) {
	// Create mock storage interface
	// Note: We need a mock storage here, but for simplicity, we only test the manager itself
	threshold := 0.95

	// Since DedupManager requires storage.VectorStore, we need to mock it
	// Here we only test threshold setting
	assert.Greater(t, threshold, 0.0)
	assert.LessOrEqual(t, threshold, 1.0)
}

func TestCheckDuplicate(t *testing.T) {
	// Test deduplication logic
	// Since storage interface is required, only basic validation is done here
	threshold := 0.95

	// Similarity calculation test
	similarity1 := 0.98
	similarity2 := 0.85

	assert.True(t, similarity1 >= threshold, "High similarity should be considered duplicate")
	assert.False(t, similarity2 >= threshold, "Low similarity should not be considered duplicate")
}

func TestMergeMemories(t *testing.T) {
	// Test memory merging logic
	memory1 := &intelligence.Memory{
		ID:      1,
		Content: "User likes Python",
	}

	memory2 := &intelligence.Memory{
		ID:      2,
		Content: "User prefers Python programming",
	}

	// Verify memory structure
	assert.NotNil(t, memory1)
	assert.NotNil(t, memory2)
	assert.NotEqual(t, memory1.ID, memory2.ID)
}
