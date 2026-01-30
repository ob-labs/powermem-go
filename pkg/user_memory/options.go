// Package usermemory provides user memory management with automatic profile extraction.
package usermemory

import "github.com/oceanbase/powermem-go/pkg/core"

// AddResult contains the result of an Add operation.
//
// It includes both the created memory and profile extraction results.
type AddResult struct {
	// Memory is the created or merged memory.
	Memory *core.Memory

	// ProfileExtracted indicates whether a profile was extracted/updated.
	ProfileExtracted bool

	// ProfileContent is the extracted unstructured profile content (if extracted).
	ProfileContent *string

	// Topics is the extracted structured topics (if extracted).
	Topics map[string]interface{}
}

// AddOptions contains configuration options for Add operations.
type AddOptions struct {
	// UserID identifies the user.
	UserID string

	// AgentID identifies the agent.
	AgentID string

	// RunID identifies the run/session.
	RunID string

	// Metadata contains additional metadata about the memory.
	Metadata map[string]interface{}

	// Filters provides additional metadata filters.
	Filters map[string]interface{}

	// Scope defines the visibility scope of the memory.
	Scope core.MemoryScope

	// MemoryType specifies the type of memory (e.g., "conversation", "fact", "preference").
	MemoryType string

	// Prompt is an optional prompt used for memory processing.
	Prompt string

	// Infer enables intelligent deduplication.
	Infer bool

	// ProfileType specifies the type of profile extraction:
	//   - "content": Extract unstructured profile content (default)
	//   - "topics": Extract structured topics
	ProfileType string

	// CustomTopics is a JSON string defining custom topic structure (for "topics" mode).
	CustomTopics string

	// StrictMode enables strict mode for topic extraction (for "topics" mode).
	StrictMode bool

	// IncludeRoles specifies which roles to include when filtering messages for profile extraction.
	IncludeRoles []string

	// ExcludeRoles specifies which roles to exclude when filtering messages for profile extraction.
	ExcludeRoles []string
}

// AddOption is a function type for configuring Add operations.
type AddOption func(*AddOptions)

// WithUserID sets the user ID for Add operations.
//
// Example:
//
//	result, _ := client.Add(ctx, messages, usermemory.WithUserID("user_001"))
func WithUserID(userID string) AddOption {
	return func(opts *AddOptions) {
		opts.UserID = userID
	}
}

// WithAgentID sets the agent ID for Add operations.
//
// Example:
//
//	result, _ := client.Add(ctx, messages, usermemory.WithAgentID("agent_001"))
func WithAgentID(agentID string) AddOption {
	return func(opts *AddOptions) {
		opts.AgentID = agentID
	}
}

// WithProfileType sets the profile extraction type.
//
// Valid values:
//   - "content": Extract unstructured profile content (default)
//   - "topics": Extract structured topics
//
// Example:
//
//	result, _ := client.Add(ctx, messages,
//	    usermemory.WithProfileType("topics"),
//	    usermemory.WithCustomTopics(`{"occupation": true, "interests": true}`),
//	)
func WithProfileType(profileType string) AddOption {
	return func(opts *AddOptions) {
		opts.ProfileType = profileType
	}
}

// WithCustomTopics sets custom topic structure for structured extraction.
//
// The customTopics parameter should be a JSON string defining which topics
// to extract and their structure. Only used when ProfileType is "topics".
//
// Example:
//
//	customTopics := `{"occupation": true, "interests": ["hobby1", "hobby2"]}`
//	result, _ := client.Add(ctx, messages,
//	    usermemory.WithProfileType("topics"),
//	    usermemory.WithCustomTopics(customTopics),
//	)
func WithCustomTopics(customTopics string) AddOption {
	return func(opts *AddOptions) {
		opts.CustomTopics = customTopics
	}
}

// WithStrictMode enables strict mode for topic extraction.
//
// In strict mode, only topics that match the custom structure exactly are extracted.
// Only used when ProfileType is "topics".
func WithStrictMode(strictMode bool) AddOption {
	return func(opts *AddOptions) {
		opts.StrictMode = strictMode
	}
}

// WithRunID sets the run ID for Add operations.
//
// RunID identifies a specific run or session, useful for grouping related memories.
//
// Example:
//
//	result, _ := client.Add(ctx, messages,
//	    usermemory.WithUserID("user_001"),
//	    usermemory.WithRunID("run_001"),
//	)
func WithRunID(runID string) AddOption {
	return func(opts *AddOptions) {
		opts.RunID = runID
	}
}

// WithMetadata sets metadata for Add operations.
//
// Metadata can be used for filtering and additional context.
//
// Example:
//
//	result, _ := client.Add(ctx, messages,
//	    usermemory.WithMetadata(map[string]interface{}{
//	        "source": "conversation",
//	        "priority": "high",
//	    }),
//	)
func WithMetadata(metadata map[string]interface{}) AddOption {
	return func(opts *AddOptions) {
		opts.Metadata = metadata
	}
}

// WithFilters sets metadata filters for Add operations.
//
// Filters can be used for additional filtering and categorization.
//
// Example:
//
//	result, _ := client.Add(ctx, messages,
//	    usermemory.WithFilters(map[string]interface{}{
//	        "category": "conversation",
//	    }),
//	)
func WithFilters(filters map[string]interface{}) AddOption {
	return func(opts *AddOptions) {
		opts.Filters = filters
	}
}

// WithScope sets the memory scope for Add operations.
//
// Scope controls visibility: "private", "agent_group", or "global".
//
// Example:
//
//	result, _ := client.Add(ctx, messages, usermemory.WithScope("global"))
func WithScope(scope string) AddOption {
	return func(opts *AddOptions) {
		opts.Scope = core.MemoryScope(scope)
	}
}

// WithMemoryType sets the memory type for Add operations.
//
// MemoryType categorizes the type of memory (e.g., "conversation", "fact", "preference").
//
// Example:
//
//	result, _ := client.Add(ctx, messages, usermemory.WithMemoryType("conversation"))
func WithMemoryType(memoryType string) AddOption {
	return func(opts *AddOptions) {
		opts.MemoryType = memoryType
	}
}

// WithPrompt sets an optional prompt for Add operations.
//
// Prompt can be used to guide memory processing or extraction.
//
// Example:
//
//	result, _ := client.Add(ctx, messages, usermemory.WithPrompt("Extract key facts"))
func WithPrompt(prompt string) AddOption {
	return func(opts *AddOptions) {
		opts.Prompt = prompt
	}
}

// WithInfer enables or disables intelligent deduplication for Add operations.
//
// When enabled, the system automatically detects and merges duplicate memories
// based on vector similarity.
//
// Example:
//
//	result, _ := client.Add(ctx, messages, usermemory.WithInfer(true))
func WithInfer(infer bool) AddOption {
	return func(opts *AddOptions) {
		opts.Infer = infer
	}
}

// WithIncludeRoles sets which message roles to include for profile extraction.
//
// Example:
//
//	result, _ := client.Add(ctx, messages,
//	    usermemory.WithIncludeRoles([]string{"user", "system"}),
//	)
func WithIncludeRoles(roles []string) AddOption {
	return func(opts *AddOptions) {
		opts.IncludeRoles = roles
	}
}

// WithExcludeRoles sets which message roles to exclude from profile extraction.
//
// Example:
//
//	result, _ := client.Add(ctx, messages,
//	    usermemory.WithExcludeRoles([]string{"assistant"}),
//	)
func WithExcludeRoles(roles []string) AddOption {
	return func(opts *AddOptions) {
		opts.ExcludeRoles = roles
	}
}

// applyAddOptions applies Add options to create AddOptions.
func applyAddOptions(opts []AddOption) *AddOptions {
	options := &AddOptions{
		ProfileType:  "content", // Default to content type
		Infer:        true,      // Default to enabled
		Metadata:     make(map[string]interface{}),
		Filters:      make(map[string]interface{}),
		IncludeRoles: []string{"user"},      // Default include user role
		ExcludeRoles: []string{"assistant"}, // Default exclude assistant role
	}
	for _, opt := range opts {
		opt(options)
	}
	return options
}

// SearchOptions contains configuration options for Search operations.
type SearchOptions struct {
	// UserID filters results to a specific user.
	UserID string

	// AgentID filters results to a specific agent.
	AgentID string

	// Limit sets the maximum number of results to return.
	Limit int

	// AddProfile indicates whether to include user profile in search results.
	AddProfile bool
}

// SearchOption is a function type for configuring Search operations.
type SearchOption func(*SearchOptions)

// WithSearchUserID sets the user ID for Search operations.
//
// Example:
//
//	results, _ := client.Search(ctx, "query", usermemory.WithSearchUserID("user_001"))
func WithSearchUserID(userID string) SearchOption {
	return func(opts *SearchOptions) {
		opts.UserID = userID
	}
}

// WithSearchAgentID sets the agent ID for Search operations.
//
// Example:
//
//	results, _ := client.Search(ctx, "query", usermemory.WithSearchAgentID("agent_001"))
func WithSearchAgentID(agentID string) SearchOption {
	return func(opts *SearchOptions) {
		opts.AgentID = agentID
	}
}

// WithSearchLimit sets the maximum number of results for Search operations.
//
// Example:
//
//	results, _ := client.Search(ctx, "query", usermemory.WithSearchLimit(20))
func WithSearchLimit(limit int) SearchOption {
	return func(opts *SearchOptions) {
		opts.Limit = limit
	}
}

// WithAddProfile sets whether to include user profile in search results.
//
// If true and UserID is provided, the search result will include profile content and topics.
//
// Example:
//
//	results, _ := client.Search(ctx, "query",
//	    usermemory.WithSearchUserID("user_001"),
//	    usermemory.WithAddProfile(true),
//	)
func WithAddProfile(addProfile bool) SearchOption {
	return func(opts *SearchOptions) {
		opts.AddProfile = addProfile
	}
}

// applySearchOptions applies Search options to create SearchOptions.
func applySearchOptions(opts []SearchOption) *SearchOptions {
	options := &SearchOptions{
		Limit: 10,
	}
	for _, opt := range opts {
		opt(options)
	}
	return options
}

// GetOptions contains configuration options for Get operations.
type GetOptions struct {
	// UserID filters results to a specific user.
	UserID string

	// AgentID filters results to a specific agent.
	AgentID string
}

// GetOption is a function type for configuring Get operations.
type GetOption func(*GetOptions)

// WithGetUserID sets the user ID for Get operations.
//
// Example:
//
//	memory, _ := client.Get(ctx, memoryID, usermemory.WithGetUserID("user_001"))
func WithGetUserID(userID string) GetOption {
	return func(opts *GetOptions) {
		opts.UserID = userID
	}
}

// WithGetAgentID sets the agent ID for Get operations.
//
// Example:
//
//	memory, _ := client.Get(ctx, memoryID, usermemory.WithGetAgentID("agent_001"))
func WithGetAgentID(agentID string) GetOption {
	return func(opts *GetOptions) {
		opts.AgentID = agentID
	}
}

// UpdateOptions contains configuration options for Update operations.
type UpdateOptions struct {
	// UserID filters results to a specific user.
	UserID string

	// AgentID filters results to a specific agent.
	AgentID string

	// Metadata contains additional metadata to update.
	Metadata map[string]interface{}
}

// UpdateOption is a function type for configuring Update operations.
type UpdateOption func(*UpdateOptions)

// WithUpdateUserID sets the user ID for Update operations.
//
// Example:
//
//	memory, _ := client.Update(ctx, memoryID, "new content", usermemory.WithUpdateUserID("user_001"))
func WithUpdateUserID(userID string) UpdateOption {
	return func(opts *UpdateOptions) {
		opts.UserID = userID
	}
}

// WithUpdateAgentID sets the agent ID for Update operations.
//
// Example:
//
//	memory, _ := client.Update(ctx, memoryID, "new content", usermemory.WithUpdateAgentID("agent_001"))
func WithUpdateAgentID(agentID string) UpdateOption {
	return func(opts *UpdateOptions) {
		opts.AgentID = agentID
	}
}

// WithUpdateMetadata sets metadata for Update operations.
//
// Example:
//
//	memory, _ := client.Update(ctx, memoryID, "new content",
//	    usermemory.WithUpdateMetadata(map[string]interface{}{
//	        "source": "updated",
//	    }),
//	)
func WithUpdateMetadata(metadata map[string]interface{}) UpdateOption {
	return func(opts *UpdateOptions) {
		opts.Metadata = metadata
	}
}

// DeleteOptions contains configuration options for Delete operations.
type DeleteOptions struct {
	// UserID filters results to a specific user.
	UserID string

	// AgentID filters results to a specific agent.
	AgentID string

	// DeleteProfile indicates whether to also delete the associated user profile.
	DeleteProfile bool
}

// DeleteOption is a function type for configuring Delete operations.
type DeleteOption func(*DeleteOptions)

// WithDeleteUserID sets the user ID for Delete operations.
//
// Example:
//
//	_ = client.Delete(ctx, memoryID, usermemory.WithDeleteUserID("user_001"))
func WithDeleteUserID(userID string) DeleteOption {
	return func(opts *DeleteOptions) {
		opts.UserID = userID
	}
}

// WithDeleteAgentID sets the agent ID for Delete operations.
//
// Example:
//
//	_ = client.Delete(ctx, memoryID, usermemory.WithDeleteAgentID("agent_001"))
func WithDeleteAgentID(agentID string) DeleteOption {
	return func(opts *DeleteOptions) {
		opts.AgentID = agentID
	}
}

// WithDeleteProfile sets whether to also delete the associated user profile.
//
// If true and UserID is provided, the user profile will be deleted along with the memory.
//
// Example:
//
//	_ = client.Delete(ctx, memoryID,
//	    usermemory.WithDeleteUserID("user_001"),
//	    usermemory.WithDeleteProfile(true),
//	)
func WithDeleteProfile(deleteProfile bool) DeleteOption {
	return func(opts *DeleteOptions) {
		opts.DeleteProfile = deleteProfile
	}
}

// applyDeleteOptions applies Delete options to create DeleteOptions.
func applyDeleteOptions(opts []DeleteOption) *DeleteOptions {
	options := &DeleteOptions{}
	for _, opt := range opts {
		opt(options)
	}
	return options
}

// GetAllOptions contains configuration options for GetAll operations.
type GetAllOptions struct {
	// UserID filters results to a specific user.
	UserID string

	// AgentID filters results to a specific agent.
	AgentID string

	// RunID filters results to a specific run.
	RunID string

	// Limit sets the maximum number of results to return.
	Limit int

	// Offset sets the number of results to skip (for pagination).
	Offset int

	// Filters provides additional metadata filters.
	Filters map[string]interface{}
}

// GetAllOption is a function type for configuring GetAll operations.
type GetAllOption func(*GetAllOptions)

// WithGetAllUserID sets the user ID for GetAll operations.
//
// Example:
//
//	memories, _ := client.GetAll(ctx, usermemory.WithGetAllUserID("user_001"))
func WithGetAllUserID(userID string) GetAllOption {
	return func(opts *GetAllOptions) {
		opts.UserID = userID
	}
}

// WithGetAllAgentID sets the agent ID for GetAll operations.
//
// Example:
//
//	memories, _ := client.GetAll(ctx, usermemory.WithGetAllAgentID("agent_001"))
func WithGetAllAgentID(agentID string) GetAllOption {
	return func(opts *GetAllOptions) {
		opts.AgentID = agentID
	}
}

// WithGetAllRunID sets the run ID for GetAll operations.
//
// Example:
//
//	memories, _ := client.GetAll(ctx, usermemory.WithGetAllRunID("run_001"))
func WithGetAllRunID(runID string) GetAllOption {
	return func(opts *GetAllOptions) {
		opts.RunID = runID
	}
}

// WithGetAllLimit sets the maximum number of results for GetAll operations.
//
// Example:
//
//	memories, _ := client.GetAll(ctx, usermemory.WithGetAllLimit(100))
func WithGetAllLimit(limit int) GetAllOption {
	return func(opts *GetAllOptions) {
		opts.Limit = limit
	}
}

// WithGetAllOffset sets the offset for GetAll operations (for pagination).
//
// Example:
//
//	memories, _ := client.GetAll(ctx,
//	    usermemory.WithGetAllLimit(50),
//	    usermemory.WithGetAllOffset(50),
//	)
func WithGetAllOffset(offset int) GetAllOption {
	return func(opts *GetAllOptions) {
		opts.Offset = offset
	}
}

// WithGetAllFilters sets metadata filters for GetAll operations.
//
// Example:
//
//	memories, _ := client.GetAll(ctx,
//	    usermemory.WithGetAllFilters(map[string]interface{}{
//	        "type": "conversation",
//	    }),
//	)
func WithGetAllFilters(filters map[string]interface{}) GetAllOption {
	return func(opts *GetAllOptions) {
		opts.Filters = filters
	}
}

// applyGetAllOptions applies GetAll options to create GetAllOptions.
func applyGetAllOptions(opts []GetAllOption) *GetAllOptions {
	options := &GetAllOptions{
		Limit:   100,
		Offset:  0,
		Filters: make(map[string]interface{}),
	}
	for _, opt := range opts {
		opt(options)
	}
	return options
}

// DeleteAllOptions contains configuration options for DeleteAll operations.
type DeleteAllOptions struct {
	// UserID filters deletions to a specific user.
	UserID string

	// AgentID filters deletions to a specific agent.
	AgentID string

	// RunID filters deletions to a specific run.
	RunID string

	// DeleteProfile indicates whether to also delete associated user profiles.
	DeleteProfile bool
}

// DeleteAllOption is a function type for configuring DeleteAll operations.
type DeleteAllOption func(*DeleteAllOptions)

// WithDeleteAllUserID sets the user ID for DeleteAll operations.
//
// Example:
//
//	_ = client.DeleteAll(ctx, usermemory.WithDeleteAllUserID("user_001"))
func WithDeleteAllUserID(userID string) DeleteAllOption {
	return func(opts *DeleteAllOptions) {
		opts.UserID = userID
	}
}

// WithDeleteAllAgentID sets the agent ID for DeleteAll operations.
//
// Example:
//
//	_ = client.DeleteAll(ctx, usermemory.WithDeleteAllAgentID("agent_001"))
func WithDeleteAllAgentID(agentID string) DeleteAllOption {
	return func(opts *DeleteAllOptions) {
		opts.AgentID = agentID
	}
}

// WithDeleteAllRunID sets the run ID for DeleteAll operations.
//
// Example:
//
//	_ = client.DeleteAll(ctx, usermemory.WithDeleteAllRunID("run_001"))
func WithDeleteAllRunID(runID string) DeleteAllOption {
	return func(opts *DeleteAllOptions) {
		opts.RunID = runID
	}
}

// WithDeleteAllProfile sets whether to also delete associated user profiles.
//
// If true and UserID is provided, user profiles will be deleted along with memories.
//
// Example:
//
//	_ = client.DeleteAll(ctx,
//	    usermemory.WithDeleteAllUserID("user_001"),
//	    usermemory.WithDeleteAllProfile(true),
//	)
func WithDeleteAllProfile(deleteProfile bool) DeleteAllOption {
	return func(opts *DeleteAllOptions) {
		opts.DeleteProfile = deleteProfile
	}
}

// applyDeleteAllOptions applies DeleteAll options to create DeleteAllOptions.
func applyDeleteAllOptions(opts []DeleteAllOption) *DeleteAllOptions {
	options := &DeleteAllOptions{}
	for _, opt := range opts {
		opt(options)
	}
	return options
}
