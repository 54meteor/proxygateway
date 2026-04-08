// Package storage 数据库操作包
// 提供用户、API Key、Token 使用记录等数据持久化功能
package storage

import (
	"database/sql"
	"fmt"
	"time"

	"ai-gateway/internal/logger"
	"ai-gateway/internal/model"

	"github.com/google/uuid"
	_ "github.com/glebarez/sqlite"
)

// DB 数据库包装器
// 封装 sql.DB 提供数据库操作方法
type DB struct {
	*sql.DB
}

// NewDB 创建数据库连接
// 参数: dbPath 数据库文件路径
// 返回: *DB 数据库实例
func NewDB(dbPath string) (*DB, error) {
	// 打开 SQLite 数据库
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &DB{db}, nil
}

// InitSchema 初始化数据库表结构
// 创建 users、api_keys、token_usage、ai_models、model_pricing 表及索引
func (db *DB) InitSchema() error {
	// 定义数据库表结构
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		email TEXT UNIQUE NOT NULL,
		balance REAL DEFAULT 0,
		created_at TEXT DEFAULT (datetime('now')),
		updated_at TEXT DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS api_keys (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		key_hash TEXT NOT NULL UNIQUE,
		name TEXT,
		rate_limit INTEGER DEFAULT 60,
		monthly_quota INTEGER,
		is_active INTEGER DEFAULT 1,
		created_at TEXT DEFAULT (datetime('now')),
		updated_at TEXT DEFAULT (datetime('now')),
		FOREIGN KEY(user_id) REFERENCES users(id)
	);

	CREATE TABLE IF NOT EXISTS token_usage (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id TEXT NOT NULL,
		api_key_id TEXT NOT NULL,
		model TEXT NOT NULL,
		prompt_tokens INTEGER NOT NULL,
		completion_tokens INTEGER NOT NULL,
		cost REAL NOT NULL,
		created_at TEXT DEFAULT (datetime('now')),
		FOREIGN KEY(user_id) REFERENCES users(id),
		FOREIGN KEY(api_key_id) REFERENCES api_keys(id)
	);

	CREATE TABLE IF NOT EXISTS ai_models (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		provider TEXT NOT NULL,
		base_url TEXT,
		api_key TEXT,
		enabled INTEGER DEFAULT 1,
		models TEXT,
		created_at TEXT DEFAULT (datetime('now')),
		updated_at TEXT DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS model_pricing (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		model_id TEXT NOT NULL,
		model_name TEXT NOT NULL,
		prompt_price REAL NOT NULL,
		completion_price REAL NOT NULL,
		unit INTEGER DEFAULT 1000,
		currency TEXT DEFAULT 'CNY',
		created_at TEXT DEFAULT (datetime('now')),
		updated_at TEXT DEFAULT (datetime('now')),
		FOREIGN KEY(model_id) REFERENCES ai_models(id)
	);

	CREATE INDEX IF NOT EXISTS idx_token_usage_user_id ON token_usage(user_id);
	CREATE INDEX IF NOT EXISTS idx_token_usage_created_at ON token_usage(created_at);
	CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash);
	CREATE INDEX IF NOT EXISTS idx_ai_models_provider ON ai_models(provider);
	CREATE INDEX IF NOT EXISTS idx_model_pricing_model_id ON model_pricing(model_id);
	`
	// 执行建表语句
	_, err := db.Exec(schema)
	return err
}

// ============ 用户操作 ============

// CreateUser 创建新用户
// 参数: email 用户邮箱
// 返回: *model.User 用户对象
func (db *DB) CreateUser(email string) (*model.User, error) {
	// 生成 UUID
	id := uuid.New().String()
	// 插入数据库
	_, err := db.Exec(`
		INSERT INTO users (id, email) VALUES (?, ?)
	`, id, email)
	if err != nil {
		return nil, err
	}
	// 返回用户对象
	return &model.User{
		ID:        uuid.MustParse(id),
		Email:     email,
		Balance:   0,
	}, nil
}

// GetUserByID 根据 ID 获取用户
func (db *DB) GetUserByID(id string) (*model.User, error) {
	var user model.User
	var createdAt, updatedAt string
	// 查询用户
	err := db.QueryRow(`
		SELECT id, email, balance, created_at, updated_at 
		FROM users WHERE id = ?
	`, id).Scan(&user.ID, &user.Email, &user.Balance, &createdAt, &updatedAt)
	user.CreatedAt = createdAt
	user.UpdatedAt = updatedAt
	return &user, err
}

// ============ API Key 操作 ============

// CreateAPIKey 创建 API Key
// 参数: userID 用户ID, keyHash Key哈希值, name Key名称, rateLimit 速率限制
func (db *DB) CreateAPIKey(userID, keyHash, name string, rateLimit int) (*model.APIKey, error) {
	// 生成 UUID
	id := uuid.New().String()
	logger.Info("[STORAGE] CreateAPIKey: userID=%s, keyHash=%s\n", userID, keyHash)
	// 插入数据库
	_, err := db.Exec(`
		INSERT INTO api_keys (id, user_id, key_hash, name, rate_limit)
		VALUES (?, ?, ?, ?, ?)
	`, id, userID, keyHash, name, rateLimit)
	if err != nil {
		logger.Info("[STORAGE] CreateAPIKey error: %v\n", err)
		return nil, err
	}
	return &model.APIKey{
		ID:          uuid.MustParse(id),
		UserID:      uuid.MustParse(userID),
		KeyHash:     keyHash,
		Name:        name,
		RateLimit:   rateLimit,
		IsActive:    true,
	}, nil
}

// GetAPIKeyByHash 根据哈希值获取 API Key
// 用于验证 API Key 是否有效
func (db *DB) GetAPIKeyByHash(keyHash string) (*model.APIKey, error) {
	var key model.APIKey
	var isActive int
	var monthlyQuota sql.NullInt64
	var createdAt, updatedAt string
	logger.Info("[STORAGE] GetAPIKeyByHash: keyHash=%s\n", keyHash)
	// 查询 API Key
	err := db.QueryRow(`
		SELECT id, user_id, key_hash, name, rate_limit, monthly_quota, is_active, created_at, updated_at
		FROM api_keys WHERE key_hash = ? AND is_active = 1
	`, keyHash).Scan(
		&key.ID, &key.UserID, &key.KeyHash, &key.Name,
		&key.RateLimit, &monthlyQuota, &isActive, &createdAt, &updatedAt,
	)
	if err != nil {
		logger.Info("[STORAGE] GetAPIKeyByHash error: %v\n", err)
		return nil, err
	}
	// 处理可空字段
	if monthlyQuota.Valid {
		key.MonthlyQuota = monthlyQuota.Int64
	}
	key.IsActive = isActive == 1
	key.CreatedAt = createdAt
	key.UpdatedAt = updatedAt
	return &key, nil
}

// ============ Token 使用记录操作 ============

// RecordTokenUsage 记录 Token 使用量
func (db *DB) RecordTokenUsage(userID, apiKeyID uuid.UUID, model string, promptTokens, completionTokens int, cost float64) error {
	_, err := db.Exec(`
		INSERT INTO token_usage (user_id, api_key_id, model, prompt_tokens, completion_tokens, cost)
		VALUES (?, ?, ?, ?, ?, ?)
	`, userID.String(), apiKeyID.String(), model, promptTokens, completionTokens, cost)
	return err
}

// GetUserUsage 获取用户使用量
// 参数: userID 用户ID, startDate 开始日期, endDate 结束日期
func (db *DB) GetUserUsage(userID string, startDate, endDate string) ([]model.TokenUsage, error) {
	// 查询指定时间范围内的使用记录
	rows, err := db.Query(`
		SELECT id, user_id, api_key_id, model, prompt_tokens, completion_tokens, cost, created_at
		FROM token_usage 
		WHERE user_id = ? AND created_at BETWEEN ? AND ?
		ORDER BY created_at DESC
	`, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 遍历结果
	var usages []model.TokenUsage
	for rows.Next() {
		var u model.TokenUsage
		var createdAt string
		if err := rows.Scan(&u.ID, &u.UserID, &u.APIKeyID, &u.Model, 
			&u.PromptTokens, &u.CompletionTokens, &u.Cost, &createdAt); err != nil {
			return nil, err
		}
		u.CreatedAt = createdAt
		usages = append(usages, u)
	}
	return usages, nil
}

// GetUserBalance 获取用户余额
func (db *DB) GetUserBalance(userID string) (float64, error) {
	var balance float64
	err := db.QueryRow("SELECT balance FROM users WHERE id = ?", userID).Scan(&balance)
	if err != nil {
		// 如果没有记录，返回默认余额 0
		return 0, nil
	}
	return balance, nil
}

// DeductUserBalance 扣除用户余额
func (db *DB) DeductUserBalance(userID string, amount float64) error {
	// 原子操作：先检查余额再扣减
	result, err := db.Exec(`
		UPDATE users SET balance = balance - ?, updated_at = datetime('now')
		WHERE id = ?
	`, amount, userID)
	if err != nil {
		return err
	}
	
	// 检查是否更新成功
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("user not found: %s", userID)
	}
	return nil
}

// GetAPIKeyUsage 获取 API Key 使用量统计
func (db *DB) GetAPIKeyUsage(apiKeyID string) (*model.APIKeyUsageStats, error) {
	var stats model.APIKeyUsageStats
	
	// 聚合查询统计
	err := db.QueryRow(`
		SELECT 
			api_key_id,
			COUNT(*) as total_requests,
			COALESCE(SUM(prompt_tokens), 0) as total_prompt,
			COALESCE(SUM(completion_tokens), 0) as total_completion,
			COALESCE(SUM(cost), 0) as total_cost
		FROM token_usage
		WHERE api_key_id = ?
	`, apiKeyID).Scan(
		&stats.APIKeyID,
		&stats.TotalRequests,
		&stats.TotalPromptTokens,
		&stats.TotalCompletionTokens,
		&stats.TotalCost,
	)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

// GetDailyUsage 获取每日使用量
func (db *DB) GetDailyUsage(userID, startDate, endDate string) ([]model.DailyUsage, error) {
	// 按日期分组统计
	rows, err := db.Query(`
		SELECT 
			user_id,
			model,
			date(created_at) as date,
			SUM(prompt_tokens) as prompt_tokens,
			SUM(completion_tokens) as completion_tokens,
			SUM(cost) as total_cost
		FROM token_usage
		WHERE user_id = ? AND date(created_at) BETWEEN ? AND ?
		GROUP BY user_id, model, date(created_at)
		ORDER BY date DESC
	`, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var usages []model.DailyUsage
	for rows.Next() {
		var u model.DailyUsage
		if err := rows.Scan(&u.UserID, &u.Model, &u.Date, 
			&u.PromptTokens, &u.CompletionTokens, &u.TotalCost); err != nil {
			return nil, err
		}
		usages = append(usages, u)
	}
	return usages, nil
}

// ListAllAPIKeys 列出所有 API Keys（调试用）
func (db *DB) ListAllAPIKeys() ([]map[string]string, error) {
	rows, err := db.Query("SELECT id, user_id, key_hash, name, is_active FROM api_keys")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []map[string]string
	for rows.Next() {
		var id, userID, keyHash, name string
		var isActive int
		if err := rows.Scan(&id, &userID, &keyHash, &name, &isActive); err != nil {
			return nil, err
		}
		keys = append(keys, map[string]string{
			"id":        id,
			"user_id":   userID,
			"key_hash":  keyHash,
			"name":      name,
			"is_active": fmt.Sprintf("%d", isActive),
		})
	}
	return keys, nil
}

// DebugCheckKey 检查 key 是否存在（调试用）
func (db *DB) DebugCheckKey(keyHash string) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM api_keys WHERE key_hash = ?", keyHash).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ============ AI Model 管理 ============

// CreateAIModel 创建 AI 模型配置
func (db *DB) CreateAIModel(m *model.AIModel) error {
	_, err := db.Exec(`
		INSERT INTO ai_models (id, name, provider, base_url, api_key, enabled, models)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, m.ID, m.Name, m.Provider, m.BaseURL, m.APIKey, m.Enabled, fmt.Sprintf("%v", m.Models))
	return err
}

// GetAIModelByID 根据 ID 获取模型配置
func (db *DB) GetAIModelByID(id string) (*model.AIModel, error) {
	var m model.AIModel
	var enabled int
	var createdAt, updatedAt string
	err := db.QueryRow(`
		SELECT id, name, provider, base_url, api_key, enabled, models, created_at, updated_at
		FROM ai_models WHERE id = ?
	`, id).Scan(&m.ID, &m.Name, &m.Provider, &m.BaseURL, &m.APIKey, &enabled, &m.Models, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	m.Enabled = enabled == 1
	m.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	m.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)
	return &m, nil
}

// ListAIModels 获取所有模型配置
func (db *DB) ListAIModels() ([]model.AIModel, error) {
	rows, err := db.Query(`
		SELECT id, name, provider, base_url, api_key, enabled, models, created_at, updated_at
		FROM ai_models ORDER BY provider, name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var models []model.AIModel
	for rows.Next() {
		var m model.AIModel
		var enabled int
		var createdAt, updatedAt string
		if err := rows.Scan(&m.ID, &m.Name, &m.Provider, &m.BaseURL, &m.APIKey, &enabled, &m.Models, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		m.Enabled = enabled == 1
		m.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		m.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)
		models = append(models, m)
	}
	return models, nil
}

// UpdateAIModel 更新模型配置
func (db *DB) UpdateAIModel(m *model.AIModel) error {
	_, err := db.Exec(`
		UPDATE ai_models SET name=?, provider=?, base_url=?, api_key=?, enabled=?, models=?, updated_at=datetime('now')
		WHERE id=?
	`, m.Name, m.Provider, m.BaseURL, m.APIKey, m.Enabled, fmt.Sprintf("%v", m.Models), m.ID)
	return err
}

// DeleteAIModel 删除模型配置
func (db *DB) DeleteAIModel(id string) error {
	_, err := db.Exec("DELETE FROM ai_models WHERE id=?", id)
	return err
}

// ============ Model Pricing 管理 ============

// CreateModelPricing 创建模型定价
func (db *DB) CreateModelPricing(p *model.ModelPricing) error {
	result, err := db.Exec(`
		INSERT INTO model_pricing (model_id, model_name, prompt_price, completion_price, unit, currency)
		VALUES (?, ?, ?, ?, ?, ?)
	`, p.ModelID, p.ModelName, p.PromptPrice, p.CompletionPrice, p.Unit, p.Currency)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	p.ID = id
	return nil
}

// GetModelPricing 获取模型定价
func (db *DB) GetModelPricing(modelID string) (*model.ModelPricing, error) {
	var p model.ModelPricing
	var createdAt, updatedAt string
	err := db.QueryRow(`
		SELECT id, model_id, model_name, prompt_price, completion_price, unit, currency, created_at, updated_at
		FROM model_pricing WHERE model_id = ?
	`, modelID).Scan(&p.ID, &p.ModelID, &p.ModelName, &p.PromptPrice, &p.CompletionPrice, &p.Unit, &p.Currency, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	p.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	p.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)
	return &p, nil
}

// ListModelPricings 获取所有模型定价
func (db *DB) ListModelPricings() ([]model.ModelPricing, error) {
	rows, err := db.Query(`
		SELECT id, model_id, model_name, prompt_price, completion_price, unit, currency, created_at, updated_at
		FROM model_pricing ORDER BY model_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pricings []model.ModelPricing
	for rows.Next() {
		var p model.ModelPricing
		var createdAt, updatedAt string
		if err := rows.Scan(&p.ID, &p.ModelID, &p.ModelName, &p.PromptPrice, &p.CompletionPrice, &p.Unit, &p.Currency, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		p.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		p.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)
		pricings = append(pricings, p)
	}
	return pricings, nil
}

// UpdateModelPricing 更新模型定价
func (db *DB) UpdateModelPricing(p *model.ModelPricing) error {
	_, err := db.Exec(`
		UPDATE model_pricing SET prompt_price=?, completion_price=?, unit=?, currency=?, updated_at=datetime('now')
		WHERE id=?
	`, p.PromptPrice, p.CompletionPrice, p.Unit, p.Currency, p.ID)
	return err
}

// InitMiniMaxModels 初始化 MiniMax 模型数据
func (db *DB) InitMiniMaxModels() error {
	// 检查是否已存在
	var count int
	db.QueryRow("SELECT COUNT(*) FROM ai_models WHERE provider='minimax'").Scan(&count)
	if count > 0 {
		return nil // 已初始化
	}

	// MiniMax 模型配置
	models := []struct {
		id, name, baseURL string
		modelList         []string
	}{
		{"minimax-chat", "MiniMax Chat", "https://api.minimax.chat/v1", []string{"abab6-chat", "abab5.5-chat", "abab5-chat"}},
		{"minimax-embedding", "MiniMax Embedding", "https://api.minimax.chat/v1", []string{"embo-01"}},
		{"minimax-speech", "MiniMax Speech", "https://api.minimax.chat/v1", []string{"speech-01"}},
		{"minimax-image", "MiniMax Image", "https://api.minimax.chat/v1", []string{"image-01"}},
	}

	for _, m := range models {
		_, err := db.Exec(`
			INSERT INTO ai_models (id, name, provider, base_url, enabled, models)
			VALUES (?, ?, 'minimax', ?, 1, ?)
		`, m.id, m.name, m.baseURL, fmt.Sprintf("%v", m.modelList))
		if err != nil {
			return err
		}
	}

	// MiniMax 定价（参考价，单位：元/千Token）
	pricings := []struct {
		modelID, modelName string
		prompt, completion float64
	}{
		{"minimax-chat", "abab6-chat", 0.01, 0.01},
		{"minimax-chat", "abab5.5-chat", 0.005, 0.005},
		{"minimax-chat", "abab5-chat", 0.001, 0.001},
		{"minimax-embedding", "embo-01", 0.001, 0},
		{"minimax-image", "image-01", 0, 0.05},    // 按张计费
		{"minimax-speech", "speech-01", 0, 0.01}, // 按秒计费
	}

	for _, p := range pricings {
		_, err := db.Exec(`
			INSERT INTO model_pricing (model_id, model_name, prompt_price, completion_price, unit, currency)
			VALUES (?, ?, ?, ?, 1000, 'CNY')
		`, p.modelID, p.modelName, p.prompt, p.completion)
		if err != nil {
			return err
		}
	}

	return nil
}
