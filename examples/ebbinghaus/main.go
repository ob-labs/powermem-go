package main

import (
	"context"
	"fmt"
	"log"
	"time"

	powermem "github.com/oceanbase/powermem-go/pkg/core"
	"github.com/oceanbase/powermem-go/pkg/intelligence"
)

func main() {
	fmt.Println(repeat("=", 80))
	fmt.Println("PowerMem Go SDK - Ebbinghaus Forgetting Curve Demo")
	fmt.Println(repeat("=", 80))

	// Find configuration file
	envPath, found := powermem.FindEnvFile()
	if !found {
		fmt.Println("\n⚠️  No .env file found!")
		fmt.Println("To add your API keys:")
		fmt.Println("   1. Copy: cp .env.example .env")
		fmt.Println("   2. Edit .env and add your API keys")
		fmt.Println("\nFor now, trying to load from environment variables...")
	} else {
		fmt.Printf("\n✓ Found config file: %s\n", envPath)
	}

	// Load configuration
	fmt.Println("\nLoading configuration...")
	config, err := powermem.LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Enable intelligent memory with Ebbinghaus curve
	if config.Intelligence == nil {
		config.Intelligence = &powermem.IntelligenceConfig{}
	}
	config.Intelligence.Enabled = true
	config.Intelligence.DecayRate = 0.1           // Slower decay for demonstration
	config.Intelligence.ReinforcementFactor = 0.3 // Moderate reinforcement

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

	fmt.Println("✓ Client initialized with Ebbinghaus forgetting curve!")

	ctx := context.Background()
	userID := "ebbinghaus_user"

	// Clean up existing memories
	fmt.Println("\nCleaning up existing memories...")
	err = client.DeleteAll(ctx, powermem.WithUserIDForDeleteAll(userID))
	if err != nil {
		log.Printf("Warning: failed to clean up: %v", err)
	} else {
		fmt.Println("✓ Cleanup completed!")
	}

	// Create Ebbinghaus manager for demonstration
	ebManager := intelligence.NewEbbinghausManager(0.1, 0.3)

	// Scenario 1: Memory Retention Over Time
	fmt.Println("\n" + repeat("=", 80))
	fmt.Println("SCENARIO 1: Memory Retention Over Time")
	fmt.Println(repeat("=", 80))

	fmt.Println("\nAdding a memory and observing retention decay...")
	memory, err := client.Add(ctx, "The Ebbinghaus forgetting curve describes how information is lost over time",
		powermem.WithUserID(userID),
	)
	if err != nil {
		log.Fatalf("Failed to add memory: %v", err)
	}
	fmt.Printf("✓ Added memory: ID=%d\n", memory.ID)

	// Simulate time intervals and calculate retention
	timeIntervals := []struct {
		hours       float64
		description string
	}{
		{0, "Immediately after learning"},
		{0.33, "20 minutes later"},
		{1, "1 hour later"},
		{9, "9 hours later"},
		{24, "1 day later"},
		{48, "2 days later"},
		{144, "6 days later"},
		{744, "31 days later"},
	}

	fmt.Println("\nRetention curve (Ebbinghaus formula):")
	fmt.Println(repeat("-", 60))
	for _, interval := range timeIntervals {
		simulatedTime := memory.CreatedAt.Add(time.Duration(interval.hours * float64(time.Hour)))
		retention := ebManager.CalculateRetention(memory.CreatedAt, &simulatedTime)
		memoryType := ebManager.ClassifyMemoryType(retention)

		fmt.Printf("%-25s | Retention: %.1f%% | Type: %s\n",
			interval.description,
			retention*100,
			memoryType,
		)
	}

	// Scenario 2: Memory Reinforcement
	fmt.Println("\n" + repeat("=", 80))
	fmt.Println("SCENARIO 2: Memory Reinforcement Through Access")
	fmt.Println(repeat("=", 80))

	fmt.Println("\nDemonstrating how accessing memories strengthens them...")

	initialRetention := 0.5
	fmt.Printf("Initial retention: %.1f%%\n", initialRetention*100)

	accessCount := 5
	currentRetention := initialRetention
	for i := 1; i <= accessCount; i++ {
		currentRetention = ebManager.Reinforce(currentRetention)
		fmt.Printf("After access #%d: %.1f%%\n", i, currentRetention*100)
	}

	// Scenario 3: Review Schedule Generation
	fmt.Println("\n" + repeat("=", 80))
	fmt.Println("SCENARIO 3: Spaced Repetition Review Schedule")
	fmt.Println(repeat("=", 80))

	fmt.Println("\nGenerating optimal review schedule based on importance...")

	importanceScores := []float64{0.3, 0.5, 0.7, 0.9}
	for _, importance := range importanceScores {
		schedule := ebManager.GenerateReviewSchedule(time.Now(), importance)
		fmt.Printf("\nImportance: %.1f\n", importance)
		fmt.Println("Review schedule:")
		for i, reviewTime := range schedule {
			duration := time.Until(reviewTime)
			fmt.Printf("  Review #%d: %.1f hours from now\n", i+1, duration.Hours())
		}
	}

	// Scenario 4: Memory Classification
	fmt.Println("\n" + repeat("=", 80))
	fmt.Println("SCENARIO 4: Memory Type Classification")
	fmt.Println(repeat("=", 80))

	fmt.Println("\nAdding memories with different characteristics...")

	memoriesWithImportance := []struct {
		content    string
		importance float64
	}{
		{"Critical deadline: Submit report by Friday", 0.9},
		{"Interesting article about AI trends", 0.5},
		{"Random thought about weekend plans", 0.2},
	}

	for _, mem := range memoriesWithImportance {
		added, err := client.Add(ctx, mem.content,
			powermem.WithUserID(userID),
			powermem.WithMetadata(map[string]interface{}{
				"importance": mem.importance,
			}),
		)
		if err != nil {
			log.Printf("Failed to add memory: %v", err)
			continue
		}

		// Calculate initial retention
		retention := ebManager.CalculateRetention(added.CreatedAt, nil)
		memType := ebManager.ClassifyMemoryType(retention)

		fmt.Printf("\n✓ Added: %s\n", mem.content)
		fmt.Printf("  Importance: %.1f | Retention: %.1f%% | Type: %s\n",
			mem.importance, retention*100, memType)
	}

	// Scenario 5: Memory Lifecycle Decisions
	fmt.Println("\n" + repeat("=", 80))
	fmt.Println("SCENARIO 5: Memory Lifecycle Management")
	fmt.Println(repeat("=", 80))

	fmt.Println("\nDemonstrating memory lifecycle decisions (promote, forget, archive)...")

	// Get all memories
	allMemories, err := client.GetAll(ctx, powermem.WithUserIDForGetAll(userID))
	if err != nil {
		log.Printf("Failed to get memories: %v", err)
	} else {
		fmt.Printf("\nAnalyzing %d memories:\n", len(allMemories))
		fmt.Println(repeat("-", 60))

		for _, mem := range allMemories {
			// Calculate current retention
			retention := ebManager.CalculateRetention(mem.CreatedAt, mem.LastAccessedAt)
			memType := ebManager.ClassifyMemoryType(retention)

			// Convert to map for decision functions
			memMap := map[string]interface{}{
				"id":               mem.ID,
				"content":          mem.Content,
				"created_at":       mem.CreatedAt,
				"last_accessed_at": mem.LastAccessedAt,
				"retention":        retention,
				"metadata":         mem.Metadata,
			}

			// Make lifecycle decisions
			shouldPromote := ebManager.ShouldPromote(memMap)
			shouldForget := ebManager.ShouldForget(memMap)
			shouldArchive := ebManager.ShouldArchive(memMap)

			fmt.Printf("\nMemory ID: %d\n", mem.ID)
			fmt.Printf("  Content: %s\n", truncate(mem.Content, 50))
			fmt.Printf("  Type: %s | Retention: %.1f%%\n", memType, retention*100)
			fmt.Printf("  Decisions: Promote=%v | Forget=%v | Archive=%v\n",
				shouldPromote, shouldForget, shouldArchive)
		}
	}

	// Scenario 6: Time-Based Search Weighting
	fmt.Println("\n" + repeat("=", 80))
	fmt.Println("SCENARIO 6: Search with Ebbinghaus Weighting")
	fmt.Println(repeat("=", 80))

	fmt.Println("\nSearching memories with time-based weighting...")
	results, err := client.Search(ctx, "memory",
		powermem.WithUserIDForSearch(userID),
		powermem.WithLimit(5),
	)
	if err != nil {
		log.Printf("Failed to search: %v", err)
	} else {
		fmt.Printf("Found %d results (ordered by relevance and recency):\n", len(results))
		for i, result := range results {
			retention := ebManager.CalculateRetention(result.CreatedAt, result.LastAccessedAt)
			fmt.Printf("\n%d. %s\n", i+1, truncate(result.Content, 60))
			fmt.Printf("   Score: %.3f | Retention: %.1f%% | Age: %.1f hours\n",
				result.Score,
				retention*100,
				time.Since(result.CreatedAt).Hours())
		}
	}

	fmt.Println("\n" + repeat("=", 80))
	fmt.Println("Demo completed successfully!")
	fmt.Println(repeat("=", 80))
}

// Helper function to repeat a string
func repeat(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}

// Helper function to truncate long strings
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
