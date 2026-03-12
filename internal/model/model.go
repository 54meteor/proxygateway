package model

import (
	"github.com/google/uuid"
)

// User 用户信息
type User struct {
	ID        uuid.UUID // 用户唯一ID
	Email     string    // 用户邮箱
	Balance   float64   // 账户余额
	CreatedAt string    // 创建时间
	UpdatedAt string    // 更新时间
}

// APIKey API Key 信息
type APIKey struct {
	ID          uuid.UUID // Key 唯一ID
	UserID      uuid.UUID // 所属用户ID
	KeyHash     string    // Key 的 SHA256 哈希值
	Name        string    // Key 名称
	RateLimit   int       // 速率限制（每分钟请求数）
	MonthlyQuota int64    // 月度配额（0 表示无限制）
	IsActive    bool      // 是否激活
	CreatedAt   string    // 创建时间
	UpdatedAt   string    // 更新时间
}

// TokenUsage Token 使用记录
type TokenUsage struct {
	ID              int64     // 记录ID
	UserID          uuid.UUID // 用户ID
	APIKeyID        uuid.UUID // API Key ID
	Model           string    // 使用的模型名称
	PromptTokens    int       // Prompt 消耗的 Token 数
	CompletionTokens int      // Completion 消耗的 Token 数
	Cost            float64   // 费用
	CreatedAt       string    // 创建时间
}

// DailyUsage 每日用量汇总
type DailyUsage struct {
	UserID            uuid.UUID // 用户ID
	Model             string    // 模型名称
	Date              string    // 日期
	PromptTokens      int64     // Prompt Token 总数
	CompletionTokens  int64     // Completion Token 总数
	TotalCost         float64   // 总费用
}

// APIKeyUsageStats API Key 使用量统计
type APIKeyUsageStats struct {
	APIKeyID          uuid.UUID // API Key ID
	TotalRequests     int64     // 总请求数
	TotalPromptTokens int64     // 总 Prompt Token 数
	TotalCompletionTokens int64 // 总 Completion Token 数
	TotalCost         float64   // 总费用
}

// ChatRequest 聊天请求（兼容 OpenAI）
type ChatRequest struct {
	Model       string                  // 模型名称
	Messages    []ChatMessage           // 消息列表
	Temperature float64                 // 温度参数
	MaxTokens   int                     // 最大 Token 数
	Stream      bool                    // 是否流式输出
}

// ChatMessage 聊天消息
type ChatMessage struct {
	Role    string // 角色：user/assistant/system
	Content string // 消息内容
	Name    string // 发言者名称（可选）
}

// ChatResponse 聊天响应
type ChatResponse struct {
	ID      string   // 响应ID
	Object  string   // 对象类型
	Created int64    // 创建时间戳
	Model   string   // 模型名称
	Choices []Choice // 回复选项
	Usage   Usage    // Token 使用量
}

// Choice 回复选项
type Choice struct {
	Index        int         // 选项索引
	Message      ChatMessage // 消息内容
	FinishReason string      // 结束原因
}

// Usage Token 使用量
type Usage struct {
	PromptTokens     int // Prompt Token 数
	CompletionTokens int // Completion Token 数
	TotalTokens      int // 总 Token 数
}
