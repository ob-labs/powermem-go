package openai

import (
	"context"
	"errors"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

// Client is an OpenAI Embedder client.
// It implements the embedder.Provider interface and provides text vectorization functionality based on the OpenAI Embeddings API.
type Client struct {
	client     *openai.Client
	model      openai.EmbeddingModel
	dimensions int
}

// Config is the configuration for OpenAI Embedder.
// APIKey: OpenAI API key (required)
// Model: Model name to use, currently fixed to AdaEmbeddingV2
// BaseURL: API base URL, defaults to OpenAI official address
// Dimensions: Vector dimensions, defaults to 1536 (default dimension for AdaEmbeddingV2)
type Config struct {
	APIKey     string
	Model      string
	BaseURL    string
	Dimensions int
}

// NewClient creates a new OpenAI Embedder client.
//
// Args:
//   - cfg: OpenAI Embedder configuration containing APIKey, BaseURL, Dimensions, etc.
//
// Returns:
//   - *Client: OpenAI Embedder client instance
//   - error: Returns an error if the configuration is invalid or initialization fails
func NewClient(cfg *Config) (*Client, error) {
	config := openai.DefaultConfig(cfg.APIKey)
	if cfg.BaseURL != "" {
		config.BaseURL = cfg.BaseURL
	}

	client := openai.NewClientWithConfig(config)

	// Default to Ada v2 model
	model := openai.AdaEmbeddingV2

	dimensions := cfg.Dimensions
	if dimensions == 0 {
		dimensions = 1536 // Default dimension for AdaEmbeddingV2
	}

	return &Client{
		client:     client,
		model:      model,
		dimensions: dimensions,
	}, nil
}

// Embed converts a single text to a vector.
//
// Args:
//   - ctx: Context for controlling the request lifecycle
//   - text: Text content to vectorize
//
// Returns:
//   - []float64: Vector representation of the text (dimension determined by configuration)
//   - error: Returns an error if vectorization fails
func (c *Client) Embed(ctx context.Context, text string) ([]float64, error) {
	resp, err := c.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{text},
		Model: c.model,
	})
	if err != nil {
		return nil, err
	}

	if len(resp.Data) == 0 {
		return nil, errors.New("embedding generation failed: no data returned from OpenAI API")
	}

	// Convert float32 to float64
	embedding32 := resp.Data[0].Embedding
	embedding64 := make([]float64, len(embedding32))
	for i, v := range embedding32 {
		embedding64[i] = float64(v)
	}

	return embedding64, nil
}

// EmbedBatch converts multiple texts to vectors in batch.
//
// Args:
//   - ctx: Context for controlling the request lifecycle
//   - texts: List of texts to vectorize
//
// Returns:
//   - [][]float64: Vector representation for each text (order matches input texts)
//   - error: Returns an error if vectorization fails or the number of returned results doesn't match
func (c *Client) EmbedBatch(ctx context.Context, texts []string) ([][]float64, error) {
	resp, err := c.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: texts,
		Model: c.model,
	})
	if err != nil {
		return nil, err
	}

	if len(resp.Data) != len(texts) {
		return nil, fmt.Errorf("embedding generation failed: unexpected number of results from OpenAI API (got %d, expected %d)", len(resp.Data), len(texts))
	}

	embeddings := make([][]float64, len(texts))
	for i, data := range resp.Data {
		embedding32 := data.Embedding
		embedding64 := make([]float64, len(embedding32))
		for j, v := range embedding32 {
			embedding64[j] = float64(v)
		}
		embeddings[i] = embedding64
	}

	return embeddings, nil
}

// Dimensions returns the vector dimensions.
//
// Returns:
//   - int: Number of vector dimensions
func (c *Client) Dimensions() int {
	return c.dimensions
}

// Close closes the client connection.
// The OpenAI SDK client does not require explicit closing; this method is retained for interface compatibility.
//
// Returns:
//   - error: Always returns nil
func (c *Client) Close() error {
	return nil
}
