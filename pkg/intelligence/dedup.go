// Package intelligence provides intelligent memory management features including
// deduplication, Ebbinghaus forgetting curve, and importance evaluation.
package intelligence

import (
	"context"
	"math"

	"github.com/oceanbase/powermem-go/pkg/storage"
)

// DedupManager manages memory deduplication by detecting and merging similar memories.
//
// It uses vector similarity search to find duplicate or highly similar memories,
// then merges them to avoid storing redundant information.
//
// Example usage:
//
//	manager := NewDedupManager(store, 0.95)
//	isDup, existingID, err := manager.CheckDuplicate(ctx, embedding, "user_001", "agent_001")
//	if isDup {
//	    merged, err := manager.MergeMemories(ctx, existingID, newContent, newEmbedding)
//	}
type DedupManager struct {
	// store is the vector store for similarity search.
	store storage.VectorStore

	// threshold is the similarity threshold for duplicate detection.
	// Memories with similarity >= threshold are considered duplicates.
	// Typical range: 0.9-0.98 (higher = stricter, fewer duplicates detected)
	threshold float64
}

// NewDedupManager creates a new deduplication manager.
//
// Parameters:
//   - store: Vector store for similarity search
//   - threshold: Similarity threshold (0.0-1.0). If 0, defaults to 0.95.
//
// Returns a new DedupManager with the specified threshold.
func NewDedupManager(store storage.VectorStore, threshold float64) *DedupManager {
	if threshold == 0 {
		threshold = 0.95 // Default threshold
	}
	return &DedupManager{
		store:     store,
		threshold: threshold,
	}
}

// CheckDuplicate checks if a memory is a duplicate of an existing memory.
//
// The method:
//  1. Searches for similar memories using vector similarity
//  2. Compares similarity scores against the threshold
//  3. Returns the first memory that exceeds the threshold
//
// Parameters:
//   - ctx: Context for cancellation
//   - embedding: Embedding vector of the new memory
//   - userID: User identifier for filtering
//   - agentID: Agent identifier for filtering
//
// Returns:
//   - isDuplicate: True if a duplicate is found
//   - existingID: ID of the duplicate memory (if found)
//   - error: Error if search fails
func (m *DedupManager) CheckDuplicate(ctx context.Context, embedding []float64, userID, agentID string) (bool, int64, error) {
	// Search for similar memories
	opts := &storage.SearchOptions{
		UserID:  userID,
		AgentID: agentID,
		Limit:   5, // Only check top 5 most similar
	}

	memories, err := m.store.Search(ctx, embedding, opts)
	if err != nil {
		return false, 0, err
	}

	// Check if any memory exceeds the similarity threshold
	for _, mem := range memories {
		if mem.Score >= m.threshold {
			return true, mem.ID, nil
		}
	}

	return false, 0, nil
}

// MergeMemories merges a new memory with an existing memory.
//
// The merge strategy:
//  1. Combines content by appending new content to existing content
//  2. Averages the embedding vectors
//  3. Normalizes the resulting embedding
//  4. Updates the existing memory with merged data
//
// Note: More sophisticated merge strategies (e.g., using LLM) can be implemented
// by extending this method.
//
// Parameters:
//   - ctx: Context for cancellation
//   - existingID: ID of the existing memory to merge with
//   - newContent: Content of the new memory
//   - newEmbedding: Embedding vector of the new memory
//
// Returns the merged memory, or an error if merge fails.
func (m *DedupManager) MergeMemories(ctx context.Context, existingID int64, newContent string, newEmbedding []float64) (*Memory, error) {
	// Get existing memory
	existing, err := m.store.Get(ctx, existingID)
	if err != nil {
		return nil, err
	}

	// Simple merge strategy: append new content to existing content
	// More sophisticated strategies can use LLM for intelligent merging
	mergedContent := existing.Content + " " + newContent

	// Calculate new embedding (average of both embeddings)
	mergedEmbedding := averageEmbeddings(existing.Embedding, newEmbedding)

	// Update memory
	updated, err := m.store.Update(ctx, existingID, mergedContent, mergedEmbedding)
	if err != nil {
		return nil, err
	}

	// Convert to intelligence.Memory type
	return &Memory{
		ID:                updated.ID,
		UserID:            updated.UserID,
		AgentID:           updated.AgentID,
		Content:           updated.Content,
		Embedding:         updated.Embedding,
		Metadata:          updated.Metadata,
		CreatedAt:         updated.CreatedAt,
		UpdatedAt:         updated.UpdatedAt,
		RetentionStrength: updated.RetentionStrength,
		LastAccessedAt:    updated.LastAccessedAt,
		Score:             updated.Score,
	}, nil
}

// averageEmbeddings calculates the average of two embedding vectors.
//
// If the vectors have different dimensions, returns the first vector unchanged.
// The result is normalized to unit length.
//
// Parameters:
//   - e1: First embedding vector
//   - e2: Second embedding vector
//
// Returns the normalized average of the two vectors.
func averageEmbeddings(e1, e2 []float64) []float64 {
	if len(e1) != len(e2) {
		return e1 // Return first if dimensions don't match
	}

	result := make([]float64, len(e1))
	for i := range e1 {
		result[i] = (e1[i] + e2[i]) / 2.0
	}

	// Normalize
	return normalizeVector(result)
}

// normalizeVector normalizes a vector to unit length (L2 norm).
//
// If the vector has zero norm, returns it unchanged.
//
// Parameters:
//   - v: Vector to normalize
//
// Returns the normalized vector.
func normalizeVector(v []float64) []float64 {
	var sum float64
	for _, val := range v {
		sum += val * val
	}
	norm := math.Sqrt(sum)

	if norm == 0 {
		return v
	}

	result := make([]float64, len(v))
	for i, val := range v {
		result[i] = val / norm
	}

	return result
}

// CosineSimilarity calculates the cosine similarity between two vectors.
//
// Cosine similarity measures the cosine of the angle between two vectors,
// ranging from -1 (opposite) to 1 (identical). Values close to 1 indicate
// high similarity.
//
// The formula is: similarity = (A · B) / (||A|| * ||B||)
//
// Parameters:
//   - a: First vector
//   - b: Second vector
//
// Returns cosine similarity between -1.0 and 1.0, or 0.0 if vectors have
// different dimensions or zero norm.
func CosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
