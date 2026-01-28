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
	fmt.Println("PowerMem Go SDK - Intelligent Memory Management Demo")
	fmt.Println("=" + repeat("=", 79))

	// 查找配置文件
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

	// 加载配置
	fmt.Println("\nLoading configuration...")
	config, err := powermem.LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 启用智能记忆功能
	config.Intelligence = &powermem.IntelligenceConfig{
		Enabled:             true,
		DecayRate:           0.1,
		ReinforcementFactor: 0.3,
		DuplicateThreshold:  0.95,
	}

	// 创建客户端
	client, err := powermem.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.Printf("Warning: failed to close client: %v", err)
		}
	}()

	fmt.Println("✓ Client initialized with intelligent memory features!")

	ctx := context.Background()
	userID := "user456"

	// Clean up any existing memories for this user (optional, for demo purposes)
	fmt.Println("\nCleaning up existing memories...")
	err = client.DeleteAll(ctx, powermem.WithUserIDForDeleteAll(userID))
	if err != nil {
		log.Printf("Warning: failed to clean up: %v", err)
	} else {
		fmt.Println("✓ Cleanup completed!")
	}

	// Scenario 1: Intelligent Deduplication
	fmt.Println(repeat("=", 80))
	fmt.Println("SCENARIO 1: Intelligent Deduplication")
	fmt.Println(repeat("=", 80))
	fmt.Println("Adding user's basic information...")

	memory1, err := client.Add(ctx, "User likes Python programming",
		powermem.WithUserID(userID),
		powermem.WithInfer(true), // 启用智能去重
	)
	if err != nil {
		log.Fatalf("Failed to add memory: %v", err)
	}
	fmt.Printf("✓ Added memory 1: ID=%d, Content=%s\n", memory1.ID, memory1.Content)

	// 尝试添加相似的记忆（会被合并）
	memory2, err := client.Add(ctx, "User enjoys Python coding",
		powermem.WithUserID(userID),
		powermem.WithInfer(true),
	)
	if err != nil {
		log.Fatalf("Failed to add memory: %v", err)
	}
	fmt.Printf("✓ Added memory 2: ID=%d, Content=%s\n", memory2.ID, memory2.Content)

	if memory1.ID == memory2.ID {
		fmt.Println("✓ Similar memories were automatically merged!")
	} else {
		fmt.Println("✗ Memories were considered different and stored separately")
	}

	// 场景 2: 多代理场景
	fmt.Println("\n" + repeat("=", 80))
	fmt.Println("SCENARIO 2: Multi-Agent Scenario")
	fmt.Println(repeat("=", 80))

	// Agent1 添加私有记忆
	fmt.Println("\nAgent1 (Personal Assistant) adding private memory:")
	agentMemory1, err := client.Add(ctx, "Agent1's private task: analyze user preferences",
		powermem.WithUserID(userID),
		powermem.WithAgentID("agent1_personal"),
		powermem.WithScope(powermem.ScopePrivate),
	)
	if err != nil {
		log.Fatalf("Failed to add memory: %v", err)
	}
	fmt.Printf("✓ Added: %s\n", agentMemory1.Content)

	// Agent2 添加共享记忆
	fmt.Println("\nAgent2 (Task Manager) adding shared memory:")
	agentMemory2, err := client.Add(ctx, "Shared knowledge: User prefers morning work sessions",
		powermem.WithUserID(userID),
		powermem.WithAgentID("agent2_tasks"),
		powermem.WithScope(powermem.ScopeAgentGroup),
	)
	if err != nil {
		log.Fatalf("Failed to add memory: %v", err)
	}
	fmt.Printf("✓ Added: %s (shared)\n", agentMemory2.Content)

	// Agent1 搜索（只能看到自己的私有记忆）
	fmt.Println("\nAgent1 searching for 'task':")
	agent1Results, err := client.Search(ctx, "task",
		powermem.WithUserIDForSearch(userID),
		powermem.WithAgentIDForSearch("agent1_personal"),
		powermem.WithLimit(10),
	)
	if err != nil {
		log.Fatalf("Failed to search: %v", err)
	}
	fmt.Printf("Found %d memories:\n", len(agent1Results))
	for _, mem := range agent1Results {
		fmt.Printf("  - %s\n", mem.Content)
	}

	// 场景 3: 带过滤器的高级搜索
	fmt.Println("\n" + repeat("=", 80))
	fmt.Println("SCENARIO 3: Advanced Search with Filters")
	fmt.Println(repeat("=", 80))

	// 添加带元数据的记忆
	_, err = client.Add(ctx, "User completed Python course",
		powermem.WithUserID(userID),
		powermem.WithMetadata(map[string]interface{}{
			"type":       "achievement",
			"importance": "high",
			"date":       "2024-01-15",
		}),
	)
	if err != nil {
		log.Fatalf("Failed to add memory: %v", err)
	}
	fmt.Println("✓ Added memory with metadata")

	// 使用过滤器搜索
	fmt.Println("\nSearching with filters (type=achievement):")
	filteredResults, err := client.Search(ctx, "Python",
		powermem.WithUserIDForSearch(userID),
		powermem.WithFilters(map[string]interface{}{
			"type": "achievement",
		}),
		powermem.WithMinScore(0.7),
		powermem.WithLimit(5),
	)
	if err != nil {
		log.Fatalf("Failed to search: %v", err)
	}

	fmt.Printf("Found %d filtered results:\n", len(filteredResults))
	for _, mem := range filteredResults {
		fmt.Printf("  - [Score: %.3f] %s\n", mem.Score, mem.Content)
		if mem.Metadata != nil {
			fmt.Printf("    Metadata: %v\n", mem.Metadata)
		}
	}

	fmt.Println("\n" + repeat("=", 80))
	fmt.Println("Demo completed successfully!")
	fmt.Println(repeat("=", 80))
}

func repeat(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}
