package core_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/oceanbase/powermem-go/pkg/core"
)

// getEnvOrDefault gets an environment variable or returns a default value.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func setupStreamingTest(t *testing.T) (*core.Client, func()) {
	testDBPath := "./test_streaming.db"
	_ = os.Remove(testDBPath)

	config, err := core.LoadConfigFromEnv()
	if err != nil {
		config = &core.Config{
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
		if config.VectorStore.Provider == "sqlite" {
			if config.VectorStore.Config == nil {
				config.VectorStore.Config = make(map[string]interface{})
			}
			config.VectorStore.Config["db_path"] = testDBPath
		}
	}

	client, err := core.NewClient(config)
	require.NoError(t, err)
	require.NotNil(t, client)

	cleanup := func() {
		_ = client.Close()
		_ = os.Remove(testDBPath)
	}

	return client, cleanup
}

func TestSearchStream(t *testing.T) {
	client, cleanup := setupStreamingTest(t)
	defer cleanup()

	ctx := context.Background()

	// Add some test memories
	contents := []string{
		"User likes Python programming",
		"User prefers email communication",
		"User works in tech industry",
		"User enjoys reading books",
		"User loves hiking on weekends",
	}

	for _, content := range contents {
		_, err := client.Add(ctx, content, core.WithUserID("user_stream_001"))
		require.NoError(t, err)
	}

	// Test streaming search
	resultChan := client.SearchStream(ctx, "programming", 2, // batch size 2
		core.WithUserIDForSearch("user_stream_001"),
		core.WithLimit(10),
	)

	totalReceived := 0
	batchCount := 0
	lastBatchIndex := -1

	for result := range resultChan {
		if result.Error != nil {
			t.Fatalf("SearchStream error: %v", result.Error)
		}

		assert.NotNil(t, result.Memories)
		totalReceived += len(result.Memories)
		batchCount++

		// Verify batch index is sequential
		assert.Equal(t, lastBatchIndex+1, result.BatchIndex)
		lastBatchIndex = result.BatchIndex

		// Verify batch size (except possibly the last batch)
		if !result.IsLastBatch {
			assert.LessOrEqual(t, len(result.Memories), 2)
		}
	}

	assert.Greater(t, totalReceived, 0)
	assert.Greater(t, batchCount, 0)
}

func TestGetAllStream(t *testing.T) {
	client, cleanup := setupStreamingTest(t)
	defer cleanup()

	ctx := context.Background()

	// Add multiple test memories
	contents := []string{
		"Memory 1: User likes Python",
		"Memory 2: User prefers email",
		"Memory 3: User works in tech",
		"Memory 4: User enjoys reading",
		"Memory 5: User loves hiking",
		"Memory 6: User plays guitar",
		"Memory 7: User travels frequently",
		"Memory 8: User speaks multiple languages",
	}

	for _, content := range contents {
		_, err := client.Add(ctx, content, core.WithUserID("user_getall_stream_001"))
		require.NoError(t, err)
	}

	// Test streaming GetAll
	resultChan := client.GetAllStream(ctx, 3, // batch size 3
		core.WithUserIDForGetAll("user_getall_stream_001"),
		core.WithLimitForGetAll(10),
	)

	totalReceived := 0
	batchCount := 0
	lastBatchIndex := -1

	for result := range resultChan {
		if result.Error != nil {
			t.Fatalf("GetAllStream error: %v", result.Error)
		}

		assert.NotNil(t, result.Memories)
		totalReceived += len(result.Memories)
		batchCount++

		// Verify batch index is sequential
		assert.Equal(t, lastBatchIndex+1, result.BatchIndex)
		lastBatchIndex = result.BatchIndex

		// Verify batch size (except possibly the last batch)
		if !result.IsLastBatch {
			assert.LessOrEqual(t, len(result.Memories), 3)
		}

		// Last batch should be marked
		if result.IsLastBatch {
			assert.True(t, result.IsLastBatch)
		}
	}

	assert.GreaterOrEqual(t, totalReceived, len(contents))
	assert.Greater(t, batchCount, 0)
}

func TestBatchAdd(t *testing.T) {
	client, cleanup := setupStreamingTest(t)
	defer cleanup()

	ctx := context.Background()

	// Test batch add
	contents := []string{
		"Batch memory 1",
		"Batch memory 2",
		"Batch memory 3",
		"Batch memory 4",
		"Batch memory 5",
	}

	result, err := client.BatchAdd(ctx, contents,
		core.WithUserID("user_batch_001"),
	)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, len(contents), result.Total)
	assert.Equal(t, len(contents), result.CreatedCount)
	assert.Equal(t, 0, result.FailedCount)
	assert.Equal(t, len(contents), len(result.Created))
	assert.Equal(t, 0, len(result.Failed))

	// Verify all memories were created (order not guaranteed due to concurrent execution)
	createdContents := make(map[string]bool)
	for _, mem := range result.Created {
		assert.NotNil(t, mem)
		assert.Equal(t, "user_batch_001", mem.UserID)
		createdContents[mem.Content] = true
	}

	// Check that all contents are present
	for _, content := range contents {
		assert.True(t, createdContents[content], "Content '%s' not found in results", content)
	}
}

func TestBatchAdd_Empty(t *testing.T) {
	client, cleanup := setupStreamingTest(t)
	defer cleanup()

	ctx := context.Background()

	// Test batch add with empty slice
	result, err := client.BatchAdd(ctx, []string{})
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 0, result.Total)
	assert.Equal(t, 0, result.CreatedCount)
	assert.Equal(t, 0, result.FailedCount)
}

func TestBatchUpdate(t *testing.T) {
	client, cleanup := setupStreamingTest(t)
	defer cleanup()

	ctx := context.Background()

	// Add some memories first
	contents := []string{
		"Original content 1",
		"Original content 2",
		"Original content 3",
	}

	var memoryIDs []int64
	for _, content := range contents {
		mem, err := client.Add(ctx, content, core.WithUserID("user_batch_update_001"))
		require.NoError(t, err)
		memoryIDs = append(memoryIDs, mem.ID)
	}

	// Test batch update
	items := []core.BatchUpdateItem{
		{ID: memoryIDs[0], Content: "Updated content 1"},
		{ID: memoryIDs[1], Content: "Updated content 2"},
		{ID: memoryIDs[2], Content: "Updated content 3"},
	}

	result, err := client.BatchUpdate(ctx, items)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, len(items), result.Total)
	assert.Equal(t, len(items), result.UpdatedCount)
	assert.Equal(t, 0, result.FailedCount)
	assert.Equal(t, len(items), len(result.Updated))

	// Verify all memories were updated (order not guaranteed due to concurrent execution)
	updatedMap := make(map[int64]string)
	for _, mem := range result.Updated {
		assert.NotNil(t, mem)
		updatedMap[mem.ID] = mem.Content
	}

	// Check that all IDs are present with correct content
	for _, item := range items {
		content, exists := updatedMap[item.ID]
		assert.True(t, exists, "Memory ID %d not found in results", item.ID)
		assert.Equal(t, item.Content, content, "Content mismatch for ID %d", item.ID)
	}
}

func TestBatchUpdate_Empty(t *testing.T) {
	client, cleanup := setupStreamingTest(t)
	defer cleanup()

	ctx := context.Background()

	// Test batch update with empty slice
	result, err := client.BatchUpdate(ctx, []core.BatchUpdateItem{})
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 0, result.Total)
	assert.Equal(t, 0, result.UpdatedCount)
	assert.Equal(t, 0, result.FailedCount)
}

func TestBatchDelete(t *testing.T) {
	client, cleanup := setupStreamingTest(t)
	defer cleanup()

	ctx := context.Background()

	// Add some memories first
	contents := []string{
		"Memory to delete 1",
		"Memory to delete 2",
		"Memory to delete 3",
	}

	var memoryIDs []int64
	for _, content := range contents {
		mem, err := client.Add(ctx, content, core.WithUserID("user_batch_delete_001"))
		require.NoError(t, err)
		memoryIDs = append(memoryIDs, mem.ID)
	}

	// Test batch delete
	result, err := client.BatchDelete(ctx, memoryIDs)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, len(memoryIDs), result.Total)
	assert.Equal(t, len(memoryIDs), result.DeletedCount)
	assert.Equal(t, 0, result.FailedCount)
	assert.Equal(t, len(memoryIDs), len(result.DeletedIDs))

	// Verify all memories were deleted
	for _, id := range result.DeletedIDs {
		_, err := client.Get(ctx, id)
		assert.Error(t, err) // Should not be found
	}
}

func TestBatchDelete_Empty(t *testing.T) {
	client, cleanup := setupStreamingTest(t)
	defer cleanup()

	ctx := context.Background()

	// Test batch delete with empty slice
	result, err := client.BatchDelete(ctx, []int64{})
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 0, result.Total)
	assert.Equal(t, 0, result.DeletedCount)
	assert.Equal(t, 0, result.FailedCount)
}

func TestBatchAdd_WithFailures(t *testing.T) {
	client, cleanup := setupStreamingTest(t)
	defer cleanup()

	ctx := context.Background()

	// Test batch add with some invalid content (empty strings might fail)
	contents := []string{
		"Valid memory 1",
		"", // Empty string might cause issues
		"Valid memory 2",
	}

	result, err := client.BatchAdd(ctx, contents,
		core.WithUserID("user_batch_fail_001"),
	)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, len(contents), result.Total)
	// Some might succeed, some might fail
	assert.GreaterOrEqual(t, result.CreatedCount, 0)
	assert.LessOrEqual(t, result.CreatedCount+result.FailedCount, result.Total)
}

func TestSearchStream_ContextCancellation(t *testing.T) {
	client, cleanup := setupStreamingTest(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Add some memories
	for i := 0; i < 10; i++ {
		_, err := client.Add(ctx, "Test memory", core.WithUserID("user_cancel_001"))
		require.NoError(t, err)
	}

	// Start streaming search
	resultChan := client.SearchStream(ctx, "test", 2)

	// Cancel context after receiving first batch
	receivedFirst := false
	for result := range resultChan {
		if result.Error != nil {
			// Context cancellation error is expected
			assert.ErrorIs(t, result.Error, context.Canceled)
			break
		}
		if !receivedFirst {
			receivedFirst = true
			cancel() // Cancel after first batch
		}
	}
}

func TestGetAllStream_ContextCancellation(t *testing.T) {
	client, cleanup := setupStreamingTest(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Add some memories
	for i := 0; i < 10; i++ {
		_, err := client.Add(ctx, "Test memory", core.WithUserID("user_cancel_002"))
		require.NoError(t, err)
	}

	// Start streaming GetAll
	resultChan := client.GetAllStream(ctx, 2)

	// Cancel context after receiving first batch
	receivedFirst := false
	for result := range resultChan {
		if result.Error != nil {
			// Context cancellation error is expected
			assert.ErrorIs(t, result.Error, context.Canceled)
			break
		}
		if !receivedFirst {
			receivedFirst = true
			cancel() // Cancel after first batch
		}
	}
}
