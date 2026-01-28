// Package intelligence provides intelligent memory management features.
package intelligence

import (
	"context"
	"strings"
	"time"

	"github.com/oceanbase/powermem-go/pkg/llm"
)

// IntelligentMemoryManager manages intelligent memory processing.
//
// It integrates multiple components:
//   - ImportanceEvaluator: Evaluates memory importance
//   - EbbinghausManager: Manages retention and decay
//   - FactExtractor: Extracts facts from messages
//
// The manager processes memories through the following pipeline:
//  1. Extract facts from messages
//  2. Evaluate importance of each fact
//  3. Classify memory type (working/short-term/long-term)
//  4. Generate review schedule
//  5. Apply Ebbinghaus decay and reinforcement
//
// Example usage:
//
//	manager := NewIntelligentMemoryManager(llmProvider, &Config{
//	    DecayRate:           0.1,
//	    ReinforcementFactor: 0.3,
//	})
//	metadata := manager.ProcessMetadata(ctx, "User likes Python", nil, nil)
type IntelligentMemoryManager struct {
	// importanceEvaluator evaluates the importance of memory content.
	importanceEvaluator *ImportanceEvaluator

	// ebbinghausManager manages retention using Ebbinghaus curve.
	ebbinghausManager *EbbinghausManager

	// factExtractor extracts facts from messages.
	factExtractor *FactExtractor

	// config contains the configuration for intelligent memory.
	config *Config
}

// Config contains configuration for intelligent memory management.
type Config struct {
	// DecayRate is the rate at which memories decay over time.
	DecayRate float64

	// ReinforcementFactor determines how much memories are strengthened on access.
	ReinforcementFactor float64

	// WorkingThreshold is the threshold for working memory classification.
	WorkingThreshold float64

	// ShortTermThreshold is the threshold for short-term memory classification.
	ShortTermThreshold float64

	// LongTermThreshold is the threshold for long-term memory classification.
	LongTermThreshold float64

	// InitialRetention is the initial retention strength for new memories.
	InitialRetention float64

	// FallbackToSimpleAdd indicates whether to fallback to simple add mode
	// when intelligent processing fails.
	FallbackToSimpleAdd bool
}

// DefaultConfig returns a default configuration for intelligent memory.
func DefaultConfig() *Config {
	return &Config{
		DecayRate:           0.1,
		ReinforcementFactor: 0.3,
		WorkingThreshold:    0.3,
		ShortTermThreshold:  0.6,
		LongTermThreshold:   0.8,
		InitialRetention:    1.0,
		FallbackToSimpleAdd: false,
	}
}

// NewIntelligentMemoryManager creates a new intelligent memory manager.
//
// Parameters:
//   - llm: LLM provider for importance evaluation and fact extraction
//   - config: Configuration for intelligent memory (nil uses defaults)
//
// Returns a new IntelligentMemoryManager with all components initialized.
func NewIntelligentMemoryManager(llm llm.Provider, config *Config) *IntelligentMemoryManager {
	if config == nil {
		config = DefaultConfig()
	}

	// Initialize components
	importanceEvaluator := NewImportanceEvaluator(llm)
	factExtractor := NewFactExtractor(llm)
	ebbinghausManager := NewEbbinghausManagerWithConfig(
		config.DecayRate,
		config.ReinforcementFactor,
		config.WorkingThreshold,
		config.ShortTermThreshold,
		config.LongTermThreshold,
		config.InitialRetention,
	)

	return &IntelligentMemoryManager{
		importanceEvaluator: importanceEvaluator,
		ebbinghausManager:   ebbinghausManager,
		factExtractor:       factExtractor,
		config:              config,
	}
}

// ProcessMetadata processes memory metadata with intelligent analysis.
//
// This method:
//  1. Evaluates importance of the content
//  2. Determines memory type based on importance
//  3. Generates intelligence metadata including:
//     - Importance score
//     - Memory type (working/short-term/long-term)
//     - Initial retention strength
//     - Review schedule
//     - Decay rate
//
// Parameters:
//   - ctx: Context for cancellation
//   - content: Content to analyze
//   - metadata: Existing metadata (optional)
//   - context: Additional context (optional)
//
// Returns enhanced metadata with intelligence analysis.
func (m *IntelligentMemoryManager) ProcessMetadata(
	ctx context.Context,
	content string,
	metadata map[string]interface{},
	context map[string]interface{},
) map[string]interface{} {
	// Initialize metadata if nil
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	// Evaluate importance
	importanceScore := m.importanceEvaluator.EvaluateImportance(ctx, content, metadata, context)

	// Determine memory type based on importance
	var memoryType string
	if importanceScore >= m.config.LongTermThreshold {
		memoryType = "long_term"
	} else if importanceScore >= m.config.ShortTermThreshold {
		memoryType = "short_term"
	} else {
		memoryType = "working"
	}

	// Calculate initial retention based on importance
	initialRetention := m.config.InitialRetention * importanceScore

	// Get decay rate for memory type
	decayRate := m.ebbinghausManager.GetDecayRateForType(memoryType)

	// Generate review schedule
	reviewSchedule := m.ebbinghausManager.GenerateReviewSchedule(time.Now(), importanceScore)

	// Build intelligence metadata
	intelligenceData := map[string]interface{}{
		"importance_score":     importanceScore,
		"memory_type":          memoryType,
		"initial_retention":    initialRetention,
		"current_retention":    initialRetention,
		"decay_rate":           decayRate,
		"review_schedule":      reviewSchedule,
		"last_reviewed":        time.Now(),
		"review_count":         0,
		"access_count":         0,
		"reinforcement_factor": m.config.ReinforcementFactor,
	}

	memoryManagement := map[string]interface{}{
		"should_promote": false,
		"should_forget":  false,
		"should_archive": false,
		"is_active":      true,
	}

	// Merge intelligence metadata
	enhancedMetadata := make(map[string]interface{})
	for k, v := range metadata {
		enhancedMetadata[k] = v
	}

	enhancedMetadata["intelligence"] = intelligenceData
	enhancedMetadata["memory_management"] = memoryManagement
	enhancedMetadata["created_at"] = time.Now()
	enhancedMetadata["updated_at"] = time.Now()

	return enhancedMetadata
}

// ExtractFacts extracts facts from messages.
//
// This is a convenience method that delegates to the FactExtractor.
//
// Parameters:
//   - ctx: Context for cancellation
//   - messages: Messages to extract facts from
//
// Returns a list of extracted fact strings.
func (m *IntelligentMemoryManager) ExtractFacts(ctx context.Context, messages interface{}) ([]string, error) {
	return m.factExtractor.ExtractFacts(ctx, messages)
}

// ProcessSearchResults processes search results with intelligent ranking.
//
// This method:
//  1. Calculates relevance score for each result
//  2. Applies Ebbinghaus decay based on age
//  3. Combines relevance and decay for final score
//  4. Sorts results by final score
//
// Parameters:
//   - ctx: Context for cancellation
//   - results: Search results (list of memory maps)
//   - query: Original search query
//
// Returns processed and ranked results.
func (m *IntelligentMemoryManager) ProcessSearchResults(
	ctx context.Context,
	results []map[string]interface{},
	query string,
) []map[string]interface{} {
	processed := make([]map[string]interface{}, 0, len(results))

	for _, result := range results {
		// Calculate relevance (simple keyword matching)
		relevanceScore := m.calculateRelevance(result, query)

		// Calculate decay based on age
		var decayFactor float64
		if createdAt, ok := result["created_at"].(time.Time); ok {
			var lastAccessedAt *time.Time
			if lastAccess, ok := result["last_accessed_at"].(time.Time); ok {
				lastAccessedAt = &lastAccess
			}
			decayFactor = m.ebbinghausManager.CalculateRetention(createdAt, lastAccessedAt)
		} else {
			decayFactor = 1.0 // No decay if no creation time
		}

		// Calculate final score
		finalScore := relevanceScore * decayFactor

		// Update result
		processedResult := make(map[string]interface{})
		for k, v := range result {
			processedResult[k] = v
		}
		processedResult["relevance_score"] = relevanceScore
		processedResult["decay_factor"] = decayFactor
		processedResult["final_score"] = finalScore

		processed = append(processed, processedResult)
	}

	// Sort by final score (descending)
	for i := 0; i < len(processed)-1; i++ {
		for j := i + 1; j < len(processed); j++ {
			scoreI, _ := processed[i]["final_score"].(float64)
			scoreJ, _ := processed[j]["final_score"].(float64)
			if scoreI < scoreJ {
				processed[i], processed[j] = processed[j], processed[i]
			}
		}
	}

	return processed
}

// calculateRelevance calculates relevance score for a memory given a query.
func (m *IntelligentMemoryManager) calculateRelevance(memory map[string]interface{}, query string) float64 {
	content, ok := memory["content"].(string)
	if !ok {
		return 0.0
	}

	// Simple keyword matching
	queryLower := strings.ToLower(query)
	contentLower := strings.ToLower(content)

	queryWords := strings.Fields(queryLower)
	contentWords := strings.Fields(contentLower)

	matches := 0
	for _, word := range queryWords {
		for _, contentWord := range contentWords {
			if word == contentWord {
				matches++
				break
			}
		}
	}

	if len(queryWords) == 0 {
		return 0.0
	}

	relevanceScore := float64(matches) / float64(len(queryWords))
	if relevanceScore > 1.0 {
		return 1.0
	}
	return relevanceScore
}

// ShouldPromote checks if a memory should be promoted to a higher tier.
func (m *IntelligentMemoryManager) ShouldPromote(memory map[string]interface{}) bool {
	return m.ebbinghausManager.ShouldPromote(memory)
}

// ShouldForget checks if a memory should be forgotten (deleted).
func (m *IntelligentMemoryManager) ShouldForget(memory map[string]interface{}) bool {
	return m.ebbinghausManager.ShouldForget(memory)
}

// ShouldArchive checks if a memory should be archived.
func (m *IntelligentMemoryManager) ShouldArchive(memory map[string]interface{}) bool {
	return m.ebbinghausManager.ShouldArchive(memory)
}

// GetImportanceEvaluator returns the importance evaluator.
func (m *IntelligentMemoryManager) GetImportanceEvaluator() *ImportanceEvaluator {
	return m.importanceEvaluator
}

// GetEbbinghausManager returns the Ebbinghaus manager.
func (m *IntelligentMemoryManager) GetEbbinghausManager() *EbbinghausManager {
	return m.ebbinghausManager
}

// GetFactExtractor returns the fact extractor.
func (m *IntelligentMemoryManager) GetFactExtractor() *FactExtractor {
	return m.factExtractor
}
