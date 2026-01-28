# PowerMem Go SDK - API Reference

This document provides a comprehensive API reference for the PowerMem Go SDK.

## Table of Contents

- [Client Initialization](#client-initialization)
- [Core Operations](#core-operations)
- [Async Operations](#async-operations)
- [Intelligent Memory](#intelligent-memory)
- [Multi-Agent Support](#multi-agent-support)
- [User Memory](#user-memory)
- [Configuration](#configuration)
- [Types](#types)

---

## Client Initialization

### LoadConfigFromEnv

Loads configuration from environment variables or `.env` file.

```go
func LoadConfigFromEnv() (*Config, error)
```

**Example:**

```go
config, err := powermem.LoadConfigFromEnv()
if err != nil {
    log.Fatal(err)
}
```

### NewClient

Creates a new PowerMem client instance.

```go
func NewClient(config *Config) (*Client, error)
```

**Parameters:**

- `config`: Configuration object containing LLM, embedder, and vector store settings

**Returns:**

- `*Client`: Initialized client instance
- `error`: Error if initialization fails

**Example:**

```go
client, err := powermem.NewClient(config)
if err != nil {
    log.Fatal(err)
}
defer client.Close()
```

---

## Core Operations

### Add

Adds a new memory to the system.

```go
func (c *Client) Add(ctx context.Context, content string, opts ...Option) (*Memory, error)
```

**Parameters:**

- `ctx`: Context for cancellation and timeouts
- `content`: Memory content (text, conversation, fact)
- `opts`: Optional parameters (user ID, agent ID, metadata)

**Options:**

- `WithUserID(userID string)`: Associate memory with a user
- `WithAgentID(agentID string)`: Associate memory with an agent
- `WithMetadata(metadata map[string]interface{})`: Add custom metadata

**Returns:**

- `*Memory`: Created memory object with ID and timestamp
- `error`: Error if operation fails

**Example:**

```go
memory, err := client.Add(ctx, "User prefers dark mode",
    powermem.WithUserID("user123"),
    powermem.WithMetadata(map[string]interface{}{
        "source": "settings",
    }),
)
```

### Search

Searches for relevant memories based on a query.

```go
func (c *Client) Search(ctx context.Context, query string, opts ...SearchOption) ([]*SearchResult, error)
```

**Parameters:**

- `ctx`: Context for cancellation and timeouts
- `query`: Search query text
- `opts`: Search options (filters, limits, etc.)

**Search Options:**

- `WithUserIDForSearch(userID string)`: Filter by user
- `WithAgentIDForSearch(agentID string)`: Filter by agent
- `WithLimit(limit int)`: Maximum number of results (default: 10)
- `WithScoreThreshold(threshold float64)`: Minimum relevance score (0-1)

**Returns:**

- `[]*SearchResult`: Array of search results with memories and scores
- `error`: Error if search fails

**Example:**

```go
results, err := client.Search(ctx, "user preferences",
    powermem.WithUserIDForSearch("user123"),
    powermem.WithLimit(5),
    powermem.WithScoreThreshold(0.7),
)
```

### Get

Retrieves a specific memory by ID.

```go
func (c *Client) Get(ctx context.Context, memoryID int64, opts ...GetOption) (*Memory, error)
```

**Parameters:**

- `ctx`: Context for cancellation and timeouts
- `memoryID`: Unique memory identifier
- `opts`: Optional filters

**Returns:**

- `*Memory`: Memory object or nil if not found
- `error`: Error if operation fails

**Example:**

```go
memory, err := client.Get(ctx, memoryID)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Content: %s\n", memory.Content)
```

### GetAll

Retrieves all memories matching the filter criteria.

```go
func (c *Client) GetAll(ctx context.Context, opts ...GetAllOption) ([]*Memory, error)
```

**Options:**

- `WithUserIDForGetAll(userID string)`: Filter by user
- `WithAgentIDForGetAll(agentID string)`: Filter by agent
- `WithFilters(filters map[string]interface{})`: Custom metadata filters

**Example:**

```go
memories, err := client.GetAll(ctx,
    powermem.WithUserIDForGetAll("user123"),
)
```

### Update

Updates an existing memory's content or metadata.

```go
func (c *Client) Update(ctx context.Context, memoryID int64, content string, opts ...UpdateOption) (*Memory, error)
```

**Parameters:**

- `ctx`: Context for cancellation and timeouts
- `memoryID`: ID of memory to update
- `content`: New content (empty string to keep existing)
- `opts`: Update options

**Options:**

- `WithMetadataForUpdate(metadata map[string]interface{})`: Update metadata

**Example:**

```go
updated, err := client.Update(ctx, memoryID, "User strongly prefers dark mode",
    powermem.WithMetadataForUpdate(map[string]interface{}{
        "confidence": "high",
    }),
)
```

### Delete

Deletes a specific memory by ID.

```go
func (c *Client) Delete(ctx context.Context, memoryID int64) error
```

**Example:**

```go
err := client.Delete(ctx, memoryID)
```

### DeleteAll

Deletes all memories matching the filter criteria.

```go
func (c *Client) DeleteAll(ctx context.Context, opts ...DeleteAllOption) error
```

**Options:**

- `WithUserIDForDeleteAll(userID string)`: Delete all memories for a user
- `WithAgentIDForDeleteAll(agentID string)`: Delete all memories for an agent

**Example:**

```go
// Delete all memories for a user
err := client.DeleteAll(ctx,
    powermem.WithUserIDForDeleteAll("user123"),
)
```

---

## Async Operations

For high-performance scenarios, use async operations that return channels.

### AddAsync

```go
func (c *Client) AddAsync(ctx context.Context, content string, opts ...Option) <-chan *AsyncResult
```

**Example:**

```go
resultChan := client.AddAsync(ctx, "User likes coffee",
    powermem.WithUserID("user123"),
)

result := <-resultChan
if result.Error != nil {
    log.Fatal(result.Error)
}
fmt.Printf("Added: %v\n", result.Memory)
```

### SearchAsync

```go
func (c *Client) SearchAsync(ctx context.Context, query string, opts ...SearchOption) <-chan *AsyncSearchResult
```

### Streaming Search

For real-time results as they become available:

```go
func (c *Client) SearchStreaming(ctx context.Context, query string, opts ...SearchOption) (<-chan *SearchResult, <-chan error)
```

**Example:**

```go
resultChan, errChan := client.SearchStreaming(ctx, "user preferences",
    powermem.WithUserIDForSearch("user123"),
)

for result := range resultChan {
    fmt.Printf("- %s (score: %.4f)\n", result.Memory, result.Score)
}

if err := <-errChan; err != nil {
    log.Fatal(err)
}
```

---

## Intelligent Memory

### AddWithIntelligence

Adds memory with intelligent processing (fact extraction, deduplication, merging).

```go
func (c *Client) AddWithIntelligence(ctx context.Context, content string, opts ...Option) ([]*Memory, error)
```

**Features:**

- Automatic fact extraction from conversations
- Duplicate detection and merging
- Conflict resolution
- Related memory merging

**Example:**

```go
memories, err := client.AddWithIntelligence(ctx, 
    "User mentioned they love coffee and tea, especially in the morning",
    powermem.WithUserID("user123"),
)
// Returns multiple extracted facts as separate memories
```

### Intelligence Manager

Direct access to intelligence features:

```go
// Extract facts from text
facts, err := client.ExtractFacts(ctx, content)

// Detect duplicates
isDuplicate, err := client.DetectDuplicate(ctx, newMemory, existingMemories)

// Calculate importance score
score, err := client.CalculateImportance(ctx, memory)
```

---

## Multi-Agent Support

### Agent Isolation

Each agent has its own memory space:

```go
// Agent A's memory
client.Add(ctx, "User likes Python",
    powermem.WithUserID("user123"),
    powermem.WithAgentID("agent_a"),
)

// Agent B's memory
client.Add(ctx, "User prefers Go",
    powermem.WithUserID("user123"),
    powermem.WithAgentID("agent_b"),
)

// Search only in Agent A's space
results, _ := client.Search(ctx, "programming language",
    powermem.WithUserIDForSearch("user123"),
    powermem.WithAgentIDForSearch("agent_a"),
)
```

### Shared Memories

Memories without agent ID are shared across all agents:

```go
// Shared memory (no agent ID)
client.Add(ctx, "Important: User's birthday is June 15",
    powermem.WithUserID("user123"),
)
```

---

## User Memory

User memory provides user profile management and query rewriting capabilities.

### CreateUserMemory

```go
func NewUserMemoryClient(client *Client) (*UserMemoryClient, error)
```

**Example:**

```go
userMem, err := powermem.NewUserMemoryClient(client)
if err != nil {
    log.Fatal(err)
}
defer userMem.Close()
```

### AddUserMemory

```go
func (u *UserMemoryClient) Add(ctx context.Context, content string, userID string, opts ...UserMemoryOption) error
```

### GetUserProfile

```go
func (u *UserMemoryClient) GetProfile(ctx context.Context, userID string) (string, error)
```

**Example:**

```go
profile, err := userMem.GetProfile(ctx, "user123")
fmt.Println("User Profile:", profile)
```

### RewriteQuery

Rewrites user queries with context from user profile:

```go
func (u *UserMemoryClient) RewriteQuery(ctx context.Context, query string, userID string) (string, error)
```

**Example:**

```go
rewritten, err := userMem.RewriteQuery(ctx, "What's the weather?", "user123")
// Adds user context: "What's the weather in San Francisco?" (if user location is SF)
```

---

## Configuration

### Config Structure

```go
type Config struct {
    LLM         LLMConfig         // LLM provider configuration
    Embedder    EmbedderConfig    // Embedding model configuration
    VectorStore VectorStoreConfig // Vector database configuration
    Intelligence *IntelligenceConfig // Optional intelligence features
}

type LLMConfig struct {
    Provider    string  // "openai", "qwen", "anthropic", "deepseek", "ollama"
    APIKey      string  // API key
    Model       string  // Model name
    Temperature float64 // Sampling temperature (0-1)
    MaxTokens   int     // Maximum tokens in response
}

type EmbedderConfig struct {
    Provider string // "openai", "qwen"
    APIKey   string // API key
    Model    string // Model name
    Dimension int   // Embedding dimension (auto-detected)
}

type VectorStoreConfig struct {
    Provider       string                 // "sqlite", "postgres", "oceanbase"
    CollectionName string                 // Table/collection name
    ConnectionArgs map[string]interface{} // Connection parameters
}
```

### Environment Variables

See [`.env.example`](../../../.env.example) for all available configuration options.

---

## Types

### Memory

```go
type Memory struct {
    ID        int64                  // Unique identifier
    Content   string                 // Memory content
    UserID    string                 // User identifier
    AgentID   string                 // Agent identifier
    Metadata  map[string]interface{} // Custom metadata
    CreatedAt time.Time              // Creation timestamp
    UpdatedAt time.Time              // Last update timestamp
}
```

### SearchResult

```go
type SearchResult struct {
    Memory string  // Memory content
    Score  float64 // Relevance score (0-1)
    ID     int64   // Memory ID
}
```

### AsyncResult

```go
type AsyncResult struct {
    Memory *Memory // Result memory
    Error  error   // Error if operation failed
}
```

---

## Error Handling

All operations return errors that should be checked:

```go
memory, err := client.Add(ctx, content)
if err != nil {
    // Handle error
    if errors.Is(err, powermem.ErrInvalidConfig) {
        // Configuration error
    } else if errors.Is(err, powermem.ErrConnectionFailed) {
        // Database connection error
    }
    return err
}
```

Common error types:

- `ErrInvalidConfig`: Configuration validation failed
- `ErrConnectionFailed`: Database connection failed
- `ErrNotFound`: Memory not found
- `ErrInvalidInput`: Invalid input parameters

---

## Best Practices

1. **Always use context**: Pass context for cancellation and timeouts
2. **Close clients**: Use `defer client.Close()` to release resources
3. **Handle errors**: Check all error returns
4. **Use options**: Leverage option functions for cleaner API calls
5. **Async for performance**: Use async operations for bulk processing
6. **Agent isolation**: Use agent IDs to isolate multi-agent memories
7. **Intelligent processing**: Enable intelligence features for better memory quality

---

For more examples and detailed guides, see the [examples](../examples/) directory and the [main documentation](../../../docs/).
