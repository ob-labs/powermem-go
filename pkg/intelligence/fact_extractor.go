// Package intelligence provides intelligent memory management features.
package intelligence

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/oceanbase/powermem-go/pkg/llm"
)

// FactExtractor extracts facts from messages using LLM.
//
// Facts are self-contained pieces of information extracted from conversations,
// including personal preferences, details, plans, intentions, needs, and activities.
//
// Example usage:
//
//	extractor := NewFactExtractor(llmProvider)
//	facts := extractor.ExtractFacts(ctx, messages)
//	// facts will be a list of extracted fact strings
type FactExtractor struct {
	// llm is the LLM provider for fact extraction.
	llm llm.Provider

	// customPrompt is an optional custom prompt for fact extraction.
	// If empty, uses the default prompt.
	customPrompt string
}

// NewFactExtractor creates a new fact extractor.
//
// Parameters:
//   - llm: LLM provider for fact extraction (required)
//
// Returns a new FactExtractor with default prompt.
func NewFactExtractor(llm llm.Provider) *FactExtractor {
	return &FactExtractor{
		llm:          llm,
		customPrompt: "",
	}
}

// NewFactExtractorWithPrompt creates a new fact extractor with custom prompt.
//
// Parameters:
//   - llm: LLM provider for fact extraction (required)
//   - customPrompt: Custom prompt for fact extraction (optional, uses default if empty)
//
// Returns a new FactExtractor with custom prompt.
func NewFactExtractorWithPrompt(llm llm.Provider, customPrompt string) *FactExtractor {
	return &FactExtractor{
		llm:          llm,
		customPrompt: customPrompt,
	}
}

// ExtractFacts extracts facts from messages.
//
// The extraction process:
//  1. Parses messages into conversation format
//  2. Calls LLM with fact extraction prompt
//  3. Parses JSON response to extract facts list
//
// Facts are extracted with the following rules:
//   - TEMPORAL: Always extract time info (dates, relative refs like "yesterday")
//   - COMPLETE: Extract self-contained facts with who/what/when/where
//   - SEPARATE: Extract distinct facts separately
//   - INTENTIONS: Always extract user intentions, needs, and requests
//
// Parameters:
//   - ctx: Context for cancellation
//   - messages: Messages to extract facts from (can be string, []map[string]interface{}, or single map)
//
// Returns a list of extracted fact strings, or empty list if extraction fails.
func (e *FactExtractor) ExtractFacts(ctx context.Context, messages interface{}) ([]string, error) {
	// Parse messages into conversation format
	conversation := e.parseMessages(messages)

	// Get prompt
	systemPrompt := e.getSystemPrompt()
	userPrompt := fmt.Sprintf("Input:\n%s", conversation)

	// Call LLM
	llmMessages := []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	response, err := e.llm.GenerateWithMessages(ctx, llmMessages)
	if err != nil {
		return nil, fmt.Errorf("failed to extract facts: %w", err)
	}

	// Parse response
	facts, err := e.parseFactsResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse facts response: %w", err)
	}

	return facts, nil
}

// parseMessages parses messages into conversation format.
func (e *FactExtractor) parseMessages(messages interface{}) string {
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
		return strings.Join(parts, "\n")
	case map[string]interface{}:
		role, _ := v["role"].(string)
		content, _ := v["content"].(string)
		if role != "" && content != "" {
			return fmt.Sprintf("%s: %s", role, content)
		}
		return ""
	default:
		return fmt.Sprintf("%v", messages)
	}
}

// getSystemPrompt returns the system prompt for fact extraction.
func (e *FactExtractor) getSystemPrompt() string {
	if e.customPrompt != "" {
		return e.customPrompt
	}

	// Default fact extraction prompt
	today := time.Now().Format("2006-01-02")
	return fmt.Sprintf(`You are a Personal Information Organizer. Extract relevant facts, memories, preferences, intentions, and needs from conversations into distinct, manageable facts.

Information Types: Personal preferences, details (names, relationships, dates), plans, intentions, needs, requests, activities, health/wellness (including medical appointments, symptoms, treatments), professional, miscellaneous.

CRITICAL Rules:
1. TEMPORAL: ALWAYS extract time info (dates, relative refs like "yesterday", "last week"). Include in facts (e.g., "Went to Hawaii in May 2023" or "Went to Hawaii last year", not just "Went to Hawaii"). Preserve relative time refs for later calculation.
2. COMPLETE: Extract self-contained facts with who/what/when/where when available.
3. SEPARATE: Extract distinct facts separately, especially when they have different time periods.
4. INTENTIONS & NEEDS: ALWAYS extract user intentions, needs, and requests even without time information. Examples: "Want to book a doctor appointment", "Need to call someone", "Plan to visit a place".

Examples:
Input: Hi.
Output: {"facts" : []}

Input: Yesterday, I met John at 3pm. We discussed the project.
Output: {"facts" : ["Met John at 3pm yesterday", "Discussed project with John yesterday"]}

Input: Last May, I went to India. Visited Mumbai and Goa.
Output: {"facts" : ["Went to India in May", "Visited Mumbai in May", "Visited Goa in May"]}

Input: I met Sarah last year and became friends. We went to movies last month.
Output: {"facts" : ["Met Sarah last year and became friends", "Went to movies with Sarah last month"]}

Input: I'm John, a software engineer.
Output: {"facts" : ["Name is John", "John is a software engineer"]}

Input: I want to book an appointment with a cardiologist.
Output: {"facts" : ["Want to book an appointment with a cardiologist"]}

Rules:
- Today: %s
- Return JSON: {"facts": ["fact1", "fact2"]}
- Extract from user/assistant messages only
- Extract intentions, needs, and requests even without time information
- If no relevant facts, return empty list
- Preserve input language

Extract facts from the conversation below:`, today)
}

// parseFactsResponse parses LLM response to extract facts.
func (e *FactExtractor) parseFactsResponse(response string) ([]string, error) {
	// Remove code blocks if present
	response = e.removeCodeBlocks(response)

	// Try to parse as JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("invalid JSON response: %w", err)
	}

	// Extract facts array
	factsInterface, ok := result["facts"]
	if !ok {
		return []string{}, nil
	}

	factsArray, ok := factsInterface.([]interface{})
	if !ok {
		return nil, fmt.Errorf("facts is not an array")
	}

	// Convert to string slice
	facts := make([]string, 0, len(factsArray))
	for _, fact := range factsArray {
		if factStr, ok := fact.(string); ok && factStr != "" {
			facts = append(facts, factStr)
		}
	}

	return facts, nil
}

// removeCodeBlocks removes code blocks (```json ... ```) from response.
func (e *FactExtractor) removeCodeBlocks(response string) string {
	// Remove ```json and ``` markers
	response = strings.ReplaceAll(response, "```json", "")
	response = strings.ReplaceAll(response, "```", "")
	return strings.TrimSpace(response)
}
