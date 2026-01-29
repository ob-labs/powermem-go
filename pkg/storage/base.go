// Package storage provides interfaces and types for vector storage backends.
//
// It defines the VectorStore interface that all storage implementations must satisfy,
// along with memory types and configuration options.
package storage

import (
	"context"
	"time"
)

// Memory represents a memory stored in the vector store.
//
// This type is defined in the storage package to avoid circular dependencies
// with the core package. It mirrors the core.Memory structure.
type Memory struct {
	// ID is the unique identifier of the memory.
	ID int64

	// UserID identifies the user who owns this memory.
	UserID string

	// AgentID identifies the agent associated with this memory.
	AgentID string

	// Content is the text content of the memory.
	Content string

	// Embedding is the vector embedding for similarity search.
	Embedding []float64

	// SparseEmbedding is the sparse vector embedding (for hybrid search).
	SparseEmbedding map[int]float64

	// Metadata contains additional structured information.
	Metadata map[string]interface{}

	// CreatedAt is when the memory was created.
	CreatedAt time.Time

	// UpdatedAt is when the memory was last updated.
	UpdatedAt time.Time

	// RetentionStrength is the current retention strength (0.0-1.0).
	RetentionStrength float64

	// LastAccessedAt is when the memory was last accessed (nil if never accessed).
	LastAccessedAt *time.Time

	// Score is the similarity score from search operations.
	Score float64
}

// VectorIndexType defines the type of vector index for efficient similarity search.
type VectorIndexType string

const (
	// IndexTypeHNSW uses Hierarchical Navigable Small World graph.
	IndexTypeHNSW VectorIndexType = "HNSW"

	// IndexTypeIVFFlat uses Inverted File Index with flat vectors.
	IndexTypeIVFFlat VectorIndexType = "IVF_FLAT"

	// IndexTypeIVFPQ uses Inverted File Index with Product Quantization.
	IndexTypeIVFPQ VectorIndexType = "IVF_PQ"
)

// MetricType defines the distance metric for vector similarity.
type MetricType string

const (
	// MetricCosine uses cosine similarity.
	MetricCosine MetricType = "cosine"

	// MetricL2 uses Euclidean distance (L2 norm).
	MetricL2 MetricType = "l2"

	// MetricIP uses inner product (dot product).
	MetricIP MetricType = "ip"
)

// HNSWParams contains parameters for HNSW index configuration.
type HNSWParams struct {
	// M is the maximum number of connections for each node.
	M int

	// EfConstruction is the search depth during index construction.
	EfConstruction int

	// EfSearch is the search depth during queries.
	EfSearch int
}

// IVFParams contains parameters for IVF (Inverted File) index configuration.
type IVFParams struct {
	// Nlist is the number of clusters (centroids).
	Nlist int

	// Nprobe is the number of clusters to search during queries.
	Nprobe int
}

// VectorIndexConfig contains configuration for creating a vector index.
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

// VectorStore defines the interface for vector storage backends.
//
// All storage implementations (SQLite, PostgreSQL, OceanBase) must implement this interface.
type VectorStore interface {
	// Insert inserts a memory into the store.
	Insert(ctx context.Context, memory *Memory) error

	// Search performs vector similarity search.
	//
	// Parameters:
	//   - ctx: Context for cancellation
	//   - embedding: Query embedding vector
	//   - opts: Search options (UserID, AgentID, Limit, MinScore, Filters)
	//
	// Returns matching memories sorted by similarity (highest first).
	Search(ctx context.Context, embedding []float64, opts *SearchOptions) ([]*Memory, error)

	// Get retrieves a memory by ID with optional access control.
	//
	// If opts.UserID or opts.AgentID is specified, the memory will only be returned
	// if it matches the specified user/agent (multi-tenant isolation).
	Get(ctx context.Context, id int64, opts *GetOptions) (*Memory, error)

	// Update updates a memory's content and embedding with optional access control.
	//
	// If opts.UserID or opts.AgentID is specified, the update will only succeed
	// if the memory belongs to the specified user/agent (access control).
	Update(ctx context.Context, id int64, content string, embedding []float64, opts *UpdateOptions) (*Memory, error)

	// Delete deletes a memory by ID with optional access control.
	//
	// If opts.UserID or opts.AgentID is specified, the delete will only succeed
	// if the memory belongs to the specified user/agent (access control).
	Delete(ctx context.Context, id int64, opts *DeleteOptions) error

	// GetAll retrieves all memories with optional filtering and pagination.
	GetAll(ctx context.Context, opts *GetAllOptions) ([]*Memory, error)

	// DeleteAll deletes all memories matching the given filters.
	DeleteAll(ctx context.Context, opts *DeleteAllOptions) error

	// Close closes the store and releases resources.
	Close() error

	// CreateIndex creates a vector index for improved search performance.
	CreateIndex(ctx context.Context, config *VectorIndexConfig) error
}

// SearchOptions contains options for search operations.
type SearchOptions struct {
	// UserID filters results to a specific user.
	UserID string

	// AgentID filters results to a specific agent.
	AgentID string

	// Limit sets the maximum number of results to return.
	Limit int

	// MinScore sets the minimum similarity score for results.
	// This is the same as Threshold for backward compatibility.
	MinScore float64

	// Threshold sets the minimum similarity score for results.
	// This is an alias for MinScore, following Python SDK naming.
	// If both are set, the higher value is used.
	Threshold float64

	// Query is the original query text for hybrid search.
	// When provided, implementations can use it for:
	//   - Full-text search (keyword matching)
	//   - Sparse embedding generation
	//   - Hybrid retrieval (vector + text + sparse)
	// If empty, only vector search is performed.
	Query string

	// SparseEmbedding is the sparse embedding for hybrid search.
	// When provided together with dense embedding, implementations
	// can perform hybrid retrieval combining both representations.
	SparseEmbedding map[int]float64

	// Filters provides additional metadata filters.
	Filters map[string]interface{}
}

// GetOptions contains options for get operations with access control.
type GetOptions struct {
	// UserID restricts access to memories belonging to this user.
	// If specified, Get will return an error if the memory's UserID doesn't match.
	// This enables multi-tenant isolation.
	UserID string

	// AgentID restricts access to memories belonging to this agent.
	// If specified, Get will return an error if the memory's AgentID doesn't match.
	// This enables agent-level access control.
	AgentID string
}

// UpdateOptions contains options for update operations with access control.
type UpdateOptions struct {
	// UserID restricts updates to memories belonging to this user.
	// If specified, Update will fail if the memory's UserID doesn't match.
	// This prevents unauthorized modifications across tenants.
	UserID string

	// AgentID restricts updates to memories belonging to this agent.
	// If specified, Update will fail if the memory's AgentID doesn't match.
	// This prevents unauthorized modifications across agents.
	AgentID string
}

// DeleteOptions contains options for delete operations with access control.
type DeleteOptions struct {
	// UserID restricts deletions to memories belonging to this user.
	// If specified, Delete will fail if the memory's UserID doesn't match.
	// This prevents unauthorized deletions across tenants.
	UserID string

	// AgentID restricts deletions to memories belonging to this agent.
	// If specified, Delete will fail if the memory's AgentID doesn't match.
	// This prevents unauthorized deletions across agents.
	AgentID string
}

// GetAllOptions contains options for GetAll operations.
type GetAllOptions struct {
	// UserID filters results to a specific user.
	UserID string

	// AgentID filters results to a specific agent.
	AgentID string

	// Limit sets the maximum number of results to return.
	Limit int

	// Offset sets the number of results to skip (for pagination).
	Offset int
}

// DeleteAllOptions contains options for DeleteAll operations.
type DeleteAllOptions struct {
	// UserID filters deletions to a specific user.
	UserID string

	// AgentID filters deletions to a specific agent.
	AgentID string
}
