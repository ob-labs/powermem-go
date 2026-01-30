// Package qwen provides Qwen LLM implementation using Alibaba Cloud DashScope API.
//
// Qwen is a large language model developed by Alibaba Cloud. This package
// implements the llm.Provider interface for text generation using DashScope API.
package qwen

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

// Client implements llm.Provider using Alibaba Cloud DashScope API.
//
// It provides text generation capabilities based on Qwen models.
type Client struct {
	// client is the HTTP client for API requests.
	client *http.Client

	// apiKey is the DashScope API key.
	apiKey string

	// model is the Qwen model name to use.
	model string

	// baseURL is the base URL for DashScope API.
	baseURL string
}

// Config contains configuration for creating a Qwen LLM client.
type Config struct {
	// APIKey is the DashScope API key (required).
	APIKey string

	// Model is the model name to use (default: "qwen-plus").
	Model string

	// BaseURL is the API base URL (default: DashScope official address).
	BaseURL string

	// HTTPClient is a custom HTTP client (uses default if nil).
	HTTPClient *http.Client
}

// NewClient creates a new Qwen LLM client.
//
// Parameters:
//   - cfg: Qwen configuration containing APIKey, Model, BaseURL, etc.
//
// Returns:
//   - *Client: Qwen client instance
//   - error: Error if configuration is invalid (e.g., missing APIKey) or initialization fails
func NewClient(cfg *Config) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("API key is required")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://dashscope.aliyuncs.com/api/v1"
	}

	model := cfg.Model
	if model == "" {
		model = "qwen-plus"
	}

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	return &Client{
		client:  client,
		apiKey:  cfg.APIKey,
		model:   model,
		baseURL: baseURL,
	}, nil
}

// Generate generates text from a prompt.
//
// Parameters:
//   - ctx: Context for controlling request lifecycle
//   - prompt: User input prompt
//   - opts: Optional generation parameters (temperature, max_tokens, top_p, etc.)
//
// Returns:
//   - string: Generated text content
//   - error: Error if generation fails
func (c *Client) Generate(ctx context.Context, prompt string, opts ...llm.GenerateOption) (string, error) {
	messages := []llm.Message{
		{Role: "user", Content: prompt},
	}
	return c.GenerateWithMessages(ctx, messages, opts...)
}

// GenerateWithMessages generates text from a conversation history.
//
// Supports multi-turn conversations with complete message history
// (including system, user, and assistant messages).
//
// Parameters:
//   - ctx: Context for controlling request lifecycle
//   - messages: Message history list, each message contains role and content
//   - opts: Optional generation parameters (temperature, max_tokens, top_p, etc.)
//
// Returns:
//   - string: Generated text content
//   - error: Error if generation fails
func (c *Client) GenerateWithMessages(ctx context.Context, messages []llm.Message, opts ...llm.GenerateOption) (string, error) {
	options := llm.ApplyGenerateOptions(opts)

	// Convert message format
	chatMessages := make([]map[string]string, len(messages))
	for i, msg := range messages {
		chatMessages[i] = map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}

	// Build request
	reqBody := map[string]interface{}{
		"model": c.model,
		"input": map[string]interface{}{"messages": chatMessages},
		"parameters": map[string]interface{}{
			"temperature": options.Temperature,
			"max_tokens":  options.MaxTokens,
			"top_p":       options.TopP,
		},
	}

	if len(options.Stop) > 0 {
		reqBody["parameters"].(map[string]interface{})["stop"] = options.Stop
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/services/aigc/text-generation/generation", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

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
		Output struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		} `json:"output"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if len(response.Output.Choices) == 0 {
		return "", errors.New("llm generation failed: no choices returned from Qwen API")
	}

	return response.Output.Choices[0].Message.Content, nil
}

// Close closes the client connection.
//
// HTTP clients do not need explicit closing, this method is retained for interface compatibility.
//
// Returns:
//   - error: Always returns nil
func (c *Client) Close() error {
	return nil
}
