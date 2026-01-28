package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/oceanbase/powermem-go/pkg/llm"
)

// Client is an Anthropic LLM client.
// It implements the llm.Provider interface and provides text generation functionality based on the Anthropic Claude API.
// Supports system message separation, conforming to the Anthropic Messages API specification.
type Client struct {
	client  *http.Client
	apiKey  string
	model   string
	baseURL string
}

// Config is the configuration for Anthropic LLM.
// APIKey: Anthropic API key (required)
// Model: Model name to use, defaults to "claude-3-5-sonnet-20240620"
// BaseURL: API base URL, defaults to "https://api.anthropic.com"
// HTTPClient: Custom HTTP client, if nil uses default client (120 seconds timeout)
type Config struct {
	APIKey     string
	Model      string
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a new Anthropic LLM client.
//
// Args:
//   - cfg: Anthropic configuration containing APIKey, Model, BaseURL, etc.
//
// Returns:
//   - *Client: Anthropic client instance
//   - error: Returns an error if the configuration is invalid (e.g., missing APIKey) or initialization fails
func NewClient(cfg *Config) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("API key is required")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}

	model := cfg.Model
	if model == "" {
		model = "claude-3-5-sonnet-20240620"
	}

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{
			Timeout: 120 * time.Second,
		}
	}

	return &Client{
		client:  client,
		apiKey:  cfg.APIKey,
		model:   model,
		baseURL: baseURL,
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
// Note: Anthropic API requires system messages to be passed separately, not in the messages array.
//
// Args:
//   - ctx: Context for controlling the request lifecycle
//   - messages: Message history list, each message contains role and content (system messages will be automatically separated)
//   - opts: Optional generation parameters (temperature, max_tokens, top_p, etc.)
//
// Returns:
//   - string: Generated text content
//   - error: Returns an error if generation fails
func (c *Client) GenerateWithMessages(ctx context.Context, messages []llm.Message, opts ...llm.GenerateOption) (string, error) {
	options := llm.ApplyGenerateOptions(opts)

	// Separate system messages from other messages
	var systemMessage string
	var filteredMessages []map[string]string

	for _, msg := range messages {
		if msg.Role == "system" {
			systemMessage = msg.Content
		} else {
			filteredMessages = append(filteredMessages, map[string]string{
				"role":    msg.Role,
				"content": msg.Content,
			})
		}
	}

	// Build request body
	reqBody := map[string]interface{}{
		"model":       c.model,
		"max_tokens":  options.MaxTokens,
		"temperature": options.Temperature,
		"top_p":       options.TopP,
		"messages":    filteredMessages,
	}

	if systemMessage != "" {
		reqBody["system"] = systemMessage
	}

	if len(options.Stop) > 0 {
		reqBody["stop_sequences"] = options.Stop
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/v1/messages", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// Send request
	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if len(response.Content) == 0 {
		return "", errors.New("llm generation failed: no content returned from Anthropic API")
	}

	return response.Content[0].Text, nil
}

// Close closes the client connection.
// HTTP client does not require explicit closing; this method is retained for interface compatibility.
//
// Returns:
//   - error: Always returns nil
func (c *Client) Close() error {
	return nil
}
