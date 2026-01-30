// Package core provides the main PowerMem client and memory management functionality.
package core

// AddOption is a function type for configuring Add operations.
//
// Options are applied using the functional options pattern, allowing
// flexible configuration without requiring all parameters.
type AddOption func(*AddOptions)

// AddOptions contains configuration options for Add operations.
type AddOptions struct {
	// UserID identifies the user who owns this memory.
	UserID string

	// AgentID identifies the agent associated with this memory.
	AgentID string

	// RunID identifies the run/session associated with this memory.
	RunID string

	// Metadata contains additional metadata about the memory.
	Metadata map[string]interface{}

	// Filters provides additional metadata filters for the memory.
	Filters map[string]interface{}

	// Scope defines the visibility scope of the memory.
	// See MemoryScope constants for available scopes.
	Scope MemoryScope

	// MemoryType specifies the type of memory (e.g., "conversation", "fact", "preference").
	MemoryType string

	// Prompt is an optional prompt used for memory processing.
	Prompt string

	// Infer enables intelligent deduplication.
	// When true, the system checks for duplicate memories and merges them.
	Infer bool
}

// WithUserID sets the user ID for Add operations.
//
// Example:
//
//	memory, _ := client.Add(ctx, "content", core.WithUserID("user_001"))
func WithUserID(userID string) AddOption {
	return func(opts *AddOptions) {
		opts.UserID = userID
	}
}

// WithUserIDForSearch sets the user ID for Search operations.
//
// Example:
//
//	results, _ := client.Search(ctx, "query", core.WithUserIDForSearch("user_001"))
func WithUserIDForSearch(userID string) SearchOption {
	return func(opts *SearchOptions) {
		opts.UserID = userID
	}
}

// WithUserIDForGetAll sets the user ID for GetAll operations.
//
// Example:
//
//	memories, _ := client.GetAll(ctx, core.WithUserIDForGetAll("user_001"))
func WithUserIDForGetAll(userID string) GetAllOption {
	return func(opts *GetAllOptions) {
		opts.UserID = userID
	}
}

// WithUserIDForDeleteAll sets the user ID for DeleteAll operations.
//
// Example:
//
//	_ = client.DeleteAll(ctx, core.WithUserIDForDeleteAll("user_001"))
func WithUserIDForDeleteAll(userID string) DeleteAllOption {
	return func(opts *DeleteAllOptions) {
		opts.UserID = userID
	}
}

// WithAgentID sets the agent ID for Add operations.
//
// Example:
//
//	memory, _ := client.Add(ctx, "content", core.WithAgentID("agent_001"))
func WithAgentID(agentID string) AddOption {
	return func(opts *AddOptions) {
		opts.AgentID = agentID
	}
}

// WithAgentIDForSearch sets the agent ID for Search operations.
//
// Example:
//
//	results, _ := client.Search(ctx, "query", core.WithAgentIDForSearch("agent_001"))
func WithAgentIDForSearch(agentID string) SearchOption {
	return func(opts *SearchOptions) {
		opts.AgentID = agentID
	}
}

// WithAgentIDForGetAll sets the agent ID for GetAll operations.
//
// Example:
//
//	memories, _ := client.GetAll(ctx, core.WithAgentIDForGetAll("agent_001"))
func WithAgentIDForGetAll(agentID string) GetAllOption {
	return func(opts *GetAllOptions) {
		opts.AgentID = agentID
	}
}

// WithAgentIDForDeleteAll sets the agent ID for DeleteAll operations.
//
// Example:
//
//	_ = client.DeleteAll(ctx, core.WithAgentIDForDeleteAll("agent_001"))
func WithAgentIDForDeleteAll(agentID string) DeleteAllOption {
	return func(opts *DeleteAllOptions) {
		opts.AgentID = agentID
	}
}

// WithMetadata sets metadata for Add operations.
//
// Metadata can be used for filtering and additional context.
//
// Example:
//
//	memory, _ := client.Add(ctx, "content",
//	    core.WithMetadata(map[string]interface{}{
//	        "source": "conversation",
//	        "priority": "high",
//	    }),
//	)
func WithMetadata(metadata map[string]interface{}) AddOption {
	return func(opts *AddOptions) {
		opts.Metadata = metadata
	}
}

// WithRunID sets the run ID for Add operations.
//
// RunID identifies a specific run or session, useful for grouping related memories.
//
// Example:
//
//	memory, _ := client.Add(ctx, "content", core.WithRunID("run_001"))
func WithRunID(runID string) AddOption {
	return func(opts *AddOptions) {
		opts.RunID = runID
	}
}

// WithFiltersForAdd sets metadata filters for Add operations.
//
// Filters can be used for additional filtering and categorization.
//
// Example:
//
//	memory, _ := client.Add(ctx, "content",
//	    core.WithFiltersForAdd(map[string]interface{}{
//	        "category": "conversation",
//	        "priority": "high",
//	    }),
//	)
func WithFiltersForAdd(filters map[string]interface{}) AddOption {
	return func(opts *AddOptions) {
		opts.Filters = filters
	}
}

// WithMemoryType sets the memory type for Add operations.
//
// MemoryType categorizes the type of memory (e.g., "conversation", "fact", "preference").
//
// Example:
//
//	memory, _ := client.Add(ctx, "content", core.WithMemoryType("conversation"))
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
//	memory, _ := client.Add(ctx, "content", core.WithPrompt("Extract key facts"))
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
//	memory, _ := client.Add(ctx, "content", core.WithInfer(true))
func WithInfer(infer bool) AddOption {
	return func(opts *AddOptions) {
		opts.Infer = infer
	}
}

// WithScope sets the memory scope for Add operations.
//
// Scope determines visibility:
//   - ScopePrivate: Only visible to the creating agent
//   - ScopeAgentGroup: Visible to all agents in the group
//   - ScopeGlobal: Visible to all agents
//
// Example:
//
//	memory, _ := client.Add(ctx, "content", core.WithScope(core.ScopeGlobal))
func WithScope(scope MemoryScope) AddOption {
	return func(opts *AddOptions) {
		opts.Scope = scope
	}
}

// SearchOption is a function type for configuring Search operations.
type SearchOption func(*SearchOptions)

// SearchOptions contains configuration options for Search operations.
type SearchOptions struct {
	// UserID filters results to a specific user.
	UserID string

	// AgentID filters results to a specific agent.
	AgentID string

	// Limit sets the maximum number of results to return.
	// Default: 10
	Limit int

	// Filters provides additional metadata filters.
	Filters map[string]interface{}

	// MinScore sets the minimum similarity score for results.
	// Results with scores below this threshold are excluded.
	// Default: 0.0 (no minimum)
	MinScore float64

	// IncludeArchived indicates whether to include archived memories.
	IncludeArchived bool
}

// WithLimit sets the maximum number of results for Search operations.
//
// Example:
//
//	results, _ := client.Search(ctx, "query", core.WithLimit(20))
func WithLimit(limit int) SearchOption {
	return func(opts *SearchOptions) {
		opts.Limit = limit
	}
}

// WithLimitForGetAll sets the maximum number of results for GetAll operations.
//
// Example:
//
//	memories, _ := client.GetAll(ctx, core.WithLimitForGetAll(100))
func WithLimitForGetAll(limit int) GetAllOption {
	return func(opts *GetAllOptions) {
		opts.Limit = limit
	}
}

// WithFilters sets metadata filters for Search operations.
//
// Filters allow searching by custom metadata fields.
//
// Example:
//
//	results, _ := client.Search(ctx, "query",
//	    core.WithFilters(map[string]interface{}{
//	        "type": "conversation",
//	        "priority": "high",
//	    }),
//	)
func WithFilters(filters map[string]interface{}) SearchOption {
	return func(opts *SearchOptions) {
		opts.Filters = filters
	}
}

// WithMinScore sets the minimum similarity score for Search results.
//
// Only results with similarity scores >= minScore are returned.
// Typical range: 0.0-1.0, where 1.0 is identical.
//
// Example:
//
//	results, _ := client.Search(ctx, "query", core.WithMinScore(0.7))
func WithMinScore(score float64) SearchOption {
	return func(opts *SearchOptions) {
		opts.MinScore = score
	}
}

// WithIncludeArchived sets whether to include archived memories in Search results.
//
// Example:
//
//	results, _ := client.Search(ctx, "query", core.WithIncludeArchived(true))
func WithIncludeArchived(include bool) SearchOption {
	return func(opts *SearchOptions) {
		opts.IncludeArchived = include
	}
}

// GetAllOption is a function type for configuring GetAll operations.
type GetAllOption func(*GetAllOptions)

// GetAllOptions contains configuration options for GetAll operations.
type GetAllOptions struct {
	// UserID filters results to a specific user.
	UserID string

	// AgentID filters results to a specific agent.
	AgentID string

	// Limit sets the maximum number of results to return.
	// Default: 100
	Limit int

	// Offset sets the number of results to skip (for pagination).
	// Default: 0
	Offset int
}

// WithOffset sets the offset for GetAll operations (for pagination).
//
// Example:
//
//	// Get second page of results
//	memories, _ := client.GetAll(ctx,
//	    core.WithLimitForGetAll(50),
//	    core.WithOffset(50),
//	)
func WithOffset(offset int) GetAllOption {
	return func(opts *GetAllOptions) {
		opts.Offset = offset
	}
}

// DeleteAllOption is a function type for configuring DeleteAll operations.
type DeleteAllOption func(*DeleteAllOptions)

// DeleteAllOptions contains configuration options for DeleteAll operations.
type DeleteAllOptions struct {
	// UserID filters deletions to a specific user.
	UserID string

	// AgentID filters deletions to a specific agent.
	AgentID string
}

// applyAddOptions applies Add options to create AddOptions.
func applyAddOptions(opts []AddOption) *AddOptions {
	options := &AddOptions{
		Infer:    false,
		Scope:    ScopePrivate,
		Metadata: make(map[string]interface{}),
		Filters:  make(map[string]interface{}),
	}
	for _, opt := range opts {
		opt(options)
	}
	return options
}

// applySearchOptions applies Search options to create SearchOptions.
func applySearchOptions(opts []SearchOption) *SearchOptions {
	options := &SearchOptions{
		Limit:    10,
		MinScore: 0.0,
	}
	for _, opt := range opts {
		opt(options)
	}
	return options
}

// applyGetAllOptions applies GetAll options to create GetAllOptions.
func applyGetAllOptions(opts []GetAllOption) *GetAllOptions {
	options := &GetAllOptions{
		Limit:  100,
		Offset: 0,
	}
	for _, opt := range opts {
		opt(options)
	}
	return options
}

// applyDeleteAllOptions applies DeleteAll options to create DeleteAllOptions.
func applyDeleteAllOptions(opts []DeleteAllOption) *DeleteAllOptions {
	options := &DeleteAllOptions{}
	for _, opt := range opts {
		opt(options)
	}
	return options
}

// GetOption is a function type for configuring Get operations.
type GetOption func(*GetOptions)

// GetOptions contains configuration options for Get operations with access control.
type GetOptions struct {
	// UserID restricts access to memories belonging to this user (multi-tenant isolation).
	UserID string

	// AgentID restricts access to memories belonging to this agent (agent-level access control).
	AgentID string
}

// WithUserIDForGet sets the user ID for Get operations (access control).
func WithUserIDForGet(userID string) GetOption {
	return func(opts *GetOptions) {
		opts.UserID = userID
	}
}

// WithAgentIDForGet sets the agent ID for Get operations (access control).
func WithAgentIDForGet(agentID string) GetOption {
	return func(opts *GetOptions) {
		opts.AgentID = agentID
	}
}

// UpdateOption is a function type for configuring Update operations.
type UpdateOption func(*UpdateOptions)

// UpdateOptions contains configuration options for Update operations with access control.
type UpdateOptions struct {
	// UserID restricts updates to memories belonging to this user (prevents cross-tenant updates).
	UserID string

	// AgentID restricts updates to memories belonging to this agent (agent-level access control).
	AgentID string
}

// WithUserIDForUpdate sets the user ID for Update operations (access control).
func WithUserIDForUpdate(userID string) UpdateOption {
	return func(opts *UpdateOptions) {
		opts.UserID = userID
	}
}

// WithAgentIDForUpdate sets the agent ID for Update operations (access control).
func WithAgentIDForUpdate(agentID string) UpdateOption {
	return func(opts *UpdateOptions) {
		opts.AgentID = agentID
	}
}

// DeleteOption is a function type for configuring Delete operations.
type DeleteOption func(*DeleteOptions)

// DeleteOptions contains configuration options for Delete operations with access control.
type DeleteOptions struct {
	// UserID restricts deletions to memories belonging to this user (prevents cross-tenant deletions).
	UserID string

	// AgentID restricts deletions to memories belonging to this agent (agent-level access control).
	AgentID string
}

// WithUserIDForDelete sets the user ID for Delete operations (access control).
func WithUserIDForDelete(userID string) DeleteOption {
	return func(opts *DeleteOptions) {
		opts.UserID = userID
	}
}

// WithAgentIDForDelete sets the agent ID for Delete operations (access control).
func WithAgentIDForDelete(agentID string) DeleteOption {
	return func(opts *DeleteOptions) {
		opts.AgentID = agentID
	}
}

// applyGetOptions applies Get options to create GetOptions.
func applyGetOptions(opts []GetOption) *GetOptions {
	options := &GetOptions{}
	for _, opt := range opts {
		opt(options)
	}
	return options
}

// applyUpdateOptions applies Update options to create UpdateOptions.
func applyUpdateOptions(opts []UpdateOption) *UpdateOptions {
	options := &UpdateOptions{}
	for _, opt := range opts {
		opt(options)
	}
	return options
}

// applyDeleteOptions applies Delete options to create DeleteOptions.
func applyDeleteOptions(opts []DeleteOption) *DeleteOptions {
	options := &DeleteOptions{}
	for _, opt := range opts {
		opt(options)
	}
	return options
}
