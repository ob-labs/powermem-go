package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	powermem "github.com/oceanbase/powermem-go/pkg/core"
)

func main() {
	fmt.Println("=" + repeat("=", 59))
	fmt.Println("PowerMem Go SDK - Async Usage Example")
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
		if filepath.Base(envPath) == ".env.example" {
			fmt.Println("⚠️  Using .env.example file. Consider copying it to .env:")
			fmt.Println("   cp .env.example .env")
		}
	}

	// 加载配置
	fmt.Println("\nInitializing async memory...")
	config, err := powermem.LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 创建异步客户端
	asyncClient, err := powermem.NewAsyncClient(config)
	if err != nil {
		log.Fatalf("Failed to create async client: %v", err)
	}
	defer func() {
		if err := asyncClient.Close(); err != nil {
			log.Printf("Warning: failed to close async client: %v", err)
		}
	}()

	fmt.Println("✓ Async memory initialized successfully!")

	ctx := context.Background()
	userID := "user123"

	// 示例 1: 并发添加多个记忆
	fmt.Println("Example 1: Concurrently adding multiple memories...")
	contents := []string{
		"User likes coffee",
		"User prefers Python over Java",
		"User works as a software engineer",
		"User enjoys reading technical books",
		"User is learning Go programming",
	}

	var wg sync.WaitGroup
	memories := make([]*powermem.Memory, len(contents))
	errors := make([]error, len(contents))

	for i, content := range contents {
		wg.Add(1)
		go func(idx int, text string) {
			defer wg.Done()
			resultChan := asyncClient.AddAsync(ctx, text, powermem.WithUserID(userID))
			result := <-resultChan
			memories[idx] = result.Memory
			errors[idx] = result.Error
		}(i, content)
	}

	wg.Wait()

	// 检查结果
	for i, content := range contents {
		if errors[i] != nil {
			log.Printf("Failed to add memory %d: %v", i, errors[i])
		} else {
			fmt.Printf("✓ Added: %s (ID: %d)\n", content, memories[i].ID)
		}
	}

	// 示例 2: 并发搜索
	fmt.Println("\nExample 2: Concurrently searching for memories...")
	queries := []string{
		"programming language",
		"work",
		"hobbies",
	}

	searchResults := make([][]*powermem.Memory, len(queries))
	searchErrors := make([]error, len(queries))

	wg = sync.WaitGroup{}
	for i, query := range queries {
		wg.Add(1)
		go func(idx int, q string) {
			defer wg.Done()
			resultChan := asyncClient.SearchAsync(ctx, q,
				powermem.WithUserIDForSearch(userID),
				powermem.WithLimit(5),
			)
			result := <-resultChan
			searchResults[idx] = result.Memories
			searchErrors[idx] = result.Error
		}(i, query)
	}

	wg.Wait()

	// 显示搜索结果
	for i, query := range queries {
		if searchErrors[i] != nil {
			log.Printf("Failed to search for '%s': %v", query, searchErrors[i])
		} else {
			fmt.Printf("\nQuery: '%s'\n", query)
			fmt.Printf("Found %d results:\n", len(searchResults[i]))
			for j, mem := range searchResults[i] {
				if j >= 3 { // 只显示前 3 个
					break
				}
				fmt.Printf("  %d. %s (ID: %d, Score: %.4f)\n", j+1, mem.Content, mem.ID, mem.Score)
			}
		}
	}

	// 示例 3: 使用 select 和 timeout
	fmt.Println("\nExample 3: Using select with timeout...")
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	resultChan := asyncClient.SearchAsync(ctxWithTimeout, "coffee",
		powermem.WithUserIDForSearch(userID),
	)

	select {
	case result := <-resultChan:
		if result.Error != nil {
			log.Printf("Search error: %v", result.Error)
		} else {
			fmt.Printf("✓ Found %d results for 'coffee'\n", len(result.Memories))
		}
	case <-ctxWithTimeout.Done():
		fmt.Println("⚠️  Search timed out")
	}

	// 示例 4: 批量操作
	fmt.Println("\nExample 4: Batch operations...")
	if len(memories) > 0 {
		// 并发获取多个记忆
		getResults := make([]*powermem.Memory, len(memories))
		getErrors := make([]error, len(memories))

		wg = sync.WaitGroup{}
		for i, mem := range memories {
			if mem == nil {
				continue
			}
			wg.Add(1)
			go func(idx int, id int64) {
				defer wg.Done()
				resultChan := asyncClient.GetAsync(ctx, id)
				result := <-resultChan
				getResults[idx] = result.Memory
				getErrors[idx] = result.Error
			}(i, mem.ID)
		}

		wg.Wait()

		successCount := 0
		for i := range memories {
			if memories[i] != nil && getErrors[i] == nil {
				successCount++
			}
		}
		fmt.Printf("✓ Successfully retrieved %d/%d memories\n", successCount, len(memories))
	}

	fmt.Println("\n" + repeat("=", 60))
	fmt.Println("All async operations completed!")
	fmt.Println(repeat("=", 60))
}

func repeat(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}
