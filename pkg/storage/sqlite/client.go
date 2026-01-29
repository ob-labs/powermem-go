// Package sqlite provides SQLite implementation for vector storage.
//
// SQLite is a lightweight, file-based database suitable for local development
// and small-scale applications. Vectors are stored as JSON strings in TEXT fields,
// and similarity search uses in-memory cosine similarity calculation.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/oceanbase/powermem-go/pkg/storage"
)

// Client implements VectorStore using SQLite as the backend.
type Client struct {
	// db is the SQLite database connection.
	db *sql.DB

	// collectionName is the name of the table storing memories.
	collectionName string

	// dimensions is the dimension of embedding vectors.
	dimensions int
}

// Config contains configuration for creating a SQLite VectorStore.
type Config struct {
	// DBPath is the path to the SQLite database file.
	DBPath string

	// CollectionName is the name of the table to use.
	CollectionName string

	// EmbeddingModelDims is the dimension of embedding vectors.
	EmbeddingModelDims int
}

// NewClient creates a new SQLite VectorStore client.
//
// Parameters:
//   - cfg: Configuration containing database path, table name, and embedding dimensions
//
// Returns:
//   - *Client: The SQLite client instance
//   - error: Error if database connection or table creation fails
func NewClient(cfg *Config) (*Client, error) {
	// Create parent directory if it doesn't exist
	dbDir := filepath.Dir(cfg.DBPath)
	if dbDir != "" && dbDir != "." {
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			return nil, fmt.Errorf("NewSQLiteClient: failed to create directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite3", cfg.DBPath+"?_foreign_keys=1&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("NewSQLiteClient: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("NewSQLiteClient: %w", err)
	}

	client := &Client{
		db:             db,
		collectionName: cfg.CollectionName,
		dimensions:     cfg.EmbeddingModelDims,
	}

	// Initialize table structure
	if err := client.initTables(context.Background()); err != nil {
		return nil, err
	}

	return client, nil
}

// initTables initializes the database table structure.
//
// SQLite stores vectors as JSON strings in TEXT fields.
func (c *Client) initTables(ctx context.Context) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id INTEGER PRIMARY KEY,
			user_id TEXT NOT NULL,
			agent_id TEXT,
			content TEXT NOT NULL,
			embedding TEXT NOT NULL,
			metadata TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			retention_strength REAL DEFAULT 1.0,
			last_accessed_at DATETIME
		)
	`, c.collectionName)

	_, err := c.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("initTables: %w", err)
	}

	// Create index
	indexQuery := fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS idx_%s_user_agent ON %s(user_id, agent_id)
	`, c.collectionName, c.collectionName)
	_, err = c.db.ExecContext(ctx, indexQuery)
	if err != nil {
		return fmt.Errorf("initTables: %w", err)
	}

	return nil
}

// Insert inserts a memory into the SQLite database.
//
// Vectors are stored as JSON strings in TEXT fields.
func (c *Client) Insert(ctx context.Context, memory *storage.Memory) error {
	query := fmt.Sprintf(`
		INSERT INTO %s 
		(id, user_id, agent_id, content, embedding, metadata, created_at, retention_strength)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, c.collectionName)

	embeddingJSON, err := json.Marshal(memory.Embedding)
	if err != nil {
		return fmt.Errorf("Insert: %w", err)
	}

	metadataJSON, err := json.Marshal(memory.Metadata)
	if err != nil {
		return fmt.Errorf("Insert: %w", err)
	}

	_, err = c.db.ExecContext(ctx, query,
		memory.ID,
		memory.UserID,
		memory.AgentID,
		memory.Content,
		string(embeddingJSON),
		string(metadataJSON),
		time.Now(),
		memory.RetentionStrength,
	)

	if err != nil {
		return fmt.Errorf("Insert: %w", err)
	}

	return nil
}

// Search performs vector similarity search using cosine similarity.
//
// SQLite does not have native vector operations, so similarity is calculated
// in memory after loading all matching records.
//
// The method supports hybrid search parameters for future enhancement:
//   - opts.Query: Original query text (reserved for full-text search)
//   - opts.SparseEmbedding: Sparse vector (reserved for sparse + dense hybrid)
//   - opts.Threshold: Minimum similarity score (alias for MinScore)
//
// Currently, only vector similarity search is implemented.
// Hybrid search (vector + full-text + sparse) will be added in future versions.
func (c *Client) Search(ctx context.Context, embedding []float64, opts *storage.SearchOptions) ([]*storage.Memory, error) {
	// Use Threshold if MinScore is not set (Python SDK compatibility)
	minScore := opts.MinScore
	if minScore == 0 && opts.Threshold > 0 {
		minScore = opts.Threshold
	}

	whereClause, args := buildWhereClause(opts.UserID, opts.AgentID, opts.Filters)

	// SQLite requires manual cosine similarity calculation
	query := fmt.Sprintf(`
		SELECT 
			id, user_id, agent_id, content, embedding, metadata,
			created_at, updated_at, retention_strength, last_accessed_at
		FROM %s
		%s
		ORDER BY id
	`, c.collectionName, whereClause)

	// TODO: Future enhancement - add full-text search support using opts.Query
	// This would enable hybrid retrieval combining vector similarity and keyword matching

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("Search: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var memories []*storage.Memory
	for rows.Next() {
		memory, err := c.scanMemory(rows)
		if err != nil {
			return nil, err
		}

		// Calculate cosine similarity
		score := cosineSimilarity(embedding, memory.Embedding)
		memory.Score = score

		// Apply threshold filter
		if score >= minScore {
			memories = append(memories, memory)
		}

		// TODO: Future enhancement - combine with sparse embedding similarity
		// if opts.SparseEmbedding != nil {
		//     sparseScore := calculateSparseSimilarity(opts.SparseEmbedding, memory.SparseEmbedding)
		//     combinedScore := (score + sparseScore) / 2.0
		//     memory.Score = combinedScore
		// }
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Sort by score and limit results
	memories = sortByScore(memories, opts.Limit)

	return memories, nil
}

// Get retrieves a memory by ID with optional access control.
func (c *Client) Get(ctx context.Context, id int64, opts *storage.GetOptions) (*storage.Memory, error) {
	if opts == nil {
		opts = &storage.GetOptions{}
	}

	// Build WHERE clause with access control
	whereClause := "WHERE id = ?"
	args := []interface{}{id}

	if opts.UserID != "" {
		whereClause += " AND user_id = ?"
		args = append(args, opts.UserID)
	}
	if opts.AgentID != "" {
		whereClause += " AND agent_id = ?"
		args = append(args, opts.AgentID)
	}

	query := fmt.Sprintf(`
		SELECT id, user_id, agent_id, content, embedding, metadata,
		       created_at, updated_at, retention_strength, last_accessed_at
		FROM %s
		%s
	`, c.collectionName, whereClause)

	row := c.db.QueryRowContext(ctx, query, args...)

	memory, err := c.scanMemory(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Get: not found or access denied")
	}
	if err != nil {
		return nil, fmt.Errorf("Get: %w", err)
	}

	return memory, nil
}

// Update updates a memory with optional access control.
func (c *Client) Update(ctx context.Context, id int64, content string, embedding []float64, opts *storage.UpdateOptions) (*storage.Memory, error) {
	if opts == nil {
		opts = &storage.UpdateOptions{}
	}

	embeddingJSON, err := json.Marshal(embedding)
	if err != nil {
		return nil, fmt.Errorf("Update: %w", err)
	}

	// Build WHERE clause with access control
	whereClause := "WHERE id = ?"
	args := []interface{}{content, string(embeddingJSON), time.Now(), id}

	if opts.UserID != "" {
		whereClause += " AND user_id = ?"
		args = append(args, opts.UserID)
	}
	if opts.AgentID != "" {
		whereClause += " AND agent_id = ?"
		args = append(args, opts.AgentID)
	}

	query := fmt.Sprintf(`
		UPDATE %s
		SET content = ?, embedding = ?, updated_at = ?
		%s
	`, c.collectionName, whereClause)

	result, err := c.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("Update: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("Update: %w", err)
	}

	if rowsAffected == 0 {
		return nil, fmt.Errorf("Update: not found or access denied")
	}

	return c.Get(ctx, id, &storage.GetOptions{
		UserID:  opts.UserID,
		AgentID: opts.AgentID,
	})
}

// Delete deletes a memory by ID with optional access control.
func (c *Client) Delete(ctx context.Context, id int64, opts *storage.DeleteOptions) error {
	if opts == nil {
		opts = &storage.DeleteOptions{}
	}

	// Build WHERE clause with access control
	whereClause := "WHERE id = ?"
	args := []interface{}{id}

	if opts.UserID != "" {
		whereClause += " AND user_id = ?"
		args = append(args, opts.UserID)
	}
	if opts.AgentID != "" {
		whereClause += " AND agent_id = ?"
		args = append(args, opts.AgentID)
	}

	query := fmt.Sprintf("DELETE FROM %s %s", c.collectionName, whereClause)

	result, err := c.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("Delete: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("Delete: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("Delete: not found or access denied")
	}

	return nil
}

// GetAll retrieves all memories with optional filtering and pagination.
func (c *Client) GetAll(ctx context.Context, opts *storage.GetAllOptions) ([]*storage.Memory, error) {
	whereClause, args := buildWhereClause(opts.UserID, opts.AgentID, nil)

	query := fmt.Sprintf(`
		SELECT id, user_id, agent_id, content, embedding, metadata,
		       created_at, updated_at, retention_strength, last_accessed_at
		FROM %s
		%s
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, c.collectionName, whereClause)

	args = append(args, opts.Limit, opts.Offset)

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("GetAll: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var memories []*storage.Memory
	for rows.Next() {
		memory, err := c.scanMemory(rows)
		if err != nil {
			return nil, err
		}
		memories = append(memories, memory)
	}

	return memories, nil
}

// DeleteAll deletes all memories matching the given filters.
func (c *Client) DeleteAll(ctx context.Context, opts *storage.DeleteAllOptions) error {
	whereClause, args := buildWhereClause(opts.UserID, opts.AgentID, nil)

	query := fmt.Sprintf("DELETE FROM %s %s", c.collectionName, whereClause)

	_, err := c.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("DeleteAll: %w", err)
	}

	return nil
}

// Close closes the database connection.
func (c *Client) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// CreateIndex creates a vector index.
//
// SQLite does not support vector indexes, so this method is a no-op.
// Similarity search uses full table scan with in-memory calculation.
func (c *Client) CreateIndex(ctx context.Context, config *storage.VectorIndexConfig) error {
	// SQLite does not support vector indexes, uses full table scan
	return nil
}

// scanMemory scans a memory from a database row or rows.
func (c *Client) scanMemory(scanner interface{}) (*storage.Memory, error) {
	var memory storage.Memory
	var embeddingStr string
	var metadataStr string
	var lastAccessedAt sql.NullTime

	var err error
	switch s := scanner.(type) {
	case *sql.Row:
		err = s.Scan(
			&memory.ID,
			&memory.UserID,
			&memory.AgentID,
			&memory.Content,
			&embeddingStr,
			&metadataStr,
			&memory.CreatedAt,
			&memory.UpdatedAt,
			&memory.RetentionStrength,
			&lastAccessedAt,
		)
	case *sql.Rows:
		err = s.Scan(
			&memory.ID,
			&memory.UserID,
			&memory.AgentID,
			&memory.Content,
			&embeddingStr,
			&metadataStr,
			&memory.CreatedAt,
			&memory.UpdatedAt,
			&memory.RetentionStrength,
			&lastAccessedAt,
		)
	default:
		return nil, fmt.Errorf("unsupported scanner type")
	}

	if err != nil {
		return nil, err
	}

	// Parse embedding
	if err := json.Unmarshal([]byte(embeddingStr), &memory.Embedding); err != nil {
		return nil, fmt.Errorf("parse embedding: %w", err)
	}

	// Parse metadata
	if metadataStr != "" {
		if err := json.Unmarshal([]byte(metadataStr), &memory.Metadata); err != nil {
			return nil, fmt.Errorf("parse metadata: %w", err)
		}
	}

	// Handle last_accessed_at
	if lastAccessedAt.Valid {
		memory.LastAccessedAt = &lastAccessedAt.Time
	}

	return &memory, nil
}

// cosineSimilarity calculates the cosine similarity between two vectors.
func cosineSimilarity(a, b []float64) float64 {
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

// sortByScore sorts memories by score (descending) and limits the number of results.
//
// Uses a simple bubble sort which is sufficient for small datasets.
func sortByScore(memories []*storage.Memory, limit int) []*storage.Memory {
	n := len(memories)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if memories[j].Score < memories[j+1].Score {
				memories[j], memories[j+1] = memories[j+1], memories[j]
			}
		}
	}

	if limit > 0 && len(memories) > limit {
		return memories[:limit]
	}

	return memories
}
