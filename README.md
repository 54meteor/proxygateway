# AI Gateway 大模型流量中继平台

## 项目概述

AI Gateway 是一个用于代理 LLM 请求、追踪 Token 使用量、支持多模型计费的网关服务。

## 核心功能

| 功能 | 说明 |
|------|------|
| **LLM 代理转发** | 支持 MiniMax、OpenAI、Anthropic 多种模型 |
| **API Key 鉴权** | 基于 Bearer Token 的认证机制 |
| **Token 计费** | 自动计算费用并扣除用户余额 |
| **用量统计** | 记录每次请求的 Token 消耗 |
| **请求日志** | 完整记录请求/响应 JSON 到独立日志文件 |
| **管理后台** | Web 界面管理用户、API Keys、查看用量统计 |
| **健康检查** | 检查数据库和上游服务状态 |

## 技术栈

- **语言**: Go 1.18+
- **框架**: Gin
- **数据库**: SQLite (glebarez/sqlite - 纯 Go，无 CGO)
- **前端**: Bootstrap 5 + HTML 模板

## 项目结构

```
ai-gateway/
├── cmd/server/main.go      # 入口文件
├── config.yaml             # 配置文件
├── go.mod
├── go.sum
├── internal/
│   ├── config/             # 配置加载
│   ├── handler/            # HTTP 处理器
│   ├── middleware/         # 中间件 (API Key 鉴权、CORS)
│   ├── model/             # 数据模型
│   ├── router/            # 路由配置
│   ├── logger/             # 日志模块
│   ├── service/           # 业务逻辑
│   │   ├── adapter/       # LLM 适配器 (MiniMax/OpenAI/Anthropic)
│   │   ├── auth/         # 鉴权服务
│   │   └── billing/      # 计费服务
│   ├── storage/          # 数据库操作
│   └── admin/            # 管理后台处理器
├── templates/admin/        # 管理后台 HTML 模板
└── tests/
    └── test_gateway.py    # Python 测试脚本
```

## 快速开始

### 1. 编译

```bash
# Linux/WSL
export PATH=/usr/local/go/bin:$PATH
go build -o server cmd/server/main.go

# Windows
go build -o server.exe cmd\server\main.go
```

### 2. 配置

```bash
# 设置环境变量
export MINIMAX_API_KEY="your-minimax-api-key"
```

编辑 `config.yaml`:

```yaml
server:
  host: "0.0.0.0"
  port: 8080

database:
  path: "./ai_gateway.db"

models:
  minimax:
    enabled: true
    base_url: "https://api.minimaxi.com/v1"
    api_key: "${MINIMAX_API_KEY}"  # 从环境变量读取
    timeout: 120
```

### 3. 运行

```bash
# Linux/WSL
./server

# Windows
server.exe
```

### 4. 访问

- 管理后台: http://localhost:8080/admin/
- 健康检查: http://localhost:8080/health

## API 接口

### 公开接口

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /health | 健康检查 |
| GET | /v1/models | 可用模型列表 |

### 需要认证

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /v1/chat/completions | 聊天完成 |
| GET | /v1/usage | 用量查询 |
| POST | /v1/keys | 创建 API Key |

### 管理后台

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /admin/ | 仪表盘 |
| GET | /admin/users | 用户管理 |
| GET | /admin/keys | API Keys 管理 |
| GET | /admin/usage | 用量统计 |

## 使用示例

### 调用聊天接口

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer YOUR-API-KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MiniMax-M2.5",
    "messages": [
      {"role": "user", "content": "你好"}
    ],
    "max_tokens": 100
  }'
```

### 响应格式

```json
{
  "id": "chatcmpl-xxx",
  "object": "chat.completion",
  "created": 1234567890,
  "model": "MiniMax-M2.5",
  "choices": [{
    "message": {
      "role": "assistant",
      "content": "你好！有什么可以帮你的吗？"
    }
  }],
  "usage": {
    "prompt_tokens": 42,
    "completion_tokens": 30,
    "total_tokens": 72
  }
}
```

## 计费说明

### 价格配置 (config.yaml)

```yaml
pricing:
  MiniMax-M2.5:
    prompt: 0.01      # $/1K tokens
    completion: 0.01
```

### 日志记录

- **数据库**: `token_usage` 表存储每次请求的 Token 数量和费用
- **文件日志**: `logs/chat_YYYY-MM-DD.log` 存储完整的请求/响应 JSON

## 管理后台

访问 http://localhost:8080/admin/ 查看：

- **仪表盘**: 总用户数、API Keys 数量、今日请求数、今日消费
- **用户管理**: 查看用户、充值余额
- **API Keys**: 查看、启用/禁用、删除 Keys
- **用量统计**: 按日期筛选、查看明细

## 注意事项

1. **API Key 安全**: Key 存储为 SHA256 哈希值，不保存明文
2. **环境变量**: 生产环境建议使用环境变量配置敏感信息
3. **日志文件**: 详细日志存储在 `logs/` 目录，按日期分割
