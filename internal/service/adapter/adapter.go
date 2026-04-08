// Package adapter LLM 适配器包
// 提供多种大模型提供商的适配器实现
package adapter

import (
	"ai-gateway/internal/config"
	"ai-gateway/internal/model"
)

// LLMAdapter LLM 适配器接口
// 定义大模型交互的抽象方法
type LLMAdapter interface {
	// ChatComplete 聊天完成
	// 参数: req 聊天请求
	// 返回: *model.ChatResponse 聊天响应
	ChatComplete(req model.ChatRequest) (*model.ChatResponse, error)
	// Embeddings 文本向量化
	// 参数: req Embeddings 请求
	// 返回: *model.EmbeddingResponse Embeddings 响应
	Embeddings(req model.EmbeddingRequest) (*model.EmbeddingResponse, error)
	// Images 图片生成
	// 参数: req 图片生成请求
	// 返回: *model.ImageResponse 图片生成响应
	Images(req model.ImageRequest) (*model.ImageResponse, error)
	// CountTokens 计算 Token 数量
	// 参数: model 模型名称, text 文本内容
	// 返回: token 数量
	CountTokens(model, text string) (int, error)
	// GetModelName 获取模型名称
	GetModelName() string
}

// Factory 适配器工厂
// 根据配置创建和管理不同的 LLM 适配器
type Factory struct {
	adapters map[string]LLMAdapter // 模型名称到适配器的映射
	cfg      *config.Config         // 全局配置
}

// NewFactory 创建适配器工厂
// 根据配置初始化已启用的模型适配器
func NewFactory(cfg *config.Config) *Factory {
	f := &Factory{
		adapters: make(map[string]LLMAdapter),
		cfg:      cfg,
	}

	// 注册 OpenAI 模型
	if cfg.Models.OpenAI.Enabled {
		f.adapters["gpt-4"] = NewOpenAIAdapter(cfg)
		f.adapters["gpt-4o"] = NewOpenAIAdapter(cfg)
		f.adapters["gpt-3.5-turbo"] = NewOpenAIAdapter(cfg)
	}

	// 注册 Anthropic 模型
	if cfg.Models.Anthropic.Enabled {
		f.adapters["claude-3-opus"] = NewAnthropicAdapter(cfg)
		f.adapters["claude-3-sonnet"] = NewAnthropicAdapter(cfg)
	}

	// 注册 MiniMax 模型（支持多个别名）
	if cfg.Models.MiniMax.Enabled {
		adapter := NewMiniMaxAdapter(cfg)
		f.adapters["MiniMax-M2.5"] = adapter
		f.adapters["abab6.5s-chat"] = adapter
		f.adapters["abab6.5g-chat"] = adapter
		f.adapters["MiniMax-Text-01"] = adapter
	}

	return f
}

// Get 获取适配器
// 参数: model 模型名称
// 返回: (LLMAdapter, bool) 适配器实例和是否找到
func (f *Factory) Get(model string) (LLMAdapter, bool) {
	adapter, ok := f.adapters[model]
	return adapter, ok
}

// ListModels 列出所有支持的模型
// 返回: []string 模型名称列表
func (f *Factory) ListModels() []string {
	models := make([]string, 0, len(f.adapters))
	for m := range f.adapters {
		models = append(models, m)
	}
	return models
}
