// Package query_rewrite provides query rewriting functionality based on user profiles.
package query_rewrite

import "fmt"

// DefaultQueryRewriteInstructions is the default instruction text for query rewriting.
const DefaultQueryRewriteInstructions = `Use the user information to fill in any vague or ambiguous parts of the query.
Preserve the original intent of the query.
If the query is already clear and unambiguous, leave it unchanged.`

// QueryRewriteTemplate is the template for building query rewrite prompts.
const QueryRewriteTemplate = `# Task
Rewrite the query by clarifying any ambiguous or underspecified references based on the provided user information, making the query more precise.

# User Information
%s

# Requirements
%s

# Output
Output only the rewritten query—do not add any explanations.

# Query
%s`

// buildQueryRewritePrompt builds a query rewrite prompt with user profile and query.
//
// Parameters:
//   - profileContent: User profile text
//   - query: Original query string
//   - customInstructions: Optional custom instructions (uses default if empty)
//
// Returns the complete prompt string for the LLM.
func buildQueryRewritePrompt(profileContent, query, customInstructions string) string {
	instructions := customInstructions
	if instructions == "" {
		instructions = DefaultQueryRewriteInstructions
	}

	return fmt.Sprintf(QueryRewriteTemplate, profileContent, instructions, query)
}
