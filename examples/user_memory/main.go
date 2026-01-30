package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/oceanbase/powermem-go/pkg/core"
	usermemory "github.com/oceanbase/powermem-go/pkg/user_memory"
	queryrewrite "github.com/oceanbase/powermem-go/pkg/user_memory/query_rewrite"
	usermemorySQLite "github.com/oceanbase/powermem-go/pkg/user_memory/sqlite"
)

func main() {
	// Load configuration from environment variables or config file
	memoryConfig, err := core.LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("Failed to load config from env: %v", err)
	}

	// Override example database path
	if memoryConfig.VectorStore.Provider == "sqlite" {
		if memoryConfig.VectorStore.Config == nil {
			memoryConfig.VectorStore.Config = make(map[string]interface{})
		}
		memoryConfig.VectorStore.Config["db_path"] = "./usermemory_example.db"
		memoryConfig.VectorStore.Config["collection_name"] = "memories"
	}

	// Create UserMemory configuration (optional: enable query rewrite)
	queryRewriteEnabled := os.Getenv("QUERY_REWRITE_ENABLED") == "true"
	var queryRewriteConfig *queryrewrite.Config
	if queryRewriteEnabled {
		customPrompt := os.Getenv("QUERY_REWRITE_PROMPT")
		queryRewriteConfig = &queryrewrite.Config{
			Enabled:            true,
			CustomInstructions: customPrompt,
		}
	}

	userMemoryConfig := &usermemory.Config{
		MemoryConfig:     memoryConfig,
		ProfileStoreType: "sqlite",
		ProfileStoreConfig: &usermemorySQLite.Config{
			DBPath:    "./user_profiles_example.db",
			TableName: "user_profiles",
		},
		QueryRewriteConfig: queryRewriteConfig,
	}

	// Create UserMemory client
	client, err := usermemory.NewClient(userMemoryConfig)
	if err != nil {
		log.Fatalf("Failed to create UserMemory client: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.Printf("Warning: failed to close client: %v", err)
		}
	}()

	ctx := context.Background()
	userID := "alice"

	// Clean up any existing memories for this user (optional, for demo purposes)
	fmt.Println("\nCleaning up existing memories...")
	err = client.DeleteAll(ctx, usermemory.WithDeleteAllUserID(userID))
	if err != nil {
		log.Printf("Warning: failed to clean up: %v", err)
	} else {
		fmt.Println("✓ Cleanup completed!")
	}

	// Example 1: Add conversation and extract user profile
	fmt.Println("\n=== Example 1: Add conversation and extract user profile ===")
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
	if err != nil {
		log.Fatalf("Failed to add conversation: %v", err)
	}

	fmt.Printf("✓ Conversation added, Memory ID: %d\n", result.Memory.ID)
	fmt.Printf("✓ Profile extraction status: %v\n", result.ProfileExtracted)
	if result.ProfileContent != nil {
		fmt.Printf("✓ Extracted profile content: %s\n", *result.ProfileContent)
	}

	// Example 2: Get user profile
	fmt.Println("\n=== Example 2: Get user profile ===")
	profile, err := client.GetProfile(ctx, "user_001")
	if err != nil {
		log.Fatalf("Failed to get profile: %v", err)
	}

	if profile != nil {
		fmt.Printf("✓ User ID: %s\n", profile.UserID)
		fmt.Printf("✓ Profile content: %s\n", profile.ProfileContent)
		fmt.Printf("✓ Created at: %s\n", profile.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("✓ Updated at: %s\n", profile.UpdatedAt.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Println("User profile not found")
	}

	// Example 3: Update user profile (by adding new conversation)
	fmt.Println("\n=== Example 3: Update user profile ===")
	newConversation := []map[string]interface{}{
		{
			"role":    "user",
			"content": "I also enjoy reading science fiction books and playing guitar in my spare time.",
		},
	}

	updateResult, err := client.Add(ctx, newConversation,
		usermemory.WithUserID("user_001"),
	)
	if err != nil {
		log.Fatalf("Failed to update profile: %v", err)
	}

	fmt.Printf("✓ Profile update status: %v\n", updateResult.ProfileExtracted)

	// Get updated profile
	updatedProfile, err := client.GetProfile(ctx, "user_001")
	if err != nil {
		log.Fatalf("Failed to get updated profile: %v", err)
	}

	if updatedProfile != nil {
		fmt.Printf("✓ Updated profile content: %s\n", updatedProfile.ProfileContent)
	}

	// Example 4: Search memories
	fmt.Println("\n=== Example 4: Search memories ===")
	searchResult, err := client.Search(ctx, "programming",
		usermemory.WithSearchUserID("user_001"),
		usermemory.WithSearchLimit(5),
	)
	if err != nil {
		log.Fatalf("Failed to search: %v", err)
	}

	fmt.Printf("✓ Found %d related memories\n", len(searchResult.Memories))
	for i, mem := range searchResult.Memories {
		fmt.Printf("  %d. %s (ID: %d)\n", i+1, mem.Content, mem.ID)
	}

	// Example 5: Get single memory
	fmt.Println("\n=== Example 5: Get single memory ===")
	if len(searchResult.Memories) > 0 {
		memoryID := searchResult.Memories[0].ID
		memory, err := client.Get(ctx, memoryID)
		if err != nil {
			log.Fatalf("Failed to get memory: %v", err)
		}
		fmt.Printf("✓ Got memory ID: %d\n", memory.ID)
		fmt.Printf("  Content: %s\n", memory.Content)
	}

	// Example 6: Update memory
	fmt.Println("\n=== Example 6: Update memory ===")
	if len(searchResult.Memories) > 0 {
		memoryID := searchResult.Memories[0].ID
		updatedContent := "I'm Alice, a software engineer. I love Python and Go programming, and I enjoy reading science fiction."
		updated, err := client.Update(ctx, memoryID, updatedContent)
		if err != nil {
			log.Fatalf("Failed to update memory: %v", err)
		}
		fmt.Printf("✓ Memory updated (ID: %d)\n", updated.ID)
		fmt.Printf("  New content: %s\n", updated.Content)
	}

	// Example 7: Query rewrite feature (if enabled)
	if queryRewriteEnabled {
		fmt.Println("\n=== Example 7: Query rewrite feature ===")
		// Use vague query, should be rewritten to more specific query
		rewriteResult, err := client.Search(ctx, "my work",
			usermemory.WithSearchUserID("user_001"),
			usermemory.WithSearchLimit(5),
			usermemory.WithAddProfile(true), // Also get user profile
		)
		if err != nil {
			log.Fatalf("Failed to search with query rewrite: %v", err)
		}
		fmt.Printf("✓ Search results after query rewrite: Found %d memories\n", len(rewriteResult.Memories))
		if rewriteResult.ProfileContent != nil {
			fmt.Printf("✓ User profile included in results\n")
		}
	}

	// Example 8: Get all memories
	fmt.Println("\n=== Example 8: Get all memories ===")
	allMemories, err := client.GetAll(ctx,
		usermemory.WithGetAllUserID("user_001"),
		usermemory.WithGetAllLimit(10),
	)
	if err != nil {
		log.Fatalf("Failed to get all memories: %v", err)
	}
	fmt.Printf("✓ Got %d memories\n", len(allMemories))

	// Example 8: Delete memory (without deleting profile)
	fmt.Println("\n=== Example 8: Delete memory ===")
	if len(allMemories) > 0 {
		memoryID := allMemories[0].ID
		err = client.Delete(ctx, memoryID)
		if err != nil {
			log.Fatalf("Failed to delete memory: %v", err)
		}
		fmt.Printf("✓ Memory deleted (ID: %d)\n", memoryID)

		// Verify deletion
		_, err = client.Get(ctx, memoryID)
		if err != nil {
			fmt.Println("✓ Verification: Memory successfully deleted")
		}
	}

	// Example 9: Delete all memories (optionally delete profile too)
	fmt.Println("\n=== Example 9: Delete all memories ===")
	// Note: Not actually deleting here, just demonstrating usage
	fmt.Println("  Usage example:")
	fmt.Println("  client.DeleteAll(ctx,")
	fmt.Println("    usermemory.WithDeleteAllUserID(\"user_001\"),")
	fmt.Println("    usermemory.WithDeleteAllProfile(true), // Also delete profile")
	fmt.Println("  )")

	// Example 10: Reset storage
	fmt.Println("\n=== Example 10: Reset storage ===")
	fmt.Println("  Usage example:")
	fmt.Println("  client.Reset(ctx) // Delete all memories")
	fmt.Println("  Note: Reset will delete all memories, but not user profiles")

	fmt.Println("\n✓ All examples completed!")
}
