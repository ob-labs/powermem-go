# PowerMem Go SDK

Official Go SDK for [PowerMem](https://github.com/oceanbase/powermem).

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
- Vector database (SQLite/OceanBase/PostgreSQL)

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

2. **Use the SDK**:

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

- **[API Reference](docs/api.md)** - Complete API documentation

- **[Examples](examples/)** - Working code examples for various scenarios

## 🔧 Configuration

### Environment Variables

PowerMem supports automatic configuration loading from `.env` files. Copy [`.env.example`](.env.example) to `.env` and configure.

```bash
cp .env.example .env
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
```

## 🛠️ Development

```bash
# Install dependencies
make install

# Run linter
make lint

# Build project
make build

# Build examples
make examples

```

## 📄 License

Apache License 2.0. See [LICENSE](LICENSE) for details.

## 🔗 Links

- **[Main Repository](https://github.com/oceanbase/powermem)** - Python SDK and documentation
- **[OceanBase](https://github.com/oceanbase/oceanbase)** - Recommended production database
- **[Documentation](docs/)** - Comprehensive guides and examples
