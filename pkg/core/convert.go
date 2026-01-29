// Package core provides the main PowerMem client and memory management functionality.
package core

import (
	"time"

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

// memoriesToMaps converts Memory structs to map[string]interface{} for intelligent processing.
//
// This function is used to prepare memories for processing by IntelligentMemoryManager.ProcessSearchResults.
func memoriesToMaps(memories []*Memory) []map[string]interface{} {
	results := make([]map[string]interface{}, len(memories))
	for i, mem := range memories {
		m := map[string]interface{}{
			"id":       mem.ID,
			"user_id":  mem.UserID,
			"agent_id": mem.AgentID,
			"content":  mem.Content,
		}

		// Add optional fields if they exist
		if mem.Embedding != nil {
			m["embedding"] = mem.Embedding
		}
		if mem.SparseEmbedding != nil {
			m["sparse_embedding"] = mem.SparseEmbedding
		}
		if mem.Metadata != nil {
			m["metadata"] = mem.Metadata
		}
		if !mem.CreatedAt.IsZero() {
			m["created_at"] = mem.CreatedAt
		}
		if !mem.UpdatedAt.IsZero() {
			m["updated_at"] = mem.UpdatedAt
		}
		if mem.LastAccessedAt != nil && !mem.LastAccessedAt.IsZero() {
			m["last_accessed_at"] = *mem.LastAccessedAt
		}
		if mem.RetentionStrength != 0 {
			m["retention_strength"] = mem.RetentionStrength
		}
		if mem.Score != 0 {
			m["score"] = mem.Score
		}

		results[i] = m
	}
	return results
}

// mapsToMemories converts map[string]interface{} back to Memory structs.
//
// This function is used to convert processed results from IntelligentMemoryManager back to Memory format.
func mapsToMemories(results []map[string]interface{}) []*Memory {
	memories := make([]*Memory, len(results))
	for i, r := range results {
		mem := &Memory{}

		// Required fields
		if id, ok := r["id"].(int64); ok {
			mem.ID = id
		}
		if userID, ok := r["user_id"].(string); ok {
			mem.UserID = userID
		}
		if agentID, ok := r["agent_id"].(string); ok {
			mem.AgentID = agentID
		}
		if content, ok := r["content"].(string); ok {
			mem.Content = content
		}

		// Optional fields
		if embedding, ok := r["embedding"].([]float64); ok {
			mem.Embedding = embedding
		}
		if sparseEmbedding, ok := r["sparse_embedding"].(map[int]float64); ok {
			mem.SparseEmbedding = sparseEmbedding
		}
		if metadata, ok := r["metadata"].(map[string]interface{}); ok {
			mem.Metadata = make(map[string]interface{})
			for k, v := range metadata {
				mem.Metadata[k] = v
			}
		} else {
			mem.Metadata = make(map[string]interface{})
		}
		if createdAt, ok := r["created_at"].(time.Time); ok {
			mem.CreatedAt = createdAt
		}
		if updatedAt, ok := r["updated_at"].(time.Time); ok {
			mem.UpdatedAt = updatedAt
		}
		if lastAccessedAt, ok := r["last_accessed_at"].(time.Time); ok {
			mem.LastAccessedAt = &lastAccessedAt
		}
		if retentionStrength, ok := r["retention_strength"].(float64); ok {
			mem.RetentionStrength = retentionStrength
		}
		if score, ok := r["score"].(float64); ok {
			mem.Score = score
		}

		// Add intelligent processing scores to metadata
		if relevanceScore, ok := r["relevance_score"].(float64); ok {
			mem.Metadata["relevance_score"] = relevanceScore
		}
		if decayFactor, ok := r["decay_factor"].(float64); ok {
			mem.Metadata["decay_factor"] = decayFactor
		}
		if finalScore, ok := r["final_score"].(float64); ok {
			mem.Metadata["final_score"] = finalScore
			// Use final_score as the new score for ranking
			mem.Score = finalScore
		}

		memories[i] = mem
	}
	return memories
}
