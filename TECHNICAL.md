# AI Gateway 技术方案

## 需求分析

构建一个大模型流量中继平台（AI Gateway），实现：
1. 代理 LLM 请求
2. 追踪 Token 使用量
3. 支持多模型计费

## 架构设计

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   客户端    │────>│ AI Gateway  │────>│  LLM 服务   │
│  (用户)     │     │   (Go)      │     │ (MiniMax)   │
└─────────────┘     └─────────────┘     └─────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │  SQLite    │
                    │ (用量记录)  │
                    └─────────────┘
```

## 核心模块

### 1. 适配器模式 (Adapter Pattern)

支持多 LLM 提供商：

```go
// LLMAdapter 接口
type LLMAdapter interface {
    ChatComplete(req ChatRequest) (*ChatResponse, error)
    CountTokens(model, text string) (int, error)
    GetModelName() string
}
```

实现：
- `MiniMaxAdapter` - MiniMax 模型
- `OpenAIAdapter` - OpenAI 模型 (预留)
- `AnthropicAdapter` - Anthropic 模型 (预留)

### 2. API Key 鉴权

- Key 生成：UUID v4
- Key 存储：SHA256 哈希
- 验证：Bearer Token 方式

### 3. Token 计费

- 实时计算 Token 数量
- 按模型单价计费
- 每日/每月用量统计

### 4. 数据模型

```go
// 用户
type User struct {
    ID        uuid.UUID
    Email     string
    Balance   float64
    CreatedAt time.Time
}

// API Key
type APIKey struct {
    ID           uuid.UUID
    UserID       uuid.UUID
    KeyHash      string
    RateLimit    int
    IsActive     bool
}

// Token 使用记录
type TokenUsage struct {
    ID              int64
    UserID          uuid.UUID
    APIKeyID        uuid.UUID
    Model           string
    PromptTokens    int
    CompletionTokens int
    Cost            float64
    CreatedAt       time.Time
}
```

## 技术选型

| 组件 | 选型 | 原因 |
|------|------|------|
| 语言 | Go | 高性能、并发好 |
| 框架 | Gin | 轻量、简单 |
| 数据库 | SQLite | 轻量、易部署 |
| SQL驱动 | glebarez/sqlite | 纯 Go、无 CGO |

## 关键实现

### MiniMax API 对接

使用原生 HTTP Client 而非 go-openai 库：

```go
func (a *MiniMaxAdapter) ChatComplete(req model.ChatRequest) (*model.ChatResponse, error) {
    // 构建请求
    url := a.cfg.Models.MiniMax.BaseURL + "/chat/completions"
    
    httpReq, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
    httpReq.Header.Set("Authorization", "Bearer "+a.cfg.Models.MiniMax.APIKey)
    httpReq.Header.Set("Content-Type", "application/json")
    
    // 发送请求
    client := &http.Client{}
    resp, _ := client.Do(httpReq)
    // ...
}
```

### API Key 验证流程

```
1. 客户端发送请求: Authorization: Bearer <API-KEY>
2. 中间件提取 KEY
3. 计算 KEY 的 SHA256 哈希
4. 查询数据库匹配哈希
5. 返回用户ID或 401 错误
```

## 测试方案

- 单元测试：核心逻辑
- 集成测试：API 端到端
- Python 测试脚本：自动化回归

## 部署建议

1. 开发环境：本地运行
2. 生产环境：Docker 容器
3. 监控：日志 + 指标
