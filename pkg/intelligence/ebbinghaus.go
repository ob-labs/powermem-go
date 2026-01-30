// Package intelligence provides intelligent memory management features including
// deduplication, Ebbinghaus forgetting curve, and importance evaluation.
package intelligence

import (
	"math"
	"time"
)

// EbbinghausManager manages memory retention using the Ebbinghaus forgetting curve algorithm.
//
// The Ebbinghaus forgetting curve describes how information is lost over time
// when there is no attempt to retain it. This manager implements:
//   - Retention strength calculation based on time elapsed
//   - Memory reinforcement on access
//   - Memory classification (working, short-term, long-term)
//   - Review schedule generation
//   - Promotion, forgetting, and archiving decisions
//
// Example usage:
//
//	manager := NewEbbinghausManager(0.1, 0.3)
//	retention := manager.CalculateRetention(createdAt, lastAccessedAt)
//	if manager.ShouldPromote(memory) {
//	    // Promote memory to higher tier
//	}
type EbbinghausManager struct {
	// decayRate is the rate at which memories decay over time.
	// Higher values mean faster decay. Typical range: 0.05-0.2
	decayRate float64

	// reinforcementFactor determines how much memories are strengthened on access.
	// Higher values mean stronger reinforcement. Typical range: 0.2-0.5
	reinforcementFactor float64

	// workingThreshold is the threshold for working memory classification.
	// Memories with retention below this are considered working memory.
	workingThreshold float64

	// shortTermThreshold is the threshold for short-term memory classification.
	// Memories with retention between workingThreshold and shortTermThreshold
	// are considered short-term memory.
	shortTermThreshold float64

	// longTermThreshold is the threshold for long-term memory classification.
	// Memories with retention above longTermThreshold are considered long-term memory.
	longTermThreshold float64

	// initialRetention is the initial retention strength for new memories.
	initialRetention float64

	// reviewIntervals defines the review intervals in hours for spaced repetition.
	// Default: [1, 6, 24, 72, 168] (1 hour, 6 hours, 1 day, 3 days, 1 week)
	reviewIntervals []float64
}

// NewEbbinghausManager creates a new Ebbinghaus forgetting curve manager.
//
// Parameters:
//   - decayRate: Rate at which memories decay (0.05-0.2 recommended)
//   - reinforcementFactor: How much memories strengthen on access (0.2-0.5 recommended)
//
// Returns a new EbbinghausManager with default thresholds:
//   - workingThreshold: 0.3
//   - shortTermThreshold: 0.6
//   - longTermThreshold: 0.8
//   - initialRetention: 1.0
func NewEbbinghausManager(decayRate, reinforcementFactor float64) *EbbinghausManager {
	return &EbbinghausManager{
		decayRate:           decayRate,
		reinforcementFactor: reinforcementFactor,
		workingThreshold:    0.3,
		shortTermThreshold:  0.6,
		longTermThreshold:   0.8,
		initialRetention:    1.0,
		reviewIntervals:     []float64{1, 6, 24, 72, 168}, // 1h, 6h, 1d, 3d, 1w
	}
}

// NewEbbinghausManagerWithConfig creates a new Ebbinghaus manager with custom configuration.
//
// Parameters:
//   - decayRate: Rate at which memories decay
//   - reinforcementFactor: How much memories strengthen on access
//   - workingThreshold: Threshold for working memory (default: 0.3)
//   - shortTermThreshold: Threshold for short-term memory (default: 0.6)
//   - longTermThreshold: Threshold for long-term memory (default: 0.8)
//   - initialRetention: Initial retention strength (default: 1.0)
func NewEbbinghausManagerWithConfig(
	decayRate, reinforcementFactor float64,
	workingThreshold, shortTermThreshold, longTermThreshold, initialRetention float64,
) *EbbinghausManager {
	return &EbbinghausManager{
		decayRate:           decayRate,
		reinforcementFactor: reinforcementFactor,
		workingThreshold:    workingThreshold,
		shortTermThreshold:  shortTermThreshold,
		longTermThreshold:   longTermThreshold,
		initialRetention:    initialRetention,
		reviewIntervals:     []float64{1, 6, 24, 72, 168},
	}
}

// CalculateRetention calculates the current retention strength of a memory
// based on the Ebbinghaus forgetting curve.
//
// The formula used is: R = e^(-decay_rate * hours_elapsed / 24)
// where R is retention (0-1), and hours_elapsed is time since last access
// (or creation if never accessed).
//
// Parameters:
//   - createdAt: When the memory was created
//   - lastAccessedAt: When the memory was last accessed (nil if never accessed)
//
// Returns retention strength between 0.0 and 1.0, where:
//   - 1.0 = perfect retention (just created/accessed)
//   - 0.0 = completely forgotten
func (m *EbbinghausManager) CalculateRetention(createdAt time.Time, lastAccessedAt *time.Time) float64 {
	now := time.Now()
	var timeElapsed time.Duration

	if lastAccessedAt != nil {
		timeElapsed = now.Sub(*lastAccessedAt)
	} else {
		timeElapsed = now.Sub(createdAt)
	}

	// Convert to hours
	hoursElapsed := timeElapsed.Hours()

	// Apply Ebbinghaus formula: R = e^(-decay_rate * hours_elapsed / 24)
	retention := math.Exp(-m.decayRate * hoursElapsed / 24.0)

	// Ensure retention is within valid range
	if retention > 1.0 {
		return 1.0
	}
	if retention < 0.0 {
		return 0.0
	}

	return retention
}

// Reinforce strengthens a memory when it is accessed.
//
// The reinforcement formula is:
//
//	new_strength = min(1.0, current_strength + reinforcement_factor * (1 - current_strength))
//
// This means:
//   - Memories with low strength get more reinforcement
//   - Memories with high strength get less reinforcement
//   - Strength is capped at 1.0
//
// Parameters:
//   - currentStrength: Current retention strength (0.0-1.0)
//
// Returns the new retention strength after reinforcement.
func (m *EbbinghausManager) Reinforce(currentStrength float64) float64 {
	// Reinforcement formula: new_strength = min(1.0, current_strength + reinforcement_factor * (1 - current_strength))
	newStrength := currentStrength + m.reinforcementFactor*(1.0-currentStrength)
	if newStrength > 1.0 {
		return 1.0
	}
	return newStrength
}

// ClassifyMemoryType classifies a memory based on its retention strength.
//
// Memory types:
//   - "working": retention < workingThreshold (0.3)
//   - "short_term": workingThreshold <= retention < shortTermThreshold (0.3-0.6)
//   - "long_term": retention >= longTermThreshold (>= 0.8)
//
// Parameters:
//   - retentionStrength: Current retention strength (0.0-1.0)
//
// Returns the memory type as a string.
func (m *EbbinghausManager) ClassifyMemoryType(retentionStrength float64) string {
	if retentionStrength >= m.longTermThreshold {
		return "long_term"
	} else if retentionStrength >= m.shortTermThreshold {
		return "short_term"
	}
	return "working"
}

// ShouldPromote determines if a memory should be promoted to a higher tier.
//
// A memory is promoted if:
//   - Access count >= 3 (frequently accessed)
//   - Age > 24 hours (survived initial period)
//   - Importance score >= shortTermThreshold (high importance)
//
// Parameters:
//   - memory: Memory data containing access_count, created_at, importance_score
//
// Returns true if the memory should be promoted.
func (m *EbbinghausManager) ShouldPromote(memory map[string]interface{}) bool {
	// Check access frequency
	if accessCount, ok := memory["access_count"].(int); ok && accessCount >= 3 {
		return true
	}

	// Check recency
	if createdAt, ok := memory["created_at"].(time.Time); ok {
		timeElapsed := time.Since(createdAt)
		if timeElapsed > 24*time.Hour {
			return true
		}
	}

	// Check importance
	if importance, ok := memory["importance_score"].(float64); ok {
		if importance >= m.shortTermThreshold {
			return true
		}
	}

	return false
}

// ShouldForget determines if a memory should be forgotten (deleted).
//
// A memory is forgotten if:
//   - Retention strength < workingThreshold (too weak)
//   - Never accessed AND age > 7 days (unused old memory)
//
// Parameters:
//   - memory: Memory data containing created_at, access_count, retention_strength
//
// Returns true if the memory should be forgotten.
func (m *EbbinghausManager) ShouldForget(memory map[string]interface{}) bool {
	// Check retention strength
	if retention, ok := memory["retention_strength"].(float64); ok {
		if retention < m.workingThreshold {
			return true
		}
	}

	// Check if never accessed and old enough
	if accessCount, ok := memory["access_count"].(int); ok && accessCount == 0 {
		if createdAt, ok := memory["created_at"].(time.Time); ok {
			timeElapsed := time.Since(createdAt)
			if timeElapsed > 7*24*time.Hour { // 7 days
				return true
			}
		}
	}

	return false
}

// ShouldArchive determines if a memory should be archived.
//
// A memory is archived if:
//   - Age > 30 days (very old)
//   - Importance score < workingThreshold (low importance)
//
// Parameters:
//   - memory: Memory data containing created_at, importance_score
//
// Returns true if the memory should be archived.
func (m *EbbinghausManager) ShouldArchive(memory map[string]interface{}) bool {
	// Check age
	if createdAt, ok := memory["created_at"].(time.Time); ok {
		timeElapsed := time.Since(createdAt)
		if timeElapsed > 30*24*time.Hour { // 30 days
			return true
		}
	}

	// Check importance
	if importance, ok := memory["importance_score"].(float64); ok {
		if importance < m.workingThreshold {
			return true
		}
	}

	return false
}

// GenerateReviewSchedule generates a review schedule for spaced repetition
// based on the Ebbinghaus curve and importance score.
//
// Higher importance memories get more frequent reviews (shorter intervals).
// The schedule is adjusted based on importance: intervals are reduced by
// (importance * 0.3) to make important memories reviewed more often.
//
// Parameters:
//   - createdAt: When the memory was created
//   - importanceScore: Importance score (0.0-1.0)
//
// Returns a list of review times in chronological order.
func (m *EbbinghausManager) GenerateReviewSchedule(createdAt time.Time, importanceScore float64) []time.Time {
	// Adjust intervals based on importance
	// Higher importance = shorter intervals (more frequent reviews)
	adjustedIntervals := make([]float64, len(m.reviewIntervals))
	for i, interval := range m.reviewIntervals {
		// Reduce interval by importance factor (max 30% reduction)
		adjustedInterval := interval * (1 - importanceScore*0.3)
		// Minimum interval is 0.5 hours
		if adjustedInterval < 0.5 {
			adjustedInterval = 0.5
		}
		adjustedIntervals[i] = adjustedInterval
	}

	// Calculate review times
	reviewTimes := make([]time.Time, len(adjustedIntervals))
	for i, interval := range adjustedIntervals {
		reviewTimes[i] = createdAt.Add(time.Duration(interval) * time.Hour)
	}

	return reviewTimes
}

// CalculateNextReview calculates the next review time for a memory.
//
// The next review time is based on the current retention strength:
//
//	hours_until_review = 24 * (1 + strength * 10)
//
// This means:
//   - Strong memories (strength=1.0) have longer intervals (24 * 11 = 264 hours ≈ 11 days)
//   - Weak memories (strength=0.0) have shorter intervals (24 * 1 = 24 hours)
//
// Parameters:
//   - retentionStrength: Current retention strength (0.0-1.0)
//
// Returns the next review time.
func (m *EbbinghausManager) CalculateNextReview(retentionStrength float64) time.Time {
	// Review interval (hours) = 24 * (1 + strength * 10)
	// Higher strength = longer interval
	hoursUntilReview := 24.0 * (1.0 + retentionStrength*10.0)
	return time.Now().Add(time.Duration(hoursUntilReview) * time.Hour)
}

// GetDecayRateForType returns the decay rate for a specific memory type.
//
// Different memory types have different decay rates:
//   - working: 2x base decay rate (faster decay)
//   - short_term: 1.5x base decay rate (medium decay)
//   - long_term: 1x base decay rate (standard decay)
//
// Parameters:
//   - memoryType: Memory type ("working", "short_term", "long_term")
//
// Returns the adjusted decay rate for the memory type.
func (m *EbbinghausManager) GetDecayRateForType(memoryType string) float64 {
	switch memoryType {
	case "working":
		return m.decayRate * 2.0 // Faster decay for working memory
	case "short_term":
		return m.decayRate * 1.5 // Medium decay for short-term
	case "long_term":
		return m.decayRate // Standard decay for long-term
	default:
		return m.decayRate
	}
}

// ShouldArchive is a convenience method that checks if retention strength
// is below a threshold (for backward compatibility).
//
// Deprecated: Use ShouldArchive with full memory map instead.
func (m *EbbinghausManager) ShouldArchiveByThreshold(retentionStrength float64, threshold float64) bool {
	if threshold == 0 {
		threshold = 0.2 // Default threshold
	}
	return retentionStrength < threshold
}
