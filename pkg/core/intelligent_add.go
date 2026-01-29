package core

import (
	"context"
	"fmt"
	"log"

	"github.com/oceanbase/powermem-go/pkg/intelligence"
	"github.com/oceanbase/powermem-go/pkg/storage"
)

// IntelligentAddResult represents the result of an intelligent add operation.
type IntelligentAddResult struct {
	// Results contains the list of memory operations performed
	Results []MemoryActionResult `json:"results"`
}

// MemoryActionResult represents a single memory operation result.
type MemoryActionResult struct {
	// ID is the memory ID
	ID int64 `json:"id"`

	// Memory is the memory content
	Memory string `json:"memory"`

	// Event is the operation type: ADD, UPDATE, DELETE, NONE
	Event string `json:"event"`

	// PreviousMemory is the previous content (for UPDATE operations)
	PreviousMemory string `json:"previous_memory,omitempty"`

	// Metadata contains additional information
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// IntelligentAdd performs intelligent memory addition with fact extraction and LLM decision making.
//
// This method implements the complete intelligent add flow similar to Python SDK:
//  1. Extract facts from messages using FactExtractor
//  2. For each fact, search for similar existing memories
//  3. Use LLM (DecisionMaker) to decide operations: ADD / UPDATE / DELETE / NONE
//  4. Execute the decided operations
//
// Parameters:
//   - ctx: Context for cancellation
//   - messages: Messages to process (can be string, []map[string]interface{}, or single map)
//   - opts: Optional parameters (UserID, AgentID, RunID, Metadata, etc.)
//
// Returns IntelligentAddResult with details of all operations performed.
//
// Example:
//
//	result, err := client.IntelligentAdd(ctx, []map[string]interface{}{
//	    {"role": "user", "content": "I'm Alice, a software engineer"},
//	    {"role": "assistant", "content": "Nice to meet you!"},
//	},
//	    core.WithUserID("user_001"),
//	    core.WithAgentID("agent_001"),
//	)
func (c *Client) IntelligentAdd(ctx context.Context, messages interface{}, opts ...AddOption) (*IntelligentAddResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Apply options
	addOpts := applyAddOptions(opts)

	// Check if intelligent manager is available
	if c.intelligentManager == nil {
		return nil, fmt.Errorf("IntelligentAdd requires intelligent memory features to be enabled")
	}

	// Get LLM provider
	if c.llm == nil {
		return nil, fmt.Errorf("IntelligentAdd requires LLM provider to be configured")
	}

	// Step 1: Extract facts from messages
	log.Println("Extracting facts from messages...")
	facts, err := c.intelligentManager.ExtractFacts(ctx, messages)
	if err != nil {
		// Check if fallback to simple add is enabled
		if c.config.Intelligence != nil && c.config.Intelligence.FallbackToSimpleAdd {
			log.Printf("Failed to extract facts, falling back to simple add: %v", err)
			return c.fallbackToSimpleAdd(ctx, messages, opts...)
		}
		return nil, fmt.Errorf("failed to extract facts: %w", err)
	}

	if len(facts) == 0 {
		log.Println("No facts extracted, skip intelligent add")
		if c.config.Intelligence != nil && c.config.Intelligence.FallbackToSimpleAdd {
			log.Println("No facts extracted, falling back to simple add")
			return c.fallbackToSimpleAdd(ctx, messages, opts...)
		}
		return &IntelligentAddResult{Results: []MemoryActionResult{}}, nil
	}

	log.Printf("Extracted %d facts: %v", len(facts), facts)

	// Step 2: Search for similar memories for each fact
	existingMemories := make([]*Memory, 0)
	factEmbeddings := make(map[string][]float64)

	for _, fact := range facts {
		// Generate embedding for the fact
		embedding, err := c.embedder.Embed(ctx, fact)
		if err != nil {
			log.Printf("Failed to generate embedding for fact '%s': %v", fact, err)
			continue
		}
		factEmbeddings[fact] = embedding

		// Search for similar memories
		searchOpts := &storage.SearchOptions{
			UserID:   addOpts.UserID,
			AgentID:  addOpts.AgentID,
			Limit:    5, // Limit to reduce noise
			MinScore: 0.0,
			Query:    fact, // Pass fact text for future hybrid search
			Filters:  addOpts.Filters,
		}

		similar, err := c.storage.Search(ctx, embedding, searchOpts)
		if err != nil {
			log.Printf("Failed to search for similar memories: %v", err)
			continue
		}

		existingMemories = append(existingMemories, fromStorageMemories(similar)...)
	}

	// Deduplicate existing memories by ID
	uniqueMemories := make(map[int64]*Memory)
	for _, mem := range existingMemories {
		if _, exists := uniqueMemories[mem.ID]; !exists {
			uniqueMemories[mem.ID] = mem
		}
	}

	// Convert to slice and limit to max 10 memories
	existingMemoriesList := make([]*Memory, 0, len(uniqueMemories))
	for _, mem := range uniqueMemories {
		existingMemoriesList = append(existingMemoriesList, mem)
		if len(existingMemoriesList) >= 10 {
			break
		}
	}

	log.Printf("Found %d unique existing memories to consider", len(existingMemoriesList))

	// Create temporary ID mapping (index -> real ID)
	tempIDMapping := make(map[string]int64)
	existingForDecision := make([]intelligence.ExistingMemory, len(existingMemoriesList))
	for i, mem := range existingMemoriesList {
		tempID := fmt.Sprintf("%d", i)
		tempIDMapping[tempID] = mem.ID
		existingForDecision[i] = intelligence.ExistingMemory{
			ID:   tempID,
			Text: mem.Content,
		}
	}

	// Step 3: Let LLM decide memory actions
	decisionMaker := intelligence.NewDecisionMaker(c.llm)
	actions, err := decisionMaker.DecideActions(ctx, facts, existingForDecision)
	if err != nil {
		if c.config.Intelligence != nil && c.config.Intelligence.FallbackToSimpleAdd {
			log.Printf("Failed to get LLM decisions, falling back to simple add: %v", err)
			return c.fallbackToSimpleAdd(ctx, messages, opts...)
		}
		return nil, fmt.Errorf("failed to get LLM decisions: %w", err)
	}

	log.Printf("LLM decided on %d memory actions", len(actions))

	if len(actions) == 0 {
		log.Println("No actions returned from LLM, skip intelligent add")
		if c.config.Intelligence != nil && c.config.Intelligence.FallbackToSimpleAdd {
			log.Println("No actions from LLM, falling back to simple add")
			return c.fallbackToSimpleAdd(ctx, messages, opts...)
		}
		return &IntelligentAddResult{Results: []MemoryActionResult{}}, nil
	}

	// Step 4: Execute actions
	results := make([]MemoryActionResult, 0)
	actionCounts := map[string]int{"ADD": 0, "UPDATE": 0, "DELETE": 0, "NONE": 0}

	for _, action := range actions {
		actionText := action.Text
		if actionText == "" {
			actionText = action.Memory
		}
		eventType := action.Event

		// Skip actions with empty text UNLESS it's a NONE event
		if actionText == "" && eventType != "NONE" {
			log.Printf("Skipping action with empty text: %+v", action)
			continue
		}

		log.Printf("Processing action: %s - '%s' (id: %s)", eventType, truncate(actionText, 50), action.ID)

		switch eventType {
		case "ADD":
			// Add new memory
			embedding := factEmbeddings[actionText]
			if embedding == nil {
				// Generate new embedding if not in cache
				embedding, err = c.embedder.Embed(ctx, actionText)
				if err != nil {
					log.Printf("Failed to generate embedding for ADD action: %v", err)
					continue
				}
			}

			metadata := copyMetadata(addOpts.Metadata)
			addMetadataFields(metadata, addOpts)

			memory := &Memory{
				ID:                c.snowflakeNode.Generate().Int64(),
				UserID:            addOpts.UserID,
				AgentID:           addOpts.AgentID,
				Content:           actionText,
				Embedding:         embedding,
				Metadata:          metadata,
				RetentionStrength: 1.0,
			}

			if err := c.storage.Insert(ctx, toStorageMemory(memory)); err != nil {
				log.Printf("Failed to insert memory: %v", err)
				continue
			}

			results = append(results, MemoryActionResult{
				ID:       memory.ID,
				Memory:   actionText,
				Event:    eventType,
				Metadata: metadata,
			})
			actionCounts["ADD"]++

		case "UPDATE":
			// Update existing memory
			realMemoryID, ok := tempIDMapping[action.ID]
			if !ok {
				log.Printf("Could not find real memory ID for action ID: %s", action.ID)
				continue
			}

			// Generate new embedding
			embedding, err := c.embedder.Embed(ctx, actionText)
			if err != nil {
				log.Printf("Failed to generate embedding for UPDATE action: %v", err)
				continue
			}

			// Update the memory (without access control restrictions)
			_, err = c.storage.Update(ctx, realMemoryID, actionText, embedding, nil)
			if err != nil {
				log.Printf("Failed to update memory %d: %v", realMemoryID, err)
				continue
			}

			results = append(results, MemoryActionResult{
				ID:             realMemoryID,
				Memory:         actionText,
				Event:          eventType,
				PreviousMemory: action.OldMemory,
			})
			actionCounts["UPDATE"]++

		case "DELETE":
			// Delete existing memory
			realMemoryID, ok := tempIDMapping[action.ID]
			if !ok {
				log.Printf("Could not find real memory ID for action ID: %s", action.ID)
				continue
			}

			if err := c.storage.Delete(ctx, realMemoryID, nil); err != nil {
				log.Printf("Failed to delete memory %d: %v", realMemoryID, err)
				continue
			}

			results = append(results, MemoryActionResult{
				ID:     realMemoryID,
				Memory: actionText,
				Event:  eventType,
			})
			actionCounts["DELETE"]++

		case "NONE":
			// No action needed (duplicate)
			log.Println("No action needed for memory (duplicate detected)")
			actionCounts["NONE"]++

		default:
			log.Printf("Unknown event type: %s", eventType)
		}
	}

	log.Printf("Action counts: ADD=%d, UPDATE=%d, DELETE=%d, NONE=%d",
		actionCounts["ADD"], actionCounts["UPDATE"], actionCounts["DELETE"], actionCounts["NONE"])

	return &IntelligentAddResult{Results: results}, nil
}

// fallbackToSimpleAdd falls back to simple add when intelligent add fails.
func (c *Client) fallbackToSimpleAdd(ctx context.Context, messages interface{}, opts ...AddOption) (*IntelligentAddResult, error) {
	// Convert messages to string content
	content := parseMessagesToString(messages)

	// Use the regular Add method
	memory, err := c.Add(ctx, content, opts...)
	if err != nil {
		return nil, fmt.Errorf("fallback to simple add failed: %w", err)
	}

	return &IntelligentAddResult{
		Results: []MemoryActionResult{
			{
				ID:     memory.ID,
				Memory: memory.Content,
				Event:  "ADD",
			},
		},
	}, nil
}

// parseMessagesToString converts various message formats to a string.
func parseMessagesToString(messages interface{}) string {
	switch v := messages.(type) {
	case string:
		return v
	case []map[string]interface{}:
		var parts []string
		for _, msg := range v {
			role, _ := msg["role"].(string)
			content, _ := msg["content"].(string)
			if role != "" && content != "" && role != "system" {
				parts = append(parts, fmt.Sprintf("%s: %s", role, content))
			}
		}
		return fmt.Sprintf("%v", parts)
	case map[string]interface{}:
		content, _ := v["content"].(string)
		return content
	default:
		return fmt.Sprintf("%v", messages)
	}
}

// copyMetadata creates a deep copy of metadata.
func copyMetadata(metadata map[string]interface{}) map[string]interface{} {
	if metadata == nil {
		return make(map[string]interface{})
	}
	result := make(map[string]interface{})
	for k, v := range metadata {
		result[k] = v
	}
	return result
}

// addMetadataFields adds additional fields from options to metadata.
func addMetadataFields(metadata map[string]interface{}, opts *AddOptions) {
	if opts.RunID != "" {
		metadata["run_id"] = opts.RunID
	}
	if opts.MemoryType != "" {
		metadata["memory_type"] = opts.MemoryType
	}
	if opts.Scope != "" {
		metadata["scope"] = string(opts.Scope)
	}
	if opts.Prompt != "" {
		metadata["prompt"] = opts.Prompt
	}
	// Merge filters into metadata
	if opts.Filters != nil {
		for k, v := range opts.Filters {
			metadata[k] = v
		}
	}
}

// truncate truncates a string to the specified length.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
