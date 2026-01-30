package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	powermem "github.com/oceanbase/powermem-go/pkg/core"
)

func main() {
	fmt.Println("=" + repeat("=", 79))
	fmt.Println("PowerMem Go SDK - Multi-Agent Collaboration Example")
	fmt.Println("=" + repeat("=", 79))

	// Find config file
	envPath, found := powermem.FindEnvFile()
	if !found {
		fmt.Println("\n⚠️  No .env file found!")
		fmt.Println("To add your API keys:")
		fmt.Println("   1. Copy: cp .env.example .env")
		fmt.Println("   2. Edit .env and add your API keys")
		fmt.Println("\nFor now, trying to load from environment variables...")
	} else {
		fmt.Printf("\n✓ Found config file: %s\n", envPath)
		if filepath.Base(envPath) == ".env.example" {
			fmt.Println("⚠️  Using .env.example file. Consider copying it to .env:")
			fmt.Println("   cp .env.example .env")
		}
	}

	// Load configuration
	fmt.Println("\nLoading configuration...")
	config, err := powermem.LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Configure multi-agent support
	config.AgentMemory = &powermem.AgentMemoryConfig{
		DefaultScope:          powermem.ScopePrivate,
		AllowCrossAgentAccess: false,
		CollaborationLevel:    "read_only",
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

	fmt.Println("✓ Client initialized with multi-agent support!")

	ctx := context.Background()
	userID := "user789"

	// Clean up any existing memories for this user (optional, for demo purposes)
	fmt.Println("\nCleaning up existing memories...")
	err = client.DeleteAll(ctx, powermem.WithUserIDForDeleteAll(userID))
	if err != nil {
		log.Printf("Warning: failed to clean up: %v", err)
	} else {
		fmt.Println("✓ Cleanup completed!")
	}

	fmt.Println("\n=== Multi-Agent Collaboration Scenario ===")

	// Scenario: Three AI agents collaborating to handle user requests
	// - Agent1: Personal Assistant (private memory)
	// - Agent2: Learning Assistant (private memory)
	// - Agent3: Task Manager (shared memory)

	// Agent1: Personal Assistant adds private information
	fmt.Println("1. Agent1 (Personal Assistant) adding private memory:")
	_, err = client.Add(ctx, "User's birthday is on March 15th",
		powermem.WithUserID(userID),
		powermem.WithAgentID("agent1_personal"),
		powermem.WithScope(powermem.ScopePrivate),
		powermem.WithMetadata(map[string]interface{}{
			"type":        "personal_info",
			"sensitivity": "high",
		}),
	)
	if err != nil {
		log.Fatalf("Failed to add memory: %v", err)
	}
	fmt.Println("   ✓ Added: User's birthday is on March 15th")

	_, err = client.Add(ctx, "User prefers email notifications",
		powermem.WithUserID(userID),
		powermem.WithAgentID("agent1_personal"),
		powermem.WithScope(powermem.ScopePrivate),
	)
	if err != nil {
		log.Fatalf("Failed to add memory: %v", err)
	}
	fmt.Println("   ✓ Added: User prefers email notifications")

	// Agent2: Learning Assistant adds learning-related memory
	fmt.Println("\n2. Agent2 (Learning Assistant) adding learning memory:")
	_, err = client.Add(ctx, "User is learning Go programming",
		powermem.WithUserID(userID),
		powermem.WithAgentID("agent2_learning"),
		powermem.WithScope(powermem.ScopePrivate),
		powermem.WithMetadata(map[string]interface{}{
			"type":  "learning",
			"topic": "programming",
		}),
	)
	if err != nil {
		log.Fatalf("Failed to add memory: %v", err)
	}
	fmt.Println("   ✓ Added: User is learning Go programming")

	_, err = client.Add(ctx, "User completed chapter 3 of Go tutorial",
		powermem.WithUserID(userID),
		powermem.WithAgentID("agent2_learning"),
		powermem.WithScope(powermem.ScopePrivate),
	)
	if err != nil {
		log.Fatalf("Failed to add memory: %v", err)
	}
	fmt.Println("   ✓ Added: User completed chapter 3 of Go tutorial")

	// Agent3: Task Manager adds shared memory
	fmt.Println("\n3. Agent3 (Task Manager) adding shared memory:")
	_, err = client.Add(ctx, "User has a meeting at 2 PM tomorrow",
		powermem.WithUserID(userID),
		powermem.WithAgentID("agent3_tasks"),
		powermem.WithScope(powermem.ScopeAgentGroup), // Agent group shared
		powermem.WithMetadata(map[string]interface{}{
			"type":     "schedule",
			"priority": "high",
		}),
	)
	if err != nil {
		log.Fatalf("Failed to add memory: %v", err)
	}
	fmt.Println("   ✓ Added: User has a meeting at 2 PM tomorrow (shared)")

	_, err = client.Add(ctx, "User's timezone is UTC+8",
		powermem.WithUserID(userID),
		powermem.WithAgentID("agent3_tasks"),
		powermem.WithScope(powermem.ScopeGlobal), // Global shared
	)
	if err != nil {
		log.Fatalf("Failed to add memory: %v", err)
	}
	fmt.Println("   ✓ Added: User's timezone is UTC+8 (global shared)")

	// Each agent searches for its visible memories
	fmt.Println("\n4. Each agent searching for memories:")

	// Agent1 searches
	fmt.Println("\n   Agent1 (Personal Assistant) searching for 'user':")
	agent1Results, err := client.Search(ctx, "user",
		powermem.WithUserIDForSearch(userID),
		powermem.WithAgentIDForSearch("agent1_personal"),
		powermem.WithLimit(10),
	)
	if err != nil {
		log.Fatalf("Failed to search: %v", err)
	}
	fmt.Printf("   Found %d memories:\n", len(agent1Results))
	for _, mem := range agent1Results {
		fmt.Printf("     - %s\n", mem.Content)
	}

	// Agent2 searches
	fmt.Println("\n   Agent2 (Learning Assistant) searching for 'learning':")
	agent2Results, err := client.Search(ctx, "learning",
		powermem.WithUserIDForSearch(userID),
		powermem.WithAgentIDForSearch("agent2_learning"),
		powermem.WithLimit(10),
	)
	if err != nil {
		log.Fatalf("Failed to search: %v", err)
	}
	fmt.Printf("   Found %d memories:\n", len(agent2Results))
	for _, mem := range agent2Results {
		fmt.Printf("     - %s\n", mem.Content)
	}

	// Agent3 searches
	fmt.Println("\n   Agent3 (Task Manager) searching for 'user':")
	agent3Results, err := client.Search(ctx, "user",
		powermem.WithUserIDForSearch(userID),
		powermem.WithAgentIDForSearch("agent3_tasks"),
		powermem.WithLimit(10),
	)
	if err != nil {
		log.Fatalf("Failed to search: %v", err)
	}
	fmt.Printf("   Found %d memories:\n", len(agent3Results))
	for _, mem := range agent3Results {
		fmt.Printf("     - %s\n", mem.Content)
	}

	// Get all memories for user (no agent restriction)
	fmt.Println("\n5. Getting all memories for user:")
	allMemories, err := client.GetAll(ctx,
		powermem.WithUserIDForGetAll(userID),
		powermem.WithLimitForGetAll(20),
	)
	if err != nil {
		log.Fatalf("Failed to get all memories: %v", err)
	}
	fmt.Printf("   User has %d total memories\n", len(allMemories))

	// Clean up specific agent's memories
	fmt.Println("\n6. Cleaning up Agent2's memories:")
	err = client.DeleteAll(ctx,
		powermem.WithUserIDForDeleteAll(userID),
		powermem.WithAgentIDForDeleteAll("agent2_learning"),
	)
	if err != nil {
		log.Fatalf("Failed to delete: %v", err)
	}
	fmt.Println("   ✓ Agent2's memories cleaned up")

	fmt.Println("\n" + repeat("=", 79))
	fmt.Println("Example completed successfully!")
	fmt.Println(repeat("=", 79))
}

func repeat(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}
