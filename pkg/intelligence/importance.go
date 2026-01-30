// Package intelligence provides intelligent memory management features.
package intelligence

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/oceanbase/powermem-go/pkg/llm"
)

// ImportanceEvaluator evaluates the importance of memory content.
//
// It supports two evaluation modes:
//   - LLM-based: Uses LLM to evaluate importance (more accurate, requires LLM)
//   - Rule-based: Uses keyword matching and heuristics (faster, no LLM required)
//
// The evaluator considers multiple criteria:
//   - Relevance: How relevant the content is to the user
//   - Novelty: Whether the content contains new information
//   - Emotional impact: Emotional significance of the content
//   - Actionable: Whether the content requires action
//   - Factual: Whether the content contains factual information
//   - Personal: Whether the content is personal to the user
//
// Example usage:
//
//	evaluator := NewImportanceEvaluator(llmProvider)
//	score := evaluator.EvaluateImportance(ctx, "User's birthday is March 15th", nil, nil)
//	// score will be between 0.0 and 1.0
type ImportanceEvaluator struct {
	// llm is the LLM provider for LLM-based evaluation.
	// If nil, falls back to rule-based evaluation.
	llm llm.Provider

	// criteriaWeights defines the weight of each evaluation criterion.
	criteriaWeights map[string]float64

	// useLLM indicates whether to use LLM-based evaluation.
	// If false, always uses rule-based evaluation.
	useLLM bool
}

// NewImportanceEvaluator creates a new importance evaluator.
//
// Parameters:
//   - llm: LLM provider for LLM-based evaluation (can be nil for rule-based only)
//
// Returns a new ImportanceEvaluator with default criterion weights:
//   - relevance: 0.3
//   - novelty: 0.2
//   - emotional_impact: 0.15
//   - actionable: 0.15
//   - factual: 0.1
//   - personal: 0.1
func NewImportanceEvaluator(llm llm.Provider) *ImportanceEvaluator {
	return &ImportanceEvaluator{
		llm:    llm,
		useLLM: llm != nil,
		criteriaWeights: map[string]float64{
			"relevance":        0.3,
			"novelty":          0.2,
			"emotional_impact": 0.15,
			"actionable":       0.15,
			"factual":          0.1,
			"personal":         0.1,
		},
	}
}

// EvaluateImportance evaluates the importance of content.
//
// The evaluation uses LLM-based evaluation if available, otherwise falls back
// to rule-based evaluation. The result is a score between 0.0 and 1.0, where:
//   - 1.0 = extremely important
//   - 0.5 = moderately important
//   - 0.0 = not important
//
// Parameters:
//   - ctx: Context for cancellation
//   - content: Content to evaluate
//   - metadata: Additional metadata (optional)
//   - context: Additional context (optional)
//
// Returns importance score between 0.0 and 1.0.
func (e *ImportanceEvaluator) EvaluateImportance(
	ctx context.Context,
	content string,
	metadata map[string]interface{},
	context map[string]interface{},
) float64 {
	if e.useLLM && e.llm != nil {
		score, err := e.evaluateWithLLM(ctx, content, metadata, context)
		if err == nil {
			return score
		}
		// Fall back to rule-based if LLM fails
	}

	return e.evaluateWithRules(content, metadata, context)
}

// evaluateWithLLM evaluates importance using LLM.
func (e *ImportanceEvaluator) evaluateWithLLM(
	ctx context.Context,
	content string,
	metadata map[string]interface{},
	context map[string]interface{},
) (float64, error) {
	// Build evaluation prompt
	systemPrompt := `You are an importance evaluator for memory content. 
Evaluate the importance of the given content on a scale from 0.0 to 1.0.
Consider factors like relevance, novelty, emotional impact, actionability, and personal significance.
Return a JSON object with an "importance_score" field.`

	userPrompt := fmt.Sprintf("Content: %s\n\nEvaluate the importance and return JSON: {\"importance_score\": 0.0-1.0}", content)

	// Call LLM
	messages := []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	response, err := e.llm.GenerateWithMessages(ctx, messages)
	if err != nil {
		return 0.5, err
	}

	// Parse response
	score := e.parseImportanceResponse(response)
	return score, nil
}

// evaluateWithRules evaluates importance using rule-based heuristics.
func (e *ImportanceEvaluator) evaluateWithRules(
	content string,
	metadata map[string]interface{},
	context map[string]interface{},
) float64 {
	score := 0.0
	contentLower := strings.ToLower(content)

	// Length factor
	if len(content) > 100 {
		score += 0.1
	} else if len(content) > 50 {
		score += 0.05
	}

	// Keyword importance
	importantKeywords := []string{
		"important", "critical", "urgent", "remember", "note",
		"preference", "like", "dislike", "hate", "love",
		"password", "secret", "private", "confidential",
	}
	for _, keyword := range importantKeywords {
		if strings.Contains(contentLower, keyword) {
			score += 0.1
		}
	}

	// Question factor
	if strings.Contains(content, "?") {
		score += 0.05
	}

	// Exclamation factor
	if strings.Contains(content, "!") {
		score += 0.05
	}

	// Metadata factors
	if metadata != nil {
		if priority, ok := metadata["priority"].(string); ok {
			switch priority {
			case "high":
				score += 0.2
			case "medium":
				score += 0.1
			}
		}

		if tags, ok := metadata["tags"].([]interface{}); ok && len(tags) > 0 {
			score += 0.05
		}
	}

	// Context factors
	if context != nil {
		if engagement, ok := context["user_engagement"].(string); ok {
			switch engagement {
			case "high":
				score += 0.1
			case "medium":
				score += 0.05
			}
		}
	}

	// Cap at 1.0
	return math.Min(score, 1.0)
}

// parseImportanceResponse parses LLM response to extract importance score.
func (e *ImportanceEvaluator) parseImportanceResponse(response string) float64 {
	// Try to extract JSON
	if strings.Contains(response, "{") && strings.Contains(response, "}") {
		start := strings.Index(response, "{")
		end := strings.LastIndex(response, "}") + 1
		if start >= 0 && end > start {
			jsonStr := response[start:end]
			var result map[string]interface{}
			if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
				if score, ok := result["importance_score"].(float64); ok {
					return math.Max(0.0, math.Min(1.0, score))
				}
			}
		}
	}

	// Fallback: extract number from response
	re := regexp.MustCompile(`\d+\.?\d*`)
	matches := re.FindAllString(response, -1)
	if len(matches) > 0 {
		var score float64
		if _, err := fmt.Sscanf(matches[0], "%f", &score); err == nil {
			return math.Max(0.0, math.Min(1.0, score))
		}
	}

	// Default medium importance
	return 0.5
}

// GetImportanceBreakdown returns a detailed breakdown of importance by criterion.
//
// This method evaluates each criterion separately and returns a map with
// criterion names as keys and scores (0.0-1.0) as values.
//
// Parameters:
//   - content: Content to evaluate
//   - metadata: Additional metadata (optional)
//   - context: Additional context (optional)
//
// Returns a map with criterion scores.
func (e *ImportanceEvaluator) GetImportanceBreakdown(
	content string,
	metadata map[string]interface{},
	context map[string]interface{},
) map[string]float64 {
	breakdown := make(map[string]float64)

	for criterion := range e.criteriaWeights {
		var score float64
		switch criterion {
		case "relevance":
			score = e.evaluateRelevance(content, context)
		case "novelty":
			score = e.evaluateNovelty(content, metadata)
		case "emotional_impact":
			score = e.evaluateEmotionalImpact(content)
		case "actionable":
			score = e.evaluateActionable(content)
		case "factual":
			score = e.evaluateFactual(content)
		case "personal":
			score = e.evaluatePersonal(content, metadata)
		default:
			score = 0.0
		}
		breakdown[criterion] = score
	}

	return breakdown
}

// evaluateRelevance evaluates how relevant the content is.
func (e *ImportanceEvaluator) evaluateRelevance(content string, context map[string]interface{}) float64 {
	contentLower := strings.ToLower(content)
	relevanceKeywords := []string{"relevant", "related", "connected", "associated"}

	score := 0.0
	for _, keyword := range relevanceKeywords {
		if strings.Contains(contentLower, keyword) {
			score += 0.25
		}
	}

	return math.Min(score, 1.0)
}

// evaluateNovelty evaluates whether content contains new information.
func (e *ImportanceEvaluator) evaluateNovelty(content string, metadata map[string]interface{}) float64 {
	contentLower := strings.ToLower(content)
	noveltyIndicators := []string{"new", "first", "never", "unprecedented", "unique"}

	score := 0.0
	for _, indicator := range noveltyIndicators {
		if strings.Contains(contentLower, indicator) {
			score += 0.2
		}
	}

	return math.Min(score, 1.0)
}

// evaluateEmotionalImpact evaluates the emotional significance of content.
func (e *ImportanceEvaluator) evaluateEmotionalImpact(content string) float64 {
	contentLower := strings.ToLower(content)
	emotionalWords := []string{
		"happy", "sad", "angry", "excited", "worried", "scared",
		"love", "hate", "fear", "joy", "sorrow", "anger",
	}

	score := 0.0
	for _, word := range emotionalWords {
		if strings.Contains(contentLower, word) {
			score += 0.1
		}
	}

	return math.Min(score, 1.0)
}

// evaluateActionable evaluates whether content requires action.
func (e *ImportanceEvaluator) evaluateActionable(content string) float64 {
	contentLower := strings.ToLower(content)
	actionWords := []string{
		"do", "make", "create", "build", "fix", "solve",
		"implement", "execute", "perform", "complete",
	}

	score := 0.0
	for _, word := range actionWords {
		if strings.Contains(contentLower, word) {
			score += 0.1
		}
	}

	return math.Min(score, 1.0)
}

// evaluateFactual evaluates whether content contains factual information.
func (e *ImportanceEvaluator) evaluateFactual(content string) float64 {
	contentLower := strings.ToLower(content)
	factualIndicators := []string{
		"fact", "data", "statistic", "research", "study",
		"evidence", "proof", "confirmed", "verified",
	}

	score := 0.0
	for _, indicator := range factualIndicators {
		if strings.Contains(contentLower, indicator) {
			score += 0.15
		}
	}

	return math.Min(score, 1.0)
}

// evaluatePersonal evaluates whether content is personal to the user.
func (e *ImportanceEvaluator) evaluatePersonal(content string, metadata map[string]interface{}) float64 {
	contentLower := strings.ToLower(content)
	personalIndicators := []string{
		"i ", "me ", "my ", "mine ", "myself",
		"personal", "private", "confidential",
	}

	score := 0.0
	for _, indicator := range personalIndicators {
		if strings.Contains(contentLower, indicator) {
			score += 0.1
		}
	}

	return math.Min(score, 1.0)
}
