// Package core provides the main PowerMem client and memory management functionality.
package core

import (
	"github.com/oceanbase/powermem-go/pkg/intelligence"
	"github.com/oceanbase/powermem-go/pkg/storage"
)

// toStorageMemory converts a core.Memory to storage.Memory.
//
// This function is used internally to convert between package types
// to avoid circular dependencies.
func toStorageMemory(m *Memory) *storage.Memory {
	return &storage.Memory{
		ID:                m.ID,
		UserID:            m.UserID,
		AgentID:           m.AgentID,
		Content:           m.Content,
		Embedding:         m.Embedding,
		SparseEmbedding:   m.SparseEmbedding,
		Metadata:          m.Metadata,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
		RetentionStrength: m.RetentionStrength,
		LastAccessedAt:    m.LastAccessedAt,
		Score:             m.Score,
	}
}

// fromStorageMemory converts a storage.Memory to core.Memory.
//
// This function is used internally to convert between package types
// to avoid circular dependencies.
func fromStorageMemory(m *storage.Memory) *Memory {
	return &Memory{
		ID:                m.ID,
		UserID:            m.UserID,
		AgentID:           m.AgentID,
		Content:           m.Content,
		Embedding:         m.Embedding,
		SparseEmbedding:   m.SparseEmbedding,
		Metadata:          m.Metadata,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
		RetentionStrength: m.RetentionStrength,
		LastAccessedAt:    m.LastAccessedAt,
		Score:             m.Score,
	}
}

// fromStorageMemories converts a slice of storage.Memory to a slice of core.Memory.
//
// This function is used internally for batch conversion between package types.
func fromStorageMemories(memories []*storage.Memory) []*Memory {
	result := make([]*Memory, len(memories))
	for i, m := range memories {
		result[i] = fromStorageMemory(m)
	}
	return result
}

// fromIntelligenceMemory converts an intelligence.Memory to core.Memory.
//
// This function is used internally to convert between package types
// to avoid circular dependencies.
func fromIntelligenceMemory(m *intelligence.Memory) *Memory {
	return &Memory{
		ID:                m.ID,
		UserID:            m.UserID,
		AgentID:           m.AgentID,
		Content:           m.Content,
		Embedding:         m.Embedding,
		Metadata:          m.Metadata,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
		RetentionStrength: m.RetentionStrength,
		LastAccessedAt:    m.LastAccessedAt,
		Score:             m.Score,
	}
}
