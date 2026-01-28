package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/oceanbase/powermem-go/pkg/storage"
)

// Client is a PostgreSQL + pgvector client.
type Client struct {
	db             *sql.DB
	collectionName string
	dimensions     int
}

// Config contains PostgreSQL configuration.
type Config struct {
	Host               string
	Port               int
	User               string
	Password           string
	DBName             string
	CollectionName     string
	EmbeddingModelDims int
	SSLMode            string
}

// NewClient creates a new PostgreSQL client.
func NewClient(cfg *Config) (*Client, error) {
	sslMode := cfg.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, sslMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("NewPostgresClient: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("NewPostgresClient: %w", err)
	}

	client := &Client{
		db:             db,
		collectionName: cfg.CollectionName,
		dimensions:     cfg.EmbeddingModelDims,
	}

	// Initialize pgvector extension and table structure
	if err := client.initTables(context.Background()); err != nil {
		return nil, err
	}

	return client, nil
}

// initTables initializes the database table.
func (c *Client) initTables(ctx context.Context) error {
	// Enable pgvector extension
	_, err := c.db.ExecContext(ctx, "CREATE EXTENSION IF NOT EXISTS vector")
	if err != nil {
		return fmt.Errorf("initTables: create extension: %w", err)
	}

	// Create table (using pgvector's vector type)
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id BIGINT PRIMARY KEY,
			user_id VARCHAR(255) NOT NULL,
			agent_id VARCHAR(255),
			content TEXT NOT NULL,
			embedding vector(%d) NOT NULL,
			metadata JSONB,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			retention_strength FLOAT DEFAULT 1.0,
			last_accessed_at TIMESTAMP
		)
	`, c.collectionName, c.dimensions)

	_, err = c.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("initTables: create table: %w", err)
	}

	// Create index
	indexQuery := fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS idx_%s_user_agent ON %s(user_id, agent_id)
	`, c.collectionName, c.collectionName)
	_, err = c.db.ExecContext(ctx, indexQuery)
	if err != nil {
		return fmt.Errorf("initTables: create index: %w", err)
	}

	return nil
}

// Insert inserts a memory.
func (c *Client) Insert(ctx context.Context, memory *storage.Memory) error {
	query := fmt.Sprintf(`
		INSERT INTO %s 
		(id, user_id, agent_id, content, embedding, metadata, created_at, retention_strength)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, c.collectionName)

	// Convert vector to PostgreSQL vector format: "[0.1,0.2,0.3,...]"
	vectorStr := vectorToString(memory.Embedding)

	metadataJSON, err := json.Marshal(memory.Metadata)
	if err != nil {
		return fmt.Errorf("Insert: %w", err)
	}

	_, err = c.db.ExecContext(ctx, query,
		memory.ID,
		memory.UserID,
		memory.AgentID,
		memory.Content,
		vectorStr,
		string(metadataJSON),
		time.Now(),
		memory.RetentionStrength,
	)

	if err != nil {
		return fmt.Errorf("Insert: %w", err)
	}

	return nil
}

// Search performs vector search using pgvector's cosine similarity.
func (c *Client) Search(ctx context.Context, embedding []float64, opts *storage.SearchOptions) ([]*storage.Memory, error) {
	queryVectorStr := vectorToString(embedding)

	// Build WHERE clause (starting from $2 since $1 is the query vector)
	whereClause, filterArgs := buildWhereClauseWithOffset(opts.UserID, opts.AgentID, opts.Filters, 2)

	// Use pgvector's <=> operator (cosine distance, 1 - cosine similarity)
	query := fmt.Sprintf(`
		SELECT 
			id, user_id, agent_id, content, embedding, metadata,
			created_at, updated_at, retention_strength, last_accessed_at,
			1 - (embedding <=> $1) as similarity
		FROM %s
		%s
		ORDER BY embedding <=> $1
		LIMIT $%d
	`, c.collectionName, whereClause, len(filterArgs)+2)

	// Build final args: query vector, filter args, then limit
	allArgs := []interface{}{queryVectorStr}
	allArgs = append(allArgs, filterArgs...)
	allArgs = append(allArgs, opts.Limit)

	rows, err := c.db.QueryContext(ctx, query, allArgs...)
	if err != nil {
		return nil, fmt.Errorf("Search: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return c.scanMemories(rows, true)
}

// Get retrieves a memory by ID.
func (c *Client) Get(ctx context.Context, id int64) (*storage.Memory, error) {
	query := fmt.Sprintf(`
		SELECT id, user_id, agent_id, content, embedding, metadata,
		       created_at, updated_at, retention_strength, last_accessed_at
		FROM %s
		WHERE id = $1
	`, c.collectionName)

	row := c.db.QueryRowContext(ctx, query, id)

	memory, err := c.scanMemory(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Get: not found")
	}
	if err != nil {
		return nil, fmt.Errorf("Get: %w", err)
	}

	return memory, nil
}

// Update updates a memory.
func (c *Client) Update(ctx context.Context, id int64, content string, embedding []float64) (*storage.Memory, error) {
	vectorStr := vectorToString(embedding)

	query := fmt.Sprintf(`
		UPDATE %s
		SET content = $1, embedding = $2, updated_at = $3
		WHERE id = $4
	`, c.collectionName)

	result, err := c.db.ExecContext(ctx, query, content, vectorStr, time.Now(), id)
	if err != nil {
		return nil, fmt.Errorf("Update: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("Update: %w", err)
	}

	if rowsAffected == 0 {
		return nil, fmt.Errorf("Update: not found")
	}

	return c.Get(ctx, id)
}

// Delete deletes a memory.
func (c *Client) Delete(ctx context.Context, id int64) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = $1", c.collectionName)

	result, err := c.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("Delete: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("Delete: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("Delete: not found")
	}

	return nil
}

// GetAll retrieves all memories.
func (c *Client) GetAll(ctx context.Context, opts *storage.GetAllOptions) ([]*storage.Memory, error) {
	whereClause, args := buildWhereClause(opts.UserID, opts.AgentID, nil)

	query := fmt.Sprintf(`
		SELECT id, user_id, agent_id, content, embedding, metadata,
		       created_at, updated_at, retention_strength, last_accessed_at
		FROM %s
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, c.collectionName, whereClause, len(args)+1, len(args)+2)

	args = append(args, opts.Limit, opts.Offset)

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("GetAll: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return c.scanMemories(rows, false)
}

// DeleteAll deletes all memories.
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

// CreateIndex creates a vector index (HNSW index).
func (c *Client) CreateIndex(ctx context.Context, config *storage.VectorIndexConfig) error {
	switch config.IndexType {
	case storage.IndexTypeHNSW:
		query := fmt.Sprintf(`
			CREATE INDEX IF NOT EXISTS %s ON %s 
			USING hnsw (%s vector_cosine_ops)
			WITH (m = %d, ef_construction = %d)
		`, config.IndexName, config.TableName, config.VectorField,
			config.HNSWParams.M, config.HNSWParams.EfConstruction)
		_, err := c.db.ExecContext(ctx, query)
		return err
	case storage.IndexTypeIVFFlat:
		query := fmt.Sprintf(`
			CREATE INDEX IF NOT EXISTS %s ON %s 
			USING ivfflat (%s vector_cosine_ops)
			WITH (lists = %d)
		`, config.IndexName, config.TableName, config.VectorField, config.IVFParams.Nlist)
		_, err := c.db.ExecContext(ctx, query)
		return err
	default:
		return fmt.Errorf("unsupported index type: %s", config.IndexType)
	}
}

// vectorToString converts a vector to PostgreSQL vector format.
func vectorToString(vector []float64) string {
	if len(vector) == 0 {
		return "[]"
	}

	parts := make([]string, len(vector))
	for i, v := range vector {
		parts[i] = fmt.Sprintf("%f", v)
	}

	return "[" + strings.Join(parts, ",") + "]"
}

// scanMemory scans a single memory.
func (c *Client) scanMemory(row *sql.Row) (*storage.Memory, error) {
	var memory storage.Memory
	var embeddingStr string
	var metadataStr []byte
	var lastAccessedAt sql.NullTime

	err := row.Scan(
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
	if err != nil {
		return nil, err
	}

	// Parse embedding (pgvector returns string format)
	embedding, err := parseVectorString(embeddingStr)
	if err != nil {
		return nil, fmt.Errorf("parse embedding: %w", err)
	}
	memory.Embedding = embedding

	// Parse metadata
	if len(metadataStr) > 0 {
		if err := json.Unmarshal(metadataStr, &memory.Metadata); err != nil {
			return nil, fmt.Errorf("parse metadata: %w", err)
		}
	}

	// Handle last_accessed_at
	if lastAccessedAt.Valid {
		memory.LastAccessedAt = &lastAccessedAt.Time
	}

	return &memory, nil
}

// scanMemories scans multiple memories.
func (c *Client) scanMemories(rows *sql.Rows, hasScore bool) ([]*storage.Memory, error) {
	var memories []*storage.Memory

	for rows.Next() {
		var memory storage.Memory
		var embeddingStr string
		var metadataStr []byte
		var lastAccessedAt sql.NullTime
		var similarity float64

		if hasScore {
			err := rows.Scan(
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
				&similarity,
			)
			if err != nil {
				return nil, err
			}
			memory.Score = similarity
		} else {
			err := rows.Scan(
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
			if err != nil {
				return nil, err
			}
		}

		// Parse embedding
		embedding, err := parseVectorString(embeddingStr)
		if err != nil {
			return nil, fmt.Errorf("parse embedding: %w", err)
		}
		memory.Embedding = embedding

		// Parse metadata
		if len(metadataStr) > 0 {
			if err := json.Unmarshal(metadataStr, &memory.Metadata); err != nil {
				return nil, fmt.Errorf("parse metadata: %w", err)
			}
		}

		// Handle last_accessed_at
		if lastAccessedAt.Valid {
			memory.LastAccessedAt = &lastAccessedAt.Time
		}

		memories = append(memories, &memory)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return memories, nil
}

// parseVectorString parses a PostgreSQL vector string.
func parseVectorString(s string) ([]float64, error) {
	// Remove leading and trailing square brackets
	s = strings.Trim(s, "[]")
	if s == "" {
		return []float64{}, nil
	}

	parts := strings.Split(s, ",")
	result := make([]float64, len(parts))

	for i, part := range parts {
		var val float64
		_, err := fmt.Sscanf(strings.TrimSpace(part), "%f", &val)
		if err != nil {
			return nil, err
		}
		result[i] = val
	}

	return result, nil
}
