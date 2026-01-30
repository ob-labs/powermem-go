// Package llm provides interfaces and utilities for Large Language Model (LLM) providers.
//
// It defines the Provider interface that all LLM implementations must satisfy,
// along with message types and generation options.
package llm

import "context"

// Provider defines the interface for LLM providers.
//
// All LLM implementations (OpenAI, Qwen, Anthropic, etc.) must implement this interface.
type Provider interface {
	// Generate generates text from a prompt.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout
	//   - prompt: The input prompt text
	//   - opts: Optional generation parameters (temperature, max tokens, etc.)
	//
	// Returns the generated text and any error.
	Generate(ctx context.Context, prompt string, opts ...GenerateOption) (string, error)

	// GenerateWithMessages generates text from a conversation history.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout
	//   - messages: Conversation history (system, user, assistant messages)
	//   - opts: Optional generation parameters
	//
	// Returns the generated text and any error.
	GenerateWithMessages(ctx context.Context, messages []Message, opts ...GenerateOption) (string, error)

	// Close closes the provider and releases resources.
	Close() error
}

// Message represents a single message in a conversation.
type Message struct {
	// Role is the message role: "system", "user", or "assistant".
	Role string `json:"role"`

	// Content is the message content text.
	Content string `json:"content"`
}

// GenerateOptions contains options for text generation.
type GenerateOptions struct {
	// Temperature controls randomness (0.0-2.0). Higher = more random.
	Temperature float64

	// MaxTokens limits the maximum number of tokens in the response.
	MaxTokens int

	// TopP controls nucleus sampling (0.0-1.0). Higher = more diverse.
	TopP float64

	// Stop contains stop sequences that will end generation.
	Stop []string
}

// GenerateOption is a function type for configuring generation options.
type GenerateOption func(*GenerateOptions)

// WithTemperature sets the temperature for text generation.
//
// Temperature controls randomness: 0.0 = deterministic, 2.0 = very random.
//
// Example:
//
//	text, _ := llm.Generate(ctx, "Hello", llm.WithTemperature(0.7))
func WithTemperature(temp float64) GenerateOption {
	return func(opts *GenerateOptions) {
		opts.Temperature = temp
	}
}

// WithMaxTokens sets the maximum number of tokens in the response.
//
// Example:
//
//	text, _ := llm.Generate(ctx, "Hello", llm.WithMaxTokens(100))
func WithMaxTokens(max int) GenerateOption {
	return func(opts *GenerateOptions) {
		opts.MaxTokens = max
	}
}

// WithTopP sets the top-p (nucleus sampling) parameter.
//
// TopP controls diversity: 0.0 = most likely tokens only, 1.0 = all tokens.
//
// Example:
//
//	text, _ := llm.Generate(ctx, "Hello", llm.WithTopP(0.9))
func WithTopP(topP float64) GenerateOption {
	return func(opts *GenerateOptions) {
		opts.TopP = topP
	}
}

// ApplyGenerateOptions applies a slice of GenerateOption functions to create GenerateOptions.
//
// This is a helper function used internally by LLM implementations.
// Default values: Temperature=0.7, MaxTokens=1000, TopP=1.0.
func ApplyGenerateOptions(opts []GenerateOption) *GenerateOptions {
	options := &GenerateOptions{
		Temperature: 0.7,
		MaxTokens:   1000,
		TopP:        1.0,
	}
	for _, opt := range opts {
		opt(options)
	}
	return options
}
