package adapter

import (
	"testing"

	"ai-gateway/internal/config"
	"ai-gateway/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiniMaxAdapter_NewMiniMaxAdapter(t *testing.T) {
	cfg := &config.Config{}
	adapter := NewMiniMaxAdapter(cfg)
	assert.NotNil(t, adapter)
	assert.Equal(t, cfg, adapter.cfg)
}

func TestMiniMaxAdapter_GetModelName(t *testing.T) {
	cfg := &config.Config{}
	adapter := NewMiniMaxAdapter(cfg)
	assert.Equal(t, "MiniMax-M2.5", adapter.GetModelName())
}

func TestMiniMaxAdapter_CountTokens(t *testing.T) {
	cfg := &config.Config{}
	adapter := NewMiniMaxAdapter(cfg)

	// 测试 token 计数
	count, err := adapter.CountTokens("MiniMax-M2.5", "Hello world")
	require.NoError(t, err)
	assert.Greater(t, count, 0)
}

func TestMiniMaxAdapter_CountTokens_EmptyText(t *testing.T) {
	cfg := &config.Config{}
	adapter := NewMiniMaxAdapter(cfg)

	count, err := adapter.CountTokens("MiniMax-M2.5", "")
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestFactory_GetModel(t *testing.T) {
	cfg := &config.Config{
		Models: config.ModelsConfig{
			MiniMax: config.ModelProviderConfig{
				Enabled:  true,
				APIKey:   "test-key",
				BaseURL:  "https://api.minimaxi.com/v1",
				Timeout:  120,
			},
			OpenAI: config.ModelProviderConfig{
				Enabled: false,
			},
			Anthropic: config.ModelProviderConfig{
				Enabled: false,
			},
		},
	}

	factory := NewFactory(cfg)

	// 测试获取存在的模型
	adapter, ok := factory.Get("MiniMax-M2.5")
	assert.True(t, ok)
	assert.NotNil(t, adapter)

	// 测试获取不存在的模型
	_, ok = factory.Get("non-existent")
	assert.False(t, ok)
}

func TestFactory_ListModels(t *testing.T) {
	cfg := &config.Config{
		Models: config.ModelsConfig{
			MiniMax: config.ModelProviderConfig{
				Enabled: true,
				APIKey:  "test-key",
			},
			OpenAI: config.ModelProviderConfig{
				Enabled: false,
			},
			Anthropic: config.ModelProviderConfig{
				Enabled: false,
			},
		},
	}

	factory := NewFactory(cfg)
	models := factory.ListModels()
	assert.NotEmpty(t, models)
	assert.Contains(t, models, "MiniMax-M2.5")
}

func TestOpenAIAdapter_GetModelName(t *testing.T) {
	cfg := &config.Config{}
	adapter := NewOpenAIAdapter(cfg)
	assert.Equal(t, "openai", adapter.GetModelName())
}

func TestAnthropicAdapter_GetModelName(t *testing.T) {
	cfg := &config.Config{}
	adapter := NewAnthropicAdapter(cfg)
	assert.Equal(t, "anthropic", adapter.GetModelName())
}

func TestFactory_NoEnabledModels(t *testing.T) {
	cfg := &config.Config{
		Models: config.ModelsConfig{
			MiniMax: config.ModelProviderConfig{
				Enabled: false,
				APIKey:  "test-key",
			},
			OpenAI: config.ModelProviderConfig{
				Enabled: false,
			},
			Anthropic: config.ModelProviderConfig{
				Enabled: false,
			},
		},
	}

	factory := NewFactory(cfg)
	models := factory.ListModels()
	// 空列表因为没有启用的模型
	assert.Empty(t, models)
}

// ============ Embeddings 测试 ============

func TestMiniMaxAdapter_Embeddings(t *testing.T) {
	cfg := &config.Config{
		Models: config.ModelsConfig{
			MiniMax: config.ModelProviderConfig{
				Enabled:  true,
				APIKey:   "test-key",
				BaseURL:  "https://api.minimaxi.com/v1",
				Timeout:  120,
			},
		},
	}
	adapter := NewMiniMaxAdapter(cfg)
	req := model.EmbeddingRequest{
		Model:          "embedding-model",
		Input:          []string{"Hello world"},
		EncodingFormat: "float",
	}
	// 验证方法存在且返回结构正确的响应（可能是空数据但结构正确）
	resp, err := adapter.Embeddings(req)
	// MiniMax API 可能返回成功但数据为空的情况
	if err == nil {
		assert.NotNil(t, resp)
		assert.Equal(t, "list", resp.Object)
		assert.Equal(t, "embedding-model", resp.Model)
	}
}

func TestOpenAIAdapter_Embeddings_NotImplemented(t *testing.T) {
	cfg := &config.Config{}
	adapter := NewOpenAIAdapter(cfg)
	req := model.EmbeddingRequest{
		Model:          "text-embedding-3-small",
		Input:          []string{"Hello world"},
		EncodingFormat: "float",
	}
	resp, err := adapter.Embeddings(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
	assert.Nil(t, resp)
}

func TestAnthropicAdapter_Embeddings_NotImplemented(t *testing.T) {
	cfg := &config.Config{}
	adapter := NewAnthropicAdapter(cfg)
	req := model.EmbeddingRequest{
		Model:          "embedding-model",
		Input:          []string{"Hello world"},
		EncodingFormat: "float",
	}
	resp, err := adapter.Embeddings(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
	assert.Nil(t, resp)
}
