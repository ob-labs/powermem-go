package oceanbase

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/oceanbase/powermem-go/pkg/storage"
)

// Client is an OceanBase client.
type Client struct {
	db             *sql.DB
	config         *Config
	collectionName string
}

// Config contains OceanBase configuration.
type Config struct {
	Host               string
	Port               int
	User               string
	Password           string
	DBName             string
	CollectionName     string
	EmbeddingModelDims int
}

// NewClient creates a new OceanBase client.
func NewClient(cfg *Config) (*Client, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("NewOceanBaseClient: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("NewOceanBaseClient: %w", err)
	}

	client := &Client{
		db:             db,
		config:         cfg,
		collectionName: cfg.CollectionName,
	}

	// Initialize table structure
	if err := client.initTables(context.Background()); err != nil {
		return nil, err
	}

	return client, nil
}

// initTables initializes the database table.
// Compatible with Python SDK table structure
func (c *Client) initTables(ctx context.Context) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id BIGINT PRIMARY KEY,
			embedding VECTOR(%d),
			document LONGTEXT,
			metadata JSON,
			user_id VARCHAR(128),
			agent_id VARCHAR(128),
			run_id VARCHAR(128),
			actor_id VARCHAR(128),
			hash VARCHAR(32),
			created_at VARCHAR(128),
			updated_at VARCHAR(128),
			category VARCHAR(64),
			fulltext_content LONGTEXT,
			INDEX idx_user_agent (user_id, agent_id)
		)
	`, c.collectionName, c.config.EmbeddingModelDims)

	_, err := c.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("initTables: %w", err)
	}

	return nil
}

// Insert inserts a memory.
// Compatible with Python SDK: uses 'document' field instead of 'content'
func (c *Client) Insert(ctx context.Context, memory *storage.Memory) error {
	query := fmt.Sprintf(`
		INSERT INTO %s 
		(id, user_id, agent_id, document, embedding, metadata, created_at, updated_at, hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, c.collectionName)

	vectorStr := vectorToString(memory.Embedding)

	// Add retention_strength to metadata if it exists
	metadataMap := memory.Metadata
	if metadataMap == nil {
		metadataMap = make(map[string]interface{})
	}
	if memory.RetentionStrength > 0 {
		metadataMap["retention_strength"] = memory.RetentionStrength
	}

	metadataJSON, err := json.Marshal(metadataMap)
	if err != nil {
		return fmt.Errorf("Insert: %w", err)
	}

	// Generate hash for content (compatible with Python SDK)
	hash := generateHash(memory.Content)

	now := time.Now().Format(time.RFC3339)

	_, err = c.db.ExecContext(ctx, query,
		memory.ID,
		memory.UserID,
		memory.AgentID,
		memory.Content,
		vectorStr,
		metadataJSON,
		now,
		now,
		hash,
	)

	if err != nil {
		return fmt.Errorf("Insert: %w", err)
	}

	return nil
}

// Search performs vector search.
//
// Compatible with Python SDK: uses 'document' field for content storage.
//
// The method supports hybrid search parameters for future enhancement:
//   - opts.Query: Original query text (reserved for full-text search)
//   - opts.SparseEmbedding: Sparse vector (reserved for hybrid retrieval)
//   - opts.Threshold: Minimum similarity score (alias for MinScore)
//
// Currently, only vector similarity search is implemented using OceanBase's cosine_distance.
// Hybrid search (vector + full-text + sparse) will be added in future versions when
// OceanBase supports additional retrieval modes.
func (c *Client) Search(ctx context.Context, embedding []float64, opts *storage.SearchOptions) ([]*storage.Memory, error) {
	// Use Threshold if MinScore is not set (Python SDK compatibility)
	minScore := opts.MinScore
	if minScore == 0 && opts.Threshold > 0 {
		minScore = opts.Threshold
	}

	queryVectorStr := vectorToString(embedding)

	whereClause, args := buildWhereClause(opts.UserID, opts.AgentID, opts.Filters)

	// Add similarity threshold filter if specified
	if minScore > 0 {
		if whereClause == "" {
			whereClause = "WHERE 1 - cosine_distance(embedding, ?) >= ?"
		} else {
			whereClause += " AND 1 - cosine_distance(embedding, ?) >= ?"
		}
		// Note: We'll add queryVectorStr to args twice (once for distance calc, once for threshold check)
		args = append(args, queryVectorStr, minScore)
	}

	query := fmt.Sprintf(`
		SELECT 
			id, user_id, agent_id, run_id, document, embedding, metadata,
			created_at, updated_at, hash,
			cosine_distance(embedding, ?) as distance
		FROM %s
		%s
		ORDER BY distance ASC
		LIMIT ?
	`, c.collectionName, whereClause)

	// Build args: query vector (for SELECT and distance), then filter args, then limit
	allArgs := []interface{}{queryVectorStr}
	allArgs = append(allArgs, args...)
	allArgs = append(allArgs, opts.Limit)

	// TODO: Future enhancement - add full-text search support using opts.Query
	// This would enable hybrid retrieval combining vector similarity and keyword matching
	// if opts.Query != "" {
	//     // Add full-text search condition to WHERE clause
	//     // Combine vector distance with text relevance scoring
	// }

	// TODO: Future enhancement - add sparse embedding support using opts.SparseEmbedding
	// This would enable sparse + dense hybrid retrieval
	// if opts.SparseEmbedding != nil {
	//     // Calculate sparse similarity
	//     // Combine with dense vector score
	// }

	rows, err := c.db.QueryContext(ctx, query, allArgs...)
	if err != nil {
		return nil, fmt.Errorf("Search: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return c.scanMemories(rows, true)
}

// Get retrieves a memory by ID with optional access control.
// Compatible with Python SDK: uses 'document' field
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
		SELECT id, user_id, agent_id, run_id, document, embedding, metadata,
		       created_at, updated_at, hash
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
// Compatible with Python SDK: uses 'document' field
func (c *Client) Update(ctx context.Context, id int64, content string, embedding []float64, opts *storage.UpdateOptions) (*storage.Memory, error) {
	if opts == nil {
		opts = &storage.UpdateOptions{}
	}

	vectorStr := vectorToString(embedding)
	hash := generateHash(content)
	now := time.Now().Format(time.RFC3339)

	// Build WHERE clause with access control
	whereClause := "WHERE id = ?"
	args := []interface{}{content, vectorStr, now, hash, id}

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
		SET document = ?, embedding = ?, updated_at = ?, hash = ?
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

	// Return updated memory
	return c.Get(ctx, id, &storage.GetOptions{
		UserID:  opts.UserID,
		AgentID: opts.AgentID,
	})
}

// Delete deletes a memory with optional access control.
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

// GetAll retrieves all memories.
// Compatible with Python SDK: uses 'document' field
func (c *Client) GetAll(ctx context.Context, opts *storage.GetAllOptions) ([]*storage.Memory, error) {
	whereClause, args := buildWhereClause(opts.UserID, opts.AgentID, nil)

	query := fmt.Sprintf(`
		SELECT id, user_id, agent_id, run_id, document, embedding, metadata,
		       created_at, updated_at, hash
		FROM %s
		%s
		ORDER BY id DESC
		LIMIT ? OFFSET ?
	`, c.collectionName, whereClause)

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

// CreateIndex creates a vector index.
func (c *Client) CreateIndex(ctx context.Context, config *storage.VectorIndexConfig) error {
	var query string

	switch config.IndexType {
	case storage.IndexTypeHNSW:
		query = fmt.Sprintf(`
			CREATE VECTOR INDEX %s ON %s (%s) WITH (
				index_type = HNSW,
				M = %d,
				efConstruction = %d,
				metric_type = %s
			)`,
			config.IndexName, config.TableName, config.VectorField,
			config.HNSWParams.M,
			config.HNSWParams.EfConstruction,
			config.MetricType,
		)
	case storage.IndexTypeIVFFlat:
		query = fmt.Sprintf(`
			CREATE VECTOR INDEX %s ON %s (%s) WITH (
				index_type = IVF_FLAT,
				nlist = %d,
				metric_type = %s
			)`,
			config.IndexName, config.TableName, config.VectorField,
			config.IVFParams.Nlist,
			config.MetricType,
		)
	default:
		return fmt.Errorf("CreateIndex: invalid index type")
	}

	_, err := c.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("CreateIndex: %w", err)
	}

	return nil
}

// scanMemory scans a single memory.
// Compatible with Python SDK: reads from 'document' field
func (c *Client) scanMemory(row *sql.Row) (*storage.Memory, error) {
	var memory storage.Memory
	var embeddingStr string
	var metadataJSON []byte
	var userID sql.NullString
	var agentID sql.NullString
	var runID sql.NullString
	var hash sql.NullString
	var createdAt sql.NullString
	var updatedAt sql.NullString

	err := row.Scan(
		&memory.ID,
		&userID,
		&agentID,
		&runID,
		&memory.Content,
		&embeddingStr,
		&metadataJSON,
		&createdAt,
		&updatedAt,
		&hash,
	)
	if err != nil {
		return nil, err
	}

	// Handle nullable fields
	if userID.Valid {
		memory.UserID = userID.String
	}
	if agentID.Valid {
		memory.AgentID = agentID.String
	}

	// Parse embedding
	if embeddingStr != "" {
		embedding, err := stringToVector(embeddingStr)
		if err != nil {
			return nil, err
		}
		memory.Embedding = embedding
	}

	// Parse metadata
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &memory.Metadata); err != nil {
			return nil, err
		}

		// Extract retention_strength from metadata if present
		if memory.Metadata != nil {
			if rs, ok := memory.Metadata["retention_strength"].(float64); ok {
				memory.RetentionStrength = rs
			}
		}
	}

	// Parse timestamps
	if createdAt.Valid {
		if t, err := time.Parse(time.RFC3339, createdAt.String); err == nil {
			memory.CreatedAt = t
		}
	}
	if updatedAt.Valid {
		if t, err := time.Parse(time.RFC3339, updatedAt.String); err == nil {
			memory.UpdatedAt = t
		}
	}

	return &memory, nil
}

// scanMemories scans multiple memories.
// Compatible with Python SDK: reads from 'document' field
func (c *Client) scanMemories(rows *sql.Rows, hasScore bool) ([]*storage.Memory, error) {
	var memories []*storage.Memory

	for rows.Next() {
		var memory storage.Memory
		var embeddingStr string
		var metadataJSON []byte
		var userID sql.NullString
		var agentID sql.NullString
		var runID sql.NullString
		var hash sql.NullString
		var createdAt sql.NullString
		var updatedAt sql.NullString
		var distance float64

		if hasScore {
			err := rows.Scan(
				&memory.ID,
				&userID,
				&agentID,
				&runID,
				&memory.Content,
				&embeddingStr,
				&metadataJSON,
				&createdAt,
				&updatedAt,
				&hash,
				&distance,
			)
			if err != nil {
				return nil, err
			}
			// Convert distance to similarity score (1 - distance)
			memory.Score = 1.0 - distance
		} else {
			err := rows.Scan(
				&memory.ID,
				&userID,
				&agentID,
				&runID,
				&memory.Content,
				&embeddingStr,
				&metadataJSON,
				&createdAt,
				&updatedAt,
				&hash,
			)
			if err != nil {
				return nil, err
			}
		}

		// Handle nullable fields
		if userID.Valid {
			memory.UserID = userID.String
		}
		if agentID.Valid {
			memory.AgentID = agentID.String
		}

		// Parse embedding
		if embeddingStr != "" {
			embedding, err := stringToVector(embeddingStr)
			if err != nil {
				return nil, err
			}
			memory.Embedding = embedding
		}

		// Parse metadata
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &memory.Metadata); err != nil {
				return nil, err
			}

			// Extract retention_strength from metadata if present
			if memory.Metadata != nil {
				if rs, ok := memory.Metadata["retention_strength"].(float64); ok {
					memory.RetentionStrength = rs
				}
			}
		}

		// Parse timestamps
		if createdAt.Valid {
			if t, err := time.Parse(time.RFC3339, createdAt.String); err == nil {
				memory.CreatedAt = t
			}
		}
		if updatedAt.Valid {
			if t, err := time.Parse(time.RFC3339, updatedAt.String); err == nil {
				memory.UpdatedAt = t
			}
		}

		memories = append(memories, &memory)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return memories, nil
}
