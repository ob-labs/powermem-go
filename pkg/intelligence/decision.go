// Package intelligence provides intelligent memory management features.
package intelligence

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/oceanbase/powermem-go/pkg/llm"
)

// MemoryAction represents a memory operation decision from LLM.
type MemoryAction struct {
	// ID is the memory ID (for UPDATE/DELETE operations)
	ID string `json:"id"`

	// Text is the memory content
	Text string `json:"text"`

	// Memory is an alternative field name for Text (for compatibility)
	Memory string `json:"memory"`

	// Event is the operation type: ADD, UPDATE, DELETE, or NONE
	Event string `json:"event"`

	// OldMemory is the previous memory content (for UPDATE operations)
	OldMemory string `json:"old_memory,omitempty"`
}

// DecisionMaker makes intelligent decisions about memory operations.
//
// It uses LLM to analyze new facts against existing memories and decides:
//   - ADD: Create a new memory for novel information
//   - UPDATE: Merge/update existing memory with new information
//   - DELETE: Remove outdated or incorrect memory
//   - NONE: Skip duplicate or irrelevant information
//
// Example usage:
//
//	maker := NewDecisionMaker(llmProvider)
//	actions, err := maker.DecideActions(ctx, facts, existingMemories)
//	for _, action := range actions {
//	    switch action.Event {
//	    case "ADD":
//	        // Create new memory
//	    case "UPDATE":
//	        // Update existing memory
//	    case "DELETE":
//	        // Delete memory
//	    case "NONE":
//	        // Skip (duplicate)
//	    }
//	}
type DecisionMaker struct {
	// llm is the LLM provider for decision making.
	llm llm.Provider

	// customPrompt is an optional custom prompt for decision making.
	customPrompt string
}

// ExistingMemory represents an existing memory for decision making.
type ExistingMemory struct {
	// ID is the temporary ID (will be mapped to real ID)
	ID string `json:"id"`

	// Text is the memory content
	Text string `json:"text"`
}

// NewDecisionMaker creates a new decision maker.
func NewDecisionMaker(llm llm.Provider) *DecisionMaker {
	return &DecisionMaker{
		llm:          llm,
		customPrompt: "",
	}
}

// NewDecisionMakerWithPrompt creates a new decision maker with custom prompt.
func NewDecisionMakerWithPrompt(llm llm.Provider, customPrompt string) *DecisionMaker {
	return &DecisionMaker{
		llm:          llm,
		customPrompt: customPrompt,
	}
}

// DecideActions decides memory actions for new facts against existing memories.
//
// Parameters:
//   - ctx: Context for cancellation
//   - newFacts: List of newly extracted facts
//   - existingMemories: List of existing similar memories
//
// Returns a list of MemoryAction decisions from the LLM.
func (d *DecisionMaker) DecideActions(
	ctx context.Context,
	newFacts []string,
	existingMemories []ExistingMemory,
) ([]MemoryAction, error) {
	if len(newFacts) == 0 {
		return []MemoryAction{}, nil
	}

	// Generate decision prompt
	prompt := d.generateDecisionPrompt(newFacts, existingMemories)

	// Call LLM
	messages := []llm.Message{
		{Role: "user", Content: prompt},
	}

	response, err := d.llm.GenerateWithMessages(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM decision: %w", err)
	}

	// Parse response
	actions, err := d.parseActionsResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	return actions, nil
}

// generateDecisionPrompt generates the prompt for LLM decision making.
func (d *DecisionMaker) generateDecisionPrompt(
	newFacts []string,
	existingMemories []ExistingMemory,
) string {
	if d.customPrompt != "" {
		// Use custom prompt (user should format it themselves)
		return d.customPrompt
	}

	// Format existing memories
	existingMemoriesJSON, _ := json.Marshal(existingMemories)

	// Format new facts
	newFactsJSON, _ := json.Marshal(newFacts)

	// Default prompt (aligned with Python SDK)
	prompt := fmt.Sprintf(`You are a Personal Information Organizer, specialized in managing and organizing personal information. You create, update, or delete memories based on new information and existing memories.

# Existing Memories
%s

# New Facts
%s

# Task
Analyze the new facts against existing memories and decide the appropriate action for each:

## Actions:
- **ADD**: Create a new memory if the fact is novel and doesn't overlap with existing memories
- **UPDATE**: Update an existing memory if the new fact provides additional or corrected information. Merge and consolidate information, keeping the updated memory self-contained and complete.
- **DELETE**: Remove a memory if it's outdated, incorrect, or contradicted by new information
- **NONE**: Skip if the fact is already captured or is not worth storing (e.g., greetings, small talk)

## Important Guidelines:
1. **Deduplication**: Mark facts as NONE if they duplicate existing memories
2. **Consolidation**: When updating, merge information to create complete, self-contained memories
3. **Temporal Information**: Always preserve time references (dates, "yesterday", "last week", etc.)
4. **Completeness**: Updated memories should include who/what/when/where
5. **Clarity**: Each memory should be understandable on its own
6. **ID Accuracy**: When UPDATE/DELETE, use the exact ID from existing memories

## Output Format (JSON):
Return a JSON object with a "memory" array containing action objects:

{
  "memory": [
    {
      "id": "0",
      "text": "Updated memory text",
      "event": "UPDATE",
      "old_memory": "Previous memory text"
    },
    {
      "text": "New memory text",
      "event": "ADD"
    },
    {
      "id": "2",
      "event": "DELETE"
    },
    {
      "text": "Duplicate fact",
      "event": "NONE"
    }
  ]
}

Note: 
- For UPDATE/DELETE, "id" is required and must match an existing memory ID
- For ADD, only "text" and "event" are required
- For NONE, include "text" to show what was skipped

Now analyze the facts and provide your decision:`, string(existingMemoriesJSON), string(newFactsJSON))

	return prompt
}

// parseActionsResponse parses the LLM response to extract memory actions.
func (d *DecisionMaker) parseActionsResponse(response string) ([]MemoryAction, error) {
	// Remove code blocks if present
	response = removeCodeBlocks(response)

	// Try to parse as JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("invalid JSON response: %w", err)
	}

	// Extract memory array
	memoryInterface, ok := result["memory"]
	if !ok {
		return []MemoryAction{}, nil
	}

	memoryArray, ok := memoryInterface.([]interface{})
	if !ok {
		return nil, fmt.Errorf("memory is not an array")
	}

	// Convert to MemoryAction slice
	actions := make([]MemoryAction, 0, len(memoryArray))
	for _, item := range memoryArray {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		action := MemoryAction{}

		// Extract fields
		if id, ok := itemMap["id"].(string); ok {
			action.ID = id
		}
		if text, ok := itemMap["text"].(string); ok {
			action.Text = text
		}
		if memory, ok := itemMap["memory"].(string); ok {
			action.Memory = memory
		}
		if event, ok := itemMap["event"].(string); ok {
			action.Event = strings.ToUpper(event)
		}
		if oldMemory, ok := itemMap["old_memory"].(string); ok {
			action.OldMemory = oldMemory
		}

		// Use Memory field if Text is empty (compatibility)
		if action.Text == "" && action.Memory != "" {
			action.Text = action.Memory
		}

		actions = append(actions, action)
	}

	return actions, nil
}

// removeCodeBlocks removes code blocks (```json ... ```) from response.
func removeCodeBlocks(response string) string {
	// Remove ```json and ``` markers
	response = strings.ReplaceAll(response, "```json", "")
	response = strings.ReplaceAll(response, "```", "")
	return strings.TrimSpace(response)
}
