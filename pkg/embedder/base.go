// Package embedder provides interfaces for text embedding providers.
//
// It defines the Provider interface that all embedding implementations must satisfy,
// enabling text-to-vector conversion for similarity search.
package embedder

import "context"

// Provider defines the interface for embedding providers.
//
// All embedding implementations (OpenAI, Qwen, etc.) must implement this interface.
type Provider interface {
	// Embed converts a text string into a vector embedding.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout
	//   - text: The input text to embed
	//
	// Returns the embedding vector and any error.
	Embed(ctx context.Context, text string) ([]float64, error)

	// EmbedBatch converts multiple text strings into vector embeddings.
	//
	// This method is more efficient than calling Embed multiple times,
	// as it can batch process requests.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout
	//   - texts: Slice of input texts to embed
	//
	// Returns a slice of embedding vectors and any error.
	EmbedBatch(ctx context.Context, texts []string) ([][]float64, error)

	// Dimensions returns the dimension of embedding vectors produced by this provider.
	//
	// For example, OpenAI's text-embedding-ada-002 produces 1536-dimensional vectors.
	Dimensions() int

	// Close closes the provider and releases resources.
	Close() error
}
