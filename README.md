# PowerMem Go SDK

<p align="center">
    <a href="https://github.com/oceanbase/powermem">
        <img alt="PowerMem" src="../../docs/images/powermem_en.png" width="60%" />
    </a>
</p>

<p align="center">
    <a href="https://pkg.go.dev/github.com/oceanbase/powermem-go">
        <img src="https://pkg.go.dev/badge/github.com/oceanbase/powermem-go.svg" alt="Go Reference">
    </a>
    <a href="https://github.com/oceanbase/powermem/blob/master/LICENSE">
        <img alt="license" src="https://img.shields.io/badge/license-Apache%202.0-green.svg" />
    </a>
    <img alt="Go version" src="https://img.shields.io/badge/go-%3E%3D1.19-blue.svg" />
</p>

Official Go SDK for [PowerMem](https://github.com/oceanbase/powermem) - an intelligent memory system for AI applications. PowerMem enables large language models to persistently "remember" historical conversations, user preferences, and contextual information through a hybrid storage architecture combining vector retrieval, full-text search, and graph databases.

## ✨ Features

- 🔌 **Simple Integration**: Lightweight SDK with automatic `.env` configuration loading
- 🧠 **Intelligent Memory**: Automatic fact extraction, duplicate detection, and memory merging
- 📉 **Ebbinghaus Curve**: Time-decay weighting based on cognitive science principles
- 🤖 **Multi-Agent Support**: Independent memory spaces with flexible sharing and isolation
- ⚡ **Async Operations**: Full async/await support for high-performance scenarios
- 🎨 **Multimodal Memory**: Support for text, images, and audio content
- 💾 **Flexible Storage**: SQLite for development, PostgreSQL/OceanBase for production
- 🔍 **Hybrid Retrieval**: Vector search, full-text search, and graph traversal

## 📦 Installation

```bash
go get github.com/oceanbase/powermem-go
```

## 🚀 Quick Start

### Prerequisites

- Go 1.19 or higher
- API keys for LLM and embedding providers (OpenAI, Qwen, etc.)
- Vector database (SQLite for local development, OceanBase/PostgreSQL for production)

### Basic Usage

**✨ Simplest Way**: Create memory from `.env` file automatically!

1. **Create a `.env` file** (see [`.env.example`](../../.env.example) for reference):

```env
# LLM Configuration
LLM_PROVIDER=qwen
LLM_API_KEY=your_api_key_here
LLM_MODEL=qwen-plus

# Embedding Configuration
EMBEDDER_PROVIDER=qwen
EMBEDDER_API_KEY=your_api_key_here
EMBEDDER_MODEL=text-embedding-v4

# Vector Store Configuration
VECTOR_STORE_PROVIDER=sqlite
VECTOR_STORE_COLLECTION_NAME=memories
```

1. **Use the SDK**:

```go
package main

import (
    "context"
    "fmt"
    "log"

    powermem "github.com/oceanbase/powermem-go/pkg/core"
)

func main() {
    // Load configuration from .env file
    config, err := powermem.LoadConfigFromEnv()
    if err != nil {
        log.Fatal(err)
    }

    // Create memory client
    client, err := powermem.NewClient(config)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    ctx := context.Background()
    userID := "user123"

    // Add memory
    memory, err := client.Add(ctx, "User likes coffee",
        powermem.WithUserID(userID),
    )
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Added memory: %s (ID: %d)\n", memory.Content, memory.ID)

    // Search memories
    results, err := client.Search(ctx, "user preferences",
        powermem.WithUserIDForSearch(userID),
        powermem.WithLimit(10),
    )
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("\nSearch results:")
    for _, result := range results {
        fmt.Printf("- %s (score: %.4f)\n", result.Memory, result.Score)
    }
}
```

## 📚 Documentation

### Core Concepts

- **[API Reference](docs/api.md)** - Complete API documentation
- **[Configuration Guide](../../docs/guides/0003-configuration.md)** - Detailed configuration options
- **[Examples](examples/)** - Working code examples for various scenarios

### Examples

Explore the [examples](examples/) directory for complete working examples:

- **[Basic Usage](examples/basic/)** - Simple memory operations (add, search, get, update, delete)
- **[Advanced Usage](examples/advanced/)** - Intelligent memory with fact extraction and deduplication
- **[Async Operations](examples/async/)** - High-performance async memory operations
- **[Multi-Agent](examples/multi_agent/)** - Agent isolation and memory sharing
- **[Streaming](examples/streaming/)** - Real-time streaming search results
- **[Ebbinghaus Curve](examples/ebbinghaus/)** - Memory decay and review scheduling
- **[User Memory](examples/user_memory/)** - User profile management and query rewriting

## 🔧 Configuration

### Environment Variables

PowerMem supports automatic configuration loading from `.env` files. Copy [`.env.example`](../../.env.example) to `.env` and configure:

```bash
cp ../../.env.example .env
```

### Programmatic Configuration

You can also create configuration programmatically:

```go
config := &powermem.Config{
    LLM: powermem.LLMConfig{
        Provider: "qwen",
        APIKey:   "your_api_key",
        Model:    "qwen-plus",
    },
    Embedder: powermem.EmbedderConfig{
        Provider: "qwen",
        APIKey:   "your_api_key",
        Model:    "text-embedding-v4",
    },
    VectorStore: powermem.VectorStoreConfig{
        Provider:       "sqlite",
        CollectionName: "memories",
    },
}
```

### Supported Providers

**LLM Providers:**

- OpenAI (GPT-4, GPT-3.5)
- Qwen (Qwen-Plus, Qwen-Turbo)
- Anthropic (Claude)
- DeepSeek
- Ollama (local)

**Embedding Providers:**

- OpenAI
- Qwen

**Vector Stores:**

- SQLite (development)
- PostgreSQL (production)
- OceanBase (production, recommended)

## 🧪 Testing

Run tests:

```bash
# Run all tests
make test

# Run specific test suite
make test-core
make test-storage
make test-intelligence

# Run with coverage
make test-coverage
```

## 🛠️ Development

```bash
# Install dependencies
make install

# Format code
make fmt

# Run linter
make lint

# Run go vet
make vet

# Build project
make build

# Build examples
make examples

# Run all checks (fmt, vet, lint, test)
make check
```

## 📄 License

Apache License 2.0. See [LICENSE](../../LICENSE) for details.

## 🔗 Links

- **[Main Repository](https://github.com/oceanbase/powermem)** - Python SDK and documentation
- **[OceanBase](https://github.com/oceanbase/oceanbase)** - Recommended production database
- **[Documentation](../../docs/)** - Comprehensive guides and examples
