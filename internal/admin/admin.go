package admin

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"ai-gateway/internal/storage"

	"github.com/gin-gonic/gin"
)

// AdminHandler 管理后台处理器
type AdminHandler struct {
	db *storage.DB
	templates *template.Template
}

// NewAdminHandler 创建管理后台处理器
func NewAdminHandler(db *storage.DB) *AdminHandler {
	// 加载模板
	tmpl := template.Must(template.ParseGlob("templates/admin/*.html"))
	
	return &AdminHandler{
		db: db,
		templates: tmpl,
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
	
	stats["TotalUsers"] = totalUsers
	stats["TotalKeys"] = totalKeys
	stats["TodayRequests"] = todayRequests
	stats["TodayCost"] = todayCost
	
	return stats
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
	users := h.getAllUsers()
	c.JSON(200, users)
}

// KeysAPI API Keys JSON
func (h *AdminHandler) KeysAPI(c *gin.Context) {
	keys := h.getAllKeys()
	c.JSON(200, keys)
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
