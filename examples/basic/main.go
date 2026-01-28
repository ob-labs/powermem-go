package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	powermem "github.com/oceanbase/powermem-go/pkg/core"
)

func main() {
	fmt.Println("=" + repeat("=", 59))
	fmt.Println("PowerMem Go SDK - Basic Usage Example")
	fmt.Println("=" + repeat("=", 59))

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
		// 如果找到 .env.example，提示用户复制
		if filepath.Base(envPath) == ".env.example" {
			fmt.Println("⚠️  Using .env.example file. Consider copying it to .env:")
			fmt.Println("   cp .env.example .env")
		}
	}

	// 加载配置
	fmt.Println("\nInitializing memory...")
	config, err := powermem.LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
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

	fmt.Println("✓ Memory initialized successfully!")

	ctx := context.Background()
	userID := "user123"

	// Clean up any existing memories for this user (optional, for demo purposes)
	fmt.Println("\nCleaning up existing memories...")
	err = client.DeleteAll(ctx, powermem.WithUserIDForDeleteAll(userID))
	if err != nil {
		log.Printf("Warning: failed to clean up: %v", err)
	} else {
		fmt.Println("✓ Cleanup completed!")
	}

	// Add memories
	fmt.Println("\nAdding memories...")
	memory1, err := client.Add(ctx, "User likes coffee",
		powermem.WithUserID(userID),
	)
	if err != nil {
		log.Fatalf("Failed to add memory: %v", err)
	}
	fmt.Printf("✓ Added: %s (ID: %d)\n", memory1.Content, memory1.ID)

	memory2, err := client.Add(ctx, "User prefers Python over Java",
		powermem.WithUserID(userID),
	)
	if err != nil {
		log.Fatalf("Failed to add memory: %v", err)
	}
	fmt.Printf("✓ Added: %s (ID: %d)\n", memory2.Content, memory2.ID)

	memory3, err := client.Add(ctx, "User works as a software engineer",
		powermem.WithUserID(userID),
	)
	if err != nil {
		log.Fatalf("Failed to add memory: %v", err)
	}
	fmt.Printf("✓ Added: %s (ID: %d)\n", memory3.Content, memory3.ID)
	fmt.Println("✓ Memories added!")

	// 搜索记忆
	fmt.Println("Searching memories...")
	results, err := client.Search(ctx, "user preferences",
		powermem.WithUserIDForSearch(userID),
		powermem.WithLimit(5),
	)
	if err != nil {
		log.Fatalf("Failed to search: %v", err)
	}

	fmt.Printf("Found %d results:\n", len(results))
	for i, mem := range results {
		fmt.Printf("  %d. [Score: %.3f] %s\n", i+1, mem.Score, mem.Content)
	}

	// 获取所有记忆
	fmt.Println("\nGetting all memories...")
	allMemories, err := client.GetAll(ctx,
		powermem.WithUserIDForGetAll(userID),
		powermem.WithLimitForGetAll(10),
	)
	if err != nil {
		log.Fatalf("Failed to get all memories: %v", err)
	}
	fmt.Printf("✓ Total memories: %d\n", len(allMemories))
}

func repeat(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}
