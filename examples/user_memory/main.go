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
	fmt.Println("\n=== 示例 1: 添加对话并提取用户画像 ===")
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

	fmt.Printf("✓ 对话已添加，记忆 ID: %d\n", result.Memory.ID)
	fmt.Printf("✓ 画像提取状态: %v\n", result.ProfileExtracted)
	if result.ProfileContent != nil {
		fmt.Printf("✓ 提取的画像内容: %s\n", *result.ProfileContent)
	}

	// 示例 2: 获取用户画像
	fmt.Println("\n=== 示例 2: 获取用户画像 ===")
	profile, err := client.GetProfile(ctx, "user_001")
	if err != nil {
		log.Fatalf("Failed to get profile: %v", err)
	}

	if profile != nil {
		fmt.Printf("✓ 用户 ID: %s\n", profile.UserID)
		fmt.Printf("✓ 画像内容: %s\n", profile.ProfileContent)
		fmt.Printf("✓ 创建时间: %s\n", profile.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("✓ 更新时间: %s\n", profile.UpdatedAt.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Println("未找到用户画像")
	}

	// 示例 3: 更新用户画像（通过添加新对话）
	fmt.Println("\n=== 示例 3: 更新用户画像 ===")
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

	fmt.Printf("✓ 画像更新状态: %v\n", updateResult.ProfileExtracted)

	// 获取更新后的画像
	updatedProfile, err := client.GetProfile(ctx, "user_001")
	if err != nil {
		log.Fatalf("Failed to get updated profile: %v", err)
	}

	if updatedProfile != nil {
		fmt.Printf("✓ 更新后的画像内容: %s\n", updatedProfile.ProfileContent)
	}

	// 示例 4: 搜索记忆
	fmt.Println("\n=== 示例 4: 搜索记忆 ===")
	searchResult, err := client.Search(ctx, "programming",
		usermemory.WithSearchUserID("user_001"),
		usermemory.WithSearchLimit(5),
	)
	if err != nil {
		log.Fatalf("Failed to search: %v", err)
	}

	fmt.Printf("✓ 找到 %d 条相关记忆\n", len(searchResult.Memories))
	for i, mem := range searchResult.Memories {
		fmt.Printf("  %d. %s (ID: %d)\n", i+1, mem.Content, mem.ID)
	}

	// 示例 5: 获取单个记忆
	fmt.Println("\n=== 示例 5: 获取单个记忆 ===")
	if len(searchResult.Memories) > 0 {
		memoryID := searchResult.Memories[0].ID
		memory, err := client.Get(ctx, memoryID)
		if err != nil {
			log.Fatalf("Failed to get memory: %v", err)
		}
		fmt.Printf("✓ 获取记忆 ID: %d\n", memory.ID)
		fmt.Printf("  内容: %s\n", memory.Content)
	}

	// 示例 6: 更新记忆
	fmt.Println("\n=== 示例 6: 更新记忆 ===")
	if len(searchResult.Memories) > 0 {
		memoryID := searchResult.Memories[0].ID
		updatedContent := "I'm Alice, a software engineer. I love Python and Go programming, and I enjoy reading science fiction."
		updated, err := client.Update(ctx, memoryID, updatedContent)
		if err != nil {
			log.Fatalf("Failed to update memory: %v", err)
		}
		fmt.Printf("✓ 记忆已更新 (ID: %d)\n", updated.ID)
		fmt.Printf("  新内容: %s\n", updated.Content)
	}

	// 示例 7: 查询重写功能（如果启用）
	if queryRewriteEnabled {
		fmt.Println("\n=== 示例 7: 查询重写功能 ===")
		// 使用模糊查询，应该会被重写为更具体的查询
		rewriteResult, err := client.Search(ctx, "my work",
			usermemory.WithSearchUserID("user_001"),
			usermemory.WithSearchLimit(5),
			usermemory.WithAddProfile(true), // 同时获取用户画像
		)
		if err != nil {
			log.Fatalf("Failed to search with query rewrite: %v", err)
		}
		fmt.Printf("✓ 查询重写后的搜索结果: 找到 %d 条记忆\n", len(rewriteResult.Memories))
		if rewriteResult.ProfileContent != nil {
			fmt.Printf("✓ 用户画像已包含在结果中\n")
		}
	}

	// 示例 8: 获取所有记忆
	fmt.Println("\n=== 示例 8: 获取所有记忆 ===")
	allMemories, err := client.GetAll(ctx,
		usermemory.WithGetAllUserID("user_001"),
		usermemory.WithGetAllLimit(10),
	)
	if err != nil {
		log.Fatalf("Failed to get all memories: %v", err)
	}
	fmt.Printf("✓ 获取到 %d 条记忆\n", len(allMemories))

	// 示例 8: 删除记忆（不删除画像）
	fmt.Println("\n=== 示例 8: 删除记忆 ===")
	if len(allMemories) > 0 {
		memoryID := allMemories[0].ID
		err = client.Delete(ctx, memoryID)
		if err != nil {
			log.Fatalf("Failed to delete memory: %v", err)
		}
		fmt.Printf("✓ 记忆已删除 (ID: %d)\n", memoryID)

		// 验证删除
		_, err = client.Get(ctx, memoryID)
		if err != nil {
			fmt.Println("✓ 验证：记忆已成功删除")
		}
	}

	// 示例 9: 删除所有记忆（可选择同时删除画像）
	fmt.Println("\n=== 示例 9: 删除所有记忆 ===")
	// 注意：这里不实际删除，只是演示用法
	fmt.Println("  用法示例：")
	fmt.Println("  client.DeleteAll(ctx,")
	fmt.Println("    usermemory.WithDeleteAllUserID(\"user_001\"),")
	fmt.Println("    usermemory.WithDeleteAllProfile(true), // 同时删除画像")
	fmt.Println("  )")

	// 示例 10: 重置存储
	fmt.Println("\n=== 示例 10: 重置存储 ===")
	fmt.Println("  用法示例：")
	fmt.Println("  client.Reset(ctx) // 删除所有记忆")
	fmt.Println("  注意：Reset 会删除所有记忆，但不会删除用户画像")

	fmt.Println("\n✓ 所有示例执行完成！")
}
