package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	powermem "github.com/oceanbase/powermem-go/pkg/core"
)

func TestAddOptions(t *testing.T) {
	// Test WithUserID
	opt := powermem.WithUserID("user123")
	assert.NotNil(t, opt)

	// Test WithAgentID
	opt = powermem.WithAgentID("agent456")
	assert.NotNil(t, opt)

	// Test WithMetadata
	metadata := map[string]interface{}{"key": "value"}
	opt = powermem.WithMetadata(metadata)
	assert.NotNil(t, opt)
}

func TestSearchOptions(t *testing.T) {
	// Test WithUserIDForSearch
	opt := powermem.WithUserIDForSearch("user123")
	assert.NotNil(t, opt)

	// Test WithAgentIDForSearch
	opt = powermem.WithAgentIDForSearch("agent456")
	assert.NotNil(t, opt)

	// Test WithLimit
	opt = powermem.WithLimit(10)
	assert.NotNil(t, opt)

	// Test WithFilters (SearchOptions uses Filters instead of MetadataFilter)
	filter := map[string]interface{}{"key": "value"}
	opt = powermem.WithFilters(filter)
	assert.NotNil(t, opt)
}

func TestGetAllOptions(t *testing.T) {
	// Test WithUserIDForGetAll
	opt := powermem.WithUserIDForGetAll("user123")
	assert.NotNil(t, opt)

	// Test WithAgentIDForGetAll
	opt = powermem.WithAgentIDForGetAll("agent456")
	assert.NotNil(t, opt)

	// Test WithLimitForGetAll
	opt = powermem.WithLimitForGetAll(10)
	assert.NotNil(t, opt)

	// Test WithOffset (GetAll operation)
	opt = powermem.WithOffset(5)
	assert.NotNil(t, opt)
}

func TestDeleteAllOptions(t *testing.T) {
	// Test WithUserIDForDeleteAll
	opt := powermem.WithUserIDForDeleteAll("user123")
	assert.NotNil(t, opt)

	// Test WithAgentIDForDeleteAll
	opt = powermem.WithAgentIDForDeleteAll("agent456")
	assert.NotNil(t, opt)
}
