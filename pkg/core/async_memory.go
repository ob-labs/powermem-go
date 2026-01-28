// Package core provides the main PowerMem client and memory management functionality.
package core

import (
	"context"
	"sync"
)

// AsyncClient provides asynchronous PowerMem operations.
//
// It wraps the synchronous Client and executes all operations in separate goroutines,
// making it suitable for scenarios requiring concurrent processing of multiple operations.
//
// All async methods return channels that will receive the results when operations complete.
// The client tracks all goroutines and provides Wait() to ensure all operations finish.
//
// Example:
//
//	asyncClient, _ := core.NewAsyncClient(config)
//	defer asyncClient.Close()
//
//	resultChan := asyncClient.AddAsync(ctx, "User likes Python", core.WithUserID("user_001"))
//	result := <-resultChan
//	if result.Error != nil {
//	    log.Fatal(result.Error)
//	}
type AsyncClient struct {
	*Client
	wg sync.WaitGroup
}

// NewAsyncClient creates a new asynchronous PowerMem client.
//
// Parameters:
//   - cfg: PowerMem configuration
//
// Returns:
//   - *AsyncClient: The asynchronous client instance
//   - error: Error if configuration is invalid or initialization fails
func NewAsyncClient(cfg *Config) (*AsyncClient, error) {
	client, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}

	return &AsyncClient{
		Client: client,
	}, nil
}

// AddAsync adds a memory asynchronously.
//
// The operation executes in a separate goroutine and returns results via a channel.
//
// Parameters:
//   - ctx: Context for controlling request lifecycle
//   - content: Memory content to add
//   - opts: Optional add options (UserID, AgentID, Metadata, etc.)
//
// Returns:
//   - <-chan *MemoryResult: Channel that receives the result containing Memory and error
func (ac *AsyncClient) AddAsync(ctx context.Context, content string, opts ...AddOption) <-chan *MemoryResult {
	resultChan := make(chan *MemoryResult, 1)
	ac.wg.Add(1)

	go func() {
		defer ac.wg.Done()
		memory, err := ac.Add(ctx, content, opts...)
		resultChan <- &MemoryResult{
			Memory: memory,
			Error:  err,
		}
		close(resultChan)
	}()

	return resultChan
}

// SearchAsync searches memories asynchronously.
//
// The operation executes in a separate goroutine and returns results via a channel.
//
// Parameters:
//   - ctx: Context for controlling request lifecycle
//   - query: Search query text
//   - opts: Optional search options (UserID, AgentID, Limit, Filters, etc.)
//
// Returns:
//   - <-chan *AsyncSearchResult: Channel that receives search results containing Memories and error
func (ac *AsyncClient) SearchAsync(ctx context.Context, query string, opts ...SearchOption) <-chan *AsyncSearchResult {
	resultChan := make(chan *AsyncSearchResult, 1)
	ac.wg.Add(1)

	go func() {
		defer ac.wg.Done()
		memories, err := ac.Search(ctx, query, opts...)
		resultChan <- &AsyncSearchResult{
			Memories: memories,
			Error:    err,
		}
		close(resultChan)
	}()

	return resultChan
}

// GetAsync retrieves a memory by ID asynchronously.
//
// The operation executes in a separate goroutine and returns results via a channel.
//
// Parameters:
//   - ctx: Context for controlling request lifecycle
//   - id: Memory ID
//
// Returns:
//   - <-chan *MemoryResult: Channel that receives the result containing Memory and error
func (ac *AsyncClient) GetAsync(ctx context.Context, id int64) <-chan *MemoryResult {
	resultChan := make(chan *MemoryResult, 1)
	ac.wg.Add(1)

	go func() {
		defer ac.wg.Done()
		memory, err := ac.Get(ctx, id)
		resultChan <- &MemoryResult{
			Memory: memory,
			Error:  err,
		}
		close(resultChan)
	}()

	return resultChan
}

// UpdateAsync updates a memory asynchronously.
//
// The operation executes in a separate goroutine and returns results via a channel.
//
// Parameters:
//   - ctx: Context for controlling request lifecycle
//   - id: Memory ID
//   - content: New memory content
//
// Returns:
//   - <-chan *MemoryResult: Channel that receives the result containing Memory and error
func (ac *AsyncClient) UpdateAsync(ctx context.Context, id int64, content string) <-chan *MemoryResult {
	resultChan := make(chan *MemoryResult, 1)
	ac.wg.Add(1)

	go func() {
		defer ac.wg.Done()
		memory, err := ac.Update(ctx, id, content)
		resultChan <- &MemoryResult{
			Memory: memory,
			Error:  err,
		}
		close(resultChan)
	}()

	return resultChan
}

// DeleteAsync deletes a memory asynchronously.
//
// The operation executes in a separate goroutine and returns results via a channel.
//
// Parameters:
//   - ctx: Context for controlling request lifecycle
//   - id: Memory ID
//
// Returns:
//   - <-chan error: Channel that receives error (nil if deletion succeeds)
func (ac *AsyncClient) DeleteAsync(ctx context.Context, id int64) <-chan error {
	errChan := make(chan error, 1)
	ac.wg.Add(1)

	go func() {
		defer ac.wg.Done()
		err := ac.Delete(ctx, id)
		errChan <- err
		close(errChan)
	}()

	return errChan
}

// GetAllAsync retrieves all memories asynchronously.
//
// The operation executes in a separate goroutine and returns results via a channel.
//
// Parameters:
//   - ctx: Context for controlling request lifecycle
//   - opts: Optional retrieval options (UserID, AgentID, Limit, Offset, etc.)
//
// Returns:
//   - <-chan *AsyncGetAllResult: Channel that receives results containing Memories and error
func (ac *AsyncClient) GetAllAsync(ctx context.Context, opts ...GetAllOption) <-chan *AsyncGetAllResult {
	resultChan := make(chan *AsyncGetAllResult, 1)
	ac.wg.Add(1)

	go func() {
		defer ac.wg.Done()
		memories, err := ac.GetAll(ctx, opts...)
		resultChan <- &AsyncGetAllResult{
			Memories: memories,
			Error:    err,
		}
		close(resultChan)
	}()

	return resultChan
}

// DeleteAllAsync deletes all memories asynchronously.
//
// The operation executes in a separate goroutine and returns results via a channel.
//
// Parameters:
//   - ctx: Context for controlling request lifecycle
//   - opts: Optional deletion options (UserID, AgentID, etc.)
//
// Returns:
//   - <-chan error: Channel that receives error (nil if deletion succeeds)
func (ac *AsyncClient) DeleteAllAsync(ctx context.Context, opts ...DeleteAllOption) <-chan error {
	errChan := make(chan error, 1)
	ac.wg.Add(1)

	go func() {
		defer ac.wg.Done()
		err := ac.DeleteAll(ctx, opts...)
		errChan <- err
		close(errChan)
	}()

	return errChan
}

// Wait waits for all asynchronous operations to complete.
//
// This method blocks until all goroutines started by async methods have finished.
// It should be called before program exit to ensure all operations complete.
func (ac *AsyncClient) Wait() {
	ac.wg.Wait()
}

// Close closes the asynchronous client.
//
// It first waits for all asynchronous operations to complete, then closes the underlying client.
func (ac *AsyncClient) Close() error {
	ac.Wait()
	return ac.Client.Close()
}

// MemoryResult contains the result of a memory operation.
type MemoryResult struct {
	// Memory is the memory returned by the operation (nil if error occurred).
	Memory *Memory

	// Error is the error returned by the operation (nil if operation succeeded).
	Error error
}

// AsyncSearchResult contains the result of an asynchronous search operation.
type AsyncSearchResult struct {
	// Memories is the list of matching memories.
	Memories []*Memory

	// Error is the error returned by the operation (nil if operation succeeded).
	Error error
}

// AsyncGetAllResult contains the result of an asynchronous GetAll operation.
type AsyncGetAllResult struct {
	// Memories is the list of memories.
	Memories []*Memory

	// Error is the error returned by the operation (nil if operation succeeded).
	Error error
}
