package billing

import (
	"fmt"
	"time"

	"ai-gateway/internal/config"
	"ai-gateway/internal/model"
	"ai-gateway/internal/storage"

	"github.com/google/uuid"
)

// BillingService 计费服务
// 提供 Token 计费、余额管理、用量统计等功能
type BillingService struct {
	db  *storage.DB       // 数据库实例
	cfg *config.Config    // 配置实例
}

// NewBillingService 创建计费服务
func NewBillingService(db *storage.DB, cfg *config.Config) *BillingService {
	return &BillingService{
		db:  db,
		cfg: cfg,
	}
}

// CalculateCost 计算费用
// 根据模型单价和 Token 数量计算费用
// 参数: model 模型名称, promptTokens Prompt Token 数, completionTokens Completion Token 数
// 返回: (float64, error) 费用和错误
func (s *BillingService) CalculateCost(model string, promptTokens, completionTokens int) (float64, error) {
	// 获取模型单价配置
	pricing, ok := s.cfg.Pricing[model]
	if !ok {
		return 0, fmt.Errorf("unknown model: %s", model)
	}

	// 计算费用：Prompt费用 + Completion费用
	promptCost := float64(promptTokens) / 1000 * pricing.Prompt
	completionCost := float64(completionTokens) / 1000 * pricing.Completion

	return promptCost + completionCost, nil
}

// CheckBalance 检查余额是否足够
// 参数: userID 用户ID, model 模型名称, promptTokens Prompt Token 数, completionTokens Completion Token 数
// 返回: (float64, error) 当前余额和错误
func (s *BillingService) CheckBalance(userID uuid.UUID, model string, promptTokens, completionTokens int) (float64, error) {
	// 计算需要扣除的费用
	cost, err := s.CalculateCost(model, promptTokens, completionTokens)
	if err != nil {
		return 0, err
	}

	// 获取用户当前余额
	balance, err := s.db.GetUserBalance(userID.String())
	if err != nil {
		return 0, err
	}

	// 检查余额是否足够
	if balance < cost {
		return balance, fmt.Errorf("insufficient balance: have %.2f, need %.2f", balance, cost)
	}

	return balance, nil
}

// DeductBalance 扣除余额
// 参数: userID 用户ID, amount 扣除金额
func (s *BillingService) DeductBalance(userID uuid.UUID, amount float64) error {
	return s.db.DeductUserBalance(userID.String(), amount)
}

// RecordUsage 记录使用量并扣费
// 完整的计费流程：计算费用 -> 扣除余额 -> 记录用量
func (s *BillingService) RecordUsage(userID, apiKeyID uuid.UUID, req model.ChatRequest, resp *model.ChatResponse) error {
	// 获取 Token 使用量
	promptTokens := resp.Usage.PromptTokens
	completionTokens := resp.Usage.CompletionTokens

	// 计算费用
	cost, err := s.CalculateCost(req.Model, promptTokens, completionTokens)
	if err != nil {
		return err
	}

	// 扣除余额
	if err := s.DeductBalance(userID, cost); err != nil {
		return fmt.Errorf("deduct balance failed: %v", err)
	}

	// 记录使用量到数据库
	return s.db.RecordTokenUsage(userID, apiKeyID, req.Model, promptTokens, completionTokens, cost)
}

// GetUserUsage 获取用户使用量
// 参数: userID 用户ID, startDate 开始日期, endDate 结束日期
// 返回: []model.TokenUsage 使用记录列表
func (s *BillingService) GetUserUsage(userID, startDate, endDate string) ([]model.TokenUsage, error) {
	return s.db.GetUserUsage(userID, startDate, endDate)
}

// GetAPIKeyUsage 获取 API Key 使用量统计
func (s *BillingService) GetAPIKeyUsage(apiKeyID string) (*model.APIKeyUsageStats, error) {
	return s.db.GetAPIKeyUsage(apiKeyID)
}

// GetDailyUsage 获取每日使用量统计
// 参数: userID 用户ID, days 前几天
// 返回: []model.DailyUsage 每日用量列表
func (s *BillingService) GetDailyUsage(userID string, days int) ([]model.DailyUsage, error) {
	// 计算日期范围
	startDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	endDate := time.Now().Format("2006-01-02")
	return s.db.GetDailyUsage(userID, startDate, endDate)
}
