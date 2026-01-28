package core

import (
	"context"
	"sync"

	"github.com/bwmarrin/snowflake"
	"github.com/oceanbase/powermem-go/pkg/embedder"
	openaiEmbedder "github.com/oceanbase/powermem-go/pkg/embedder/openai"
	qwenEmbedder "github.com/oceanbase/powermem-go/pkg/embedder/qwen"
	"github.com/oceanbase/powermem-go/pkg/intelligence"
	"github.com/oceanbase/powermem-go/pkg/llm"
	anthropicLLM "github.com/oceanbase/powermem-go/pkg/llm/anthropic"
	deepseekLLM "github.com/oceanbase/powermem-go/pkg/llm/deepseek"
	ollamaLLM "github.com/oceanbase/powermem-go/pkg/llm/ollama"
	openaiLLM "github.com/oceanbase/powermem-go/pkg/llm/openai"
	qwenLLM "github.com/oceanbase/powermem-go/pkg/llm/qwen"
	"github.com/oceanbase/powermem-go/pkg/storage"
	"github.com/oceanbase/powermem-go/pkg/storage/oceanbase"
	postgresStore "github.com/oceanbase/powermem-go/pkg/storage/postgres"
	sqliteStore "github.com/oceanbase/powermem-go/pkg/storage/sqlite"
)

// Client is the main PowerMem client for memory management.
//
// It provides a complete interface for storing, retrieving, and managing memories
// with support for:
//   - Vector similarity search
//   - Intelligent deduplication
//   - Ebbinghaus forgetting curve
//   - Multi-agent support
//   - Metadata filtering
//
// The client is thread-safe and can be used concurrently from multiple goroutines.
//
// Example usage:
//
//	config, _ := core.LoadConfigFromEnv()
//	client, _ := core.NewClient(config)
//	defer client.Close()
//
//	memory, _ := client.Add(ctx, "User likes Python",
//	    core.WithUserID("user_001"),
//	    core.WithInfer(true), // Enable intelligent deduplication
//	)
type Client struct {
	// config contains the client configuration.
	config *Config

	// storage is the vector store for memory persistence.
	storage storage.VectorStore

	// llm is the LLM provider for intelligent features.
	llm llm.Provider

	// embedder is the embedding provider for vector generation.
	embedder embedder.Provider

	// dedupManager manages memory deduplication (nil if not enabled).
	dedupManager *intelligence.DedupManager

	// ebbinghausManager manages retention using Ebbinghaus curve (nil if not enabled).
	ebbinghausManager *intelligence.EbbinghausManager

	// intelligentManager manages complete intelligent memory processing (nil if not enabled).
	intelligentManager *intelligence.IntelligentMemoryManager

	// snowflakeNode generates unique IDs for memories.
	snowflakeNode *snowflake.Node

	// mu protects concurrent access to the client.
	mu sync.RWMutex
}

// NewClient creates a new PowerMem client.
//
// The client is initialized with:
//   - Vector store (SQLite, OceanBase, or PostgreSQL)
//   - LLM provider (OpenAI, Qwen, DeepSeek, Ollama, Anthropic)
//   - Embedding provider (OpenAI, Qwen)
//   - Intelligent features (if enabled in config)
//
// Parameters:
//   - cfg: Configuration containing storage, LLM, and embedding settings
//
// Returns a new Client instance, or an error if initialization fails.
//
// Example:
//
//	config := &core.Config{
//	    VectorStore: core.VectorStoreConfig{...},
//	    LLM: core.LLMConfig{...},
//	    Embedder: core.EmbedderConfig{...},
//	    Intelligence: &core.IntelligenceConfig{
//	        Enabled: true,
//	        DecayRate: 0.1,
//	    },
//	}
//	client, err := core.NewClient(config)
func NewClient(cfg *Config) (*Client, error) {
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Initialize storage
	store, err := initStorage(cfg.VectorStore)
	if err != nil {
		return nil, err
	}

	// Initialize LLM
	llmProvider, err := initLLM(cfg.LLM)
	if err != nil {
		return nil, err
	}

	// Initialize Embedder
	embedderProvider, err := initEmbedder(cfg.Embedder)
	if err != nil {
		return nil, err
	}

	// Initialize Snowflake ID generator
	node, err := snowflake.NewNode(1)
	if err != nil {
		return nil, NewMemoryError("NewClient", err)
	}

	client := &Client{
		config:        cfg,
		storage:       store,
		llm:           llmProvider,
		embedder:      embedderProvider,
		snowflakeNode: node,
	}

	// Initialize intelligent features (if enabled)
	if cfg.Intelligence != nil && cfg.Intelligence.Enabled {
		// Initialize deduplication manager
		client.dedupManager = intelligence.NewDedupManager(
			store,
			cfg.Intelligence.DuplicateThreshold,
		)

		// Initialize Ebbinghaus manager
		client.ebbinghausManager = intelligence.NewEbbinghausManager(
			cfg.Intelligence.DecayRate,
			cfg.Intelligence.ReinforcementFactor,
		)

		// Initialize intelligent memory manager (for full intelligent processing)
		intelligenceConfig := &intelligence.Config{
			DecayRate:           cfg.Intelligence.DecayRate,
			ReinforcementFactor: cfg.Intelligence.ReinforcementFactor,
			WorkingThreshold:    cfg.Intelligence.WorkingThreshold,
			ShortTermThreshold:  cfg.Intelligence.ShortTermThreshold,
			LongTermThreshold:   cfg.Intelligence.LongTermThreshold,
			InitialRetention:    cfg.Intelligence.InitialRetention,
			FallbackToSimpleAdd: cfg.Intelligence.FallbackToSimpleAdd,
		}
		// Set defaults if not specified
		if intelligenceConfig.WorkingThreshold == 0 {
			intelligenceConfig.WorkingThreshold = 0.3
		}
		if intelligenceConfig.ShortTermThreshold == 0 {
			intelligenceConfig.ShortTermThreshold = 0.6
		}
		if intelligenceConfig.LongTermThreshold == 0 {
			intelligenceConfig.LongTermThreshold = 0.8
		}
		if intelligenceConfig.InitialRetention == 0 {
			intelligenceConfig.InitialRetention = 1.0
		}

		client.intelligentManager = intelligence.NewIntelligentMemoryManager(
			llmProvider,
			intelligenceConfig,
		)
	}

	return client, nil
}

// Add adds a new memory to the store.
//
// The method:
//  1. Generates an embedding vector for the content
//  2. Optionally checks for duplicates (if Infer option is enabled)
//  3. Stores the memory with metadata
//
// If intelligent deduplication is enabled and a duplicate is found,
// the memories are merged instead of creating a new one.
//
// Parameters:
//   - ctx: Context for cancellation
//   - content: Memory content (text string)
//   - opts: Optional parameters (UserID, AgentID, Metadata, Infer, etc.)
//
// Returns the created or merged Memory, or an error if the operation fails.
//
// Example:
//
//	memory, err := client.Add(ctx, "User likes Python programming",
//	    core.WithUserID("user_001"),
//	    core.WithAgentID("agent_001"),
//	    core.WithInfer(true), // Enable intelligent deduplication
//	    core.WithMetadata(map[string]interface{}{
//	        "source": "conversation",
//	    }),
//	)
func (c *Client) Add(ctx context.Context, content string, opts ...AddOption) (*Memory, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Apply options
	addOpts := applyAddOptions(opts)

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Generate embedding
	embedding, err := c.embedder.Embed(ctx, content)
	if err != nil {
		return nil, NewMemoryError("Add", err)
	}

	// Intelligent deduplication (if enabled)
	if addOpts.Infer && c.dedupManager != nil {
		isDup, existingID, err := c.dedupManager.CheckDuplicate(ctx, embedding, addOpts.UserID, addOpts.AgentID)
		if err != nil {
			return nil, NewMemoryError("Add", err)
		}
		if isDup {
			// Merge memories
			merged, err := c.dedupManager.MergeMemories(ctx, existingID, content, embedding)
			if err != nil {
				return nil, NewMemoryError("Add", err)
			}
			// Convert back to core.Memory type
			return fromIntelligenceMemory(merged), nil
		}
	}

	// Build metadata, merge all additional parameters
	metadata := make(map[string]interface{})
	if addOpts.Metadata != nil {
		for k, v := range addOpts.Metadata {
			metadata[k] = v
		}
	}
	// Add extra parameters to metadata (if provided)
	if addOpts.RunID != "" {
		metadata["run_id"] = addOpts.RunID
	}
	if addOpts.MemoryType != "" {
		metadata["memory_type"] = addOpts.MemoryType
	}
	if addOpts.Scope != "" {
		metadata["scope"] = string(addOpts.Scope)
	}
	if addOpts.Prompt != "" {
		metadata["prompt"] = addOpts.Prompt
	}
	// Merge filters into metadata
	if addOpts.Filters != nil {
		for k, v := range addOpts.Filters {
			metadata[k] = v
		}
	}

	// Insert into storage
	memory := &Memory{
		ID:                c.snowflakeNode.Generate().Int64(),
		UserID:            addOpts.UserID,
		AgentID:           addOpts.AgentID,
		Content:           content,
		Embedding:         embedding,
		Metadata:          metadata,
		RetentionStrength: 1.0, // Initial strength: 1.0
	}

	if err := c.storage.Insert(ctx, toStorageMemory(memory)); err != nil {
		return nil, NewMemoryError("Add", err)
	}

	return memory, nil
}

// Search searches for memories using vector similarity.
//
// The method:
//  1. Generates an embedding vector for the query
//  2. Performs vector similarity search in the store
//  3. Returns results sorted by similarity score
//
// Results can be filtered by UserID, AgentID, and custom metadata filters.
//
// Parameters:
//   - ctx: Context for cancellation
//   - query: Search query (text string)
//   - opts: Optional parameters (UserID, AgentID, Limit, MinScore, Filters)
//
// Returns a list of memories sorted by relevance (highest first), or an error.
//
// Example:
//
//	results, err := client.Search(ctx, "Python programming",
//	    core.WithUserIDForSearch("user_001"),
//	    core.WithLimit(10),
//	    core.WithMinScore(0.7),
//	)
func (c *Client) Search(ctx context.Context, query string, opts ...SearchOption) ([]*Memory, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Apply search options
	searchOpts := applySearchOptions(opts)

	// Generate query embedding
	queryEmbedding, err := c.embedder.Embed(ctx, query)
	if err != nil {
		return nil, NewMemoryError("Search", err)
	}

	// Execute vector similarity search
	storageOpts := &storage.SearchOptions{
		UserID:   searchOpts.UserID,
		AgentID:  searchOpts.AgentID,
		Limit:    searchOpts.Limit,
		MinScore: searchOpts.MinScore,
		Filters:  searchOpts.Filters,
	}

	memories, err := c.storage.Search(ctx, queryEmbedding, storageOpts)
	if err != nil {
		return nil, NewMemoryError("Search", err)
	}

	return fromStorageMemories(memories), nil
}

// Get retrieves a memory by its ID.
//
// Parameters:
//   - ctx: Context for cancellation
//   - id: Memory ID
//
// Returns the Memory if found, or an error if not found or retrieval fails.
func (c *Client) Get(ctx context.Context, id int64) (*Memory, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	memory, err := c.storage.Get(ctx, id)
	if err != nil {
		return nil, NewMemoryError("Get", err)
	}

	return fromStorageMemory(memory), nil
}

// Update updates an existing memory's content.
//
// The method:
//  1. Generates a new embedding vector for the updated content
//  2. Updates the memory in the store
//
// Parameters:
//   - ctx: Context for cancellation
//   - id: Memory ID to update
//   - content: New content for the memory
//
// Returns the updated Memory, or an error if update fails.
func (c *Client) Update(ctx context.Context, id int64, content string) (*Memory, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Generate new embedding
	embedding, err := c.embedder.Embed(ctx, content)
	if err != nil {
		return nil, NewMemoryError("Update", err)
	}

	// Update storage
	memory, err := c.storage.Update(ctx, id, content, embedding)
	if err != nil {
		return nil, NewMemoryError("Update", err)
	}

	return fromStorageMemory(memory), nil
}

// Delete deletes a memory by its ID.
//
// Parameters:
//   - ctx: Context for cancellation
//   - id: Memory ID to delete
//
// Returns an error if deletion fails.
func (c *Client) Delete(ctx context.Context, id int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.storage.Delete(ctx, id); err != nil {
		return NewMemoryError("Delete", err)
	}

	return nil
}

// GetAll retrieves all memories with optional filtering.
//
// Results can be filtered by UserID, AgentID, and paginated using Limit and Offset.
//
// Parameters:
//   - ctx: Context for cancellation
//   - opts: Optional parameters (UserID, AgentID, Limit, Offset)
//
// Returns a list of memories, or an error if retrieval fails.
//
// Example:
//
//	memories, err := client.GetAll(ctx,
//	    core.WithUserIDForGetAll("user_001"),
//	    core.WithLimitForGetAll(100),
//	    core.WithOffset(0),
//	)
func (c *Client) GetAll(ctx context.Context, opts ...GetAllOption) ([]*Memory, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	getAllOpts := applyGetAllOptions(opts)

	storageOpts := &storage.GetAllOptions{
		UserID:  getAllOpts.UserID,
		AgentID: getAllOpts.AgentID,
		Limit:   getAllOpts.Limit,
		Offset:  getAllOpts.Offset,
	}

	memories, err := c.storage.GetAll(ctx, storageOpts)
	if err != nil {
		return nil, NewMemoryError("GetAll", err)
	}

	return fromStorageMemories(memories), nil
}

// DeleteAll deletes all memories matching the given filters.
//
// If no filters are provided, deletes ALL memories (use with caution).
//
// Parameters:
//   - ctx: Context for cancellation
//   - opts: Optional parameters (UserID, AgentID)
//
// Returns an error if deletion fails.
//
// Example:
//
//	err := client.DeleteAll(ctx,
//	    core.WithUserIDForDeleteAll("user_001"),
//	    core.WithAgentIDForDeleteAll("agent_001"),
//	)
func (c *Client) DeleteAll(ctx context.Context, opts ...DeleteAllOption) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	deleteAllOpts := applyDeleteAllOptions(opts)

	storageOpts := &storage.DeleteAllOptions{
		UserID:  deleteAllOpts.UserID,
		AgentID: deleteAllOpts.AgentID,
	}

	if err := c.storage.DeleteAll(ctx, storageOpts); err != nil {
		return NewMemoryError("DeleteAll", err)
	}

	return nil
}

// Close closes the client and releases all resources.
//
// This method:
//   - Closes the vector store connection
//   - Closes the LLM provider
//   - Closes the embedder provider
//
// Returns the first error encountered during cleanup, or nil if all resources
// were closed successfully.
//
// Example:
//
//	defer client.Close()
func (c *Client) Close() error {
	var errs []error

	if c.storage != nil {
		if err := c.storage.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if c.llm != nil {
		if err := c.llm.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if c.embedder != nil {
		if err := c.embedder.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0] // Return the first error
	}

	return nil
}

// initStorage initializes the storage backend.
func initStorage(cfg VectorStoreConfig) (storage.VectorStore, error) {
	switch cfg.Provider {
	case "oceanbase":
		return oceanbase.NewClient(&oceanbase.Config{
			Host:               cfg.Config["host"].(string),
			Port:               cfg.Config["port"].(int),
			User:               cfg.Config["user"].(string),
			Password:           cfg.Config["password"].(string),
			DBName:             cfg.Config["db_name"].(string),
			CollectionName:     cfg.Config["collection_name"].(string),
			EmbeddingModelDims: cfg.Config["embedding_model_dims"].(int),
		})
	case "sqlite":
		return sqliteStore.NewClient(&sqliteStore.Config{
			DBPath:             cfg.Config["db_path"].(string),
			CollectionName:     cfg.Config["collection_name"].(string),
			EmbeddingModelDims: cfg.Config["embedding_model_dims"].(int),
		})
	case "postgres":
		sslMode := "disable"
		if s, ok := cfg.Config["ssl_mode"].(string); ok {
			sslMode = s
		}
		return postgresStore.NewClient(&postgresStore.Config{
			Host:               cfg.Config["host"].(string),
			Port:               cfg.Config["port"].(int),
			User:               cfg.Config["user"].(string),
			Password:           cfg.Config["password"].(string),
			DBName:             cfg.Config["db_name"].(string),
			CollectionName:     cfg.Config["collection_name"].(string),
			EmbeddingModelDims: cfg.Config["embedding_model_dims"].(int),
			SSLMode:            sslMode,
		})
	default:
		return nil, NewMemoryError("initStorage", ErrInvalidConfig)
	}
}

// initLLM initializes the LLM provider.
func initLLM(cfg LLMConfig) (llm.Provider, error) {
	switch cfg.Provider {
	case "openai":
		return openaiLLM.NewClient(&openaiLLM.Config{
			APIKey:  cfg.APIKey,
			Model:   cfg.Model,
			BaseURL: cfg.BaseURL,
		})
	case "qwen":
		return qwenLLM.NewClient(&qwenLLM.Config{
			APIKey:  cfg.APIKey,
			Model:   cfg.Model,
			BaseURL: cfg.BaseURL,
		})
	case "deepseek":
		return deepseekLLM.NewClient(&deepseekLLM.Config{
			APIKey:  cfg.APIKey,
			Model:   cfg.Model,
			BaseURL: cfg.BaseURL,
		})
	case "ollama":
		return ollamaLLM.NewClient(&ollamaLLM.Config{
			APIKey:  cfg.APIKey,
			Model:   cfg.Model,
			BaseURL: cfg.BaseURL,
		})
	case "anthropic":
		return anthropicLLM.NewClient(&anthropicLLM.Config{
			APIKey:  cfg.APIKey,
			Model:   cfg.Model,
			BaseURL: cfg.BaseURL,
		})
	default:
		return nil, NewMemoryError("initLLM", ErrInvalidConfig)
	}
}

// initEmbedder initializes the embedder provider.
func initEmbedder(cfg EmbedderConfig) (embedder.Provider, error) {
	switch cfg.Provider {
	case "openai":
		return openaiEmbedder.NewClient(&openaiEmbedder.Config{
			APIKey:     cfg.APIKey,
			Model:      cfg.Model,
			BaseURL:    cfg.BaseURL,
			Dimensions: cfg.Dimensions,
		})
	case "qwen":
		return qwenEmbedder.NewClient(&qwenEmbedder.Config{
			APIKey:     cfg.APIKey,
			Model:      cfg.Model,
			BaseURL:    cfg.BaseURL,
			Dimensions: cfg.Dimensions,
		})
	default:
		return nil, NewMemoryError("initEmbedder", ErrInvalidConfig)
	}
}
