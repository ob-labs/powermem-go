package storage_test

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/oceanbase/powermem-go/pkg/storage"
	postgresStore "github.com/oceanbase/powermem-go/pkg/storage/postgres"
)

func setupPostgresTest(t *testing.T) (storage.VectorStore, string, func()) {
	// Load .env file from project root
	envPath := filepath.Join("..", "..", ".env")
	_ = godotenv.Load(envPath)

	// Get PostgreSQL config from environment variables
	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		host = "127.0.0.1"
	}

	portStr := os.Getenv("POSTGRES_PORT")
	if portStr == "" {
		portStr = "5432"
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Skipf("Skipping PostgreSQL test: invalid POSTGRES_PORT: %s", portStr)
	}

	user := os.Getenv("POSTGRES_USER")
	if user == "" {
		user = "postgres"
	}

	password := os.Getenv("POSTGRES_PASSWORD")
	if password == "" {
		t.Skip("Skipping PostgreSQL test: POSTGRES_PASSWORD not set")
	}

	dbName := os.Getenv("POSTGRES_DATABASE")
	if dbName == "" {
		dbName = "powermem_test"
	}

	collectionName := "test_memories_" + strconv.FormatInt(int64(t.Name()[0]), 10)

	config := &postgresStore.Config{
		Host:               host,
		Port:               port,
		User:               user,
		Password:           password,
		DBName:             dbName,
		CollectionName:     collectionName,
		EmbeddingModelDims: 1536,
		SSLMode:            "disable",
	}

	store, err := postgresStore.NewClient(config)
	if err != nil {
		t.Skipf("Skipping PostgreSQL test: failed to connect: %v", err)
	}

	cleanup := func() {
		// Clean up test data
		ctx := context.Background()
		_ = store.DeleteAll(ctx, &storage.DeleteAllOptions{})
		_ = store.Close()
	}

	return store, collectionName, cleanup
}

func TestPostgresClient_Insert(t *testing.T) {
	store, _, cleanup := setupPostgresTest(t)
	defer cleanup()

	ctx := context.Background()

	memory := &storage.Memory{
		ID:                1,
		UserID:            "test_user",
		AgentID:           "test_agent",
		Content:           "Test memory content",
		Embedding:         []float64{0.1, 0.2, 0.3, 0.4, 0.5},
		Metadata:          map[string]interface{}{"key": "value"},
		RetentionStrength: 1.0,
	}

	// Ensure vector dimensions match
	embedding := make([]float64, 1536)
	copy(embedding, memory.Embedding)
	memory.Embedding = embedding

	err := store.Insert(ctx, memory)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), memory.ID)
}

func TestPostgresClient_Get(t *testing.T) {
	store, _, cleanup := setupPostgresTest(t)
	defer cleanup()

	ctx := context.Background()

	// Insert a memory first
	embedding := make([]float64, 1536)
	embedding[0] = 0.1
	embedding[1] = 0.2
	embedding[2] = 0.3

	memory := &storage.Memory{
		ID:        1,
		UserID:    "test_user",
		Content:   "Test memory content",
		Embedding: embedding,
	}

	err := store.Insert(ctx, memory)
	require.NoError(t, err)
	id := memory.ID

	// Get memory
	retrieved, err := store.Get(ctx, id)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, id, retrieved.ID)
	assert.Equal(t, "test_user", retrieved.UserID)
	assert.Equal(t, "Test memory content", retrieved.Content)
	assert.Equal(t, 1536, len(retrieved.Embedding))
}

func TestPostgresClient_Update(t *testing.T) {
	store, _, cleanup := setupPostgresTest(t)
	defer cleanup()

	ctx := context.Background()

	// Insert a memory first
	embedding := make([]float64, 1536)
	embedding[0] = 0.1
	embedding[1] = 0.2
	embedding[2] = 0.3

	memory := &storage.Memory{
		ID:        2,
		UserID:    "test_user",
		Content:   "Original content",
		Embedding: embedding,
	}

	err := store.Insert(ctx, memory)
	require.NoError(t, err)
	id := memory.ID

	// Update memory
	updatedContent := "Updated content"
	updatedEmbedding := make([]float64, 1536)
	updatedEmbedding[0] = 0.2
	updatedEmbedding[1] = 0.3
	updatedEmbedding[2] = 0.4

	updated, err := store.Update(ctx, id, updatedContent, updatedEmbedding)
	assert.NoError(t, err)
	assert.Equal(t, updatedContent, updated.Content)
	assert.Equal(t, 1536, len(updated.Embedding))
}

func TestPostgresClient_Delete(t *testing.T) {
	store, _, cleanup := setupPostgresTest(t)
	defer cleanup()

	ctx := context.Background()

	// Insert a memory first
	embedding := make([]float64, 1536)
	embedding[0] = 0.1
	embedding[1] = 0.2
	embedding[2] = 0.3

	memory := &storage.Memory{
		ID:        3,
		UserID:    "test_user",
		Content:   "Test memory content",
		Embedding: embedding,
	}

	err := store.Insert(ctx, memory)
	require.NoError(t, err)
	id := memory.ID

	// Delete memory
	err = store.Delete(ctx, id)
	assert.NoError(t, err)

	// Verify deletion
	_, err = store.Get(ctx, id)
	assert.Error(t, err)
}

func TestPostgresClient_Search(t *testing.T) {
	store, _, cleanup := setupPostgresTest(t)
	defer cleanup()

	ctx := context.Background()

	// Insert several memories
	memories := []*storage.Memory{
		{
			ID:      4,
			UserID:  "test_user",
			Content: "Python programming",
		},
		{
			ID:      5,
			UserID:  "test_user",
			Content: "Go programming",
		},
		{
			ID:      6,
			UserID:  "test_user",
			Content: "JavaScript programming",
		},
	}

	// Create 1536-dimensional vectors for each memory
	for i, mem := range memories {
		embedding := make([]float64, 1536)
		for j := 0; j < 5; j++ {
			embedding[j] = float64(i+1)*0.1 + float64(j)*0.1
		}
		mem.Embedding = embedding
		err := store.Insert(ctx, mem)
		require.NoError(t, err)
	}

	// Search
	queryVector := make([]float64, 1536)
	for j := 0; j < 5; j++ {
		queryVector[j] = 0.15 + float64(j)*0.1
	}

	options := &storage.SearchOptions{
		Limit: 2,
	}

	results, err := store.Search(ctx, queryVector, options)
	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.LessOrEqual(t, len(results), 2)

	// Verify results contain similarity scores
	if len(results) > 0 {
		assert.Greater(t, results[0].Score, 0.0)
	}
}

func TestPostgresClient_SearchWithFilters(t *testing.T) {
	store, _, cleanup := setupPostgresTest(t)
	defer cleanup()

	ctx := context.Background()

	// Insert memories for different users
	embedding1 := make([]float64, 1536)
	embedding1[0] = 0.1
	embedding1[1] = 0.2

	embedding2 := make([]float64, 1536)
	embedding2[0] = 0.1
	embedding2[1] = 0.2

	memories := []*storage.Memory{
		{
			ID:        7,
			UserID:    "user1",
			AgentID:   "agent1",
			Content:   "User1 memory",
			Embedding: embedding1,
		},
		{
			ID:        8,
			UserID:    "user2",
			AgentID:   "agent1",
			Content:   "User2 memory",
			Embedding: embedding2,
		},
	}

	for _, mem := range memories {
		err := store.Insert(ctx, mem)
		require.NoError(t, err)
	}

	// Search memories for specific user
	queryVector := make([]float64, 1536)
	queryVector[0] = 0.1
	queryVector[1] = 0.2

	options := &storage.SearchOptions{
		UserID: "user1",
		Limit:  10,
	}

	results, err := store.Search(ctx, queryVector, options)
	assert.NoError(t, err)
	assert.NotNil(t, results)
	// Should only return user1's memories
	for _, result := range results {
		assert.Equal(t, "user1", result.UserID)
	}
}

func TestPostgresClient_GetAll(t *testing.T) {
	store, _, cleanup := setupPostgresTest(t)
	defer cleanup()

	ctx := context.Background()

	// Insert several memories
	for i := 0; i < 3; i++ {
		embedding := make([]float64, 1536)
		embedding[0] = float64(i) * 0.1

		memory := &storage.Memory{
			ID:        int64(10 + i),
			UserID:    "test_user",
			Content:   "Test memory",
			Embedding: embedding,
		}
		err := store.Insert(ctx, memory)
		require.NoError(t, err)
	}

	// Get all memories
	options := &storage.GetAllOptions{
		Limit: 10,
	}

	results, err := store.GetAll(ctx, options)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 3)
}

func TestPostgresClient_GetAllWithFilters(t *testing.T) {
	store, _, cleanup := setupPostgresTest(t)
	defer cleanup()

	ctx := context.Background()

	// Insert memories for different users
	for i := 0; i < 3; i++ {
		embedding := make([]float64, 1536)
		embedding[0] = float64(i) * 0.1

		memory := &storage.Memory{
			ID:        int64(20 + i),
			UserID:    "user1",
			AgentID:   "agent1",
			Content:   "User1 memory",
			Embedding: embedding,
		}
		err := store.Insert(ctx, memory)
		require.NoError(t, err)
	}

	// Insert user2's memory
	embedding := make([]float64, 1536)
	embedding[0] = 0.5
	memory := &storage.Memory{
		ID:        30,
		UserID:    "user2",
		AgentID:   "agent1",
		Content:   "User2 memory",
		Embedding: embedding,
	}
	err := store.Insert(ctx, memory)
	require.NoError(t, err)

	// Get all memories for user1
	options := &storage.GetAllOptions{
		UserID: "user1",
		Limit:  10,
	}

	results, err := store.GetAll(ctx, options)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 3)
	// Verify all results are for user1
	for _, result := range results {
		assert.Equal(t, "user1", result.UserID)
	}
}

func TestPostgresClient_DeleteAll(t *testing.T) {
	store, _, cleanup := setupPostgresTest(t)
	defer cleanup()

	ctx := context.Background()

	// Insert several memories
	for i := 0; i < 3; i++ {
		embedding := make([]float64, 1536)
		embedding[0] = float64(i) * 0.1

		memory := &storage.Memory{
			ID:        int64(40 + i),
			UserID:    "test_user",
			Content:   "Test memory",
			Embedding: embedding,
		}
		err := store.Insert(ctx, memory)
		require.NoError(t, err)
	}

	// Delete all memories
	options := &storage.DeleteAllOptions{}
	err := store.DeleteAll(ctx, options)
	assert.NoError(t, err)

	// Verify deletion
	getOptions := &storage.GetAllOptions{Limit: 10}
	results, err := store.GetAll(ctx, getOptions)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(results))
}

func TestPostgresClient_DeleteAllWithFilters(t *testing.T) {
	store, _, cleanup := setupPostgresTest(t)
	defer cleanup()

	ctx := context.Background()

	// Insert memories for different users
	for i := 0; i < 2; i++ {
		embedding := make([]float64, 1536)
		embedding[0] = float64(i) * 0.1

		memory := &storage.Memory{
			ID:        int64(50 + i),
			UserID:    "user1",
			Content:   "User1 memory",
			Embedding: embedding,
		}
		err := store.Insert(ctx, memory)
		require.NoError(t, err)
	}

	// Insert user2's memory
	embedding := make([]float64, 1536)
	embedding[0] = 0.5
	memory := &storage.Memory{
		ID:        60,
		UserID:    "user2",
		Content:   "User2 memory",
		Embedding: embedding,
	}
	err := store.Insert(ctx, memory)
	require.NoError(t, err)

	// Delete only user1's memories
	options := &storage.DeleteAllOptions{
		UserID: "user1",
	}
	err = store.DeleteAll(ctx, options)
	assert.NoError(t, err)

	// Verify user1's memories are deleted, user2's memories remain
	getOptions := &storage.GetAllOptions{Limit: 10}
	results, err := store.GetAll(ctx, getOptions)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(results))
	assert.Equal(t, "user2", results[0].UserID)
}

func TestPostgresClient_CreateIndex(t *testing.T) {
	store, collectionName, cleanup := setupPostgresTest(t)
	defer cleanup()

	ctx := context.Background()

	// Test creating HNSW index
	hnswConfig := &storage.VectorIndexConfig{
		IndexName:   "test_hnsw_idx",
		TableName:   collectionName, // Use actual table name
		VectorField: "embedding",
		IndexType:   storage.IndexTypeHNSW,
		MetricType:  storage.MetricCosine,
		HNSWParams: &storage.HNSWParams{
			M:              16,
			EfConstruction: 64,
		},
	}

	err := store.CreateIndex(ctx, hnswConfig)
	// Index may already exist, which is normal
	if err != nil {
		// Check if it's an index already exists error
		assert.Contains(t, err.Error(), "already exists")
	}

	// Test creating IVFFlat index
	ivfConfig := &storage.VectorIndexConfig{
		IndexName:   "test_ivfflat_idx",
		TableName:   collectionName,
		VectorField: "embedding",
		IndexType:   storage.IndexTypeIVFFlat,
		MetricType:  storage.MetricCosine,
		IVFParams: &storage.IVFParams{
			Nlist: 100,
		},
	}

	err = store.CreateIndex(ctx, ivfConfig)
	// Index may already exist, which is normal
	if err != nil {
		assert.Contains(t, err.Error(), "already exists")
	}
}
