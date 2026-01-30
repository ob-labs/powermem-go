package main

import (
	"context"
	"fmt"
	"log"

	powermem "github.com/oceanbase/powermem-go/pkg/core"
)

func main() {
	fmt.Println("=" + repeat("=", 59))
	fmt.Println("PowerMem Go SDK - Streaming Support Example")
	fmt.Println("=" + repeat("=", 59))

	// Load configuration
	config, err := powermem.LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Override test database path
	if config.VectorStore.Provider == "sqlite" {
		if config.VectorStore.Config == nil {
			config.VectorStore.Config = make(map[string]interface{})
		}
		config.VectorStore.Config["db_path"] = "./streaming_example.db"
	}

	// Create client
	client, err := powermem.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.Printf("Warning: failed to close client: %v", err)
		}
	}()

	ctx := context.Background()
	userID := "user_streaming_001"

	// Clean up any existing memories for this user (optional, for demo purposes)
	fmt.Println("\nCleaning up existing memories...")
	err = client.DeleteAll(ctx, powermem.WithUserIDForDeleteAll(userID))
	if err != nil {
		log.Printf("Warning: failed to clean up: %v", err)
	} else {
		fmt.Println("✓ Cleanup completed!")
	}

	// Example 1: Streaming Search
	fmt.Println("\n=== Example 1: Streaming Search ===")

	// Add some memories first
	contents := []string{
		"User likes Python programming and machine learning",
		"User prefers email communication over phone calls",
		"User works in tech industry as a software engineer",
		"User enjoys reading science fiction books",
		"User loves hiking on weekends in the mountains",
		"User plays guitar and enjoys music",
		"User travels frequently to different countries",
		"User speaks multiple languages including English and Chinese",
	}

	fmt.Println("Adding memories...")
	for _, content := range contents {
		_, err := client.Add(ctx, content, powermem.WithUserID(userID))
		if err != nil {
			log.Printf("Warning: Failed to add memory: %v", err)
		}
	}
	fmt.Println("✓ Memories added")

	// Perform streaming search
	fmt.Println("\nPerforming streaming search (batch size: 3)...")
	resultChan := client.SearchStream(ctx, "programming", 3, // batch size 3
		powermem.WithUserIDForSearch("user_streaming_001"),
		powermem.WithLimit(10),
	)

	totalReceived := 0
	batchCount := 0
	for result := range resultChan {
		if result.Error != nil {
			log.Fatalf("SearchStream error: %v", result.Error)
		}

		fmt.Printf("  Batch %d: Received %d memories (Last: %v)\n",
			result.BatchIndex, len(result.Memories), result.IsLastBatch)
		totalReceived += len(result.Memories)
		batchCount++

		for i, mem := range result.Memories {
			fmt.Printf("    [%d] %s (Score: %.3f)\n", i+1, mem.Content, mem.Score)
		}
	}

	fmt.Printf("✓ Streaming search completed: %d batches, %d total memories\n", batchCount, totalReceived)

	// Example 2: Streaming GetAll
	fmt.Println("\n=== Example 2: Streaming GetAll ===")

	fmt.Println("Performing streaming GetAll (batch size: 2)...")
	getAllChan := client.GetAllStream(ctx, 2, // batch size 2
		powermem.WithUserIDForGetAll("user_streaming_001"),
		powermem.WithLimitForGetAll(10),
	)

	totalGetAll := 0
	getAllBatchCount := 0
	for result := range getAllChan {
		if result.Error != nil {
			log.Fatalf("GetAllStream error: %v", result.Error)
		}

		fmt.Printf("  Batch %d: Received %d memories (Last: %v)\n",
			result.BatchIndex, len(result.Memories), result.IsLastBatch)
		totalGetAll += len(result.Memories)
		getAllBatchCount++

		for i, mem := range result.Memories {
			fmt.Printf("    [%d] ID: %d, Content: %s\n", i+1, mem.ID, mem.Content)
		}
	}

	fmt.Printf("✓ Streaming GetAll completed: %d batches, %d total memories\n", getAllBatchCount, totalGetAll)

	// Example 3: Batch Add
	fmt.Println("\n=== Example 3: Batch Add ===")

	batchContents := []string{
		"Batch memory 1: User likes Go programming",
		"Batch memory 2: User prefers async operations",
		"Batch memory 3: User works with distributed systems",
		"Batch memory 4: User enjoys open source contributions",
		"Batch memory 5: User loves clean code practices",
	}

	fmt.Println("Adding memories in batch...")
	batchResult, err := client.BatchAdd(ctx, batchContents,
		powermem.WithUserID("user_batch_001"),
	)
	if err != nil {
		log.Fatalf("BatchAdd error: %v", err)
	}

	fmt.Printf("✓ Batch add completed:\n")
	fmt.Printf("  Total: %d\n", batchResult.Total)
	fmt.Printf("  Created: %d\n", batchResult.CreatedCount)
	fmt.Printf("  Failed: %d\n", batchResult.FailedCount)

	if len(batchResult.Failed) > 0 {
		fmt.Println("\n  Failed items:")
		for _, failed := range batchResult.Failed {
			fmt.Printf("    Index %d: %s - %v\n", failed.Index, failed.Content, failed.Error)
		}
	}

	// Example 4: Batch Update
	fmt.Println("\n=== Example 4: Batch Update ===")

	// Get some memory IDs first
	allMemories, err := client.GetAll(ctx,
		powermem.WithUserIDForGetAll("user_batch_001"),
		powermem.WithLimitForGetAll(5),
	)
	if err != nil || len(allMemories) == 0 {
		fmt.Println("  No memories to update, skipping...")
	} else {
		updateItems := make([]powermem.BatchUpdateItem, 0, len(allMemories))
		for i, mem := range allMemories {
			updateItems = append(updateItems, powermem.BatchUpdateItem{
				ID:      mem.ID,
				Content: fmt.Sprintf("Updated: %s (updated %d)", mem.Content, i+1),
			})
		}

		fmt.Println("Updating memories in batch...")
		updateResult, err := client.BatchUpdate(ctx, updateItems)
		if err != nil {
			log.Fatalf("BatchUpdate error: %v", err)
		}

		fmt.Printf("✓ Batch update completed:\n")
		fmt.Printf("  Total: %d\n", updateResult.Total)
		fmt.Printf("  Updated: %d\n", updateResult.UpdatedCount)
		fmt.Printf("  Failed: %d\n", updateResult.FailedCount)
	}

	// Example 5: Batch Delete
	fmt.Println("\n=== Example 5: Batch Delete ===")

	// Get memory IDs to delete
	memoriesToDelete, err := client.GetAll(ctx,
		powermem.WithUserIDForGetAll("user_batch_001"),
		powermem.WithLimitForGetAll(3),
	)
	if err != nil || len(memoriesToDelete) == 0 {
		fmt.Println("  No memories to delete, skipping...")
	} else {
		idsToDelete := make([]int64, 0, len(memoriesToDelete))
		for _, mem := range memoriesToDelete {
			idsToDelete = append(idsToDelete, mem.ID)
		}

		fmt.Printf("Deleting %d memories in batch...\n", len(idsToDelete))
		deleteResult, err := client.BatchDelete(ctx, idsToDelete)
		if err != nil {
			log.Fatalf("BatchDelete error: %v", err)
		}

		fmt.Printf("✓ Batch delete completed:\n")
		fmt.Printf("  Total: %d\n", deleteResult.Total)
		fmt.Printf("  Deleted: %d\n", deleteResult.DeletedCount)
		fmt.Printf("  Failed: %d\n", deleteResult.FailedCount)
	}

	fmt.Println("\n✓ All streaming examples completed!")
}

func repeat(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}
