package ollama

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

// Client is an Ollama LLM client.
// It implements the llm.Provider interface and provides text generation functionality based on Ollama local/remote service.
// Ollama is a tool for running large language models locally, supporting both local deployment and remote access.
type Client struct {
	client  *http.Client
	apiKey  string
	model   string
	baseURL string
}

// Config is the configuration for Ollama LLM.
// APIKey: Ollama API key (optional, usually not required for local deployment)
// Model: Model name to use, defaults to "llama3.1:70b"
// BaseURL: Ollama service address, defaults to "http://localhost:11434"
// HTTPClient: Custom HTTP client, if nil uses default client (120 seconds timeout)
type Config struct {
	APIKey     string
	Model      string
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a new Ollama LLM client.
//
// Args:
//   - cfg: Ollama configuration containing Model, BaseURL, etc. (APIKey is optional)
//
// Returns:
//   - *Client: Ollama client instance
//   - error: Returns an error if initialization fails
func NewClient(cfg *Config) (*Client, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	model := cfg.Model
	if model == "" {
		model = "llama3.1:70b"
	}

	client := cfg.HTTPClient
	if client == nil {
		// Ollama may require longer timeout, especially for large models
		client = &http.Client{
			Timeout: 120 * time.Second,
		}
	}

	return &Client{
		client:  client,
		apiKey:  cfg.APIKey, // Ollama local deployment usually doesn't require API key, but kept to support authenticated remote deployment
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
// Note: Ollama uses different parameter names (num_predict instead of max_tokens).
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
	chatMessages := make([]map[string]string, len(messages))
	for i, msg := range messages {
		chatMessages[i] = map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}

	// Build request body
	reqBody := map[string]interface{}{
		"model":    c.model,
		"messages": chatMessages,
		"options": map[string]interface{}{
			"temperature": options.Temperature,
			"num_predict": options.MaxTokens,
			"top_p":       options.TopP,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/chat", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}

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
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if response.Message.Content == "" {
		return "", errors.New("llm generation failed: empty response from Ollama API")
	}

	return response.Message.Content, nil
}

// Close closes the client connection.
// HTTP client does not require explicit closing; this method is retained for interface compatibility.
//
// Returns:
//   - error: Always returns nil
func (c *Client) Close() error {
	return nil
}
