package admin

import (
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"os"
	"path/filepath"
	"time"

	"ai-gateway/internal/model"
	"ai-gateway/internal/storage"
	"ai-gateway/internal/service/auth"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AdminHandler 管理后台处理器
type AdminHandler struct {
	db          *storage.DB
	templates  *template.Template
	authService *auth.AuthService
}

// NewAdminHandler 创建管理后台处理器
func NewAdminHandler(db *storage.DB, authService *auth.AuthService) *AdminHandler {
	// 加载模板
	tmpl := template.Must(template.ParseGlob("templates/admin/*.html"))
	
	return &AdminHandler{
		db:          db,
		templates:   tmpl,
		authService:  authService,
	}
}

// Dashboard 仪表盘
func (h *AdminHandler) Dashboard(c *gin.Context) {
	// 获取统计数据
	stats := h.getStats()
	recentUsage := h.getRecentUsage(20)
	
	h.templates.ExecuteTemplate(c.Writer, "dashboard.html", gin.H{
		"Stats": stats,
		"RecentUsage": recentUsage,
	})
}

// Users 用户管理页面
func (h *AdminHandler) Users(c *gin.Context) {
	users := h.getAllUsers()
	h.templates.ExecuteTemplate(c.Writer, "users.html", gin.H{
		"Users": users,
	})
}

// Keys API Keys 页面
func (h *AdminHandler) Keys(c *gin.Context) {
	keys := h.getAllKeys()
	h.templates.ExecuteTemplate(c.Writer, "keys.html", gin.H{
		"Keys": keys,
	})
}

// Usage 用量统计页面
func (h *AdminHandler) Usage(c *gin.Context) {
	startDate := c.DefaultQuery("start", time.Now().AddDate(0, 0, -7).Format("2006-01-02"))
	endDate := c.DefaultQuery("end", time.Now().Format("2006-01-02"))
	userID := c.Query("user")
	
	usage, stats := h.getUsageStats(startDate, endDate, userID)
	
	err := h.templates.ExecuteTemplate(c.Writer, "usage.html", gin.H{
		"StartDate": startDate,
		"EndDate": endDate,
		"UserID": userID,
		"Usage": usage,
		"Stats": stats,
	})
	if err != nil {
		c.String(500, "Template error: %v", err)
	}
}

// ============ API 接口 ============

// Recharge 用户充值
func (h *AdminHandler) Recharge(c *gin.Context) {
	var req struct {
		UserID string  `json:"user_id"`
		Amount float64 `json:"amount"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	_, err := h.db.Exec("UPDATE users SET balance = balance + ?, updated_at = datetime('now') WHERE id = ?", req.Amount, req.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ResetBalance 重置余额
func (h *AdminHandler) ResetBalance(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	_, err := h.db.Exec("UPDATE users SET balance = 0, updated_at = datetime('now') WHERE id = ?", req.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ToggleKey 切换 Key 状态
func (h *AdminHandler) ToggleKey(c *gin.Context) {
	var req struct {
		KeyID  string `json:"key_id"`
		Active bool   `json:"active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	active := 0
	if req.Active {
		active = 1
	}
	_, err := h.db.Exec("UPDATE api_keys SET is_active = ?, updated_at = datetime('now') WHERE id = ?", active, req.KeyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// DeleteKey 删除 Key
func (h *AdminHandler) DeleteKey(c *gin.Context) {
	var req struct {
		KeyID string `json:"key_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	_, err := h.db.Exec("DELETE FROM api_keys WHERE id = ?", req.KeyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ============ 内部方法 ============

func (h *AdminHandler) getStats() map[string]interface{} {
	stats := make(map[string]interface{})
	
	var totalUsers, totalKeys int
	h.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&totalUsers)
	h.db.QueryRow("SELECT COUNT(*) FROM api_keys").Scan(&totalKeys)
	
	today := time.Now().Format("2006-01-02")
	var todayRequests int
	var todayCost float64
	h.db.QueryRow("SELECT COUNT(*), COALESCE(SUM(cost), 0) FROM token_usage WHERE date(created_at) = ?", today).Scan(&todayRequests, &todayCost)
	
	// 累计统计
	var totalRequests int64
	var totalPrompt, totalCompletion int64
	var totalCost float64
	h.db.QueryRow("SELECT COUNT(*), COALESCE(SUM(prompt_tokens), 0), COALESCE(SUM(completion_tokens), 0), COALESCE(SUM(cost), 0) FROM token_usage").Scan(&totalRequests, &totalPrompt, &totalCompletion, &totalCost)
	
	stats["TotalUsers"] = totalUsers
	stats["TotalKeys"] = totalKeys
	stats["TodayRequests"] = todayRequests
	stats["TodayCost"] = todayCost
	stats["TotalRequests"] = totalRequests
	stats["TotalTokens"] = totalPrompt + totalCompletion
	stats["TotalPromptTokens"] = totalPrompt
	stats["TotalCompletionTokens"] = totalCompletion
	stats["TotalCost"] = totalCost
	
	// Top 5 用户
	topUsers := h.getTopUsers(5)
	stats["TopUsers"] = topUsers
	
	// 近7天趋势
	trend := h.getUsageTrend(7)
	stats["Trend"] = trend
	
	return stats
}

func (h *AdminHandler) getTopUsers(limit int) []map[string]interface{} {
	rows, _ := h.db.Query(`
		SELECT u.id, u.email, u.username,
			COALESCE(SUM(t.prompt_tokens), 0) as total_prompt,
			COALESCE(SUM(t.completion_tokens), 0) as total_completion,
			COALESCE(SUM(t.cost), 0) as total_cost,
			COUNT(t.id) as request_count
		FROM users u
		LEFT JOIN token_usage t ON u.id = t.user_id
		GROUP BY u.id
		ORDER BY total_cost DESC
		LIMIT ?
	`, limit)
	defer rows.Close()
	
	var result []map[string]interface{}
	for rows.Next() {
		var id, email, username string
		var totalPrompt, totalCompletion, requestCount int64
		var totalCost float64
		rows.Scan(&id, &email, &username, &totalPrompt, &totalCompletion, &totalCost, &requestCount)
		emailDisplay := email
		if emailDisplay == "" {
			emailDisplay = safeSubstr(id, 8)
		}
		result = append(result, map[string]interface{}{
			"UserID":          id,
			"Email":           emailDisplay,
			"TotalPrompt":     totalPrompt,
			"TotalCompletion": totalCompletion,
			"TotalTokens":     totalPrompt + totalCompletion,
			"TotalCost":       totalCost,
			"RequestCount":    requestCount,
		})
	}
	return result
}

func (h *AdminHandler) getUsageTrend(days int) []map[string]interface{} {
	rows, _ := h.db.Query(`
		SELECT date(created_at) as day,
			COUNT(*) as requests,
			COALESCE(SUM(prompt_tokens), 0) as prompt_tokens,
			COALESCE(SUM(completion_tokens), 0) as completion_tokens,
			COALESCE(SUM(cost), 0) as cost
		FROM token_usage
		WHERE created_at >= datetime('now', '-' || ? || ' days')
		GROUP BY date(created_at)
		ORDER BY day ASC
	`, days)
	defer rows.Close()
	
	var result []map[string]interface{}
	for rows.Next() {
		var day string
		var requests, promptTokens, completionTokens int64
		var cost float64
		rows.Scan(&day, &requests, &promptTokens, &completionTokens, &cost)
		result = append(result, map[string]interface{}{
			"Date":              day,
			"Requests":          requests,
			"PromptTokens":      promptTokens,
			"CompletionTokens":  completionTokens,
			"TotalTokens":      promptTokens + completionTokens,
			"Cost":              cost,
		})
	}
	return result
}

func (h *AdminHandler) getRecentUsage(limit int) []map[string]interface{} {
	rows, _ := h.db.Query(`
		SELECT user_id, api_key_id, model, prompt_tokens, completion_tokens, cost, created_at
		FROM token_usage ORDER BY created_at DESC LIMIT ?
	`, limit)
	defer rows.Close()
	
	var result []map[string]interface{}
	for rows.Next() {
		var u map[string]interface{}
		var userID, apiKeyID, model, createdAt string
		var prompt, completion int
		var cost float64
		rows.Scan(&userID, &apiKeyID, &model, &prompt, &completion, &cost, &createdAt)
		u = map[string]interface{}{
			"UserID": userID[:8],
			"APIKeyID": apiKeyID[:8],
			"Model": model,
			"PromptTokens": prompt,
			"CompletionTokens": completion,
			"Cost": cost,
			"CreatedAt": createdAt,
		}
		result = append(result, u)
	}
	return result
}

func (h *AdminHandler) getAllUsers() []map[string]interface{} {
	rows, _ := h.db.Query("SELECT id, email, balance, created_at FROM users ORDER BY created_at DESC")
	defer rows.Close()
	
	var users []map[string]interface{}
	for rows.Next() {
		var id, email, createdAt string
		var balance float64
		rows.Scan(&id, &email, &balance, &createdAt)
		users = append(users, map[string]interface{}{
			"ID": id,
			"Email": email,
			"Balance": balance,
			"CreatedAt": createdAt,
		})
	}
	return users
}

func (h *AdminHandler) getAllKeys() []map[string]interface{} {
	rows, _ := h.db.Query(`
		SELECT k.id, k.user_id, k.name, k.key_hash, k.is_active, k.created_at
		FROM api_keys k ORDER BY k.created_at DESC
	`)
	defer rows.Close()
	
	var keys []map[string]interface{}
	for rows.Next() {
		var id, userID, name, keyHash, createdAt string
		var isActive int
		rows.Scan(&id, &userID, &name, &keyHash, &isActive, &createdAt)
		keys = append(keys, map[string]interface{}{
			"ID": id,
			"UserID": userID[:8],
			"Name": name,
			"KeyHash": keyHash[:16]+"...",
			"IsActive": isActive,
			"CreatedAt": createdAt,
		})
	}
	return keys
}

func (h *AdminHandler) getUsageStats(startDate, endDate, userID string) ([]map[string]interface{}, map[string]interface{}) {
	// 构建查询
	where := "WHERE date(t.created_at) BETWEEN ? AND ?"
	args := []interface{}{startDate, endDate}
	
	if userID != "" {
		where += " AND t.user_id = ?"
		args = append(args, userID)
	}
	
	// 查询明细（关联用户邮箱）
	query := fmt.Sprintf(`
		SELECT t.user_id, u.email, t.api_key_id, t.model, t.prompt_tokens, t.completion_tokens, t.cost, t.created_at 
		FROM token_usage t 
		LEFT JOIN users u ON t.user_id = u.id 
		%s 
		ORDER BY t.created_at DESC LIMIT 100`, where)
	
	rows, _ := h.db.Query(query, args...)
	defer rows.Close()
	
	var usage []map[string]interface{}
	for rows.Next() {
		var uid, email, aid, model, createdAt string
		var prompt, completion int
		var cost float64
		rows.Scan(&uid, &email, &aid, &model, &prompt, &completion, &cost, &createdAt)
		
		emailDisplay := email
		if emailDisplay == "" {
			emailDisplay = safeSubstr(uid, 8)
		} else {
			// 截取邮箱前缀
			for i := range emailDisplay {
				if emailDisplay[i] == '@' {
					emailDisplay = emailDisplay[:i]
					break
				}
			}
		}
		
		usage = append(usage, map[string]interface{}{
			"UserID":      safeSubstr(uid, 8),
			"UserEmail":   emailDisplay,
			"APIKeyID":   safeSubstr(aid, 8),
			"Model":       model,
			"PromptTokens": prompt,
			"CompletionTokens": completion,
			"TotalTokens": prompt + completion,
			"Cost":        cost,
			"CreatedAt":   createdAt,
		})
	}
	
	// 统计
	stats := make(map[string]interface{})
	var totalReq, totalPrompt, totalCompletion int64
	var totalCost float64
	
	statQuery := "SELECT COUNT(*), COALESCE(SUM(prompt_tokens), 0), COALESCE(SUM(completion_tokens), 0), COALESCE(SUM(cost), 0) FROM token_usage WHERE date(created_at) BETWEEN ? AND ?"
	h.db.QueryRow(statQuery, startDate, endDate).Scan(&totalReq, &totalPrompt, &totalCompletion, &totalCost)
	
	// 成功请求数（假设所有记录都是成功的）
	successReq := totalReq
	
	stats["TotalRequests"] = totalReq
	stats["TotalPrompt"] = totalPrompt
	stats["TotalCompletion"] = totalCompletion
	stats["TotalTokens"] = totalPrompt + totalCompletion
	stats["TotalCost"] = totalCost
	if totalReq > 0 {
		stats["SuccessRate"] = float64(successReq) * 100 / float64(totalReq)
	} else {
		stats["SuccessRate"] = 0
	}
	
	return usage, stats
}

// Init 初始化目录（确保模板目录存在）
func Init() {
	filepath.Walk("templates", func(path string, info os.FileInfo, err error) error {
		return nil
	})
}

// 安全截断字符串
func safeSubstr(s string, maxLen int) string {
	if len(s) == 0 {
		return ""
	}
	if len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}

// ============ JSON API ============

// DashboardAPI 仪表盘 JSON
func (h *AdminHandler) DashboardAPI(c *gin.Context) {
	stats := h.getStats()
	c.JSON(200, stats)
}

// UsersAPI 用户列表 JSON
func (h *AdminHandler) UsersAPI(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT u.id, u.email, u.phone, u.username, u.balance, u.created_at,
			(SELECT k.key_hash FROM api_keys k WHERE k.user_id = u.id AND k.is_active = 1 ORDER BY k.created_at DESC LIMIT 1) as key_hash
		FROM users u ORDER BY u.created_at DESC
	`)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var users []map[string]interface{}
	for rows.Next() {
		var id, email, phone, username, createdAt, keyHash string
		var balance float64
		rows.Scan(&id, &email, &phone, &username, &balance, &createdAt, &keyHash)

		// 脱敏：前5位 + ***** + 后5位，无Key则为空字符串
		var maskedKey string
		if keyHash != "" {
			if len(keyHash) >= 10 {
				maskedKey = keyHash[:5] + "*****" + keyHash[len(keyHash)-5:]
			} else {
				maskedKey = keyHash[:3] + "****"
			}
		}

		users = append(users, map[string]interface{}{
			"id":          id,
			"email":       email,
			"phone":       phone,
			"username":    username,
			"balance":     balance,
			"created_at":  createdAt,
			"api_key":     maskedKey,
		})
	}
	c.JSON(200, gin.H{"success": true, "data": users})
}

// KeysAPI API Keys JSON（已废弃，保留路由兼容）
func (h *AdminHandler) KeysAPI(c *gin.Context) {
	keys := h.getAllKeys()
	c.JSON(200, gin.H{"success": true, "data": keys})
}

// UsageAPI 用量统计 JSON
func (h *AdminHandler) UsageAPI(c *gin.Context) {
	startDate := c.DefaultQuery("start", time.Now().AddDate(0, 0, -7).Format("2006-01-02"))
	endDate := c.DefaultQuery("end", time.Now().Format("2006-01-02"))
	userID := c.Query("user")
	
	usage, stats := h.getUsageStats(startDate, endDate, userID)
	
	c.JSON(200, gin.H{
		"Usage": usage,
		"Stats": stats,
	})
}

// ============ Models API ============

// ListModelsAPI 获取所有模型
func (h *AdminHandler) ListModelsAPI(c *gin.Context) {
	models, err := h.db.ListAIModels()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 转换字段以匹配前端期望的格式
	var result []map[string]interface{}
	for _, m := range models {
		status := "inactive"
		if m.Enabled {
			status = "active"
		}
		result = append(result, map[string]interface{}{
			"id":          m.ID,
			"name":        m.Name,
			"provider":    m.Provider,
			"base_url":    m.BaseURL,
			"api_key":     m.APIKey,
			"status":      status,
			"models":      m.Models,
			"createdAt":   m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			"updatedAt":   m.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"list":  result,
			"total": len(result),
		},
	})
}

// GetModelAPI 获取单个模型
func (h *AdminHandler) GetModelAPI(c *gin.Context) {
	id := c.Param("id")
	model, err := h.db.GetAIModelByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "model not found"})
		return
	}
	status := "inactive"
	if model.Enabled {
		status = "active"
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"id":        model.ID,
			"name":      model.Name,
			"provider":  model.Provider,
			"base_url":  model.BaseURL,
			"api_key":   model.APIKey,
			"status":    status,
			"models":    model.Models,
			"createdAt": model.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			"updatedAt": model.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
	})
}

// CreateModelAPI 创建模型
func (h *AdminHandler) CreateModelAPI(c *gin.Context) {
	var m struct {
		ID       string   `json:"id"`
		Name     string   `json:"name" binding:"required"`
		Provider string   `json:"provider" binding:"required"`
		BaseURL  string   `json:"base_url"`
		APIKey   string   `json:"api_key"`
		Enabled  bool     `json:"enabled"`
		Models   []string `json:"models"`
	}
	if err := c.ShouldBindJSON(&m); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 生成 UUID 如果未提供
	modelID := m.ID
	if modelID == "" {
		modelID = uuid.New().String()
	}

	model := &model.AIModel{
		ID:       modelID,
		Name:     m.Name,
		Provider: m.Provider,
		BaseURL:  m.BaseURL,
		APIKey:   m.APIKey,
		Enabled:  m.Enabled,
		Models:   m.Models,
	}

	if err := h.db.CreateAIModel(model); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, model)
}

// UpdateModelAPI 更新模型
func (h *AdminHandler) UpdateModelAPI(c *gin.Context) {
	id := c.Param("id")
	var m struct {
		Name     string   `json:"name"`
		Provider string   `json:"provider"`
		BaseURL  string   `json:"base_url"`
		APIKey   string   `json:"api_key"`
		Enabled  *bool    `json:"enabled"`
		Models   []string `json:"models"`
	}
	if err := c.ShouldBindJSON(&m); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 先获取现有模型
	existing, err := h.db.GetAIModelByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "model not found"})
		return
	}

	// 更新字段
	if m.Name != "" {
		existing.Name = m.Name
	}
	if m.Provider != "" {
		existing.Provider = m.Provider
	}
	if m.BaseURL != "" {
		existing.BaseURL = m.BaseURL
	}
	if m.APIKey != "" {
		existing.APIKey = m.APIKey
	}
	if m.Enabled != nil {
		existing.Enabled = *m.Enabled
	}
	if m.Models != nil {
		existing.Models = m.Models
	}

	if err := h.db.UpdateAIModel(existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, existing)
}

// DeleteModelAPI 删除模型
func (h *AdminHandler) DeleteModelAPI(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.DeleteAIModel(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetModelPricingAPI 获取模型定价
func (h *AdminHandler) GetModelPricingAPI(c *gin.Context) {
	id := c.Param("id")
	pricing, err := h.db.GetModelPricing(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pricing not found"})
		return
	}
	c.JSON(http.StatusOK, pricing)
}

// UpdateModelPricingAPI 更新模型定价
func (h *AdminHandler) UpdateModelPricingAPI(c *gin.Context) {
	id := c.Param("id")
	var p struct {
		PromptPrice     float64 `json:"prompt_price"`
		CompletionPrice float64 `json:"completion_price"`
		Unit            int     `json:"unit"`
		Currency        string  `json:"currency"`
	}
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 先获取现有定价
	existing, err := h.db.GetModelPricing(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pricing not found"})
		return
	}

	if p.PromptPrice > 0 {
		existing.PromptPrice = p.PromptPrice
	}
	if p.CompletionPrice > 0 {
		existing.CompletionPrice = p.CompletionPrice
	}
	if p.Unit > 0 {
		existing.Unit = p.Unit
	}
	if p.Currency != "" {
		existing.Currency = p.Currency
	}

	if err := h.db.UpdateModelPricing(existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, existing)
}

// CreateUserAPI 创建用户
func (h *AdminHandler) CreateUserAPI(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Username string `json:"username"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 至少填写一项
	if req.Email == "" && req.Phone == "" && req.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email、phone、username 至少填写一项"})
		return
	}

	// 格式校验
	if req.Email != "" && !isValidEmail(req.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "邮箱格式不正确"})
		return
	}
	if req.Phone != "" && !isValidPhone(req.Phone) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "手机号格式不正确，需为11位数字"})
		return
	}

	// 唯一性校验（代码层）
	if req.Email != "" {
		exists, _ := h.db.CheckFieldExists("email", req.Email)
		if exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "邮箱已被注册"})
			return
		}
	}
	if req.Phone != "" {
		exists, _ := h.db.CheckFieldExists("phone", req.Phone)
		if exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "手机号已被注册"})
			return
		}
	}
	if req.Username != "" {
		exists, _ := h.db.CheckFieldExists("username", req.Username)
		if exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "用户名已被注册"})
			return
		}
	}

	userID := uuid.New().String()
	_, err := h.db.Exec(`
		INSERT INTO users (id, email, phone, username, balance, created_at)
		VALUES (?, ?, ?, ?, 0, datetime('now'))
	`, userID, req.Email, req.Phone, req.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建用户失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "user_id": userID, "message": "用户创建成功"})
}

// UpdateUserAPI 更新用户
func (h *AdminHandler) UpdateUserAPI(c *gin.Context) {
	var req struct {
		UserID   string `json:"user_id" binding:"required"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Username string `json:"username"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 格式校验
	if req.Email != "" && !isValidEmail(req.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "邮箱格式不正确"})
		return
	}
	if req.Phone != "" && !isValidPhone(req.Phone) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "手机号格式不正确，需为11位数字"})
		return
	}

	// 唯一性校验（排除自己）
	if req.Email != "" {
		exists, _ := h.db.CheckFieldExistsExcludingUser("email", req.Email, req.UserID)
		if exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "邮箱已被其他用户注册"})
			return
		}
	}
	if req.Phone != "" {
		exists, _ := h.db.CheckFieldExistsExcludingUser("phone", req.Phone, req.UserID)
		if exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "手机号已被其他用户注册"})
			return
		}
	}
	if req.Username != "" {
		exists, _ := h.db.CheckFieldExistsExcludingUser("username", req.Username, req.UserID)
		if exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "用户名已被其他用户注册"})
			return
		}
	}

	_, err := h.db.Exec(`
		UPDATE users SET email=?, phone=?, username=? WHERE id=?
	`, req.Email, req.Phone, req.Username, req.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新用户失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "用户更新成功"})
}

// DeleteUserAPI 删除用户
func (h *AdminHandler) DeleteUserAPI(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 级联删除
	h.db.Exec(`DELETE FROM api_keys WHERE user_id=?`, req.UserID)
	h.db.Exec(`DELETE FROM token_usage WHERE user_id=?`, req.UserID)
	h.db.Exec(`DELETE FROM users WHERE id=?`, req.UserID)

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "用户删除成功"})
}

// CreateKeyAPI 为指定用户创建 API Key
func (h *AdminHandler) CreateKeyAPI(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查该用户是否已有激活的 Key
	var count int
	h.db.QueryRow(`SELECT COUNT(*) FROM api_keys WHERE user_id = ? AND is_active = 1`, req.UserID).Scan(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该用户已存在 API Key，请使用重置功能"})
		return
	}

	apiKey, err := h.authService.GenerateAPIKey(uuid.MustParse(req.UserID), "admin-created")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建 Key 失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"api_key":  apiKey,
		"message":  "API Key 创建成功，请妥善保管",
	})
}

// ResetKeyAPI 重置指定用户的 API Key（通过 user_id）
func (h *AdminHandler) ResetKeyAPI(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 查找该用户当前激活的 Key
	var oldKeyID string
	err := h.db.QueryRow(`SELECT id FROM api_keys WHERE user_id = ? AND is_active = 1`, req.UserID).Scan(&oldKeyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "该用户没有可重置的 Key"})
		return
	}

	// 删除旧 Key
	h.db.Exec(`DELETE FROM api_keys WHERE id = ?`, oldKeyID)

	// 生成新 Key
	newKey, err := h.authService.GenerateAPIKey(uuid.MustParse(req.UserID), "admin-reset")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "重置 Key 失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"api_key": newKey,
		"message": "Key 已重置，新 Key 请妥善保管",
	})
}

// GetAPIKeyByID 根据 ID 获取 Key 信息
func (h *AdminHandler) GetAPIKeyByID(keyID string) (*model.APIKey, error) {
	return h.db.GetAPIKeyByID(keyID)
}

// isValidEmail 校验邮箱格式
func isValidEmail(email string) bool {
	matched, _ := regexp.MatchString(`^[^@]+@[^@]+\.[^@]+$`, email)
	return matched
}

// isValidPhone 校验手机号格式（11位数字）
func isValidPhone(phone string) bool {
	matched, _ := regexp.MatchString(`^\d{11}$`, phone)
	return matched
}
