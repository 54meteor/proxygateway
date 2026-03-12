package config

import (
	"os"

	"ai-gateway/internal/logger"

	"gopkg.in/yaml.v3"
)

// Config 全局配置
type Config struct {
	Server    ServerConfig    // 服务器配置
	Database  DatabaseConfig  // 数据库配置
	Models    ModelsConfig    // 模型提供商配置
	Pricing   PricingConfig   // 计费价格配置
	RateLimit RateLimitConfig // 限流配置
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host string `yaml:"host"` // 监听地址
	Port int    `yaml:"port"` // 监听端口
	Mode string `yaml:"mode"` // 运行模式：debug/release
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver   string `yaml:"driver"`   // 驱动类型：sqlite3/postgres
	Host     string `yaml:"host"`     // 数据库主机地址
	Port     int    `yaml:"port"`     // 数据库端口
	User     string `yaml:"user"`     // 数据库用户名
	Password string `yaml:"password"` // 数据库密码
	DBName   string `yaml:"dbname"`  // 数据库名称
	Path     string `yaml:"path"`    // SQLite 文件路径
}

// ModelsConfig 模型提供商配置
type ModelsConfig struct {
	OpenAI    ModelProviderConfig // OpenAI 配置
	Anthropic ModelProviderConfig // Anthropic 配置
	MiniMax   ModelProviderConfig // MiniMax 配置
}

// ModelProviderConfig 模型提供商配置项
type ModelProviderConfig struct {
	Enabled  bool   `yaml:"enabled"`  // 是否启用
	BaseURL  string `yaml:"base_url"` // API 基础地址
	APIKey   string `yaml:"api_key"`  // API Key
	Timeout  int    `yaml:"timeout"`  // 超时时间（秒）
}

// PricingConfig 计费配置，按模型名称映射
type PricingConfig map[string]PricingItem

// PricingItem 单价配置
type PricingItem struct {
	Prompt      float64 // Prompt Token 单价（$/1K tokens）
	Completion  float64 // Completion Token 单价（$/1K tokens）
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Enabled           bool `yaml:"enabled"`            // 是否启用限流
	RequestsPerMinute int  `yaml:"requests_per_minute"` // 每分钟最大请求数
	TokensPerMinute   int  `yaml:"tokens_per_minute"`   // 每分钟最大 Token 数
}

// Load 加载配置文件
// 1. 读取 YAML 文件
// 2. 解析到 Config 结构体
// 3. 从环境变量覆盖 API Key（支持 ${ENV_VAR} 格式）
func Load(path string) (*Config, error) {
	// 读取配置文件
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// 解析 YAML
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// 处理环境变量占位符 ${VAR}
	cfg.Models.OpenAI.APIKey = resolveEnvVar(cfg.Models.OpenAI.APIKey)
	cfg.Models.Anthropic.APIKey = resolveEnvVar(cfg.Models.Anthropic.APIKey)
	cfg.Models.MiniMax.APIKey = resolveEnvVar(cfg.Models.MiniMax.APIKey)

	// 从环境变量加载 API Key（环境变量优先，覆盖配置文件）
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		cfg.Models.OpenAI.APIKey = apiKey
	}
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		cfg.Models.Anthropic.APIKey = apiKey
	}
	if apiKey := os.Getenv("MINIMAX_API_KEY"); apiKey != "" {
		cfg.Models.MiniMax.APIKey = apiKey
		logger.Info("MiniMax API Key loaded from env")
	} else if cfg.Models.MiniMax.APIKey != "" {
		logger.Info("MiniMax API Key loaded from config")
	} else {
		logger.Warn("MiniMax API Key not configured")
	}

	return &cfg, nil
}

// resolveEnvVar 解析环境变量占位符 ${VAR}
// 如果值以 ${ 开头且以 } 结尾，则读取对应的环境变量
func resolveEnvVar(value string) string {
	if len(value) > 3 && value[:2] == "${" && value[len(value)-1] == '}' {
		envVar := value[2 : len(value)-1]
		if envVal := os.Getenv(envVar); envVal != "" {
			return envVal
		}
	}
	return value
}
