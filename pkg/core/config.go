// Package core provides the main PowerMem client and memory management functionality.
package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
)

// Config contains the complete configuration for a PowerMem client.
//
// It includes settings for:
//   - LLM provider (for intelligent features)
//   - Embedding provider (for vector generation)
//   - Vector store (for memory persistence)
//   - Intelligent memory management (optional)
//   - Multi-agent support (optional)
//
// Example:
//
//	config := &core.Config{
//	    LLM: core.LLMConfig{
//	        Provider: "openai",
//	        APIKey:   "sk-...",
//	        Model:    "gpt-4",
//	    },
//	    Embedder: core.EmbedderConfig{
//	        Provider:   "openai",
//	        APIKey:     "sk-...",
//	        Model:      "text-embedding-ada-002",
//	        Dimensions: 1536,
//	    },
//	    VectorStore: core.VectorStoreConfig{
//	        Provider: "sqlite",
//	        Config: map[string]interface{}{
//	            "db_path": "./memories.db",
//	        },
//	    },
//	}
type Config struct {
	// LLM contains LLM provider configuration.
	LLM LLMConfig `json:"llm"`

	// Embedder contains embedding provider configuration.
	Embedder EmbedderConfig `json:"embedder"`

	// VectorStore contains vector store configuration.
	VectorStore VectorStoreConfig `json:"vector_store"`

	// Intelligence contains intelligent memory management configuration (optional).
	Intelligence *IntelligenceConfig `json:"intelligence,omitempty"`

	// AgentMemory contains multi-agent memory configuration (optional).
	AgentMemory *AgentMemoryConfig `json:"agent_memory,omitempty"`
}

// LLMConfig contains configuration for the LLM provider.
//
// Supported providers: openai, qwen, anthropic, deepseek, ollama
//
// Example:
//
//	llmConfig := core.LLMConfig{
//	    Provider: "openai",
//	    APIKey:   "sk-...",
//	    Model:    "gpt-4",
//	    BaseURL:  "https://api.openai.com/v1",
//	}
type LLMConfig struct {
	// Provider is the LLM provider name (openai, qwen, anthropic, deepseek, ollama).
	Provider string `json:"provider"`

	// APIKey is the API key for the LLM provider.
	APIKey string `json:"api_key"`

	// Model is the model name to use (e.g., "gpt-4", "qwen-plus").
	Model string `json:"model"`

	// BaseURL is the base URL for the API (optional, uses provider default if empty).
	BaseURL string `json:"base_url,omitempty"`

	// Parameters contains additional provider-specific parameters (optional).
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// EmbedderConfig contains configuration for the embedding provider.
//
// Supported providers: openai, qwen, huggingface, ollama
//
// Example:
//
//	embedderConfig := core.EmbedderConfig{
//	    Provider:   "openai",
//	    APIKey:     "sk-...",
//	    Model:      "text-embedding-ada-002",
//	    Dimensions: 1536,
//	}
type EmbedderConfig struct {
	// Provider is the embedding provider name (openai, qwen, huggingface, ollama).
	Provider string `json:"provider"`

	// APIKey is the API key for the embedding provider.
	APIKey string `json:"api_key"`

	// Model is the embedding model name (e.g., "text-embedding-ada-002", "text-embedding-v4").
	Model string `json:"model"`

	// BaseURL is the base URL for the API (optional, uses provider default if empty).
	BaseURL string `json:"base_url,omitempty"`

	// Dimensions is the dimension of the embedding vectors (e.g., 1536, 1024).
	Dimensions int `json:"dimensions,omitempty"`

	// Parameters contains additional provider-specific parameters (optional).
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// VectorStoreConfig contains configuration for the vector store.
//
// Supported providers: oceanbase, sqlite, postgres
//
// Example:
//
//	storeConfig := core.VectorStoreConfig{
//	    Provider: "sqlite",
//	    Config: map[string]interface{}{
//	        "db_path":         "./memories.db",
//	        "collection_name": "memories",
//	    },
//	}
type VectorStoreConfig struct {
	// Provider is the vector store provider name (oceanbase, sqlite, postgres).
	Provider string `json:"provider"`

	// Config contains provider-specific configuration.
	// For SQLite: db_path, collection_name, embedding_model_dims
	// For OceanBase: host, port, user, password, db_name, collection_name, embedding_model_dims
	// For PostgreSQL: host, port, user, password, db_name, collection_name, embedding_model_dims, ssl_mode
	Config map[string]interface{} `json:"config"`
}

// IntelligenceConfig contains configuration for intelligent memory management.
//
// Intelligent memory management includes:
//   - Deduplication: Detecting and merging similar memories
//   - Ebbinghaus forgetting curve: Managing memory retention and decay
//   - Importance evaluation: Evaluating memory importance
//   - Memory classification: Classifying memories as working/short-term/long-term
//
// Example:
//
//	config := &core.Config{
//	    Intelligence: &core.IntelligenceConfig{
//	        Enabled:             true,
//	        DecayRate:           0.1,
//	        ReinforcementFactor: 0.3,
//	        DuplicateThreshold:  0.95,
//	        WorkingThreshold:   0.3,
//	        ShortTermThreshold:  0.6,
//	        LongTermThreshold:   0.8,
//	        InitialRetention:    1.0,
//	    },
//	}
type IntelligenceConfig struct {
	// Enabled indicates whether intelligent memory management is enabled.
	Enabled bool `json:"enabled"`

	// DecayRate is the rate at which memories decay over time (Ebbinghaus curve).
	// Higher values mean faster decay. Typical range: 0.05-0.2.
	DecayRate float64 `json:"decay_rate"`

	// ReinforcementFactor determines how much memories are strengthened on access.
	// Higher values mean stronger reinforcement. Typical range: 0.2-0.5.
	ReinforcementFactor float64 `json:"reinforcement_factor"`

	// DuplicateThreshold is the similarity threshold for duplicate detection.
	// Memories with similarity >= threshold are considered duplicates.
	// Typical range: 0.9-0.98 (higher = stricter).
	DuplicateThreshold float64 `json:"duplicate_threshold"`

	// WorkingThreshold is the threshold for working memory classification.
	// Memories with retention < threshold are considered working memory.
	// Default: 0.3
	WorkingThreshold float64 `json:"working_threshold,omitempty"`

	// ShortTermThreshold is the threshold for short-term memory classification.
	// Memories with retention between WorkingThreshold and ShortTermThreshold
	// are considered short-term memory. Default: 0.6
	ShortTermThreshold float64 `json:"short_term_threshold,omitempty"`

	// LongTermThreshold is the threshold for long-term memory classification.
	// Memories with retention >= threshold are considered long-term memory.
	// Default: 0.8
	LongTermThreshold float64 `json:"long_term_threshold,omitempty"`

	// InitialRetention is the initial retention strength for new memories.
	// Default: 1.0
	InitialRetention float64 `json:"initial_retention,omitempty"`

	// FallbackToSimpleAdd indicates whether to fallback to simple add mode
	// when intelligent processing fails (e.g., no facts extracted).
	// Default: false
	FallbackToSimpleAdd bool `json:"fallback_to_simple_add,omitempty"`
}

// AgentMemoryConfig contains configuration for multi-agent memory management.
//
// This configuration controls how memories are shared and accessed across
// multiple agents in a multi-agent system.
//
// Example:
//
//	agentConfig := &core.AgentMemoryConfig{
//	    DefaultScope:          core.ScopePrivate,
//	    AllowCrossAgentAccess: false,
//	    CollaborationLevel:    "read_only",
//	}
type AgentMemoryConfig struct {
	// DefaultScope is the default scope for new memories.
	// See MemoryScope constants for available scopes.
	DefaultScope MemoryScope `json:"default_scope"`

	// AllowCrossAgentAccess indicates whether agents can access memories
	// created by other agents (beyond scope-based access).
	AllowCrossAgentAccess bool `json:"allow_cross_agent_access"`

	// CollaborationLevel defines the level of collaboration between agents.
	// Possible values: "none", "read_only", "full"
	CollaborationLevel string `json:"collaboration_level"`
}

// LoadConfigFromEnv loads configuration from environment variables.
//
// The function:
//  1. Searches for .env or .env.example files (up to 5 directory levels up)
//  2. Loads environment variables from the found file
//  3. Parses environment variables into a Config struct
//
// Supported environment variables:
//   - DATABASE_PROVIDER (sqlite, oceanbase, postgres)
//   - OCEANBASE_HOST, OCEANBASE_PORT, OCEANBASE_USER, OCEANBASE_PASSWORD, etc.
//   - SQLITE_PATH, SQLITE_COLLECTION, etc.
//   - POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, etc.
//   - LLM_PROVIDER, LLM_API_KEY, LLM_MODEL, LLM_BASE_URL
//   - EMBEDDING_PROVIDER, EMBEDDING_API_KEY, EMBEDDING_MODEL, EMBEDDING_BASE_URL
//   - INTELLIGENCE_ENABLED (to enable intelligent memory)
//
// Returns a Config instance, or an error if loading fails.
//
// Example:
//
//	config, err := core.LoadConfigFromEnv()
//	if err != nil {
//	    log.Fatal(err)
//	}
func LoadConfigFromEnv() (*Config, error) {
	// Use FindEnvFile to locate .env file (supports upward search)
	envPath, found := FindEnvFile()
	if found {
		// If .env file is found, load it
		_ = godotenv.Load(envPath)
	} else {
		// If not found, try loading from current directory (godotenv default behavior)
		_ = godotenv.Load()
	}

	// Get database provider (unified with Python SDK naming)
	provider := getEnvOrDefault("DATABASE_PROVIDER", "sqlite")

	// Build different configurations based on provider
	vectorStoreConfig := make(map[string]interface{})

	switch provider {
	case "oceanbase":
		// Use Python SDK compatible environment variables
		port, _ := strconv.Atoi(getEnvOrDefault("OCEANBASE_PORT", "2881"))
		dims, _ := strconv.Atoi(getEnvOrDefault("OCEANBASE_EMBEDDING_MODEL_DIMS", "1536"))

		vectorStoreConfig = map[string]interface{}{
			"host":                 getEnvOrDefault("OCEANBASE_HOST", "127.0.0.1"),
			"port":                 port,
			"user":                 getEnvOrDefault("OCEANBASE_USER", "root@sys"),
			"password":             os.Getenv("OCEANBASE_PASSWORD"),
			"db_name":              getEnvOrDefault("OCEANBASE_DATABASE", "powermem"),
			"collection_name":      getEnvOrDefault("OCEANBASE_COLLECTION", "memories"),
			"embedding_model_dims": dims,
		}
	case "sqlite":
		// Use Python SDK compatible environment variables
		dims, _ := strconv.Atoi(getEnvOrDefault("SQLITE_EMBEDDING_MODEL_DIMS", "1536"))

		vectorStoreConfig = map[string]interface{}{
			"db_path":              getEnvOrDefault("SQLITE_PATH", "./powermem.db"),
			"collection_name":      getEnvOrDefault("SQLITE_COLLECTION", "memories"),
			"embedding_model_dims": dims,
		}
	case "postgres":
		// Use Python SDK compatible environment variables
		port, _ := strconv.Atoi(getEnvOrDefault("POSTGRES_PORT", "5432"))
		dims, _ := strconv.Atoi(getEnvOrDefault("POSTGRES_EMBEDDING_MODEL_DIMS", "1536"))

		vectorStoreConfig = map[string]interface{}{
			"host":                 getEnvOrDefault("POSTGRES_HOST", "localhost"),
			"port":                 port,
			"user":                 getEnvOrDefault("POSTGRES_USER", "postgres"),
			"password":             os.Getenv("POSTGRES_PASSWORD"),
			"db_name":              getEnvOrDefault("POSTGRES_DATABASE", "powermem"),
			"collection_name":      getEnvOrDefault("POSTGRES_COLLECTION", "memories"),
			"embedding_model_dims": dims,
			"ssl_mode":             getEnvOrDefault("POSTGRES_SSLMODE", "disable"),
		}
	}

	// Get LLM provider to determine which base URL environment variable and default model to use
	llmProvider := getEnvOrDefault("LLM_PROVIDER", "openai")
	var llmBaseURL string
	var defaultModel string

	switch llmProvider {
	case "deepseek":
		llmBaseURL = os.Getenv("DEEPSEEK_LLM_BASE_URL")
		if llmBaseURL == "" {
			llmBaseURL = "https://api.deepseek.com"
		}
		defaultModel = "deepseek-chat"
	case "qwen":
		defaultModel = "qwen-plus"
	case "ollama":
		llmBaseURL = os.Getenv("OLLAMA_LLM_BASE_URL")
		if llmBaseURL == "" {
			llmBaseURL = "http://localhost:11434"
		}
		defaultModel = "llama3.1:70b"
	case "anthropic":
		llmBaseURL = os.Getenv("ANTHROPIC_LLM_BASE_URL")
		if llmBaseURL == "" {
			llmBaseURL = "https://api.anthropic.com"
		}
		defaultModel = "claude-3-5-sonnet-20240620"
	default:
		llmBaseURL = os.Getenv("LLM_BASE_URL")
		defaultModel = "gpt-4"
	}

	// Use Python SDK style environment variable naming: EMBEDDING_*
	embedderProvider := getEnvOrDefault("EMBEDDING_PROVIDER", "qwen")
	embedderAPIKey := os.Getenv("EMBEDDING_API_KEY")
	embedderModel := os.Getenv("EMBEDDING_MODEL")

	// Set default base URL based on provider
	var embedderFinalBaseURL string
	switch embedderProvider {
	case "qwen":
		embedderFinalBaseURL = os.Getenv("QWEN_EMBEDDING_BASE_URL")
		if embedderFinalBaseURL == "" {
			embedderFinalBaseURL = "https://dashscope.aliyuncs.com/api/v1"
		}
		if embedderModel == "" {
			embedderModel = "text-embedding-v4"
		}
	case "openai":
		embedderFinalBaseURL = os.Getenv("OPENAI_EMBEDDING_BASE_URL")
		if embedderFinalBaseURL == "" {
			embedderFinalBaseURL = "https://api.openai.com/v1"
		}
		if embedderModel == "" {
			embedderModel = "text-embedding-3-small"
		}
	default:
		embedderFinalBaseURL = os.Getenv("EMBEDDING_BASE_URL")
		if embedderModel == "" {
			embedderModel = "text-embedding-3-small"
		}
	}

	config := &Config{
		LLM: LLMConfig{
			Provider: llmProvider,
			APIKey:   os.Getenv("LLM_API_KEY"),
			Model:    getEnvOrDefault("LLM_MODEL", defaultModel),
			BaseURL:  llmBaseURL,
		},
		Embedder: EmbedderConfig{
			Provider: embedderProvider,
			APIKey:   embedderAPIKey,
			Model:    embedderModel,
			BaseURL:  embedderFinalBaseURL,
		},
		VectorStore: VectorStoreConfig{
			Provider: provider,
			Config:   vectorStoreConfig,
		},
	}

	// Intelligent memory configuration (optional)
	if os.Getenv("INTELLIGENCE_ENABLED") == "true" {
		config.Intelligence = &IntelligenceConfig{
			Enabled:             true,
			DecayRate:           0.1,
			ReinforcementFactor: 0.3,
			DuplicateThreshold:  0.95,
			WorkingThreshold:    0.3,
			ShortTermThreshold:  0.6,
			LongTermThreshold:   0.8,
			InitialRetention:    1.0,
			FallbackToSimpleAdd: false,
		}
	}

	return config, nil
}

// LoadConfigFromEnvFile loads configuration from a specific .env file.
//
// Parameters:
//   - envPath: Path to the .env file
//
// Returns a Config instance, or an error if loading fails.
func LoadConfigFromEnvFile(envPath string) (*Config, error) {
	if err := godotenv.Load(envPath); err != nil {
		return nil, fmt.Errorf("failed to load .env file: %w", err)
	}
	return LoadConfigFromEnv()
}

// LoadConfigFromJSON loads configuration from a JSON file.
//
// Parameters:
//   - path: Path to the JSON configuration file
//
// Returns a Config instance, or an error if loading or parsing fails.
func LoadConfigFromJSON(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, NewMemoryError("LoadConfigFromJSON", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, NewMemoryError("LoadConfigFromJSON", err)
	}

	return &config, nil
}

// Validate validates the configuration.
//
// Checks that all required fields are set:
//   - LLM provider must be specified
//   - Embedder provider must be specified
//   - Vector store provider must be specified
//
// Returns an error if validation fails, nil otherwise.
func (c *Config) Validate() error {
	if c.LLM.Provider == "" {
		return NewMemoryError("Validate", ErrInvalidConfig)
	}
	if c.Embedder.Provider == "" {
		return NewMemoryError("Validate", ErrInvalidConfig)
	}
	if c.VectorStore.Provider == "" {
		return NewMemoryError("Validate", ErrInvalidConfig)
	}
	return nil
}

// getEnvOrDefault gets an environment variable or returns the default value.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// FindEnvFile searches for .env or .env.example files.
//
// The search:
//  1. Checks the current directory
//  2. Searches up to 5 directory levels up
//  3. Returns the first .env or .env.example file found
//
// Returns:
//   - path: Path to the found file (empty if not found)
//   - found: True if a file was found, false otherwise
func FindEnvFile() (string, bool) {
	// First check the current directory
	if _, err := os.Stat(".env"); err == nil {
		return ".env", true
	}
	if _, err := os.Stat(".env.example"); err == nil {
		return ".env.example", true
	}

	// Check project root directory (search upward)
	dir, _ := os.Getwd()
	for i := 0; i < 5; i++ {
		envPath := filepath.Join(dir, ".env")
		envExamplePath := filepath.Join(dir, ".env.example")

		if _, err := os.Stat(envPath); err == nil {
			return envPath, true
		}
		if _, err := os.Stat(envExamplePath); err == nil {
			return envExamplePath, true
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", false
}
