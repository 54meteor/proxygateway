// Package storage 数据库操作包
// 提供用户、API Key、Token 使用记录等数据持久化功能
package storage

import (
	"database/sql"
	"fmt"

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
// 创建 users、api_keys、token_usage 三张表及索引
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

	CREATE INDEX IF NOT EXISTS idx_token_usage_user_id ON token_usage(user_id);
	CREATE INDEX IF NOT EXISTS idx_token_usage_created_at ON token_usage(created_at);
	CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash);
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
