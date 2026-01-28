# PowerMem Go SDK 设计文档

> 版本：v0.1  
> 日期：2026-01-18  
> 作者：Chris Lee
> Issue: [#143](https://github.com/oceanbase/powermem/issues/143)

## 目录

- [PowerMem Go SDK 设计文档](#powermem-go-sdk-设计文档)
  - [目录](#目录)
  - [1. 需要实现的接口和功能](#1-需要实现的接口和功能)
    - [1.1 核心 Memory 接口](#11-核心-memory-接口)
    - [1.2 配置管理](#12-配置管理)
    - [1.3 高级功能](#13-高级功能)
  - [2. 模块实现方案](#2-模块实现方案)
    - [2.1 整体架构](#21-整体架构)
    - [2.2 OceanBase 向量存储实现方案](#22-oceanbase-向量存储实现方案)
      - [2.2.1 数据库连接](#221-数据库连接)
      - [2.2.2 表结构设计](#222-表结构设计)
      - [2.2.3 向量操作实现](#223-向量操作实现)
      - [2.2.4 向量索引管理](#224-向量索引管理)
    - [2.3 配置管理模块](#23-配置管理模块)
    - [2.4 Memory 客户端实现](#24-memory-客户端实现)
    - [2.5 并发安全与 Context 支持](#25-并发安全与-context-支持)
    - [2.6 错误处理](#26-错误处理)
  - [3. 示例代码](#3-示例代码)
    - [3.1 基础使用示例](#31-基础使用示例)
    - [3.2 高级示例：智能记忆管理](#32-高级示例智能记忆管理)
    - [3.3 多代理场景示例](#33-多代理场景示例)
  - [4. 总结](#4-总结)
    - [4.1 技术要点](#41-技术要点)
    - [4.2 实现优先级](#42-实现优先级)
    - [4.3 依赖库](#43-依赖库)

---

## 1. 需要实现的接口和功能

根据 Python SDK 的核心功能，Go SDK 需要实现以下接口：

### 1.1 核心 Memory 接口

| 方法 | 功能描述 | 参数 | 返回值 |
|------|---------|------|--------|
| `Add(ctx, messages, opts)` | 添加记忆（支持智能去重） | messages（文本/多模态）、user_id、agent_id、metadata | Memory 对象或 error |
| `Search(ctx, query, opts)` | 语义搜索记忆 | query、user_id、agent_id、limit、filters | []Memory 或 error |
| `Update(ctx, memoryID, content)` | 更新指定记忆 | memory_id、新内容 | Memory 或 error |
| `Delete(ctx, memoryID)` | 删除指定记忆 | memory_id | error |
| `GetAll(ctx, opts)` | 获取所有记忆 | user_id、agent_id、limit | []Memory 或 error |
| `DeleteAll(ctx, opts)` | 删除所有记忆 | user_id、agent_id | error |

### 1.2 配置管理

需要支持的配置项：

- **LLM 配置**：支持 OpenAI、Qwen、Anthropic、Gemini、Ollama 等多种 LLM 提供商
- **Embedder 配置**：支持 OpenAI、Qwen、HuggingFace、Ollama 等 Embedding 提供商
- **Vector Store 配置**：重点支持 OceanBase、SQLite、PostgreSQL（pgvector）
- **智能记忆配置**：Ebbinghaus 遗忘曲线参数
- **多代理/多用户配置**：agent_id、user_id、作用域、协作级别等

### 1.3 高级功能

- **智能去重**：在 Add 时自动检测和合并相似记忆（通过 infer 参数控制）
- **混合搜索**：支持向量搜索 + 全文搜索 + 稀疏向量搜索的混合检索
- **Reranker 支持**：对搜索结果进行重排序
- **图存储支持**：可选的知识图谱存储（用于关系抽取）
- **多模态支持**：支持文本、图像等多模态内容

## 2. 模块实现方案

### 2.1 整体架构

```
powermem-go/
├── pkg/
│   ├── powermem/
│   │   ├── memory.go           # Memory 客户端主接口
│   │   ├── config.go           # 配置管理
│   │   ├── types.go            # 核心类型定义
│   │   └── options.go          # 选项模式
│   ├── storage/
│   │   ├── base.go             # 存储接口定义
│   │   ├── oceanbase/
│   │   │   ├── client.go       # OceanBase 客户端
│   │   │   ├── vector.go       # 向量操作实现
│   │   │   └── sql.go          # SQL 构建器
│   │   ├── sqlite/
│   │   │   └── client.go       # SQLite 实现
│   │   └── postgres/
│   │       └── client.go       # PostgreSQL 实现
│   ├── llm/
│   │   ├── base.go             # LLM 接口
│   │   ├── openai/             # OpenAI 实现
│   │   ├── qwen/               # 通义千问实现
│   │   └── ...
│   ├── embedder/
│   │   ├── base.go             # Embedder 接口
│   │   ├── openai/             # OpenAI Embedding
│   │   ├── qwen/               # Qwen Embedding
│   │   └── ...
│   └── intelligence/
│       ├── ebbinghaus.go       # 遗忘曲线算法
│       └── dedup.go            # 去重逻辑
├── examples/
│   ├── basic/                  # 基础示例
│   └── advanced/               # 高级示例
└── tests/
    └── ...
```

### 2.2 OceanBase 向量存储实现方案

**重要提示**：由于 pyobvector 没有 Go 版本，需要手动实现向量存储功能。

#### 2.2.1 数据库连接

使用标准的 MySQL/Go SQL 驱动（OceanBase 兼容 MySQL 协议）：

```go
import (
    "database/sql"
    _ "github.com/go-sql-driver/mysql"
)

type OceanBaseClient struct {
    db *sql.DB
    config *OceanBaseConfig
}

func NewOceanBaseClient(cfg *OceanBaseConfig) (*OceanBaseClient, error) {
    dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
        cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)
    
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return nil, err
    }
    
    return &OceanBaseClient{db: db, config: cfg}, nil
}
```

#### 2.2.2 表结构设计

```sql
CREATE TABLE IF NOT EXISTS memories (
    id BIGINT PRIMARY KEY,                    -- Snowflake ID
    user_id VARCHAR(255) NOT NULL,
    agent_id VARCHAR(255),
    content LONGTEXT NOT NULL,
    embedding VECTOR(1536) NOT NULL,          -- 向量列（维度可配置）
    sparse_embedding SPARSE_VECTOR,           -- 稀疏向量（可选）
    fulltext_content TEXT,                    -- 全文搜索列
    metadata JSON,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    retention_strength FLOAT DEFAULT 1.0,     -- Ebbinghaus 强度
    last_accessed_at DATETIME,
    INDEX idx_user_agent (user_id, agent_id),
    VECTOR INDEX vidx (embedding) WITH (
        index_type = HNSW,
        M = 16,
        efConstruction = 200,
        metric_type = cosine
    ),
    FULLTEXT INDEX fts_idx (fulltext_content) WITH PARSER ik
);
```

#### 2.2.3 向量操作实现

**插入向量**：

```go
func (c *OceanBaseClient) Insert(ctx context.Context, memory *Memory) error {
    query := `INSERT INTO memories 
        (id, user_id, agent_id, content, embedding, metadata, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?)`
    
    // 将向量转换为 OceanBase VECTOR 格式
    vectorStr := vectorToString(memory.Embedding)
    
    _, err := c.db.ExecContext(ctx, query,
        memory.ID,
        memory.UserID,
        memory.AgentID,
        memory.Content,
        vectorStr,  // "[0.1, 0.2, 0.3, ...]" 格式
        memory.Metadata,
        time.Now(),
    )
    return err
}
```

**向量搜索**：

OceanBase 支持向量相似度函数，构造 SQL 查询：

```go
func (c *OceanBaseClient) Search(ctx context.Context, opts *SearchOptions) ([]Memory, error) {
    // 将查询向量转换为字符串格式
    queryVectorStr := vectorToString(opts.QueryEmbedding)
    
    // 构造 SQL - 使用 OceanBase 的向量距离函数
    query := `
        SELECT 
            id, user_id, agent_id, content, embedding, metadata,
            cosine_distance(embedding, ?) as distance
        FROM memories
        WHERE user_id = ?
        ORDER BY distance ASC
        LIMIT ?
    `
    
    rows, err := c.db.QueryContext(ctx, query,
        queryVectorStr,
        opts.UserID,
        opts.Limit,
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    return scanMemories(rows)
}
```

**混合搜索**（向量 + 全文 + 稀疏向量）：

```go
func (c *OceanBaseClient) HybridSearch(ctx context.Context, opts *HybridSearchOptions) ([]Memory, error) {
    query := `
        SELECT 
            id, user_id, content, metadata,
            (? * cosine_distance(embedding, ?) + 
             ? * MATCH(fulltext_content) AGAINST(? IN BOOLEAN MODE) +
             ? * cosine_distance(sparse_embedding, ?)) as hybrid_score
        FROM memories
        WHERE user_id = ?
        ORDER BY hybrid_score DESC
        LIMIT ?
    `
    
    rows, err := c.db.QueryContext(ctx, query,
        opts.VectorWeight, vectorToString(opts.QueryEmbedding),
        opts.FTSWeight, opts.QueryText,
        opts.SparseWeight, sparseVectorToString(opts.QuerySparseEmbedding),
        opts.UserID,
        opts.Limit,
    )
    
    return scanMemories(rows)
}
```

#### 2.2.4 向量索引管理

```go
func (c *OceanBaseClient) CreateVectorIndex(ctx context.Context, cfg *VectorIndexConfig) error {
    var query string
    
    switch cfg.IndexType {
    case "HNSW":
        query = fmt.Sprintf(`
            CREATE VECTOR INDEX %s ON %s (%s) WITH (
                index_type = HNSW,
                M = %d,
                efConstruction = %d,
                metric_type = %s
            )`,
            cfg.IndexName, cfg.TableName, cfg.VectorField,
            cfg.HNSWParams.M,
            cfg.HNSWParams.EfConstruction,
            cfg.MetricType,
        )
    case "IVF_FLAT":
        query = fmt.Sprintf(`
            CREATE VECTOR INDEX %s ON %s (%s) WITH (
                index_type = IVF_FLAT,
                nlist = %d,
                metric_type = %s
            )`,
            cfg.IndexName, cfg.TableName, cfg.VectorField,
            cfg.IVFParams.Nlist,
            cfg.MetricType,
        )
    }
    
    _, err := c.db.ExecContext(ctx, query)
    return err
}
```

### 2.3 配置管理模块

```go
type Config struct {
    LLM          LLMConfig          `json:"llm"`
    Embedder     EmbedderConfig     `json:"embedder"`
    VectorStore  VectorStoreConfig  `json:"vector_store"`
    Intelligence IntelligenceConfig `json:"intelligent_memory,omitempty"`
    AgentMemory  AgentMemoryConfig  `json:"agent_memory,omitempty"`
}

// 从环境变量加载配置
func LoadConfigFromEnv() (*Config, error) {
    // 读取 .env 文件或环境变量
    return &Config{
        LLM: LLMConfig{
            Provider: os.Getenv("LLM_PROVIDER"),
            APIKey:   os.Getenv("LLM_API_KEY"),
            Model:    os.Getenv("LLM_MODEL"),
        },
        // ...
    }
}

// 从 JSON 文件加载配置
func LoadConfigFromJSON(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    
    var cfg Config
    if err := json.Unmarshal(data, &cfg); err != nil {
        return nil, err
    }
    
    return &cfg, nil
}
```

### 2.4 Memory 客户端实现

```go
type Client struct {
    config       *Config
    storage      storage.VectorStore
    llm          llm.Provider
    embedder     embedder.Provider
    intelligence *intelligence.Manager
    mu           sync.RWMutex
}

func NewClient(cfg *Config) (*Client, error) {
    // 初始化存储
    store, err := initStorage(cfg.VectorStore)
    if err != nil {
        return nil, err
    }
    
    // 初始化 LLM
    llmProvider, err := initLLM(cfg.LLM)
    if err != nil {
        return nil, err
    }
    
    // 初始化 Embedder
    embedderProvider, err := initEmbedder(cfg.Embedder)
    if err != nil {
        return nil, err
    }
    
    return &Client{
        config:   cfg,
        storage:  store,
        llm:      llmProvider,
        embedder: embedderProvider,
    }, nil
}
```

### 2.5 并发安全与 Context 支持

所有操作都支持 `context.Context`，实现超时控制和取消：

```go
func (c *Client) Add(ctx context.Context, content string, opts ...AddOption) (*Memory, error) {
    // 应用选项
    addOpts := &AddOptions{}
    for _, opt := range opts {
        opt(addOpts)
    }
    
    // 检查 context
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }
    
    // 1. 生成 embedding
    embedding, err := c.embedder.Embed(ctx, content)
    if err != nil {
        return nil, err
    }
    
    // 2. 智能去重（如果启用）
    if addOpts.Infer {
        if shouldMerge, existingID := c.checkDuplicate(ctx, embedding, addOpts); shouldMerge {
            return c.mergeWithExisting(ctx, existingID, content)
        }
    }
    
    // 3. 插入存储
    memory := &Memory{
        ID:       generateSnowflakeID(),
        UserID:   addOpts.UserID,
        AgentID:  addOpts.AgentID,
        Content:  content,
        Embedding: embedding,
        Metadata: addOpts.Metadata,
    }
    
    if err := c.storage.Insert(ctx, memory); err != nil {
        return nil, err
    }
    
    return memory, nil
}
```

### 2.6 错误处理

定义清晰的错误类型：

```go
var (
    ErrNotFound        = errors.New("memory not found")
    ErrInvalidConfig   = errors.New("invalid configuration")
    ErrConnectionFailed = errors.New("connection failed")
    ErrEmbeddingFailed = errors.New("embedding generation failed")
    ErrDuplicateMemory = errors.New("duplicate memory detected")
)

type MemoryError struct {
    Op  string // 操作名称
    Err error  // 原始错误
}

func (e *MemoryError) Error() string {
    return fmt.Sprintf("powermem: %s: %v", e.Op, e.Err)
}
```

## 3. 示例代码

### 3.1 基础使用示例

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/oceanbase/powermem-go/pkg/powermem"
)

func main() {
    // 1. 加载配置
    config := &powermem.Config{
        LLM: powermem.LLMConfig{
            Provider: "qwen",
            APIKey:   "your-api-key",
            Model:    "qwen-plus",
        },
        Embedder: powermem.EmbedderConfig{
            Provider: "qwen",
            APIKey:   "your-api-key",
            Model:    "text-embedding-v4",
        },
        VectorStore: powermem.VectorStoreConfig{
            Provider: "oceanbase",
            Config: map[string]interface{}{
                "host":                 "127.0.0.1",
                "port":                 2881,
                "user":                 "root@sys",
                "password":             "password",
                "db_name":              "powermem",
                "collection_name":      "memories",
                "embedding_model_dims": 1536,
            },
        },
    }
    
    // 2. 创建客户端
    client, err := powermem.NewClient(config)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    ctx := context.Background()
    userID := "user123"
    
    // 3. 添加记忆
    memory, err := client.Add(ctx, "User likes Python programming",
        powermem.WithUserID(userID),
        powermem.WithMetadata(map[string]interface{}{
            "category": "preference",
            "importance": "high",
        }),
    )
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Added memory: %d\n", memory.ID)
    
    // 4. 搜索记忆
    results, err := client.Search(ctx, "user preferences",
        powermem.WithUserID(userID),
        powermem.WithLimit(5),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d memories:\n", len(results))
    for _, mem := range results {
        fmt.Printf("  - %s (score: %.3f)\n", mem.Content, mem.Score)
    }
    
    // 5. 更新记忆
    updated, err := client.Update(ctx, memory.ID, 
        "User loves Python programming and data science")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Updated memory: %s\n", updated.Content)
    
    // 6. 删除记忆
    if err := client.Delete(ctx, memory.ID); err != nil {
        log.Fatal(err)
    }
    fmt.Println("Memory deleted")
}
```

### 3.2 高级示例：智能记忆管理

```go
func advancedExample() {
    config := loadConfig()
    config.Intelligence = &powermem.IntelligenceConfig{
        Enabled:           true,
        DecayRate:         0.1,
        ReinforcementFactor: 0.3,
    }
    
    client, _ := powermem.NewClient(config)
    defer client.Close()
    
    ctx := context.Background()
    
    // 添加记忆（启用智能去重）
    memory, err := client.Add(ctx, "User likes Python",
        powermem.WithUserID("user123"),
        powermem.WithInfer(true), // 启用智能去重
    )
    
    // 尝试添加相似记忆（会自动合并或更新）
    memory2, err := client.Add(ctx, "User enjoys Python programming",
        powermem.WithUserID("user123"),
        powermem.WithInfer(true),
    )
    
    // memory2 可能与 memory 合并
    fmt.Printf("Memory1 ID: %d, Memory2 ID: %d\n", memory.ID, memory2.ID)
}
```

### 3.3 多代理场景示例

```go
func multiAgentExample() {
    client, _ := powermem.NewClient(loadConfig())
    defer client.Close()
    
    ctx := context.Background()
    
    // Agent1 添加私有记忆
    _, err := client.Add(ctx, "Agent1's private data",
        powermem.WithAgentID("agent1"),
        powermem.WithUserID("user123"),
        powermem.WithScope("private"), // 私有作用域
    )
    
    // Agent2 添加共享记忆
    _, err = client.Add(ctx, "Shared knowledge",
        powermem.WithAgentID("agent2"),
        powermem.WithUserID("user123"),
        powermem.WithScope("agent_group"), // 代理组共享
    )
    
    // Agent1 搜索（只能看到自己的私有记忆 + 共享记忆）
    results, _ := client.Search(ctx, "data",
        powermem.WithAgentID("agent1"),
        powermem.WithUserID("user123"),
    )
    
    fmt.Printf("Agent1 can see %d memories\n", len(results))
}
```

## 4. 总结

### 4.1 技术要点

1. **OceanBase 向量存储**：由于没有 Go 版本的 pyobvector，需要使用标准 MySQL 驱动 + 手动构造向量 SQL
2. **向量格式**：OceanBase 的 VECTOR 类型接受字符串格式 `"[0.1, 0.2, ...]"`
3. **向量索引**：支持 HNSW、IVF_FLAT、IVF_PQ 等索引类型，通过 SQL DDL 创建
4. **并发安全**：使用 `sync.RWMutex` 保护共享资源，所有操作支持 context
5. **错误处理**：遵循 Go 习惯，返回明确的错误类型

### 4.2 实现优先级

**第一阶段（MVP）**：

- [ ] 基础配置管理
- [ ] OceanBase 存储实现（CRUD + 向量搜索）
- [ ] OpenAI LLM/Embedder 集成
- [ ] 核心 Memory API（Add, Search, Update, Delete）

**第二阶段**：

- [ ] 智能去重功能
- [ ] 混合搜索（向量 + 全文）
- [ ] 更多 LLM/Embedder 提供商
- [ ] SQLite/PostgreSQL 存储支持

**第三阶段**：

- [ ] Ebbinghaus 遗忘曲线
- [ ] 多代理/多用户管理
- [ ] Reranker 支持
- [ ] 图存储支持

### 4.3 依赖库

```go
require (
    github.com/go-sql-driver/mysql v1.7.1  // MySQL/OceanBase 驱动
    github.com/joho/godotenv v1.5.1        // .env 文件支持
    github.com/sashabaranov/go-openai v1.17.9  // OpenAI SDK
    github.com/bwmarrin/snowflake v0.3.0   // Snowflake ID 生成
    github.com/stretchr/testify v1.8.4     // 测试框架
)
```
