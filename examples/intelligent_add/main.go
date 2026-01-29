package main

import (
	"context"
	"fmt"
	"log"

	powermem "github.com/oceanbase/powermem-go/pkg/core"
)

func main() {
	fmt.Println(repeat("=", 80))
	fmt.Println("PowerMem Go SDK - Intelligent Add Demo")
	fmt.Println("Complete flow: Fact Extraction → Search → LLM Decision → Execute")
	fmt.Println(repeat("=", 80))

	// Find and load configuration
	envPath, found := powermem.FindEnvFile()
	if !found {
		fmt.Println("\n⚠️  No .env file found!")
		return
	}
	fmt.Printf("\n✓ Found config file: %s\n", envPath)

	config, err := powermem.LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Enable intelligent memory features
	config.Intelligence = &powermem.IntelligenceConfig{
		Enabled:             true,
		DecayRate:           0.1,
		ReinforcementFactor: 0.3,
		FallbackToSimpleAdd: false, // Don't fallback to simple add
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

	fmt.Println("✓ Client initialized with intelligent features!")

	ctx := context.Background()
	userID := "user_intelligent_demo"

	// Clean up
	fmt.Println("\nCleaning up existing memories...")
	_ = client.DeleteAll(ctx, powermem.WithUserIDForDeleteAll(userID))
	fmt.Println("✓ Cleanup completed!")

	// ========================================================================
	// SCENARIO 1: Fact Extraction from Conversation
	// ========================================================================
	fmt.Println()
	fmt.Println(repeat("=", 80))
	fmt.Println("SCENARIO 1: Fact Extraction from Conversation")
	fmt.Println(repeat("=", 80))

	messages1 := []map[string]interface{}{
		{"role": "user", "content": "Hi! I'm Alice, a 28-year-old software engineer from San Francisco."},
		{"role": "assistant", "content": "Nice to meet you, Alice!"},
		{"role": "user", "content": "I work at a tech startup and I love Python programming."},
		{"role": "assistant", "content": "That's great! Python is very popular."},
		{"role": "user", "content": "I also enjoy reading science fiction books and playing guitar."},
	}

	fmt.Println("Adding conversation...")
	result1, err := client.IntelligentAdd(ctx, messages1,
		powermem.WithUserID(userID),
	)
	if err != nil {
		log.Fatalf("Failed to add: %v", err)
	}

	fmt.Printf("\n✓ Processed conversation and performed %d operations:\n", len(result1.Results))
	for i, r := range result1.Results {
		fmt.Printf("  %d. [%s] %s\n", i+1, r.Event, truncate(r.Memory, 70))
		if r.PreviousMemory != "" {
			fmt.Printf("     Previous: %s\n", truncate(r.PreviousMemory, 70))
		}
	}

	// ========================================================================
	// SCENARIO 2: Deduplication (Should detect duplicates)
	// ========================================================================
	fmt.Println()
	fmt.Println(repeat("=", 80))
	fmt.Println("SCENARIO 2: Intelligent Deduplication")
	fmt.Println(repeat("=", 80))

	messages2 := []map[string]interface{}{
		{"role": "user", "content": "I'm Alice, a Python developer from SF who enjoys sci-fi novels."},
	}

	fmt.Println("Adding similar information (should detect duplicates)...")
	result2, err := client.IntelligentAdd(ctx, messages2,
		powermem.WithUserID(userID),
	)
	if err != nil {
		log.Fatalf("Failed to add: %v", err)
	}

	fmt.Printf("\n✓ Processed and performed %d operations:\n", len(result2.Results))
	noneCount := 0
	updateCount := 0
	for i, r := range result2.Results {
		fmt.Printf("  %d. [%s] %s\n", i+1, r.Event, truncate(r.Memory, 70))
		if r.Event == "NONE" {
			noneCount++
		}
		if r.Event == "UPDATE" {
			updateCount++
		}
	}

	if noneCount > 0 {
		fmt.Printf("\n✓ Successfully detected %d duplicate(s)\n", noneCount)
	}
	if updateCount > 0 {
		fmt.Printf("✓ Updated %d existing memory/memories\n", updateCount)
	}

	// ========================================================================
	// SCENARIO 3: Update Existing Memory
	// ========================================================================
	fmt.Println()
	fmt.Println(repeat("=", 80))
	fmt.Println("SCENARIO 3: Update Existing Memory")
	fmt.Println(repeat("=", 80))

	messages3 := []map[string]interface{}{
		{"role": "user", "content": "Actually, I'm 29 years old now, not 28. And I recently switched to Go programming."},
	}

	fmt.Println("Adding updated information...")
	result3, err := client.IntelligentAdd(ctx, messages3,
		powermem.WithUserID(userID),
	)
	if err != nil {
		log.Fatalf("Failed to add: %v", err)
	}

	fmt.Printf("\n✓ Processed and performed %d operations:\n", len(result3.Results))
	for i, r := range result3.Results {
		fmt.Printf("  %d. [%s] %s\n", i+1, r.Event, truncate(r.Memory, 70))
		if r.Event == "UPDATE" && r.PreviousMemory != "" {
			fmt.Printf("     → Updated from: %s\n", truncate(r.PreviousMemory, 70))
		}
	}

	// ========================================================================
	// SCENARIO 4: Search to verify final state
	// ========================================================================
	fmt.Println()
	fmt.Println(repeat("=", 80))
	fmt.Println("SCENARIO 4: Verify Final State")
	fmt.Println(repeat("=", 80))

	allMemories, err := client.GetAll(ctx, powermem.WithUserIDForGetAll(userID))
	if err != nil {
		log.Fatalf("Failed to get all memories: %v", err)
	}

	fmt.Printf("\n✓ Final state: %d memories stored:\n", len(allMemories))
	for i, mem := range allMemories {
		fmt.Printf("  %d. %s\n", i+1, truncate(mem.Content, 75))
	}

	// Search for specific information
	fmt.Println("\nSearching for 'programming'...")
	searchResults, err := client.Search(ctx, "programming",
		powermem.WithUserIDForSearch(userID),
		powermem.WithLimit(5),
	)
	if err != nil {
		log.Fatalf("Failed to search: %v", err)
	}

	fmt.Printf("Found %d result(s):\n", len(searchResults))
	for i, mem := range searchResults {
		fmt.Printf("  %d. %s\n", i+1, truncate(mem.Content, 75))
	}

	fmt.Println()
	fmt.Println(repeat("=", 80))
	fmt.Println("Demo completed successfully!")
	fmt.Println(repeat("=", 80))
}

func repeat(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
