// Package qwen provides Qwen Embedder implementation using Alibaba Cloud DashScope Text Embedding API.
//
// Qwen Embedder converts text into vector embeddings for similarity search.
// This package implements the embedder.Provider interface.
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
)

// Client implements embedder.Provider using Alibaba Cloud DashScope Text Embedding API.
//
// It provides text-to-vector conversion capabilities using Qwen embedding models.
type Client struct {
	// client is the HTTP client for API requests.
	client *http.Client

	// apiKey is the DashScope API key.
	apiKey string

	// model is the Qwen embedding model name to use.
	model string

	// baseURL is the base URL for DashScope API.
	baseURL string

	// dimensions is the dimension of embedding vectors.
	dimensions int
}

// Config contains configuration for creating a Qwen Embedder client.
type Config struct {
	// APIKey is the DashScope API key (required).
	APIKey string

	// Model is the model name to use (default: "text-embedding-v4").
	Model string

	// BaseURL is the API base URL (default: DashScope official address).
	BaseURL string

	// Dimensions is the vector dimension (default: 1536 for text-embedding-v4).
	Dimensions int

	// HTTPClient is a custom HTTP client (uses default if nil).
	HTTPClient *http.Client
}

// NewClient creates a new Qwen Embedder client.
//
// Parameters:
//   - cfg: Qwen Embedder configuration containing APIKey, Model, BaseURL, Dimensions, etc.
//
// Returns:
//   - *Client: Qwen Embedder client instance
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
		model = "text-embedding-v4"
	}

	dimensions := cfg.Dimensions
	if dimensions == 0 {
		dimensions = 1536 // text-embedding-v4 default dimension
	}

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	return &Client{
		client:     client,
		apiKey:     cfg.APIKey,
		model:      model,
		baseURL:    baseURL,
		dimensions: dimensions,
	}, nil
}

// Embed converts a single text string into a vector embedding.
//
// Parameters:
//   - ctx: Context for controlling request lifecycle
//   - text: Text content to embed
//
// Returns:
//   - []float64: Vector representation of the text (dimension determined by configuration)
//   - error: Error if embedding fails
func (c *Client) Embed(ctx context.Context, text string) ([]float64, error) {
	// Build request
	reqBody := map[string]interface{}{
		"model": c.model,
		"input": map[string]interface{}{
			"texts": []string{text},
		},
	}

	// Add dimension parameter
	if c.dimensions > 0 {
		reqBody["parameters"] = map[string]interface{}{
			"dimension": c.dimensions,
		}
	}

	// Default to document type
	reqBody["text_type"] = "document"

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/services/embeddings/text-embedding/text-embedding", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	// Send request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response struct {
		Output struct {
			Embeddings []struct {
				Embedding []float64 `json:"embedding"`
			} `json:"embeddings"`
		} `json:"output"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(response.Output.Embeddings) == 0 {
		return nil, errors.New("embedding generation failed: no embeddings returned from Qwen API")
	}

	return response.Output.Embeddings[0].Embedding, nil
}

// EmbedBatch converts multiple text strings into vector embeddings in a single batch.
//
// This method is more efficient than calling Embed multiple times,
// as it can batch process requests.
//
// Parameters:
//   - ctx: Context for controlling request lifecycle
//   - texts: List of texts to embed
//
// Returns:
//   - [][]float64: Vector representations for each text (order matches input texts)
//   - error: Error if embedding fails or number of results doesn't match input
func (c *Client) EmbedBatch(ctx context.Context, texts []string) ([][]float64, error) {
	// Build request
	reqBody := map[string]interface{}{
		"model": c.model,
		"input": map[string]interface{}{
			"texts": texts,
		},
	}

	// Add dimension parameter
	if c.dimensions > 0 {
		reqBody["parameters"] = map[string]interface{}{
			"dimension": c.dimensions,
		}
	}

	// Default to document type
	reqBody["text_type"] = "document"

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/services/embeddings/text-embedding/text-embedding", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	// Send request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response struct {
		Output struct {
			Embeddings []struct {
				Embedding []float64 `json:"embedding"`
			} `json:"embeddings"`
		} `json:"output"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(response.Output.Embeddings) != len(texts) {
		return nil, fmt.Errorf("embedding generation failed: unexpected number of results from Qwen API (got %d, expected %d)", len(response.Output.Embeddings), len(texts))
	}

	embeddings := make([][]float64, len(texts))
	for i, emb := range response.Output.Embeddings {
		embeddings[i] = emb.Embedding
	}

	return embeddings, nil
}

// Dimensions returns the dimension of embedding vectors produced by this provider.
//
// Returns:
//   - int: Vector dimension number
func (c *Client) Dimensions() int {
	return c.dimensions
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
