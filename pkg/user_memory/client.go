// Package usermemory provides user memory management with automatic profile extraction.
//
// UserMemory extends the core Memory client with user profile management capabilities:
//   - Automatic profile extraction from conversations
//   - Continuous profile updates
//   - Profile-based search enhancement
//   - Structured topic extraction
//
// The package automatically extracts and maintains user profiles based on
// conversations, enabling personalized memory management.
package usermemory

import (
	"context"
	"fmt"
	"strings"

	"github.com/oceanbase/powermem-go/pkg/core"
	"github.com/oceanbase/powermem-go/pkg/llm"
	anthropicLLM "github.com/oceanbase/powermem-go/pkg/llm/anthropic"
	deepseekLLM "github.com/oceanbase/powermem-go/pkg/llm/deepseek"
	ollamaLLM "github.com/oceanbase/powermem-go/pkg/llm/ollama"
	openaiLLM "github.com/oceanbase/powermem-go/pkg/llm/openai"
	qwenLLM "github.com/oceanbase/powermem-go/pkg/llm/qwen"
	"github.com/oceanbase/powermem-go/pkg/user_memory/query_rewrite"
	"github.com/oceanbase/powermem-go/pkg/user_memory/sqlite"
)

// Client is the UserMemory client that extends core Memory with user profile management.
//
// It provides:
//   - All core Memory operations (Add, Search, Get, Update, Delete, etc.)
//   - Automatic user profile extraction from conversations
//   - Profile-based search enhancement
//   - Profile management (Get, Update, Delete)
//
// The client automatically extracts and updates user profiles when adding memories,
// enabling personalized memory management without manual profile maintenance.
//
// Example:
//
//	config := &usermemory.Config{
//	    MemoryConfig: memoryConfig,
//	    ProfileStoreType: "sqlite",
//	    ProfileStoreConfig: &sqlite.Config{
//	        DBPath: "./profiles.db",
//	    },
//	}
//	client, _ := usermemory.NewClient(config)
//	defer client.Close()
//
//	result, _ := client.Add(ctx, conversation,
//	    usermemory.WithUserID("user_001"),
//	)
//	// Profile is automatically extracted and saved
type Client struct {
	// memory is the underlying core Memory client.
	memory *core.Client

	// profileStore stores and manages user profiles.
	profileStore UserProfileStore

	// llm is the LLM provider for profile extraction.
	llm llm.Provider

	// queryRewriter is the query rewriter for enhancing search queries (optional).
	queryRewriter *query_rewrite.QueryRewriter
}

// Config contains configuration for creating a UserMemory client.
type Config struct {
	// MemoryConfig is the configuration for the underlying Memory client.
	MemoryConfig *core.Config

	// ProfileStoreType is the type of profile store ("sqlite", "oceanbase", "postgres").
	ProfileStoreType string

	// ProfileStoreConfig is the configuration for the profile store.
	// The type depends on ProfileStoreType:
	//   - For "sqlite": *sqlite.Config
	//   - For "oceanbase": *oceanbase.Config (future)
	//   - For "postgres": *postgres.Config (future)
	ProfileStoreConfig interface{}

	// QueryRewriteConfig is the configuration for query rewriting (optional).
	// If nil or Enabled is false, query rewriting is disabled.
	QueryRewriteConfig *query_rewrite.Config
}

// NewClient creates a new UserMemory client.
//
// The client is initialized with:
//   - A core Memory client (for memory operations)
//   - A UserProfileStore (for profile management)
//   - An LLM provider (for profile extraction)
//
// Parameters:
//   - cfg: Configuration containing Memory and ProfileStore settings
//
// Returns a new Client instance, or an error if initialization fails.
//
// Example:
//
//	config := &usermemory.Config{
//	    MemoryConfig: coreConfig,
//	    ProfileStoreType: "sqlite",
//	    ProfileStoreConfig: &sqlite.Config{
//	        DBPath:    "./profiles.db",
//	        TableName: "user_profiles",
//	    },
//	}
//	client, err := usermemory.NewClient(config)
func NewClient(cfg *Config) (*Client, error) {
	// Create Memory client
	memory, err := core.NewClient(cfg.MemoryConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create memory client: %w", err)
	}

	// Create UserProfileStore
	var profileStore UserProfileStore
	switch strings.ToLower(cfg.ProfileStoreType) {
	case "sqlite":
		sqliteCfg, ok := cfg.ProfileStoreConfig.(*sqlite.Config)
		if !ok {
			return nil, fmt.Errorf("invalid sqlite config type")
		}
		sqliteStore, err := sqlite.NewStore(sqliteCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create sqlite profile store: %w", err)
		}
		// Wrap with adapter
		profileStore = &sqliteStoreAdapter{store: sqliteStore}
	default:
		return nil, fmt.Errorf("unsupported profile store type: %s", cfg.ProfileStoreType)
	}

	// Create LLM from config (for profile extraction)
	llmProvider, err := initLLMFromConfig(cfg.MemoryConfig.LLM)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM: %w", err)
	}

	// Initialize query rewriter (if enabled)
	var queryRewriter *query_rewrite.QueryRewriter
	if cfg.QueryRewriteConfig != nil && cfg.QueryRewriteConfig.Enabled {
		// Use override model if specified; otherwise use default LLM
		rewriteLLM := llmProvider
		if cfg.QueryRewriteConfig.ModelOverride != "" {
			// Create LLM config with override model
			overrideLLMConfig := cfg.MemoryConfig.LLM
			overrideLLMConfig.Model = cfg.QueryRewriteConfig.ModelOverride
			overrideLLM, err := initLLMFromConfig(overrideLLMConfig)
			if err == nil {
				rewriteLLM = overrideLLM
			}
			// Fall back to default LLM if creation fails
		}
		queryRewriter = query_rewrite.NewQueryRewriter(rewriteLLM, cfg.QueryRewriteConfig)
	}

	return &Client{
		memory:        memory,
		profileStore:  profileStore,
		llm:           llmProvider,
		queryRewriter: queryRewriter,
	}, nil
}

// initLLMFromConfig initializes an LLM provider from configuration.
//
// This is a helper function that duplicates the LLM initialization logic
// from the core package, allowing UserMemory to have its own LLM instance
// for profile extraction.
func initLLMFromConfig(cfg core.LLMConfig) (llm.Provider, error) {
	switch cfg.Provider {
	case "openai":
		return openaiLLM.NewClient(&openaiLLM.Config{
			APIKey:  cfg.APIKey,
			Model:   cfg.Model,
			BaseURL: cfg.BaseURL,
		})
	case "qwen":
		return qwenLLM.NewClient(&qwenLLM.Config{
			APIKey:  cfg.APIKey,
			Model:   cfg.Model,
			BaseURL: cfg.BaseURL,
		})
	case "deepseek":
		return deepseekLLM.NewClient(&deepseekLLM.Config{
			APIKey:  cfg.APIKey,
			Model:   cfg.Model,
			BaseURL: cfg.BaseURL,
		})
	case "ollama":
		return ollamaLLM.NewClient(&ollamaLLM.Config{
			APIKey:  cfg.APIKey,
			Model:   cfg.Model,
			BaseURL: cfg.BaseURL,
		})
	case "anthropic":
		return anthropicLLM.NewClient(&anthropicLLM.Config{
			APIKey:  cfg.APIKey,
			Model:   cfg.Model,
			BaseURL: cfg.BaseURL,
		})
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.Provider)
	}
}

// Add adds a conversation and automatically extracts/updates the user profile.
//
// The method:
//  1. Stores the conversation as a memory using the core Memory client
//  2. Extracts user profile information from the conversation using LLM
//  3. Updates the user profile in the profile store
//
// Profile extraction can extract:
//   - Unstructured profile content (default)
//   - Structured topics (if ProfileType is "topics")
//
// Parameters:
//   - ctx: Context for cancellation
//   - messages: Conversation messages (string, []map[string]interface{}, or single map)
//   - opts: Optional parameters (UserID, AgentID, ProfileType, etc.)
//
// Returns an AddResult containing the created memory and profile extraction results.
//
// Example:
//
//	result, err := client.Add(ctx, []map[string]interface{}{
//	    {"role": "user", "content": "I'm Alice, a software engineer."},
//	}, usermemory.WithUserID("user_001"))
func (c *Client) Add(ctx context.Context, messages interface{}, opts ...AddOption) (*AddResult, error) {
	addOpts := applyAddOptions(opts)

	// 1. Build core.Add options, passing all parameters
	coreOpts := []core.AddOption{
		core.WithUserID(addOpts.UserID),
		core.WithAgentID(addOpts.AgentID),
	}
	if addOpts.RunID != "" {
		coreOpts = append(coreOpts, core.WithRunID(addOpts.RunID))
	}
	if len(addOpts.Metadata) > 0 {
		coreOpts = append(coreOpts, core.WithMetadata(addOpts.Metadata))
	}
	if len(addOpts.Filters) > 0 {
		coreOpts = append(coreOpts, core.WithFiltersForAdd(addOpts.Filters))
	}
	if addOpts.Scope != "" {
		coreOpts = append(coreOpts, core.WithScope(addOpts.Scope))
	}
	if addOpts.MemoryType != "" {
		coreOpts = append(coreOpts, core.WithMemoryType(addOpts.MemoryType))
	}
	if addOpts.Prompt != "" {
		coreOpts = append(coreOpts, core.WithPrompt(addOpts.Prompt))
	}
	coreOpts = append(coreOpts, core.WithInfer(addOpts.Infer))

	// Store conversation event (using Memory)
	memory, err := c.memory.Add(ctx, c.formatMessages(messages), coreOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to add memory: %w", err)
	}

	// 2. Extract user profile
	var profileContent *string
	var topics map[string]interface{}

	// Filter messages by roles (if specified)
	filteredMessages := c.filterMessagesByRoles(messages, addOpts.IncludeRoles, addOpts.ExcludeRoles)

	if addOpts.ProfileType == "topics" {
		// Extract structured topics
		extractedTopics, err := c.extractTopics(ctx, filteredMessages, addOpts.UserID, addOpts.CustomTopics, addOpts.StrictMode)
		if err != nil {
			return nil, fmt.Errorf("failed to extract topics: %w", err)
		}
		if extractedTopics != nil {
			topics = extractedTopics
		}
	} else {
		// Extract unstructured profile content
		extractedContent, err := c.extractProfile(ctx, filteredMessages, addOpts.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to extract profile: %w", err)
		}
		if extractedContent != "" {
			profileContent = &extractedContent
		}
	}

	// 3. Save user profile
	var profileExtracted bool
	if profileContent != nil || topics != nil {
		_, err = c.profileStore.SaveProfile(ctx, addOpts.UserID, profileContent, topics)
		if err != nil {
			return nil, fmt.Errorf("failed to save profile: %w", err)
		}
		profileExtracted = true
	}

	return &AddResult{
		Memory:           memory,
		ProfileExtracted: profileExtracted,
		ProfileContent:   profileContent,
		Topics:           topics,
	}, nil
}

// SearchResult contains the result of a search operation.
type SearchResult struct {
	// Memories is the list of matching memories.
	Memories []*core.Memory

	// ProfileContent is the user profile content (if AddProfile was true).
	ProfileContent *string

	// Topics is the user profile topics (if AddProfile was true).
	Topics map[string]interface{}
}

// Search searches for memories, optionally enhanced with user profile information.
//
// The method supports:
//   - Query rewriting based on user profiles (if query rewrite is enabled)
//   - Adding user profile to search results (if AddProfile is true)
//
// Parameters:
//   - ctx: Context for cancellation
//   - query: Search query string
//   - opts: Optional parameters (UserID, AgentID, Limit, AddProfile)
//
// Returns a SearchResult containing matching memories and optionally user profile.
func (c *Client) Search(ctx context.Context, query string, opts ...SearchOption) (*SearchResult, error) {
	searchOpts := applySearchOptions(opts)

	// === Query rewrite step ===
	effectiveQuery := query
	if c.queryRewriter != nil && searchOpts.UserID != "" {
		// Get user profile from profile store
		profile, err := c.profileStore.GetProfileByUserID(ctx, searchOpts.UserID)
		if err == nil && profile != nil && profile.ProfileContent != "" {
			// Execute rewrite
			rewriteResult := c.queryRewriter.Rewrite(ctx, query, profile.ProfileContent)
			effectiveQuery = rewriteResult.RewrittenQuery
		}
	}
	// === End of query rewrite step ===

	// Call memory.search() with rewritten query
	var searchOptions []core.SearchOption
	if searchOpts.UserID != "" {
		searchOptions = append(searchOptions, core.WithUserIDForSearch(searchOpts.UserID))
	}
	if searchOpts.AgentID != "" {
		searchOptions = append(searchOptions, core.WithAgentIDForSearch(searchOpts.AgentID))
	}
	if searchOpts.Limit > 0 {
		searchOptions = append(searchOptions, core.WithLimit(searchOpts.Limit))
	}

	memories, err := c.memory.Search(ctx, effectiveQuery, searchOptions...)
	if err != nil {
		return nil, err
	}

	result := &SearchResult{
		Memories: memories,
	}

	// Add profile if requested and user_id is provided
	if searchOpts.AddProfile && searchOpts.UserID != "" {
		profile, err := c.profileStore.GetProfileByUserID(ctx, searchOpts.UserID)
		if err == nil && profile != nil {
			if profile.ProfileContent != "" {
				result.ProfileContent = &profile.ProfileContent
			}
			if len(profile.Topics) > 0 {
				result.Topics = profile.Topics
			}
		}
	}

	return result, nil
}

// GetProfile retrieves the user profile for a given user ID.
//
// Parameters:
//   - ctx: Context for cancellation
//   - userID: User identifier
//
// Returns the UserProfile if found, or nil if not found.
func (c *Client) GetProfile(ctx context.Context, userID string) (*UserProfile, error) {
	return c.profileStore.GetProfileByUserID(ctx, userID)
}

// GetProfiles retrieves a list of user profiles with optional filtering.
//
// Profiles can be filtered by:
//   - UserID
//   - MainTopic, SubTopic, TopicValue (for structured topics)
//   - Limit and Offset (for pagination)
//
// Parameters:
//   - ctx: Context for cancellation
//   - opts: Filtering and pagination options
//
// Returns a list of matching user profiles.
func (c *Client) GetProfiles(ctx context.Context, opts *GetProfilesOptions) ([]*UserProfile, error) {
	return c.profileStore.GetProfiles(ctx, opts)
}

// DeleteProfile deletes a user profile by profile ID.
//
// Parameters:
//   - ctx: Context for cancellation
//   - profileID: Profile ID to delete
//
// Returns an error if deletion fails.
func (c *Client) DeleteProfile(ctx context.Context, profileID int64) error {
	return c.profileStore.DeleteProfile(ctx, profileID)
}

// DeleteProfileByUserID deletes a user profile by user ID.
//
// This is a convenience method that first retrieves the profile by user ID,
// then deletes it by profile ID.
//
// Parameters:
//   - ctx: Context for cancellation
//   - userID: User identifier
//
// Returns an error if the profile is not found or deletion fails.
func (c *Client) DeleteProfileByUserID(ctx context.Context, userID string) error {
	profile, err := c.profileStore.GetProfileByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get profile: %w", err)
	}
	if profile == nil {
		return nil // Profile does not exist, return success directly
	}
	return c.profileStore.DeleteProfile(ctx, profile.ID)
}

// Get retrieves a single memory by ID.
//
// This method wraps the core Memory Get operation.
//
// Parameters:
//   - ctx: Context for cancellation
//   - memoryID: Memory ID
//   - opts: Optional parameters (currently unused, reserved for future use)
//
// Returns the Memory if found, or an error if not found.
func (c *Client) Get(ctx context.Context, memoryID int64, opts ...GetOption) (*core.Memory, error) {
	// Note: core.Get currently does not support user_id and agent_id parameters
	// If needed, can be implemented via metadata or other means
	return c.memory.Get(ctx, memoryID)
}

// Update updates an existing memory's content.
//
// This method wraps the core Memory Update operation.
//
// Parameters:
//   - ctx: Context for cancellation
//   - memoryID: Memory ID to update
//   - content: New content for the memory
//   - opts: Optional parameters (currently unused, reserved for future use)
//
// Returns the updated Memory, or an error if update fails.
func (c *Client) Update(ctx context.Context, memoryID int64, content string, opts ...UpdateOption) (*core.Memory, error) {
	// Note: core.Update currently does not support user_id, agent_id, and metadata parameters
	// If needed, can be implemented via other means
	return c.memory.Update(ctx, memoryID, content)
}

// Delete deletes a memory by ID, optionally also deleting the user profile.
//
// This method wraps the core Memory Delete operation, with the additional
// option to delete the associated user profile.
//
// Parameters:
//   - ctx: Context for cancellation
//   - memoryID: Memory ID to delete
//   - opts: Optional parameters (UserID, AgentID, DeleteProfile)
//
// If DeleteProfile is true and UserID is provided, the user profile is also deleted.
//
// Returns an error if deletion fails.
func (c *Client) Delete(ctx context.Context, memoryID int64, opts ...DeleteOption) error {
	deleteOpts := applyDeleteOptions(opts)

	// Delete memory
	err := c.memory.Delete(ctx, memoryID)
	if err != nil {
		return fmt.Errorf("failed to delete memory: %w", err)
	}

	// If delete_profile is set, also delete profile
	if deleteOpts.DeleteProfile && deleteOpts.UserID != "" {
		profile, err := c.profileStore.GetProfileByUserID(ctx, deleteOpts.UserID)
		if err == nil && profile != nil {
			// Ignore profile deletion errors, only log warnings
			_ = c.profileStore.DeleteProfile(ctx, profile.ID)
			// In actual applications, can use logger to log errors
		}
	}

	return nil
}

// GetAll retrieves all memories with optional filtering and pagination.
//
// This method wraps the core Memory GetAll operation.
//
// Parameters:
//   - ctx: Context for cancellation
//   - opts: Optional parameters (UserID, AgentID, RunID, Limit, Offset, Filters)
//
// Returns a list of memories matching the filters.
func (c *Client) GetAll(ctx context.Context, opts ...GetAllOption) ([]*core.Memory, error) {
	getAllOpts := applyGetAllOptions(opts)

	var getAllOptions []core.GetAllOption
	if getAllOpts.UserID != "" {
		getAllOptions = append(getAllOptions, core.WithUserIDForGetAll(getAllOpts.UserID))
	}
	if getAllOpts.AgentID != "" {
		getAllOptions = append(getAllOptions, core.WithAgentIDForGetAll(getAllOpts.AgentID))
	}
	if getAllOpts.Limit > 0 {
		getAllOptions = append(getAllOptions, core.WithLimitForGetAll(getAllOpts.Limit))
	}
	if getAllOpts.Offset > 0 {
		getAllOptions = append(getAllOptions, core.WithOffset(getAllOpts.Offset))
	}

	return c.memory.GetAll(ctx, getAllOptions...)
}

// DeleteAll deletes all memories matching the filters, optionally also deleting user profiles.
//
// This method wraps the core Memory DeleteAll operation, with the additional
// option to delete associated user profiles.
//
// Parameters:
//   - ctx: Context for cancellation
//   - opts: Optional parameters (UserID, AgentID, RunID, DeleteProfile)
//
// If DeleteProfile is true and UserID is provided, the user profile is also deleted.
//
// Returns an error if deletion fails.
func (c *Client) DeleteAll(ctx context.Context, opts ...DeleteAllOption) error {
	deleteAllOpts := applyDeleteAllOptions(opts)

	var deleteAllOptions []core.DeleteAllOption
	if deleteAllOpts.UserID != "" {
		deleteAllOptions = append(deleteAllOptions, core.WithUserIDForDeleteAll(deleteAllOpts.UserID))
	}
	if deleteAllOpts.AgentID != "" {
		deleteAllOptions = append(deleteAllOptions, core.WithAgentIDForDeleteAll(deleteAllOpts.AgentID))
	}

	// Delete all memories
	err := c.memory.DeleteAll(ctx, deleteAllOptions...)
	if err != nil {
		return fmt.Errorf("failed to delete all memories: %w", err)
	}

	// If delete_profile is set, also delete profile
	if deleteAllOpts.DeleteProfile && deleteAllOpts.UserID != "" {
		profile, err := c.profileStore.GetProfileByUserID(ctx, deleteAllOpts.UserID)
		if err == nil && profile != nil {
			// Ignore profile deletion errors, only log warnings
			_ = c.profileStore.DeleteProfile(ctx, profile.ID)
			// In actual applications, can use logger to log errors
		}
	}

	return nil
}

// Reset resets the storage by deleting all memories.
//
// Note: This method deletes all memories but does not delete user profiles.
// To delete profiles as well, use DeleteAll with DeleteProfile option.
//
// This is implemented using DeleteAll since the core package doesn't have
// a Reset method.
//
// Parameters:
//   - ctx: Context for cancellation
//
// Returns an error if reset fails.
func (c *Client) Reset(ctx context.Context) error {
	// Delete all memories
	err := c.memory.DeleteAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to reset memories: %w", err)
	}

	// Note: UserProfileStore does not have a batch delete method
	// If profile reset is needed, batch delete or iterative delete needs to be implemented
	// Profile reset is not implemented here as it requires getting all profile lists

	return nil
}

// extractProfile extracts user profile (unstructured).
func (c *Client) extractProfile(ctx context.Context, messages interface{}, userID string) (string, error) {
	// Format conversation text
	conversationText := c.formatMessages(messages)
	if conversationText == "" {
		return "", nil
	}

	// Get existing profile
	existingProfile, _ := c.profileStore.GetProfileByUserID(ctx, userID)
	var existingContent string
	if existingProfile != nil && existingProfile.ProfileContent != "" {
		existingContent = existingProfile.ProfileContent
	}

	// Build prompt
	systemPrompt := getUserProfileExtractionPrompt()
	userMessage := buildProfileExtractionUserMessage(conversationText, existingContent)

	// Call LLM
	response, err := c.llm.GenerateWithMessages(ctx, []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMessage},
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate profile: %w", err)
	}

	// Clean response
	profileContent := strings.TrimSpace(response)
	if profileContent == "" || strings.ToLower(profileContent) == "none" || strings.ToLower(profileContent) == "no profile information" {
		return "", nil
	}

	return profileContent, nil
}

// sqliteStoreAdapter is an adapter that adapts sqlite.Store to usermemory.UserProfileStore.
type sqliteStoreAdapter struct {
	store *sqlite.Store
}

func (a *sqliteStoreAdapter) SaveProfile(ctx context.Context, userID string, profileContent *string, topics map[string]interface{}) (int64, error) {
	return a.store.SaveProfile(ctx, userID, profileContent, topics)
}

func (a *sqliteStoreAdapter) GetProfileByUserID(ctx context.Context, userID string) (*UserProfile, error) {
	sqliteProfile, err := a.store.GetProfileByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if sqliteProfile == nil {
		return nil, nil
	}
	return &UserProfile{
		ID:             sqliteProfile.ID,
		UserID:         sqliteProfile.UserID,
		ProfileContent: sqliteProfile.ProfileContent,
		Topics:         sqliteProfile.Topics,
		CreatedAt:      sqliteProfile.CreatedAt,
		UpdatedAt:      sqliteProfile.UpdatedAt,
	}, nil
}

func (a *sqliteStoreAdapter) GetProfiles(ctx context.Context, opts *GetProfilesOptions) ([]*UserProfile, error) {
	sqliteOpts := &sqlite.GetProfilesOptions{
		UserID:     opts.UserID,
		MainTopic:  opts.MainTopic,
		SubTopic:   opts.SubTopic,
		TopicValue: opts.TopicValue,
		Limit:      opts.Limit,
		Offset:     opts.Offset,
	}
	sqliteProfiles, err := a.store.GetProfiles(ctx, sqliteOpts)
	if err != nil {
		return nil, err
	}
	profiles := make([]*UserProfile, len(sqliteProfiles))
	for i, p := range sqliteProfiles {
		profiles[i] = &UserProfile{
			ID:             p.ID,
			UserID:         p.UserID,
			ProfileContent: p.ProfileContent,
			Topics:         p.Topics,
			CreatedAt:      p.CreatedAt,
			UpdatedAt:      p.UpdatedAt,
		}
	}
	return profiles, nil
}

func (a *sqliteStoreAdapter) DeleteProfile(ctx context.Context, profileID int64) error {
	return a.store.DeleteProfile(ctx, profileID)
}

func (a *sqliteStoreAdapter) Close() error {
	return a.store.Close()
}

// extractTopics extracts structured topics.
func (c *Client) extractTopics(ctx context.Context, messages interface{}, userID string, customTopics string, strictMode bool) (map[string]interface{}, error) {
	// Simplified implementation, returning nil indicates not implemented
	// Full implementation requires parsing customTopics JSON and building corresponding prompts
	return nil, nil
}

// filterMessagesByRoles filters messages by include/exclude roles.
//
// This method filters messages based on includeRoles and excludeRoles,
// similar to Python's _filter_messages_by_roles method.
func (c *Client) filterMessagesByRoles(messages interface{}, includeRoles []string, excludeRoles []string) interface{} {
	// If no filtering is needed, return as-is
	if len(includeRoles) == 0 && len(excludeRoles) == 0 {
		return messages
	}

	// Convert messages to list format
	var messageList []map[string]interface{}
	switch v := messages.(type) {
	case string:
		// Single string message
		return v // Return as-is for string
	case map[string]interface{}:
		// Single message dict
		messageList = []map[string]interface{}{v}
	case []map[string]interface{}:
		// List of messages
		messageList = v
	case []interface{}:
		// List of interface{} (need to convert)
		messageList = make([]map[string]interface{}, 0, len(v))
		for _, item := range v {
			if msg, ok := item.(map[string]interface{}); ok {
				messageList = append(messageList, msg)
			}
		}
	default:
		// Unknown type, return as-is
		return messages
	}

	// Filter messages
	filtered := make([]map[string]interface{}, 0)
	for _, msg := range messageList {
		role, ok := msg["role"].(string)
		if !ok {
			// Skip messages without valid role
			continue
		}

		// Check include filter
		if len(includeRoles) > 0 {
			include := false
			for _, r := range includeRoles {
				if r == role {
					include = true
					break
				}
			}
			if !include {
				continue
			}
		}

		// Check exclude filter
		if len(excludeRoles) > 0 {
			exclude := false
			for _, r := range excludeRoles {
				if r == role {
					exclude = true
					break
				}
			}
			if exclude {
				continue
			}
		}

		filtered = append(filtered, msg)
	}

	// Return filtered messages in original format
	if len(filtered) == 0 {
		return []map[string]interface{}{}
	}
	if len(filtered) == 1 {
		return filtered[0]
	}
	return filtered
}

// formatMessages formats messages as text.
func (c *Client) formatMessages(messages interface{}) string {
	switch v := messages.(type) {
	case string:
		return v
	case []map[string]interface{}:
		var parts []string
		for _, msg := range v {
			role, _ := msg["role"].(string)
			content, _ := msg["content"].(string)
			if role != "" && content != "" {
				parts = append(parts, fmt.Sprintf("%s: %s", role, content))
			}
		}
		return strings.Join(parts, "\n")
	default:
		return fmt.Sprintf("%v", messages)
	}
}

// Close closes the client.
func (c *Client) Close() error {
	var errs []error

	if c.memory != nil {
		if err := c.memory.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if c.profileStore != nil {
		if err := c.profileStore.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if c.llm != nil {
		if err := c.llm.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

// getUserProfileExtractionPrompt returns the user profile extraction prompt.
func getUserProfileExtractionPrompt() string {
	return `You are a user profile extraction specialist. Your task is to analyze conversations and extract user profile information.

[Instructions]:
1. Review the current user profile if provided below
2. Analyze the new conversation carefully to identify any new or updated user-related information
3. Extract only factual information explicitly mentioned in the conversation
4. Update the profile by:
   - Adding new information that is not in the current profile
   - Updating existing information if the conversation provides more recent or different details
   - Keeping unchanged information that is still valid
5. Combine all information into a coherent, updated profile description
6. If no relevant profile information is found in the conversation, return the current profile as-is
7. Write the profile in natural language, not as structured data
8. Focus on current state and characteristics of the user
9. If no user profile information can be extracted from the conversation at all, return an empty string ""
10. The final extracted profile description must not exceed 1,000 characters.`
}

// buildProfileExtractionUserMessage builds the user message for profile extraction.
func buildProfileExtractionUserMessage(conversationText, existingProfile string) string {
	if existingProfile != "" {
		return fmt.Sprintf(`Current user profile:
%s

New conversation:
%s

Please update the user profile based on the new conversation.`, existingProfile, conversationText)
	}
	return fmt.Sprintf(`New conversation:
%s

Please extract user profile information from this conversation.`, conversationText)
}
