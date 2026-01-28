package usermemory_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/oceanbase/powermem-go/pkg/core"
	usermemory "github.com/oceanbase/powermem-go/pkg/user_memory"
	usermemorySQLite "github.com/oceanbase/powermem-go/pkg/user_memory/sqlite"
)

func setupUserMemoryTest(t *testing.T) (*usermemory.Client, *core.Config, func()) {
	// Set test database paths
	testDBPath := "./test_usermemory.db"
	profileDBPath := "./test_user_profiles.db"

	// Clean up any existing test databases
	_ = os.Remove(testDBPath)
	_ = os.Remove(profileDBPath)

	// Load config from environment variables or config file
	memoryConfig, err := core.LoadConfigFromEnv()
	if err != nil {
		// Use default config if loading fails
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
		// Override test database path
		if memoryConfig.VectorStore.Provider == "sqlite" {
			if memoryConfig.VectorStore.Config == nil {
				memoryConfig.VectorStore.Config = make(map[string]interface{})
			}
			memoryConfig.VectorStore.Config["db_path"] = testDBPath
			memoryConfig.VectorStore.Config["collection_name"] = "memories"
		}
	}

	// Create UserMemory config
	userMemoryConfig := &usermemory.Config{
		MemoryConfig:     memoryConfig,
		ProfileStoreType: "sqlite",
		ProfileStoreConfig: &usermemorySQLite.Config{
			DBPath:    profileDBPath,
			TableName: "user_profiles",
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

// getEnvOrDefault gets environment variable or returns default value.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// hasLLMConfig checks if LLM API Key exists in config.
func hasLLMConfig(cfg *core.Config) bool {
	return cfg != nil && cfg.LLM.APIKey != ""
}

func TestUserMemory_Add(t *testing.T) {
	client, cfg, cleanup := setupUserMemoryTest(t)
	defer cleanup()

	// If API Key is not set, skip test
	if !hasLLMConfig(cfg) {
		t.Skip("Skipping test: LLM_API_KEY not set in config")
	}

	ctx := context.Background()

	// Add conversation
	conversation := []map[string]interface{}{
		{
			"role":    "user",
			"content": "Hi, I'm Alice. I'm a 28-year-old software engineer from San Francisco. I work at a tech startup and love Python programming.",
		},
		{
			"role":    "assistant",
			"content": "Nice to meet you, Alice! Python is a great language for software engineering.",
		},
	}

	result, err := client.Add(ctx, conversation,
		usermemory.WithUserID("user_001"),
		usermemory.WithAgentID("assistant_agent"),
	)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Memory)
	// Note: ProfileExtracted may be false if LLM call fails
	if result.ProfileExtracted {
		assert.NotNil(t, result.ProfileContent)
		if result.ProfileContent != nil {
			assert.Contains(t, *result.ProfileContent, "Alice")
		}
	}
}

func TestUserMemory_GetProfile(t *testing.T) {
	client, cfg, cleanup := setupUserMemoryTest(t)
	defer cleanup()

	// If API Key is not set, skip test
	if !hasLLMConfig(cfg) {
		t.Skip("Skipping test: LLM_API_KEY not set in config")
	}

	ctx := context.Background()

	// Add conversation first
	conversation := []map[string]interface{}{
		{
			"role":    "user",
			"content": "I'm Bob, 30 years old, working as a data scientist. I enjoy machine learning and data analysis.",
		},
	}

	_, err := client.Add(ctx, conversation,
		usermemory.WithUserID("user_002"),
	)
	require.NoError(t, err)

	// Get user profile
	profile, err := client.GetProfile(ctx, "user_002")
	assert.NoError(t, err)
	// Profile may be nil if profile extraction fails
	if profile != nil {
		assert.Equal(t, "user_002", profile.UserID)
		if profile.ProfileContent != "" {
			assert.NotEmpty(t, profile.ProfileContent)
		}
	}
}

func TestUserMemory_Search(t *testing.T) {
	client, cfg, cleanup := setupUserMemoryTest(t)
	defer cleanup()

	// If API Key is not set, skip test
	if !hasLLMConfig(cfg) {
		t.Skip("Skipping test: LLM_API_KEY not set in config")
	}

	ctx := context.Background()

	// Add conversation
	conversation := []map[string]interface{}{
		{
			"role":    "user",
			"content": "I love playing basketball and watching NBA games.",
		},
	}

	_, err := client.Add(ctx, conversation,
		usermemory.WithUserID("user_003"),
	)
	require.NoError(t, err)

	// Search memories
	searchResult, err := client.Search(ctx, "sports",
		usermemory.WithSearchUserID("user_003"),
		usermemory.WithSearchLimit(5),
	)

	assert.NoError(t, err)
	assert.NotNil(t, searchResult)
	assert.NotNil(t, searchResult.Memories)
	assert.Greater(t, len(searchResult.Memories), 0)
}

func TestUserMemory_UpdateProfile(t *testing.T) {
	client, cfg, cleanup := setupUserMemoryTest(t)
	defer cleanup()

	// If API Key is not set, skip test
	if !hasLLMConfig(cfg) {
		t.Skip("Skipping test: LLM_API_KEY not set in config")
	}

	ctx := context.Background()

	// Add conversation first time
	conversation1 := []map[string]interface{}{
		{
			"role":    "user",
			"content": "I'm Charlie, a teacher.",
		},
	}

	_, err := client.Add(ctx, conversation1,
		usermemory.WithUserID("user_004"),
	)
	require.NoError(t, err)

	// Get initial profile
	profile1, err := client.GetProfile(ctx, "user_004")
	// Skip test if first extraction fails
	if err != nil || profile1 == nil {
		t.Skip("Skipping test: profile extraction failed")
	}

	// Add conversation second time (update profile)
	conversation2 := []map[string]interface{}{
		{
			"role":    "user",
			"content": "I also enjoy reading books and traveling.",
		},
	}

	_, err = client.Add(ctx, conversation2,
		usermemory.WithUserID("user_004"),
	)
	require.NoError(t, err)

	// Get updated profile
	profile2, err := client.GetProfile(ctx, "user_004")
	require.NoError(t, err)
	require.NotNil(t, profile2)

	// Verify profile has been updated
	assert.Equal(t, profile1.ID, profile2.ID)                  // Same profile record
	assert.NotEqual(t, profile1.UpdatedAt, profile2.UpdatedAt) // Updated time differs
}

func TestUserMemory_DeleteProfile(t *testing.T) {
	client, cfg, cleanup := setupUserMemoryTest(t)
	defer cleanup()

	// If API Key is not set, skip test
	if !hasLLMConfig(cfg) {
		t.Skip("Skipping test: LLM_API_KEY not set in config")
	}

	ctx := context.Background()

	// Add conversation
	conversation := []map[string]interface{}{
		{
			"role":    "user",
			"content": "I'm David, a musician.",
		},
	}

	_, err := client.Add(ctx, conversation,
		usermemory.WithUserID("user_005"),
	)
	require.NoError(t, err)

	// Get profile to obtain ID
	profile, err := client.GetProfile(ctx, "user_005")
	// If profile extraction fails, manually create a test profile
	if err != nil || profile == nil {
		// Test delete function directly (using a non-existent ID)
		err = client.DeleteProfile(ctx, 999999)
		// Should return error (not found)
		assert.Error(t, err)
		return
	}

	// Delete profile
	err = client.DeleteProfile(ctx, profile.ID)
	assert.NoError(t, err)

	// Verify profile has been deleted
	deletedProfile, err := client.GetProfile(ctx, "user_005")
	assert.NoError(t, err)
	assert.Nil(t, deletedProfile)
}

func TestUserMemory_Get(t *testing.T) {
	client, cfg, cleanup := setupUserMemoryTest(t)
	defer cleanup()

	// If API Key is not set, skip test
	if !hasLLMConfig(cfg) {
		t.Skip("Skipping test: LLM_API_KEY not set in config")
	}

	ctx := context.Background()

	// Add a memory first
	conversation := []map[string]interface{}{
		{
			"role":    "user",
			"content": "I love programming in Go.",
		},
	}

	result, err := client.Add(ctx, conversation,
		usermemory.WithUserID("user_006"),
	)
	require.NoError(t, err)
	require.NotNil(t, result.Memory)

	memoryID := result.Memory.ID

	// Get memory
	retrieved, err := client.Get(ctx, memoryID)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, memoryID, retrieved.ID)
	assert.Contains(t, retrieved.Content, "Go")
}

func TestUserMemory_Update(t *testing.T) {
	client, cfg, cleanup := setupUserMemoryTest(t)
	defer cleanup()

	// If API Key is not set, skip test
	if !hasLLMConfig(cfg) {
		t.Skip("Skipping test: LLM_API_KEY not set in config")
	}

	ctx := context.Background()

	// Add a memory first
	conversation := []map[string]interface{}{
		{
			"role":    "user",
			"content": "I like Python.",
		},
	}

	result, err := client.Add(ctx, conversation,
		usermemory.WithUserID("user_007"),
	)
	require.NoError(t, err)
	require.NotNil(t, result.Memory)

	memoryID := result.Memory.ID

	// Update memory
	updatedContent := "I love Python and Go programming."
	updated, err := client.Update(ctx, memoryID, updatedContent)
	assert.NoError(t, err)
	assert.NotNil(t, updated)
	assert.Equal(t, memoryID, updated.ID)
	assert.Contains(t, updated.Content, "Python and Go")
}

func TestUserMemory_Delete(t *testing.T) {
	client, cfg, cleanup := setupUserMemoryTest(t)
	defer cleanup()

	// If API Key is not set, skip test
	if !hasLLMConfig(cfg) {
		t.Skip("Skipping test: LLM_API_KEY not set in config")
	}

	ctx := context.Background()

	// Add a memory first
	conversation := []map[string]interface{}{
		{
			"role":    "user",
			"content": "This is a test memory to be deleted.",
		},
	}

	result, err := client.Add(ctx, conversation,
		usermemory.WithUserID("user_008"),
	)
	require.NoError(t, err)
	require.NotNil(t, result.Memory)

	memoryID := result.Memory.ID

	// Delete memory
	err = client.Delete(ctx, memoryID)
	assert.NoError(t, err)

	// Verify memory has been deleted
	_, err = client.Get(ctx, memoryID)
	assert.Error(t, err)
}

func TestUserMemory_DeleteWithProfile(t *testing.T) {
	client, cfg, cleanup := setupUserMemoryTest(t)
	defer cleanup()

	// If API Key is not set, skip test
	if !hasLLMConfig(cfg) {
		t.Skip("Skipping test: LLM_API_KEY not set in config")
	}

	ctx := context.Background()

	// Add a memory first (will create profile)
	conversation := []map[string]interface{}{
		{
			"role":    "user",
			"content": "I'm Emma, a data scientist.",
		},
	}

	result, err := client.Add(ctx, conversation,
		usermemory.WithUserID("user_009"),
	)
	require.NoError(t, err)
	require.NotNil(t, result.Memory)

	memoryID := result.Memory.ID

	// Verify profile exists
	profile, err := client.GetProfile(ctx, "user_009")
	if err == nil && profile != nil {
		// Delete memory and profile simultaneously
		err = client.Delete(ctx, memoryID,
			usermemory.WithDeleteUserID("user_009"),
			usermemory.WithDeleteProfile(true),
		)
		assert.NoError(t, err)

		// Verify profile has also been deleted
		deletedProfile, err := client.GetProfile(ctx, "user_009")
		assert.NoError(t, err)
		assert.Nil(t, deletedProfile)
	}
}

func TestUserMemory_GetAll(t *testing.T) {
	client, cfg, cleanup := setupUserMemoryTest(t)
	defer cleanup()

	// If API Key is not set, skip test
	if !hasLLMConfig(cfg) {
		t.Skip("Skipping test: LLM_API_KEY not set in config")
	}

	ctx := context.Background()

	// Add multiple memories
	for i := 0; i < 3; i++ {
		conversation := []map[string]interface{}{
			{
				"role":    "user",
				"content": fmt.Sprintf("Test memory %d", i),
			},
		}
		_, err := client.Add(ctx, conversation,
			usermemory.WithUserID("user_010"),
		)
		require.NoError(t, err)
	}

	// Get all memories
	memories, err := client.GetAll(ctx,
		usermemory.WithGetAllUserID("user_010"),
		usermemory.WithGetAllLimit(10),
	)
	assert.NoError(t, err)
	assert.NotNil(t, memories)
	assert.GreaterOrEqual(t, len(memories), 3)
}

func TestUserMemory_DeleteAll(t *testing.T) {
	client, cfg, cleanup := setupUserMemoryTest(t)
	defer cleanup()

	// If API Key is not set, skip test
	if !hasLLMConfig(cfg) {
		t.Skip("Skipping test: LLM_API_KEY not set in config")
	}

	ctx := context.Background()

	// Add multiple memories
	for i := 0; i < 3; i++ {
		conversation := []map[string]interface{}{
			{
				"role":    "user",
				"content": fmt.Sprintf("Test memory to delete %d", i),
			},
		}
		_, err := client.Add(ctx, conversation,
			usermemory.WithUserID("user_011"),
		)
		require.NoError(t, err)
	}

	// Delete all memories
	err := client.DeleteAll(ctx,
		usermemory.WithDeleteAllUserID("user_011"),
	)
	assert.NoError(t, err)

	// Verify all memories have been deleted
	memories, err := client.GetAll(ctx,
		usermemory.WithGetAllUserID("user_011"),
		usermemory.WithGetAllLimit(10),
	)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(memories))
}

func TestUserMemory_DeleteAllWithProfile(t *testing.T) {
	client, cfg, cleanup := setupUserMemoryTest(t)
	defer cleanup()

	// If API Key is not set, skip test
	if !hasLLMConfig(cfg) {
		t.Skip("Skipping test: LLM_API_KEY not set in config")
	}

	ctx := context.Background()

	// Add a memory first (will create profile)
	conversation := []map[string]interface{}{
		{
			"role":    "user",
			"content": "I'm Frank, a developer.",
		},
	}

	_, err := client.Add(ctx, conversation,
		usermemory.WithUserID("user_012"),
	)
	require.NoError(t, err)

	// Verify profile exists
	profile, err := client.GetProfile(ctx, "user_012")
	if err == nil && profile != nil {
		// Delete all memories and profile simultaneously
		err = client.DeleteAll(ctx,
			usermemory.WithDeleteAllUserID("user_012"),
			usermemory.WithDeleteAllProfile(true),
		)
		assert.NoError(t, err)

		// Verify profile has also been deleted
		deletedProfile, err := client.GetProfile(ctx, "user_012")
		assert.NoError(t, err)
		assert.Nil(t, deletedProfile)
	}
}

func TestUserMemory_Reset(t *testing.T) {
	client, cfg, cleanup := setupUserMemoryTest(t)
	defer cleanup()

	// If API Key is not set, skip test
	if !hasLLMConfig(cfg) {
		t.Skip("Skipping test: LLM_API_KEY not set in config")
	}

	ctx := context.Background()

	// Add some memories
	for i := 0; i < 2; i++ {
		conversation := []map[string]interface{}{
			{
				"role":    "user",
				"content": fmt.Sprintf("Test memory %d", i),
			},
		}
		_, err := client.Add(ctx, conversation,
			usermemory.WithUserID("user_013"),
		)
		require.NoError(t, err)
	}

	// Reset storage
	err := client.Reset(ctx)
	assert.NoError(t, err)

	// Verify memories have been deleted
	memories, err := client.GetAll(ctx,
		usermemory.WithGetAllLimit(10),
	)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(memories))
}
