package usermemory_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/oceanbase/powermem-go/pkg/core"
	usermemory "github.com/oceanbase/powermem-go/pkg/user_memory"
	queryrewrite "github.com/oceanbase/powermem-go/pkg/user_memory/query_rewrite"
	usermemorySQLite "github.com/oceanbase/powermem-go/pkg/user_memory/sqlite"
)

// setupUserMemoryTestWithQueryRewrite creates a UserMemory client with query rewrite enabled.
func setupUserMemoryTestWithQueryRewrite(t *testing.T) (*usermemory.Client, *core.Config, func()) {
	testDBPath := "./test_usermemory_queryrewrite.db"
	profileDBPath := "./test_user_profiles_queryrewrite.db"

	// Clean up test databases
	_ = os.Remove(testDBPath)
	_ = os.Remove(profileDBPath)

	// Load configuration from environment
	memoryConfig, err := core.LoadConfigFromEnv()
	if err != nil {
		memoryConfig = &core.Config{
			VectorStore: core.VectorStoreConfig{
				Provider: "sqlite",
				Config: map[string]interface{}{
					"db_path":              testDBPath,
					"collection_name":      "memories",
					"embedding_model_dims": 1536,
				},
			},
			LLM: core.LLMConfig{
				Provider: getEnvOrDefault("LLM_PROVIDER", "openai"),
				APIKey:   os.Getenv("LLM_API_KEY"),
				Model:    getEnvOrDefault("LLM_MODEL", "gpt-3.5-turbo"),
				BaseURL:  os.Getenv("LLM_BASE_URL"),
			},
			Embedder: core.EmbedderConfig{
				Provider:   getEnvOrDefault("EMBEDDING_PROVIDER", "openai"),
				APIKey:     os.Getenv("EMBEDDING_API_KEY"),
				Model:      getEnvOrDefault("EMBEDDING_MODEL", "text-embedding-ada-002"),
				Dimensions: 1536,
				BaseURL:    os.Getenv("EMBEDDING_BASE_URL"),
			},
		}
	} else {
		if memoryConfig.VectorStore.Provider == "sqlite" {
			if memoryConfig.VectorStore.Config == nil {
				memoryConfig.VectorStore.Config = make(map[string]interface{})
			}
			memoryConfig.VectorStore.Config["db_path"] = testDBPath
			memoryConfig.VectorStore.Config["collection_name"] = "memories"
		}
	}

	// Create UserMemory config with query rewrite enabled
	userMemoryConfig := &usermemory.Config{
		MemoryConfig:     memoryConfig,
		ProfileStoreType: "sqlite",
		ProfileStoreConfig: &usermemorySQLite.Config{
			DBPath:    profileDBPath,
			TableName: "user_profiles",
		},
		QueryRewriteConfig: &queryrewrite.Config{
			Enabled:            true,
			CustomInstructions: "", // Use default instructions
		},
	}

	client, err := usermemory.NewClient(userMemoryConfig)
	require.NoError(t, err)
	require.NotNil(t, client)

	cleanup := func() {
		_ = client.Close()
		_ = os.Remove(testDBPath)
		_ = os.Remove(profileDBPath)
	}

	return client, memoryConfig, cleanup
}

func TestQueryRewrite_Enabled(t *testing.T) {
	client, cfg, cleanup := setupUserMemoryTestWithQueryRewrite(t)
	defer cleanup()

	if !hasLLMConfig(cfg) {
		t.Skip("Skipping test: LLM_API_KEY not set in config")
	}

	ctx := context.Background()

	// Add conversation to create user profile
	conversation := []map[string]interface{}{
		{
			"role":    "user",
			"content": "I'm Alice, a software engineer working on Go projects. I live in San Francisco and love hiking on weekends.",
		},
	}

	_, err := client.Add(ctx, conversation, usermemory.WithUserID("user_queryrewrite_001"))
	require.NoError(t, err)

	// Verify profile was created
	profile, err := client.GetProfile(ctx, "user_queryrewrite_001")
	if err != nil || profile == nil || profile.ProfileContent == "" {
		t.Skip("Skipping test: profile extraction failed or profile content is empty")
	}

	// Search with ambiguous query - should be rewritten using profile
	searchResult, err := client.Search(ctx, "my projects",
		usermemory.WithSearchUserID("user_queryrewrite_001"),
		usermemory.WithSearchLimit(10),
	)

	assert.NoError(t, err)
	assert.NotNil(t, searchResult)
	assert.NotNil(t, searchResult.Memories)
	// Query should be rewritten to be more specific based on profile
}

func TestQueryRewrite_Disabled(t *testing.T) {
	testDBPath := "./test_usermemory_noqueryrewrite.db"
	profileDBPath := "./test_user_profiles_noqueryrewrite.db"

	_ = os.Remove(testDBPath)
	_ = os.Remove(profileDBPath)

	memoryConfig, err := core.LoadConfigFromEnv()
	if err != nil {
		memoryConfig = &core.Config{
			VectorStore: core.VectorStoreConfig{
				Provider: "sqlite",
				Config: map[string]interface{}{
					"db_path":              testDBPath,
					"collection_name":      "memories",
					"embedding_model_dims": 1536,
				},
			},
			LLM: core.LLMConfig{
				Provider: getEnvOrDefault("LLM_PROVIDER", "openai"),
				APIKey:   os.Getenv("LLM_API_KEY"),
				Model:    getEnvOrDefault("LLM_MODEL", "gpt-3.5-turbo"),
			},
			Embedder: core.EmbedderConfig{
				Provider:   getEnvOrDefault("EMBEDDING_PROVIDER", "openai"),
				APIKey:     os.Getenv("EMBEDDING_API_KEY"),
				Model:      getEnvOrDefault("EMBEDDING_MODEL", "text-embedding-ada-002"),
				Dimensions: 1536,
			},
		}
	} else {
		if memoryConfig.VectorStore.Provider == "sqlite" {
			if memoryConfig.VectorStore.Config == nil {
				memoryConfig.VectorStore.Config = make(map[string]interface{})
			}
			memoryConfig.VectorStore.Config["db_path"] = testDBPath
		}
	}

	// Create UserMemory config WITHOUT query rewrite
	userMemoryConfig := &usermemory.Config{
		MemoryConfig:     memoryConfig,
		ProfileStoreType: "sqlite",
		ProfileStoreConfig: &usermemorySQLite.Config{
			DBPath:    profileDBPath,
			TableName: "user_profiles",
		},
		// QueryRewriteConfig is nil - query rewrite disabled
	}

	client, err := usermemory.NewClient(userMemoryConfig)
	require.NoError(t, err)
	defer func() {
		_ = client.Close()
		_ = os.Remove(testDBPath)
		_ = os.Remove(profileDBPath)
	}()

	// Query rewriter should not be initialized
	// (We can't directly access private fields, but we can verify behavior)
	ctx := context.Background()

	// Search should work without query rewrite
	searchResult, err := client.Search(ctx, "test query",
		usermemory.WithSearchUserID("user_001"),
		usermemory.WithSearchLimit(10),
	)

	assert.NoError(t, err)
	assert.NotNil(t, searchResult)
}

func TestQueryRewrite_NoProfile(t *testing.T) {
	client, cfg, cleanup := setupUserMemoryTestWithQueryRewrite(t)
	defer cleanup()

	if !hasLLMConfig(cfg) {
		t.Skip("Skipping test: LLM_API_KEY not set in config")
	}

	ctx := context.Background()

	// Search without user profile - should skip rewrite and use original query
	searchResult, err := client.Search(ctx, "test query",
		usermemory.WithSearchUserID("user_no_profile"),
		usermemory.WithSearchLimit(10),
	)

	assert.NoError(t, err)
	assert.NotNil(t, searchResult)
	// Query should not be rewritten (no profile available)
}

func TestQueryRewrite_EmptyProfile(t *testing.T) {
	client, cfg, cleanup := setupUserMemoryTestWithQueryRewrite(t)
	defer cleanup()

	if !hasLLMConfig(cfg) {
		t.Skip("Skipping test: LLM_API_KEY not set in config")
	}

	ctx := context.Background()

	// Add conversation that might not generate profile content
	conversation := []map[string]interface{}{
		{
			"role":    "user",
			"content": "Hello",
		},
	}

	_, err := client.Add(ctx, conversation, usermemory.WithUserID("user_empty_profile"))
	require.NoError(t, err)

	// Search with empty profile - should skip rewrite
	searchResult, err := client.Search(ctx, "test query",
		usermemory.WithSearchUserID("user_empty_profile"),
		usermemory.WithSearchLimit(10),
	)

	assert.NoError(t, err)
	assert.NotNil(t, searchResult)
	// Query should not be rewritten (empty profile)
}

func TestQueryRewrite_ShortQuery(t *testing.T) {
	client, cfg, cleanup := setupUserMemoryTestWithQueryRewrite(t)
	defer cleanup()

	if !hasLLMConfig(cfg) {
		t.Skip("Skipping test: LLM_API_KEY not set in config")
	}

	ctx := context.Background()

	// Add conversation to create user profile
	conversation := []map[string]interface{}{
		{
			"role":    "user",
			"content": "I'm Bob, a data scientist working with Python and machine learning.",
		},
	}

	_, err := client.Add(ctx, conversation, usermemory.WithUserID("user_short_query"))
	require.NoError(t, err)

	// Search with very short query - should skip rewrite
	searchResult, err := client.Search(ctx, "hi",
		usermemory.WithSearchUserID("user_short_query"),
		usermemory.WithSearchLimit(10),
	)

	assert.NoError(t, err)
	assert.NotNil(t, searchResult)
	// Query too short, should not be rewritten
}

func TestQueryRewrite_AddProfile(t *testing.T) {
	client, cfg, cleanup := setupUserMemoryTestWithQueryRewrite(t)
	defer cleanup()

	if !hasLLMConfig(cfg) {
		t.Skip("Skipping test: LLM_API_KEY not set in config")
	}

	ctx := context.Background()

	// Add conversation to create user profile
	conversation := []map[string]interface{}{
		{
			"role":    "user",
			"content": "I'm Charlie, a teacher who loves reading and traveling.",
		},
	}

	_, err := client.Add(ctx, conversation, usermemory.WithUserID("user_addprofile_001"))
	require.NoError(t, err)

	// Search with AddProfile option - should include profile in results
	searchResult, err := client.Search(ctx, "hobbies",
		usermemory.WithSearchUserID("user_addprofile_001"),
		usermemory.WithSearchLimit(10),
		usermemory.WithAddProfile(true),
	)

	assert.NoError(t, err)
	assert.NotNil(t, searchResult)
	if searchResult.ProfileContent != nil {
		assert.NotEmpty(t, *searchResult.ProfileContent)
	}
}

func TestQueryRewrite_CustomInstructions(t *testing.T) {
	testDBPath := "./test_usermemory_custominstructions.db"
	profileDBPath := "./test_user_profiles_custominstructions.db"

	_ = os.Remove(testDBPath)
	_ = os.Remove(profileDBPath)

	memoryConfig, err := core.LoadConfigFromEnv()
	if err != nil {
		memoryConfig = &core.Config{
			VectorStore: core.VectorStoreConfig{
				Provider: "sqlite",
				Config: map[string]interface{}{
					"db_path":              testDBPath,
					"collection_name":      "memories",
					"embedding_model_dims": 1536,
				},
			},
			LLM: core.LLMConfig{
				Provider: getEnvOrDefault("LLM_PROVIDER", "openai"),
				APIKey:   os.Getenv("LLM_API_KEY"),
				Model:    getEnvOrDefault("LLM_MODEL", "gpt-3.5-turbo"),
			},
			Embedder: core.EmbedderConfig{
				Provider:   getEnvOrDefault("EMBEDDING_PROVIDER", "openai"),
				APIKey:     os.Getenv("EMBEDDING_API_KEY"),
				Model:      getEnvOrDefault("EMBEDDING_MODEL", "text-embedding-ada-002"),
				Dimensions: 1536,
			},
		}
	} else {
		if memoryConfig.VectorStore.Provider == "sqlite" {
			if memoryConfig.VectorStore.Config == nil {
				memoryConfig.VectorStore.Config = make(map[string]interface{})
			}
			memoryConfig.VectorStore.Config["db_path"] = testDBPath
		}
	}

	// Create UserMemory config with custom instructions
	userMemoryConfig := &usermemory.Config{
		MemoryConfig:     memoryConfig,
		ProfileStoreType: "sqlite",
		ProfileStoreConfig: &usermemorySQLite.Config{
			DBPath:    profileDBPath,
			TableName: "user_profiles",
		},
		QueryRewriteConfig: &queryrewrite.Config{
			Enabled:            true,
			CustomInstructions: "Make queries more specific and technical.",
		},
	}

	client, err := usermemory.NewClient(userMemoryConfig)
	require.NoError(t, err)
	defer func() {
		_ = client.Close()
		_ = os.Remove(testDBPath)
		_ = os.Remove(profileDBPath)
	}()

	if !hasLLMConfig(memoryConfig) {
		t.Skip("Skipping test: LLM_API_KEY not set in config")
	}

	ctx := context.Background()

	// Add conversation
	conversation := []map[string]interface{}{
		{
			"role":    "user",
			"content": "I'm a software engineer working on distributed systems.",
		},
	}

	_, err = client.Add(ctx, conversation, usermemory.WithUserID("user_custom_instructions"))
	require.NoError(t, err)

	// Search should use custom instructions for rewriting
	searchResult, err := client.Search(ctx, "my work",
		usermemory.WithSearchUserID("user_custom_instructions"),
		usermemory.WithSearchLimit(10),
	)

	assert.NoError(t, err)
	assert.NotNil(t, searchResult)
}
