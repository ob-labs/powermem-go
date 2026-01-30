package intelligence_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/oceanbase/powermem-go/pkg/intelligence"
)

func TestEbbinghausManager(t *testing.T) {
	decayRate := 0.1
	reinforcementFactor := 0.3

	manager := intelligence.NewEbbinghausManager(decayRate, reinforcementFactor)
	assert.NotNil(t, manager)
}

func TestCalculateRetention(t *testing.T) {
	decayRate := 0.1
	reinforcementFactor := 0.3

	manager := intelligence.NewEbbinghausManager(decayRate, reinforcementFactor)

	// Test initial strength (just created)
	createdAt := time.Now()
	retention := manager.CalculateRetention(createdAt, nil)
	assert.Greater(t, retention, 0.0, "Retention strength should be greater than 0")
	assert.LessOrEqual(t, retention, 1.0, "Retention strength should not exceed 1.0")

	// Test time decay (1 day later)
	createdAt = time.Now().Add(-24 * time.Hour)
	retention = manager.CalculateRetention(createdAt, nil)
	assert.Less(t, retention, 1.0, "Time decay should reduce strength")
	assert.Greater(t, retention, 0.0, "Strength should be greater than 0")

	// Test access reinforcement
	currentStrength := 0.5
	reinforced := manager.Reinforce(currentStrength)
	assert.Greater(t, reinforced, currentStrength, "Reinforcement should increase strength")
	assert.LessOrEqual(t, reinforced, 1.0, "Strength should not exceed 1.0")
}

func TestEbbinghausDecay(t *testing.T) {
	decayRate := 0.1
	reinforcementFactor := 0.3

	manager := intelligence.NewEbbinghausManager(decayRate, reinforcementFactor)

	// Test decay at different time points
	now := time.Now()
	testCases := []struct {
		hoursAgo  float64
		wantLower bool
	}{
		{0, false},
		{1, true},
		{24, true},
		{168, true}, // 1 week
	}

	for _, tc := range testCases {
		createdAt := now.Add(-time.Duration(tc.hoursAgo) * time.Hour)
		retention := manager.CalculateRetention(createdAt, nil)
		if tc.wantLower {
			assert.Less(t, retention, 1.0,
				"Strength should decrease after %v hours", tc.hoursAgo)
		}
		assert.Greater(t, retention, 0.0, "Strength should always be greater than 0")
		assert.LessOrEqual(t, retention, 1.0, "Strength should not exceed 1.0")
	}
}

func TestReinforcementFactor(t *testing.T) {
	decayRate := 0.1
	reinforcementFactor := 0.3

	manager := intelligence.NewEbbinghausManager(decayRate, reinforcementFactor)

	// Test reinforcement function
	currentStrength := 0.5
	reinforced := manager.Reinforce(currentStrength)

	assert.Greater(t, reinforced, currentStrength,
		"Reinforcement should increase memory strength")
	assert.LessOrEqual(t, reinforced, 1.0, "Strength should not exceed 1.0")
}

func TestEbbinghausEdgeCases(t *testing.T) {
	decayRate := 0.1
	reinforcementFactor := 0.3

	manager := intelligence.NewEbbinghausManager(decayRate, reinforcementFactor)

	// Test edge cases
	now := time.Now()

	// Created a long time ago
	oldCreatedAt := now.Add(-1000 * time.Hour)
	retention := manager.CalculateRetention(oldCreatedAt, nil)
	assert.Greater(t, retention, 0.0)
	assert.Less(t, retention, 1.0)

	// Test reinforcement upper limit
	highStrength := 0.99
	reinforced := manager.Reinforce(highStrength)
	assert.LessOrEqual(t, reinforced, 1.0, "Should not exceed 1.0 after reinforcement")
}
