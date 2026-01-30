// Package core provides the main PowerMem client and memory management functionality.
package core

import "time"

// Memory represents a single memory stored in the system.
//
// A memory contains:
//   - Content: The text content of the memory
//   - Embedding: Vector representation for similarity search
//   - Metadata: Additional structured information
//   - RetentionStrength: Current retention strength (0.0-1.0) for intelligent memory
//
// Example:
//
//	memory := &core.Memory{
//	    ID:      1234567890,
//	    UserID:  "user_001",
//	    Content: "User likes Python programming",
//	    Metadata: map[string]interface{}{
//	        "source": "conversation",
//	    },
//	}
type Memory struct {
	// ID is the unique identifier of the memory.
	ID int64 `json:"id"`

	// UserID identifies the user who owns this memory.
	UserID string `json:"user_id"`

	// AgentID identifies the agent associated with this memory (optional).
	AgentID string `json:"agent_id,omitempty"`

	// Content is the text content of the memory.
	Content string `json:"content"`

	// Embedding is the vector embedding for similarity search.
	// Omitted from JSON to reduce payload size.
	Embedding []float64 `json:"embedding,omitempty"`

	// SparseEmbedding is the sparse vector embedding (for hybrid search).
	// Omitted from JSON to reduce payload size.
	SparseEmbedding map[int]float64 `json:"sparse_embedding,omitempty"`

	// Metadata contains additional structured information about the memory.
	// Can be used for filtering and custom attributes.
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// CreatedAt is when the memory was created.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the memory was last updated.
	UpdatedAt time.Time `json:"updated_at"`

	// RetentionStrength is the current retention strength (0.0-1.0).
	// Used by intelligent memory management (Ebbinghaus curve).
	// 1.0 = perfect retention, 0.0 = completely forgotten.
	RetentionStrength float64 `json:"retention_strength"`

	// LastAccessedAt is when the memory was last accessed (nil if never accessed).
	// Used for retention calculations.
	LastAccessedAt *time.Time `json:"last_accessed_at,omitempty"`

	// Score is the similarity score from search operations (0.0-1.0).
	// Higher scores indicate better matches.
	Score float64 `json:"score,omitempty"`
}

// MemoryScope defines the visibility scope of a memory.
//
// Scopes control which agents can access a memory:
//   - ScopePrivate: Only the creating agent can access
//   - ScopeAgentGroup: All agents in the group can access
//   - ScopeGlobal: All agents can access
type MemoryScope string

const (
	// ScopePrivate makes the memory visible only to the creating agent.
	ScopePrivate MemoryScope = "private"

	// ScopeAgentGroup makes the memory visible to all agents in the group.
	ScopeAgentGroup MemoryScope = "agent_group"

	// ScopeGlobal makes the memory visible to all agents.
	ScopeGlobal MemoryScope = "global"
)

// MetricType defines the distance metric for vector similarity.
//
// Different metrics measure similarity differently:
//   - MetricCosine: Cosine similarity (angle between vectors)
//   - MetricL2: Euclidean distance (L2 norm)
//   - MetricIP: Inner product (dot product)
type MetricType string

const (
	// MetricCosine uses cosine similarity (1 - cosine_distance).
	// Best for normalized vectors. Range: -1 to 1, where 1 is identical.
	MetricCosine MetricType = "cosine"

	// MetricL2 uses Euclidean distance (L2 norm).
	// Lower values indicate higher similarity.
	MetricL2 MetricType = "l2"

	// MetricIP uses inner product (dot product).
	// Higher values indicate higher similarity.
	MetricIP MetricType = "ip"
)

// VectorIndexType defines the type of vector index for efficient similarity search.
type VectorIndexType string

const (
	// IndexTypeHNSW uses Hierarchical Navigable Small World graph.
	// Fast approximate search, good for high-dimensional vectors.
	IndexTypeHNSW VectorIndexType = "HNSW"

	// IndexTypeIVFFlat uses Inverted File Index with flat vectors.
	// Good balance of speed and accuracy.
	IndexTypeIVFFlat VectorIndexType = "IVF_FLAT"

	// IndexTypeIVFPQ uses Inverted File Index with Product Quantization.
	// Most memory-efficient, suitable for very large datasets.
	IndexTypeIVFPQ VectorIndexType = "IVF_PQ"
)

// HNSWParams contains parameters for HNSW index configuration.
//
// HNSW (Hierarchical Navigable Small World) is a graph-based index
// that provides fast approximate nearest neighbor search.
type HNSWParams struct {
	// M is the maximum number of connections for each node.
	// Higher values improve recall but increase memory usage.
	// Typical range: 16-64
	M int

	// EfConstruction is the search depth during index construction.
	// Higher values improve index quality but slow construction.
	// Typical range: 100-500
	EfConstruction int

	// EfSearch is the search depth during queries.
	// Higher values improve recall but slow queries.
	// Typical range: 50-200
	EfSearch int
}

// IVFParams contains parameters for IVF (Inverted File) index configuration.
//
// IVF indexes cluster vectors and search only relevant clusters,
// providing a good balance of speed and accuracy.
type IVFParams struct {
	// Nlist is the number of clusters (centroids).
	// Higher values improve accuracy but increase memory usage.
	// Typical range: 100-10000
	Nlist int

	// Nprobe is the number of clusters to search during queries.
	// Higher values improve recall but slow queries.
	// Typical range: 1-100
	Nprobe int
}

// VectorIndexConfig contains configuration for creating a vector index.
//
// Indexes improve search performance by organizing vectors for efficient
// similarity search.
type VectorIndexConfig struct {
	// IndexName is the name of the index.
	IndexName string

	// TableName is the name of the table/collection to index.
	TableName string

	// VectorField is the name of the vector field to index.
	VectorField string

	// IndexType is the type of index to create.
	IndexType VectorIndexType

	// MetricType is the distance metric to use.
	MetricType MetricType

	// HNSWParams contains HNSW-specific parameters (if IndexType is HNSW).
	HNSWParams *HNSWParams

	// IVFParams contains IVF-specific parameters (if IndexType is IVF_FLAT or IVF_PQ).
	IVFParams *IVFParams
}

// SearchResult contains the results of a search operation.
type SearchResult struct {
	// Memories is the list of matching memories, sorted by relevance.
	Memories []*Memory

	// TotalCount is the total number of matching memories (may be > len(Memories) if paginated).
	TotalCount int
}
