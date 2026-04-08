// Package adapter LLM 适配器包
package adapter

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"ai-gateway/internal/config"
	"ai-gateway/internal/logger"
	"ai-gateway/internal/model"
	"ai-gateway/internal/storage"
)

// DynamicFactory 动态适配器工厂
// 从数据库加载模型配置，动态注册适配器，支持热更新
type DynamicFactory struct {
	*Factory                    // 组合现有 Factory
	configLoader *ConfigLoader  // 配置加载器
	adapters    map[string]LLMAdapter // 模型名称 -> 适配器（动态注册）
	mu          sync.RWMutex
}

// ConfigLoader 配置加载器
// 从数据库动态加载模型配置，支持热更新
type ConfigLoader struct {
	db          *storage.DB
	yamlConfig  *config.Config // YAML 静态配置（fallback）
	models      map[string]*model.AIModel
	mu          sync.RWMutex
	onUpdate    []func([]*model.AIModel)
}

// NewConfigLoader 创建配置加载器
func NewConfigLoader(db *storage.DB, yamlConfig *config.Config) *ConfigLoader {
	cl := &ConfigLoader{
		db:         db,
		yamlConfig: yamlConfig,
		models:     make(map[string]*model.AIModel),
	}
	cl.Reload()
	return cl
}

// Reload 从数据库重新加载配置
func (cl *ConfigLoader) Reload() {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	dbModels, err := cl.db.ListAIModels()
	if err != nil {
		logger.Warn("Failed to load models from DB: %v, using YAML config", err)
		cl.loadFromYAML()
		return
	}

	if len(dbModels) == 0 {
		logger.Info("No models in DB, using YAML config as fallback")
		cl.loadFromYAML()
		return
	}

	cl.models = make(map[string]*model.AIModel)
	for i := range dbModels {
		m := &dbModels[i]
		cl.models[m.ID] = m
		cl.parseModelsField(m)
	}

	logger.Info("Loaded %d models from database", len(cl.models))
	cl.notifyUpdate()
}

// loadFromYAML 从 YAML 配置加载（fallback）
func (cl *ConfigLoader) loadFromYAML() {
	if cl.yamlConfig == nil {
		return
	}

	cl.models = make(map[string]*model.AIModel)

	// OpenAI
	if cl.yamlConfig.Models.OpenAI.Enabled {
		m := &model.AIModel{
			ID:       "openai",
			Name:     "OpenAI",
			Provider: "openai",
			BaseURL:  cl.yamlConfig.Models.OpenAI.BaseURL,
			APIKey:   cl.yamlConfig.Models.OpenAI.APIKey,
			Enabled:  true,
			Models:   []string{"gpt-4", "gpt-4o", "gpt-3.5-turbo"},
		}
		cl.models["openai"] = m
	}

	// Anthropic
	if cl.yamlConfig.Models.Anthropic.Enabled {
		m := &model.AIModel{
			ID:       "anthropic",
			Name:     "Anthropic",
			Provider: "anthropic",
			BaseURL:  cl.yamlConfig.Models.Anthropic.BaseURL,
			APIKey:   cl.yamlConfig.Models.Anthropic.APIKey,
			Enabled:  true,
			Models:   []string{"claude-3-opus", "claude-3-sonnet"},
		}
		cl.models["anthropic"] = m
	}

	// MiniMax
	if cl.yamlConfig.Models.MiniMax.Enabled {
		m := &model.AIModel{
			ID:       "minimax",
			Name:     "MiniMax",
			Provider: "minimax",
			BaseURL:  cl.yamlConfig.Models.MiniMax.BaseURL,
			APIKey:   cl.yamlConfig.Models.MiniMax.APIKey,
			Enabled:  true,
			Models:   []string{"MiniMax-M2.5", "abab6.5s-chat", "abab6.5g-chat", "MiniMax-Text-01"},
		}
		cl.models["minimax"] = m
	}

	logger.Info("Loaded %d models from YAML config (fallback)", len(cl.models))
}

// GetModels 获取所有模型配置
func (cl *ConfigLoader) GetModels() []*model.AIModel {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	result := make([]*model.AIModel, 0, len(cl.models))
	for _, m := range cl.models {
		result = append(result, m)
	}
	return result
}

// GetModelByID 根据 ID 获取模型
func (cl *ConfigLoader) GetModelByID(id string) *model.AIModel {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	return cl.models[id]
}

// GetModelByProvider 根据提供商获取模型
func (cl *ConfigLoader) GetModelByProvider(provider string) *model.AIModel {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	for _, m := range cl.models {
		if m.Provider == provider && m.Enabled {
			return m
		}
	}
	return nil
}

// IsEnabled 检查模型是否启用
func (cl *ConfigLoader) IsEnabled(modelName string) bool {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	for _, m := range cl.models {
		if m.Enabled {
			for _, mn := range m.Models {
				if mn == modelName {
					return true
				}
			}
		}
	}
	return false
}

// GetAPIKey 获取模型的 API Key
func (cl *ConfigLoader) GetAPIKey(modelName string) string {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	for _, m := range cl.models {
		if m.Enabled {
			for _, mn := range m.Models {
				if mn == modelName {
					return m.APIKey
				}
			}
		}
	}
	return ""
}

// GetBaseURL 获取模型的 BaseURL
func (cl *ConfigLoader) GetBaseURL(modelName string) string {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	for _, m := range cl.models {
		if m.Enabled {
			for _, mn := range m.Models {
				if mn == modelName {
					return m.BaseURL
				}
			}
		}
	}
	return ""
}

// GetProvider 获取模型对应的提供商
func (cl *ConfigLoader) GetProvider(modelName string) string {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	for _, m := range cl.models {
		if m.Enabled {
			for _, mn := range m.Models {
				if mn == modelName {
					return m.Provider
				}
			}
		}
	}
	return ""
}

// OnUpdate 注册配置变更回调
func (cl *ConfigLoader) OnUpdate(callback func([]*model.AIModel)) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	cl.onUpdate = append(cl.onUpdate, callback)
}

// notifyUpdate 通知配置变更
func (cl *ConfigLoader) notifyUpdate() {
	models := cl.GetModels()
	for _, cb := range cl.onUpdate {
		go cb(models)
	}
}

// parseModelsField 解析 models 字段
func (cl *ConfigLoader) parseModelsField(m *model.AIModel) {
	modelsStr := fmt.Sprintf("%v", m.Models)
	if modelsStr == "[]" || modelsStr == "<nil>" {
		m.Models = []string{}
		return
	}

	// 尝试 JSON 解析
	var modelList []string
	if err := json.Unmarshal([]byte(modelsStr), &modelList); err == nil {
		m.Models = modelList
		return
	}

	// 尝试 Go 格式解析: [abab6-chat abab5-chat]
	modelsStr = strings.TrimPrefix(modelsStr, "[")
	modelsStr = strings.TrimSuffix(modelsStr, "]")
	parts := strings.Fields(modelsStr)
	if len(parts) > 0 {
		m.Models = parts
	}
}

// providerConfig 提供商配置（用于创建适配器）
type providerConfig struct {
	BaseURL string
	APIKey  string
	Timeout int
}

// NewDynamicFactory 创建动态适配器工厂
// 从数据库加载配置，支持热更新
func NewDynamicFactory(db *storage.DB, yamlConfig *config.Config) *DynamicFactory {
	df := &DynamicFactory{
		Factory:     NewFactory(yamlConfig), // 用 YAML 配置初始化基础 Factory
		adapters:    make(map[string]LLMAdapter),
		configLoader: NewConfigLoader(db, yamlConfig),
	}

	// 初始注册
	df.registerFromDB()

	// 注册热更新回调
	df.configLoader.OnUpdate(func(models []*model.AIModel) {
		logger.Info("Hot update triggered, reloading adapters...")
		df.registerFromDB()
	})

	// 启动定期轮询检查（每 30 秒）
	go df.startPolling()

	return df
}

// registerFromDB 从数据库配置注册适配器
func (df *DynamicFactory) registerFromDB() {
	df.mu.Lock()
	defer df.mu.Unlock()

	// 清空现有动态注册的适配器（保留 YAML 配置的）
	df.adapters = make(map[string]LLMAdapter)

	models := df.configLoader.GetModels()
	for _, m := range models {
		if !m.Enabled {
			continue
		}

		// 根据提供商创建适配器
		adapter := df.createAdapterForProvider(m.Provider, m.BaseURL, m.APIKey)
		if adapter == nil {
			logger.Warn("No adapter for provider: %s", m.Provider)
			continue
		}

		// 注册该 provider 下的所有模型
		for _, modelName := range m.Models {
			df.adapters[modelName] = adapter
			logger.Debug("Registered adapter for model: %s (provider: %s)", modelName, m.Provider)
		}
	}

	logger.Info("Dynamic factory registered %d adapters", len(df.adapters))
}

// createAdapterForProvider 根据提供商创建适配器
func (df *DynamicFactory) createAdapterForProvider(provider, baseURL, apiKey string) LLMAdapter {
	// 构建临时 config.Config（仅包含当前 provider）
	cfg := &config.Config{
		Models: config.ModelsConfig{
			OpenAI:    config.ModelProviderConfig{Enabled: false},
			Anthropic: config.ModelProviderConfig{Enabled: false},
			MiniMax:   config.ModelProviderConfig{Enabled: false},
		},
	}

	switch provider {
	case "openai":
		cfg.Models.OpenAI = config.ModelProviderConfig{
			Enabled: true,
			BaseURL: baseURL,
			APIKey:  apiKey,
			Timeout: 120,
		}
		return NewOpenAIAdapter(cfg)
	case "anthropic":
		cfg.Models.Anthropic = config.ModelProviderConfig{
			Enabled: true,
			BaseURL: baseURL,
			APIKey:  apiKey,
			Timeout: 120,
		}
		return NewAnthropicAdapter(cfg)
	case "minimax":
		cfg.Models.MiniMax = config.ModelProviderConfig{
			Enabled: true,
			BaseURL: baseURL,
			APIKey:  apiKey,
			Timeout: 120,
		}
		return NewMiniMaxAdapter(cfg)
	default:
		logger.Warn("Unknown provider: %s", provider)
		return nil
	}
}

// startPolling 启动定期轮询检查配置变更
func (df *DynamicFactory) startPolling() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// 重新加载配置（内部会检查是否真的变更）
		df.configLoader.Reload()
	}
}

// Get 获取适配器（优先从动态注册获取，否则回退到 Factory）
func (df *DynamicFactory) Get(model string) (LLMAdapter, bool) {
	df.mu.RLock()
	defer df.mu.RUnlock()

	// 1. 优先从动态注册的适配器获取
	if adapter, ok := df.adapters[model]; ok {
		return adapter, true
	}

	// 2. 回退到 Factory（YAML 配置的适配器）
	return df.Factory.Get(model)
}

// ListModels 列出所有支持的模型
func (df *DynamicFactory) ListModels() []string {
	df.mu.RLock()
	defer df.mu.RUnlock()

	models := make(map[string]bool)
	for m := range df.adapters {
		models[m] = true
	}
	for _, m := range df.Factory.ListModels() {
		models[m] = true
	}

	result := make([]string, 0, len(models))
	for m := range models {
		result = append(result, m)
	}
	return result
}

// Reload 手动触发重新加载
func (df *DynamicFactory) Reload() {
	df.configLoader.Reload()
}

// GetConfigLoader 获取配置加载器
func (df *DynamicFactory) GetConfigLoader() *ConfigLoader {
	return df.configLoader
}
