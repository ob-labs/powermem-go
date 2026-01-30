package deepseek

import (
	"context"
	"errors"

	"github.com/oceanbase/powermem-go/pkg/llm"
	openai "github.com/sashabaranov/go-openai"
)

// Client is a DeepSeek LLM client.
// It implements the llm.Provider interface and provides text generation functionality based on the DeepSeek API.
// DeepSeek uses OpenAI-compatible API format, so it can reuse the OpenAI SDK.
type Client struct {
	client *openai.Client
	model  string
}

// Config is the configuration for DeepSeek LLM.
// APIKey: DeepSeek API key (required)
// Model: Model name to use, defaults to "deepseek-chat"
// BaseURL: API base URL, defaults to "https://api.deepseek.com"
type Config struct {
	APIKey  string
	Model   string
	BaseURL string
}

// NewClient creates a new DeepSeek LLM client.
//
// Args:
//   - cfg: DeepSeek configuration containing APIKey, Model, and BaseURL
//
// Returns:
//   - *Client: DeepSeek client instance
//   - error: Returns an error if the configuration is invalid or initialization fails
func NewClient(cfg *Config) (*Client, error) {
	config := openai.DefaultConfig(cfg.APIKey)

	// DeepSeek uses OpenAI-compatible API, but with a different base URL
	if cfg.BaseURL != "" {
		config.BaseURL = cfg.BaseURL
	} else {
		// Default DeepSeek API base URL
		config.BaseURL = "https://api.deepseek.com"
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
		return "", errors.New("llm generation failed: no choices returned from DeepSeek API")
	}

	return resp.Choices[0].Message.Content, nil
}

// Close closes the client connection.
// DeepSeek client (based on OpenAI SDK) does not require explicit closing; this method is retained for interface compatibility.
//
// Returns:
//   - error: Always returns nil
func (c *Client) Close() error {
	return nil
}
