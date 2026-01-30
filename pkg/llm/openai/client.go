package openai

import (
	"context"
	"errors"

	"github.com/oceanbase/powermem-go/pkg/llm"
	openai "github.com/sashabaranov/go-openai"
)

// Client is an OpenAI LLM client.
// It implements the llm.Provider interface and provides text generation functionality based on the OpenAI API.
type Client struct {
	client *openai.Client
	model  string
}

// Config is the configuration for OpenAI LLM.
// APIKey: OpenAI API key (required)
// Model: Model name to use, defaults to "gpt-4"
// BaseURL: API base URL, defaults to OpenAI official address
type Config struct {
	APIKey  string
	Model   string
	BaseURL string
}

// NewClient creates a new OpenAI LLM client.
//
// Args:
//   - cfg: OpenAI configuration containing APIKey, Model, and BaseURL
//
// Returns:
//   - *Client: OpenAI client instance
//   - error: Returns an error if the configuration is invalid or initialization fails
func NewClient(cfg *Config) (*Client, error) {
	config := openai.DefaultConfig(cfg.APIKey)
	if cfg.BaseURL != "" {
		config.BaseURL = cfg.BaseURL
	}

	client := openai.NewClientWithConfig(config)

	return &Client{
		client: client,
		model:  cfg.Model,
	}, nil
}

// Generate generates text based on the prompt.
//
// Args:
//   - ctx: Context for controlling the request lifecycle
//   - prompt: User input prompt
//   - opts: Optional generation parameters (temperature, max_tokens, top_p, etc.)
//
// Returns:
//   - string: Generated text content
//   - error: Returns an error if generation fails
func (c *Client) Generate(ctx context.Context, prompt string, opts ...llm.GenerateOption) (string, error) {
	messages := []llm.Message{
		{Role: "user", Content: prompt},
	}
	return c.GenerateWithMessages(ctx, messages, opts...)
}

// GenerateWithMessages generates text using message history.
// Supports multi-turn conversations and accepts complete message history (including system, user, and assistant messages).
//
// Args:
//   - ctx: Context for controlling the request lifecycle
//   - messages: Message history list, each message contains role and content
//   - opts: Optional generation parameters (temperature, max_tokens, top_p, etc.)
//
// Returns:
//   - string: Generated text content
//   - error: Returns an error if generation fails
func (c *Client) GenerateWithMessages(ctx context.Context, messages []llm.Message, opts ...llm.GenerateOption) (string, error) {
	options := llm.ApplyGenerateOptions(opts)

	// Convert message format
	chatMessages := make([]openai.ChatCompletionMessage, len(messages))
	for i, msg := range messages {
		chatMessages[i] = openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	req := openai.ChatCompletionRequest{
		Model:       c.model,
		Messages:    chatMessages,
		Temperature: float32(options.Temperature),
		MaxTokens:   options.MaxTokens,
		TopP:        float32(options.TopP),
		Stop:        options.Stop,
	}

	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", errors.New("llm generation failed: no choices returned from OpenAI API")
	}

	return resp.Choices[0].Message.Content, nil
}

// Close closes the client connection.
// The OpenAI SDK client does not require explicit closing; this method is retained for interface compatibility.
//
// Returns:
//   - error: Always returns nil
func (c *Client) Close() error {
	return nil
}
