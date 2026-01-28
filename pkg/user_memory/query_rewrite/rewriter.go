// Package query_rewrite provides query rewriting functionality based on user profiles.
//
// Query rewriting enhances search queries by using user profile information
// to clarify ambiguous or underspecified references, making queries more precise
// and improving search recall.
package query_rewrite

import (
	"context"
	"strings"
	"time"

	"github.com/oceanbase/powermem-go/pkg/llm"
)

// QueryRewriteResult contains the result of a query rewrite operation.
type QueryRewriteResult struct {
	// OriginalQuery is the original query before rewriting.
	OriginalQuery string

	// RewrittenQuery is the rewritten query (same as OriginalQuery if not rewritten).
	RewrittenQuery string

	// IsRewritten indicates whether the query was actually rewritten.
	IsRewritten bool

	// ProfileUsed is the profile content used for rewriting (if any).
	ProfileUsed *string

	// Error contains any error that occurred during rewriting (if any).
	Error *string

	// Metadata contains additional information about the rewrite operation.
	Metadata map[string]interface{}
}

// Config contains configuration for query rewriting.
type Config struct {
	// Enabled indicates whether query rewriting is enabled.
	Enabled bool

	// CustomInstructions is optional custom instructions for the rewrite prompt.
	// If empty, uses default instructions.
	CustomInstructions string

	// ModelOverride is an optional LLM model override for rewriting.
	// If empty, uses the default LLM from the client.
	ModelOverride string
}

// QueryRewriter rewrites queries based on user profiles.
//
// It uses an LLM to enhance queries by incorporating user profile information,
// making ambiguous queries more precise and improving search recall.
type QueryRewriter struct {
	// llm is the LLM provider for generating rewrites.
	llm llm.Provider

	// config contains the query rewrite configuration.
	config *Config
}

// NewQueryRewriter creates a new QueryRewriter instance.
//
// Parameters:
//   - llm: LLM provider for generating rewrites
//   - config: Query rewrite configuration
//
// Returns a new QueryRewriter instance.
func NewQueryRewriter(llm llm.Provider, config *Config) *QueryRewriter {
	return &QueryRewriter{
		llm:    llm,
		config: config,
	}
}

// Rewrite rewrites a query based on user profile content.
//
// The method:
//   - Skips rewriting if profile content is empty or query is too short
//   - Builds a prompt with user profile and query
//   - Calls LLM to generate rewritten query
//   - Falls back to original query on error
//
// Parameters:
//   - ctx: Context for cancellation
//   - query: Original query string
//   - profileContent: User profile text (optional)
//
// Returns the rewrite result containing original and rewritten queries.
func (r *QueryRewriter) Rewrite(ctx context.Context, query string, profileContent string) *QueryRewriteResult {
	// Skip if no user profile
	if profileContent == "" || strings.TrimSpace(profileContent) == "" {
		return &QueryRewriteResult{
			OriginalQuery:  query,
			RewrittenQuery: query,
			IsRewritten:    false,
			Metadata:       make(map[string]interface{}),
		}
	}

	// Skip if query is empty or too short
	trimmedQuery := strings.TrimSpace(query)
	if trimmedQuery == "" || len(trimmedQuery) < 3 {
		return &QueryRewriteResult{
			OriginalQuery:  query,
			RewrittenQuery: query,
			IsRewritten:    false,
			Metadata:       make(map[string]interface{}),
		}
	}

	startTime := time.Now()

	// Build prompt
	prompt := buildQueryRewritePrompt(profileContent, query, r.config.CustomInstructions)

	// Call LLM for rewrite
	messages := []llm.Message{
		{Role: "system", Content: "You are a helpful query rewriting assistant."},
		{Role: "user", Content: prompt},
	}

	response, err := r.llm.GenerateWithMessages(ctx, messages)
	if err != nil {
		errorMsg := err.Error()
		return &QueryRewriteResult{
			OriginalQuery:  query,
			RewrittenQuery: query,
			IsRewritten:    false,
			Error:          &errorMsg,
			Metadata: map[string]interface{}{
				"rewrite_time_seconds": time.Since(startTime).Seconds(),
			},
		}
	}

	rewritten := strings.TrimSpace(response)
	elapsed := time.Since(startTime).Seconds()

	// If rewritten query is empty or same as original, mark as not rewritten
	isRewritten := rewritten != "" && rewritten != trimmedQuery

	return &QueryRewriteResult{
		OriginalQuery:  query,
		RewrittenQuery: rewritten,
		IsRewritten:    isRewritten,
		ProfileUsed:    &profileContent,
		Metadata: map[string]interface{}{
			"rewrite_time_seconds": elapsed,
		},
	}
}
