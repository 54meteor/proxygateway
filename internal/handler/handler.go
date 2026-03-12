// Package handler HTTP 请求处理包
// 处理客户端请求并调用相应的服务
package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"ai-gateway/internal/config"
	"ai-gateway/internal/logger"
	"ai-gateway/internal/model"
	"ai-gateway/internal/service/adapter"
	"ai-gateway/internal/service/auth"
	"ai-gateway/internal/service/billing"
	"ai-gateway/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GatewayHandler 网关处理器
// 处理所有 HTTP 请求的核心处理器
type GatewayHandler struct {
	adapterFactory *adapter.Factory        // LLM 适配器工厂
	authService    *auth.AuthService      // 认证服务
	billingService *billing.BillingService // 计费服务
	db             *storage.DB             // 数据库实例
	cfg            *config.Config         // 配置实例
	chatLogger     *logger.ChatLogger     // 聊天日志记录器
}

// NewGatewayHandler 创建网关处理器
func NewGatewayHandler(
	adapterFactory *adapter.Factory,
	authService *auth.AuthService,
	billingService *billing.BillingService,
	db *storage.DB,
	cfg *config.Config,
) *GatewayHandler {
	return &GatewayHandler{
		adapterFactory: adapterFactory,
		authService:    authService,
		billingService: billingService,
		db:             db,
		cfg:            cfg,
		chatLogger:     logger.NewChatLogger("logs"),
	}
}

// HealthCheck 健康检查
// 返回服务状态、数据库状态、上游服务状态
func (h *GatewayHandler) HealthCheck(c *gin.Context) {
	result := gin.H{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// 检查数据库连接
	if err := h.db.Ping(); err != nil {
		result["status"] = "degraded"
		result["database"] = gin.H{"status": "error", "message": err.Error()}
		logger.Error("health check: database error: %v", err)
	} else {
		result["database"] = gin.H{"status": "ok"}
	}

	// 检查上游服务
	upstream := make(map[string]interface{})
	
	// MiniMax 状态
	if h.cfg.Models.MiniMax.Enabled {
		status := h.checkMiniMax()
		upstream["minimax"] = status
		if status["status"] != "ok" {
			result["status"] = "degraded"
		}
	} else {
		upstream["minimax"] = gin.H{"status": "disabled"}
	}

	// OpenAI 状态
	if h.cfg.Models.OpenAI.Enabled {
		upstream["openai"] = gin.H{"status": "enabled"}
	} else {
		upstream["openai"] = gin.H{"status": "disabled"}
	}

	// Anthropic 状态
	if h.cfg.Models.Anthropic.Enabled {
		upstream["anthropic"] = gin.H{"status": "enabled"}
	} else {
		upstream["anthropic"] = gin.H{"status": "disabled"}
	}

	result["upstream"] = upstream

	// 返回状态码
	if result["status"] == "ok" {
		c.JSON(http.StatusOK, result)
	} else {
		c.JSON(http.StatusServiceUnavailable, result)
	}
}

// checkMiniMax 检查 MiniMax 服务状态
// 发送真实请求测试上游服务是否可用
func (h *GatewayHandler) checkMiniMax() gin.H {
	// 构造测试请求
	req := model.ChatRequest{
		Model: "MiniMax-M2.5",
		Messages: []model.ChatMessage{
			{Role: "user", Content: "."},
		},
		MaxTokens: 1,
	}

	// 获取适配器
	adapter, ok := h.adapterFactory.Get("MiniMax-M2.5")
	if !ok {
		return gin.H{"status": "error", "message": "adapter not found"}
	}

	// 发送测试请求
	start := time.Now()
	_, err := adapter.ChatComplete(req)
	latency := time.Since(start)

	// 返回状态
	if err != nil {
		logger.Warn("health check: minimax error: %v", err)
		return gin.H{
			"status":  "error",
			"message": err.Error(),
			"latency": latency.String(),
		}
	}

	return gin.H{
		"status":  "ok",
		"latency": latency.String(),
	}
}

// ChatComplete 处理聊天完成请求
// 1. 验证请求
// 2. 获取模型适配器
// 3. 调用 LLM
// 4. 记录用量并扣费
// 5. 返回响应
func (h *GatewayHandler) ChatComplete(c *gin.Context) {
	// 解析请求体
	var req model.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取模型适配器
	llmAdapter, ok := h.adapterFactory.Get(req.Model)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model not supported"})
		return
	}

	// 从 Context 获取用户和 API Key ID
	userIDStr := c.GetString("user_id")
	apiKeyIDStr := c.GetString("api_key_id")
	
	userID, _ := uuid.Parse(userIDStr)
	apiKeyID, _ := uuid.Parse(apiKeyIDStr)

	// 记录请求日志
	logger.Info("Request: userID=%s, model=%s, messages=%d", userIDStr, req.Model, len(req.Messages))

	// 调用大模型
	resp, err := llmAdapter.ChatComplete(req)
	if err != nil {
		logger.Error("LLM Error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 计算费用
	cost := float64(resp.Usage.PromptTokens)/1000*0.01 + float64(resp.Usage.CompletionTokens)/1000*0.01

	// 记录响应日志（包含完整 JSON）
	logger.Info("Response: userID=%s, model=%s, prompt=%d, completion=%d, total=%d, cost=%.6f",
		userIDStr, resp.Model, resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens, cost)

	// 记录完整的请求和响应 JSON 到独立日志文件
	reqJSON, _ := json.Marshal(req)
	respJSON, _ := json.Marshal(resp)
	h.chatLogger.LogRequest(userIDStr, apiKeyIDStr, req.Model, string(reqJSON), string(respJSON), resp.Usage.PromptTokens, resp.Usage.CompletionTokens, cost)

	// 记录使用量并扣费
	if err := h.billingService.RecordUsage(userID, apiKeyID, req, resp); err != nil {
		logger.Error("Billing Error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "billing failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ListModels 列出可用模型
// 返回所有已注册的模型列表
func (h *GatewayHandler) ListModels(c *gin.Context) {
	models := h.adapterFactory.ListModels()
	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   models,
	})
}

// GetUserUsage 获取用户使用量
// 查询指定时间范围内的 Token 使用记录
func (h *GatewayHandler) GetUserUsage(c *gin.Context) {
	userID := c.GetString("user_id")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	usages, err := h.billingService.GetUserUsage(userID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, usages)
}

// CreateAPIKey 创建 API Key
// 为已认证用户创建新的 API Key
func (h *GatewayHandler) CreateAPIKey(c *gin.Context) {
	userID := c.GetString("user_id")
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Name = "default"
	}

	userUUID, _ := uuid.Parse(userID)
	apiKey, err := h.authService.GenerateAPIKey(userUUID, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"api_key": apiKey,
		"name":    req.Name,
	})
}

// InitUser 初始化测试用户 (无需认证，仅调试用)
// 用于快速测试，创建测试用户和 API Key
func (h *GatewayHandler) InitUser(c *gin.Context) {
	// 创建测试用户
	user, err := h.authService.CreateTestUser()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 创建测试 API Key
	apiKey, err := h.authService.GenerateAPIKey(user.ID, "test-key")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	// 计算 hash 用于调试
	hash := auth.HashKey(apiKey)

	c.JSON(http.StatusOK, gin.H{
		"user_id":  user.ID,
		"api_key":  apiKey,
		"key_hash": hash,
		"email":    user.Email,
	})
}

// DebugListAllKeys 调试: 列出所有 API Keys
func (h *GatewayHandler) DebugListAllKeys(c *gin.Context) {
	keys, err := h.authService.ListAllAPIKeys()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, keys)
}

// DebugCheckKey 调试: 检查指定 key
func (h *GatewayHandler) DebugCheckKey(c *gin.Context) {
	key := c.Query("key")
	hash := auth.HashKey(key)
	
	// 直接查数据库
	result, err := h.authService.DebugCheckKey(hash)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error(), "hash": hash})
		return
	}
	c.JSON(200, gin.H{"hash": hash, "found": result})
}
