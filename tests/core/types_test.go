package core_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	powermem "github.com/oceanbase/powermem-go/pkg/core"
)

func TestMemory(t *testing.T) {
	now := time.Now()
	memory := &powermem.Memory{
		ID:                12345,
		UserID:            "user123",
		AgentID:           "agent456",
		Content:           "Test memory content",
		Embedding:         []float64{0.1, 0.2, 0.3},
		Metadata:          map[string]interface{}{"key": "value"},
		CreatedAt:         now,
		UpdatedAt:         now,
		RetentionStrength: 0.8,
		Score:             0.95,
	}

	assert.Equal(t, int64(12345), memory.ID)
	assert.Equal(t, "user123", memory.UserID)
	assert.Equal(t, "agent456", memory.AgentID)
	assert.Equal(t, "Test memory content", memory.Content)
	assert.Len(t, memory.Embedding, 3)
	assert.Equal(t, 0.8, memory.RetentionStrength)
	assert.Equal(t, 0.95, memory.Score)
}

func TestMemoryScope(t *testing.T) {
	assert.Equal(t, powermem.MemoryScope("private"), powermem.ScopePrivate)
	assert.Equal(t, powermem.MemoryScope("agent_group"), powermem.ScopeAgentGroup)
	assert.Equal(t, powermem.MemoryScope("global"), powermem.ScopeGlobal)
}

func TestMetricType(t *testing.T) {
	assert.Equal(t, powermem.MetricType("cosine"), powermem.MetricCosine)
	assert.Equal(t, powermem.MetricType("l2"), powermem.MetricL2)
	assert.Equal(t, powermem.MetricType("ip"), powermem.MetricIP)
}

func TestVectorIndexType(t *testing.T) {
	assert.Equal(t, powermem.VectorIndexType("HNSW"), powermem.IndexTypeHNSW)
	assert.Equal(t, powermem.VectorIndexType("IVF_FLAT"), powermem.IndexTypeIVFFlat)
	assert.Equal(t, powermem.VectorIndexType("IVF_PQ"), powermem.IndexTypeIVFPQ)
}

func TestHNSWParams(t *testing.T) {
	params := powermem.HNSWParams{
		M:              16,
		EfConstruction: 200,
		EfSearch:       50,
	}

	assert.Equal(t, 16, params.M)
	assert.Equal(t, 200, params.EfConstruction)
	assert.Equal(t, 50, params.EfSearch)
}

func TestIVFParams(t *testing.T) {
	params := powermem.IVFParams{
		Nlist:  100,
		Nprobe: 10,
	}

	assert.Equal(t, 100, params.Nlist)
	assert.Equal(t, 10, params.Nprobe)
}

func TestVectorIndexConfig(t *testing.T) {
	config := powermem.VectorIndexConfig{
		IndexName:   "test_index",
		TableName:   "memories",
		VectorField: "embedding",
		IndexType:   powermem.IndexTypeHNSW,
		MetricType:  powermem.MetricCosine,
		HNSWParams: &powermem.HNSWParams{
			M:              16,
			EfConstruction: 200,
			EfSearch:       50,
		},
	}

	assert.Equal(t, "test_index", config.IndexName)
	assert.Equal(t, "memories", config.TableName)
	assert.Equal(t, "embedding", config.VectorField)
	assert.Equal(t, powermem.IndexTypeHNSW, config.IndexType)
	assert.Equal(t, powermem.MetricCosine, config.MetricType)
	assert.NotNil(t, config.HNSWParams)
}
