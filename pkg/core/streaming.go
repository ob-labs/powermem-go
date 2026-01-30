// Package core provides the main PowerMem client and memory management functionality.
package core

import (
	"context"
	"sync"

	"github.com/oceanbase/powermem-go/pkg/storage"
)

// StreamingSearchResult contains a batch of search results from streaming search.
type StreamingSearchResult struct {
	// Memories is a batch of matching memories.
	Memories []*Memory

	// BatchIndex is the index of this batch (0-based).
	BatchIndex int

	// IsLastBatch indicates whether this is the last batch.
	IsLastBatch bool

	// Error contains any error that occurred during streaming (if any).
	Error error
}

// StreamingGetAllResult contains a batch of memories from streaming GetAll.
type StreamingGetAllResult struct {
	// Memories is a batch of memories.
	Memories []*Memory

	// BatchIndex is the index of this batch (0-based).
	BatchIndex int

	// IsLastBatch indicates whether this is the last batch.
	IsLastBatch bool

	// Error contains any error that occurred during streaming (if any).
	Error error
}

// SearchStream performs streaming search for large datasets.
//
// Instead of returning all results at once, this method streams results in batches
// through a channel, making it suitable for processing large result sets without
// loading everything into memory at once.
//
// The method:
//  1. Performs an initial search to get the first batch
//  2. Continues fetching additional batches until all results are retrieved
//  3. Sends each batch through the channel as it becomes available
//
// Note: Vector search typically doesn't support offset-based pagination well.
// This implementation performs a single search and returns results in batches.
// For true streaming with large result sets, consider using GetAllStream with
// filtering or implementing cursor-based pagination in your storage backend.
//
// Parameters:
//   - ctx: Context for cancellation
//   - query: Search query string
//   - batchSize: Number of results per batch
//   - opts: Optional search parameters (UserID, AgentID, MinScore, Filters, Limit)
//
// Returns a channel that receives StreamingSearchResult batches.
// The channel is closed when all results have been sent or an error occurs.
//
// Example:
//
//	resultChan := client.SearchStream(ctx, "Python programming",
//	    50, // batch size
//	    core.WithUserIDForSearch("user_001"),
//	    core.WithLimit(200), // maximum total results
//	)
//
//	for result := range resultChan {
//	    if result.Error != nil {
//	        log.Fatal(result.Error)
//	    }
//	    for _, mem := range result.Memories {
//	        processMemory(mem)
//	    }
//	}
func (c *Client) SearchStream(ctx context.Context, query string, batchSize int, opts ...SearchOption) <-chan *StreamingSearchResult {
	resultChan := make(chan *StreamingSearchResult, 1)

	go func() {
		defer close(resultChan)

		c.mu.RLock()
		defer c.mu.RUnlock()

		// Apply search options
		searchOpts := applySearchOptions(opts)

		// Generate query embedding
		queryEmbedding, err := c.embedder.Embed(ctx, query)
		if err != nil {
			resultChan <- &StreamingSearchResult{
				Error: NewMemoryError("SearchStream", err),
			}
			return
		}

		// Determine maximum results
		maxResults := searchOpts.Limit
		if maxResults <= 0 {
			maxResults = 1000 // Default maximum for streaming
		}

		// Perform search with increased limit to get all results at once
		// Then we'll paginate in memory
		storageOpts := &storage.SearchOptions{
			UserID:   searchOpts.UserID,
			AgentID:  searchOpts.AgentID,
			Limit:    maxResults,
			MinScore: searchOpts.MinScore,
			Filters:  searchOpts.Filters,
		}

		// Get all matching results
		allMemories, err := c.storage.Search(ctx, queryEmbedding, storageOpts)
		if err != nil {
			resultChan <- &StreamingSearchResult{
				Error: NewMemoryError("SearchStream", err),
			}
			return
		}

		// Convert all memories
		convertedMemories := fromStorageMemories(allMemories)

		// Stream results in batches
		batchIndex := 0
		for i := 0; i < len(convertedMemories); i += batchSize {
			// Check context cancellation
			select {
			case <-ctx.Done():
				resultChan <- &StreamingSearchResult{
					BatchIndex: batchIndex,
					Error:      ctx.Err(),
				}
				return
			default:
			}

			end := i + batchSize
			if end > len(convertedMemories) {
				end = len(convertedMemories)
			}

			batch := convertedMemories[i:end]
			isLastBatch := end >= len(convertedMemories)

			resultChan <- &StreamingSearchResult{
				Memories:    batch,
				BatchIndex:  batchIndex,
				IsLastBatch: isLastBatch,
			}

			batchIndex++

			// If this was the last batch, stop
			if isLastBatch {
				break
			}
		}
	}()

	return resultChan
}

// GetAllStream performs streaming retrieval of all memories for large datasets.
//
// Instead of loading all memories into memory at once, this method streams
// results in batches through a channel, making it suitable for processing
// large datasets without exhausting system resources.
//
// Parameters:
//   - ctx: Context for cancellation
//   - batchSize: Number of memories per batch
//   - opts: Optional parameters (UserID, AgentID, Limit, Offset)
//
// Returns a channel that receives StreamingGetAllResult batches.
// The channel is closed when all memories have been sent or an error occurs.
//
// Example:
//
//	resultChan := client.GetAllStream(ctx, 100, // batch size
//	    core.WithUserIDForGetAll("user_001"),
//	    core.WithLimitForGetAll(1000), // maximum total results
//	)
//
//	for result := range resultChan {
//	    if result.Error != nil {
//	        log.Fatal(result.Error)
//	    }
//	    for _, mem := range result.Memories {
//	        processMemory(mem)
//	    }
//	}
func (c *Client) GetAllStream(ctx context.Context, batchSize int, opts ...GetAllOption) <-chan *StreamingGetAllResult {
	resultChan := make(chan *StreamingGetAllResult, 1)

	go func() {
		defer close(resultChan)

		c.mu.RLock()
		defer c.mu.RUnlock()

		// Apply options
		getAllOpts := applyGetAllOptions(opts)

		// Prepare storage options
		storageOpts := &storage.GetAllOptions{
			UserID:  getAllOpts.UserID,
			AgentID: getAllOpts.AgentID,
			Limit:   batchSize,
			Offset:  getAllOpts.Offset,
		}

		// Determine maximum results
		maxResults := getAllOpts.Limit
		if maxResults <= 0 {
			maxResults = 10000 // Default maximum for streaming
		}

		batchIndex := 0
		offset := getAllOpts.Offset

		for {
			// Check context cancellation
			select {
			case <-ctx.Done():
				resultChan <- &StreamingGetAllResult{
					Error: ctx.Err(),
				}
				return
			default:
			}

			// Update offset
			storageOpts.Offset = offset
			storageOpts.Limit = batchSize

			// Adjust batch size for the last batch
			remaining := maxResults - (offset - getAllOpts.Offset)
			if remaining <= 0 {
				break
			}
			if remaining < batchSize {
				storageOpts.Limit = remaining
			}

			// Get batch
			memories, err := c.storage.GetAll(ctx, storageOpts)
			if err != nil {
				resultChan <- &StreamingGetAllResult{
					BatchIndex: batchIndex,
					Error:      NewMemoryError("GetAllStream", err),
				}
				return
			}

			// If no more results, we're done
			if len(memories) == 0 {
				break
			}

			// Convert and send batch
			convertedMemories := fromStorageMemories(memories)
			isLastBatch := len(memories) < batchSize

			resultChan <- &StreamingGetAllResult{
				Memories:    convertedMemories,
				BatchIndex:  batchIndex,
				IsLastBatch: isLastBatch,
			}

			batchIndex++
			offset += len(memories)

			// If this was the last batch, stop
			if isLastBatch {
				break
			}

			// Check if we've reached the maximum
			if offset-getAllOpts.Offset >= maxResults {
				break
			}
		}
	}()

	return resultChan
}

// BatchAddResult contains the result of a batch add operation.
type BatchAddResult struct {
	// Created contains successfully created memories.
	Created []*Memory

	// Failed contains memories that failed to be created, along with their errors.
	Failed []BatchAddError

	// Total is the total number of items in the batch.
	Total int

	// CreatedCount is the number of successfully created memories.
	CreatedCount int

	// FailedCount is the number of failed creations.
	FailedCount int
}

// BatchAddError contains information about a failed batch add operation.
type BatchAddError struct {
	// Content is the content that failed to be added.
	Content string

	// Error is the error that occurred.
	Error error

	// Index is the index of the item in the original batch.
	Index int
}

// BatchAdd adds multiple memories in a single batch operation.
//
// This method processes memories concurrently within the batch for better performance,
// while respecting resource limits and error handling.
//
// Parameters:
//   - ctx: Context for cancellation
//   - contents: Slice of memory contents to add
//   - opts: Optional parameters (UserID, AgentID, Metadata, etc.)
//     These options apply to all memories in the batch.
//
// Returns a BatchAddResult containing created memories and any failures.
//
// Example:
//
//	contents := []string{
//	    "User likes Python",
//	    "User prefers email communication",
//	    "User works in tech industry",
//	}
//	result, err := client.BatchAdd(ctx, contents,
//	    core.WithUserID("user_001"),
//	    core.WithInfer(true),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Created %d/%d memories\n", result.CreatedCount, result.Total)
func (c *Client) BatchAdd(ctx context.Context, contents []string, opts ...AddOption) (*BatchAddResult, error) {
	if len(contents) == 0 {
		return &BatchAddResult{
			Total:        0,
			CreatedCount: 0,
			FailedCount:  0,
		}, nil
	}

	result := &BatchAddResult{
		Total:        len(contents),
		Created:      make([]*Memory, 0, len(contents)),
		Failed:       make([]BatchAddError, 0),
		CreatedCount: 0,
		FailedCount:  0,
	}

	// Use a semaphore to limit concurrent operations
	const maxConcurrency = 10
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, content := range contents {
		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore

		go func(index int, text string) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			// Check context cancellation
			select {
			case <-ctx.Done():
				mu.Lock()
				result.Failed = append(result.Failed, BatchAddError{
					Content: text,
					Error:   ctx.Err(),
					Index:   index,
				})
				result.FailedCount++
				mu.Unlock()
				return
			default:
			}

			// Add memory
			memory, err := c.Add(ctx, text, opts...)
			if err != nil {
				mu.Lock()
				result.Failed = append(result.Failed, BatchAddError{
					Content: text,
					Error:   err,
					Index:   index,
				})
				result.FailedCount++
				mu.Unlock()
				return
			}

			mu.Lock()
			result.Created = append(result.Created, memory)
			result.CreatedCount++
			mu.Unlock()
		}(i, content)
	}

	wg.Wait()

	return result, nil
}

// BatchUpdateResult contains the result of a batch update operation.
type BatchUpdateResult struct {
	// Updated contains successfully updated memories.
	Updated []*Memory

	// Failed contains memories that failed to be updated, along with their errors.
	Failed []BatchUpdateError

	// Total is the total number of items in the batch.
	Total int

	// UpdatedCount is the number of successfully updated memories.
	UpdatedCount int

	// FailedCount is the number of failed updates.
	FailedCount int
}

// BatchUpdateError contains information about a failed batch update operation.
type BatchUpdateError struct {
	// ID is the memory ID that failed to be updated.
	ID int64

	// Content is the content that failed to be updated.
	Content string

	// Error is the error that occurred.
	Error error

	// Index is the index of the item in the original batch.
	Index int
}

// BatchUpdateItem represents a single item in a batch update operation.
type BatchUpdateItem struct {
	// ID is the memory ID to update.
	ID int64

	// Content is the new content for the memory.
	Content string
}

// BatchUpdate updates multiple memories in a single batch operation.
//
// Parameters:
//   - ctx: Context for cancellation
//   - items: Slice of BatchUpdateItem containing ID and content pairs
//
// Returns a BatchUpdateResult containing updated memories and any failures.
//
// Example:
//
//	items := []core.BatchUpdateItem{
//	    {ID: 1, Content: "Updated content 1"},
//	    {ID: 2, Content: "Updated content 2"},
//	}
//	result, err := client.BatchUpdate(ctx, items)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Updated %d/%d memories\n", result.UpdatedCount, result.Total)
func (c *Client) BatchUpdate(ctx context.Context, items []BatchUpdateItem) (*BatchUpdateResult, error) {
	if len(items) == 0 {
		return &BatchUpdateResult{
			Total:        0,
			UpdatedCount: 0,
			FailedCount:  0,
		}, nil
	}

	result := &BatchUpdateResult{
		Total:        len(items),
		Updated:      make([]*Memory, 0, len(items)),
		Failed:       make([]BatchUpdateError, 0),
		UpdatedCount: 0,
		FailedCount:  0,
	}

	// Use a semaphore to limit concurrent operations
	const maxConcurrency = 10
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, item := range items {
		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore

		go func(index int, updateItem BatchUpdateItem) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			// Check context cancellation
			select {
			case <-ctx.Done():
				mu.Lock()
				result.Failed = append(result.Failed, BatchUpdateError{
					ID:      updateItem.ID,
					Content: updateItem.Content,
					Error:   ctx.Err(),
					Index:   index,
				})
				result.FailedCount++
				mu.Unlock()
				return
			default:
			}

			// Update memory
			memory, err := c.Update(ctx, updateItem.ID, updateItem.Content)
			if err != nil {
				mu.Lock()
				result.Failed = append(result.Failed, BatchUpdateError{
					ID:      updateItem.ID,
					Content: updateItem.Content,
					Error:   err,
					Index:   index,
				})
				result.FailedCount++
				mu.Unlock()
				return
			}

			mu.Lock()
			result.Updated = append(result.Updated, memory)
			result.UpdatedCount++
			mu.Unlock()
		}(i, item)
	}

	wg.Wait()

	return result, nil
}

// BatchDeleteResult contains the result of a batch delete operation.
type BatchDeleteResult struct {
	// DeletedIDs contains successfully deleted memory IDs.
	DeletedIDs []int64

	// Failed contains memory IDs that failed to be deleted, along with their errors.
	Failed []BatchDeleteError

	// Total is the total number of items in the batch.
	Total int

	// DeletedCount is the number of successfully deleted memories.
	DeletedCount int

	// FailedCount is the number of failed deletions.
	FailedCount int
}

// BatchDeleteError contains information about a failed batch delete operation.
type BatchDeleteError struct {
	// ID is the memory ID that failed to be deleted.
	ID int64

	// Error is the error that occurred.
	Error error

	// Index is the index of the item in the original batch.
	Index int
}

// BatchDelete deletes multiple memories in a single batch operation.
//
// Parameters:
//   - ctx: Context for cancellation
//   - ids: Slice of memory IDs to delete
//
// Returns a BatchDeleteResult containing deleted IDs and any failures.
//
// Example:
//
//	ids := []int64{1, 2, 3, 4, 5}
//	result, err := client.BatchDelete(ctx, ids)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Deleted %d/%d memories\n", result.DeletedCount, result.Total)
func (c *Client) BatchDelete(ctx context.Context, ids []int64) (*BatchDeleteResult, error) {
	if len(ids) == 0 {
		return &BatchDeleteResult{
			Total:        0,
			DeletedCount: 0,
			FailedCount:  0,
		}, nil
	}

	result := &BatchDeleteResult{
		Total:        len(ids),
		DeletedIDs:   make([]int64, 0, len(ids)),
		Failed:       make([]BatchDeleteError, 0),
		DeletedCount: 0,
		FailedCount:  0,
	}

	// Use a semaphore to limit concurrent operations
	const maxConcurrency = 10
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, id := range ids {
		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore

		go func(index int, memoryID int64) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			// Check context cancellation
			select {
			case <-ctx.Done():
				mu.Lock()
				result.Failed = append(result.Failed, BatchDeleteError{
					ID:    memoryID,
					Error: ctx.Err(),
					Index: index,
				})
				result.FailedCount++
				mu.Unlock()
				return
			default:
			}

			// Delete memory
			err := c.Delete(ctx, memoryID)
			if err != nil {
				mu.Lock()
				result.Failed = append(result.Failed, BatchDeleteError{
					ID:    memoryID,
					Error: err,
					Index: index,
				})
				result.FailedCount++
				mu.Unlock()
				return
			}

			mu.Lock()
			result.DeletedIDs = append(result.DeletedIDs, memoryID)
			result.DeletedCount++
			mu.Unlock()
		}(i, id)
	}

	wg.Wait()

	return result, nil
}
