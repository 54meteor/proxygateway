# AI Gateway 大模型流量中继平台

## 项目概述

AI Gateway 是一个用于代理 LLM 请求、追踪 Token 使用量、支持多模型计费的网关服务。

## 技术栈

- **语言**: Go 1.25+
- **框架**: Gin
- **数据库**: SQLite (glebarez/sqlite - 纯 Go，无 CGO)
- **模型**: MiniMax-M2.5

## 项目结构

```
ai-gateway/
├── cmd/server/main.go      # 入口文件
├── config.yaml             # 配置文件
├── ai_gateway.db           # SQLite 数据库
├── go.mod
├── go.sum
├── internal/
│   ├── config/             # 配置加载
│   ├── handler/            # HTTP 处理器
│   ├── middleware/        # 中间件 (API Key 鉴权)
│   ├── model/             # 数据模型
│   ├── router/            # 路由
│   ├── service/           # 业务逻辑
│   │   ├── adapter/       # LLM 适配器
│   │   ├── auth/          # 鉴权服务
│   │   └── billing/       # 计费服务
│   └── storage/           # 数据库操作
└── tests/
    └── test_gateway.py    # Python 测试脚本
```

## 快速开始

### 1. 编译

```bash
# WSL
export PATH=/usr/local/go/bin:$PATH
go build -o server cmd/server/main.go

# Windows
go build -o server.exe cmd\server\main.go
```

### 2. 配置

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
    api_key: "your-api-key-here"
    timeout: 120
```

### 3. 运行

```bash
# WSL
./server

# Windows
server.exe
```

### 4. 测试

```bash
# 初始化数据库后访问
curl http://localhost:8080/health
curl http://localhost:8080/v1/models
```

## API 接口

| 方法 | 路径 | 描述 | 认证 |
|------|------|------|------|
| GET | /health | 健康检查 | 否 |
| GET | /v1/models | 模型列表 | 否 |
| POST | /v1/chat/completions | 聊天完成 | API Key |
| GET | /v1/usage | 用量查询 | API Key |
| POST | /v1/keys | 创建 API Key | API Key |

## 使用示例

### 创建 API Key

```bash
curl -X POST http://localhost:8080/debug/init
```

### 调用聊天接口

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer YOUR-API-KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MiniMax-M2.5",
    "messages": [{"role": "user", "content": "你好"}]
  }'
```

## 计费说明

- Token 计费规则在 `config.yaml` 中配置
- 支持按模型设置不同的单价
- 用量记录保存在 SQLite 数据库中

## 注意事项

1. MiniMax API 使用自定义 HTTP Client，不依赖 go-openai
2. Windows 编译较快 (Go 1.25)，WSL 需要配置 Go 代理
3. 生产环境建议设置 `GIN_MODE=release`
