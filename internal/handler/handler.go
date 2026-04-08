// Package handler HTTP 请求处理包
// 处理客户端请求并调用相应的服务
package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
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
	adapterFactory adapter.AdapterRegistry // LLM 适配器注册表
	authService    *auth.AuthService      // 认证服务
	billingService *billing.BillingService // 计费服务
	db             *storage.DB             // 数据库实例
	cfg            *config.Config         // 配置实例
	chatLogger     *logger.ChatLogger     // 聊天日志记录器
}

// NewGatewayHandler 创建网关处理器
func NewGatewayHandler(
	adapterFactory adapter.AdapterRegistry,
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

// Completions 处理文本补全请求
// 1. 验证请求
// 2. 获取模型适配器
// 3. 调用 LLM
// 4. 记录用量并扣费
// 5. 返回响应
func (h *GatewayHandler) Completions(c *gin.Context) {
	// 解析请求体
	var req model.CompletionRequest
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

	// 构造与 ChatRequest 兼容的请求格式
	chatReq := model.ChatRequest{
		Model: req.Model,
		Messages: []model.ChatMessage{
			{Role: "user", Content: req.Prompt},
		},
		Temperature: req.Temperature,
		MaxTokens:  req.MaxTokens,
		Stream:     req.Stream,
	}

	logger.Info("Completion Request: userID=%s, model=%s", userIDStr, req.Model)

	// 调用大模型
	chatResp, err := llmAdapter.ChatComplete(chatReq)
	if err != nil {
		logger.Error("LLM Error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 转换为 CompletionResponse 格式
	compResp := model.CompletionResponse{
		ID:      chatResp.ID,
		Object:  "text_completion",
		Created: chatResp.Created,
		Model:   chatResp.Model,
		Choices: make([]model.CompletionChoice, len(chatResp.Choices)),
		Usage:   chatResp.Usage,
	}
	for i, choice := range chatResp.Choices {
		compResp.Choices[i] = model.CompletionChoice{
			Text:         choice.Message.Content,
			Index:        choice.Index,
			FinishReason: choice.FinishReason,
		}
	}

	// 计算费用
	cost := float64(compResp.Usage.PromptTokens)/1000*0.01 + float64(compResp.Usage.CompletionTokens)/1000*0.01

	// 记录完整的请求和响应 JSON 到独立日志文件
	reqJSON, _ := json.Marshal(req)
	respJSON, _ := json.Marshal(compResp)
	h.chatLogger.LogRequest(userIDStr, apiKeyIDStr, req.Model, string(reqJSON), string(respJSON), compResp.Usage.PromptTokens, compResp.Usage.CompletionTokens, cost)

	// 记录使用量并扣费
	if err := h.billingService.RecordUsage(userID, apiKeyID, chatReq, chatResp); err != nil {
		logger.Error("Billing Error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "billing failed: " + err.Error()})
		return
	}

	logger.Info("Completion Response: userID=%s, model=%s, prompt=%d, completion=%d, total=%d, cost=%.6f",
		userIDStr, compResp.Model, compResp.Usage.PromptTokens, compResp.Usage.CompletionTokens, compResp.Usage.TotalTokens, cost)

	c.JSON(http.StatusOK, compResp)
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

// Embeddings 处理 Embeddings 请求
// 1. 验证请求
// 2. 获取适配器
// 3. 调用 Embeddings API
// 4. 返回响应
func (h *GatewayHandler) Embeddings(c *gin.Context) {
	// 解析请求体
	var req model.EmbeddingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 设置默认值
	if req.Model == "" {
		req.Model = "MiniMax-Text-01"
	}
	if req.EncodingFormat == "" {
		req.EncodingFormat = "float"
	}

	// 获取适配器
	adapter, ok := h.adapterFactory.Get(req.Model)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model not supported"})
		return
	}

	// 从 Context 获取用户和 API Key ID
	userIDStr := c.GetString("user_id")
	apiKeyIDStr := c.GetString("api_key_id")

	userID, _ := uuid.Parse(userIDStr)
	apiKeyID, _ := uuid.Parse(apiKeyIDStr)

	// 调用 Embeddings
	resp, err := adapter.Embeddings(req)
	if err != nil {
		logger.Error("Embeddings Error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 构造计费请求（复用 ChatRequest 以调用 RecordUsage）
	chatReq := model.ChatRequest{Model: req.Model}
	chatResp := &model.ChatResponse{
		Model: resp.Model,
		Usage: model.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: 0,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}

	// 记录使用量并扣费
	if err := h.billingService.RecordUsage(userID, apiKeyID, chatReq, chatResp); err != nil {
		logger.Error("Billing Error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "billing failed: " + err.Error()})
		return
	}

	logger.Info("Embeddings: userID=%s, model=%s, inputCount=%d, promptTokens=%d", userIDStr, req.Model, len(req.Input), resp.Usage.PromptTokens)

	c.JSON(http.StatusOK, resp)
}

// Images 处理图片生成请求
// 1. 验证请求
// 2. 获取适配器
// 3. 调用 Images API
// 4. 返回响应
func (h *GatewayHandler) Images(c *gin.Context) {
	// 解析请求体
	var req model.ImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 设置默认值
	if req.Model == "" {
		req.Model = "MiniMax-Image-01"
	}
	if req.N == 0 {
		req.N = 1
	}

	// 获取适配器
	adapter, ok := h.adapterFactory.Get(req.Model)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model not supported"})
		return
	}

	// 从 Context 获取用户和 API Key ID
	userIDStr := c.GetString("user_id")
	apiKeyIDStr := c.GetString("api_key_id")

	userID, _ := uuid.Parse(userIDStr)
	apiKeyID, _ := uuid.Parse(apiKeyIDStr)

	// 调用 Images
	resp, err := adapter.Images(req)
	if err != nil {
		logger.Error("Images Error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 构造计费请求（图片生成按次计费）
	chatReq := model.ChatRequest{Model: req.Model}
	chatResp := &model.ChatResponse{
		Model: resp.Model,
		Usage: model.Usage{
			PromptTokens:     0,
			CompletionTokens: 0,
			TotalTokens:      0,
		},
	}

	// 记录使用量并扣费
	if err := h.billingService.RecordUsage(userID, apiKeyID, chatReq, chatResp); err != nil {
		logger.Error("Billing Error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "billing failed: " + err.Error()})
		return
	}

	logger.Info("Images: userID=%s, model=%s, n=%d", userIDStr, req.Model, req.N)

	c.JSON(http.StatusOK, resp)
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

// AudioTranscriptions 处理语音转文字请求
// 1. 解析 multipart/form-data 上传
// 2. 调用 MiniMax 语音识别 API
// 3. 转换响应格式并返回
func (h *GatewayHandler) AudioTranscriptions(c *gin.Context) {
	// 1. 解析 multipart/form-data
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

	// 读取文件内容
	fileContent, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read file: " + err.Error()})
		return
	}

	// 获取其他参数
	modelName := c.PostForm("model")
	if modelName == "" {
		modelName = "MiniMax-Audio"
	}
	language := c.PostForm("language")
	prompt := c.PostForm("prompt")
	responseFormat := c.PostForm("response_format")
	if responseFormat == "" {
		responseFormat = "json"
	}

	// 从 Context 获取用户和 API Key ID
	userIDStr := c.GetString("user_id")
	apiKeyIDStr := c.GetString("api_key_id")

	logger.Info("Audio Transcription: userID=%s, model=%s, filename=%s, size=%d",
		userIDStr, modelName, header.Filename, len(fileContent))

	// 2. 调用 MiniMax Audio API (multipart/form-data)
	url := h.cfg.Models.MiniMax.BaseURL + "/audio/transcriptions"

	// 构造 multipart/form-data 请求
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 添加文件
	part, err := writer.CreateFormFile("file", header.Filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create form file"})
		return
	}
	if _, err := part.Write(fileContent); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write file content"})
		return
	}

	// 添加 model 字段
	if err := writer.WriteField("model", modelName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write model field"})
		return
	}

	// 添加 language 字段（可选）
	if language != "" {
		if err := writer.WriteField("language", language); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write language field"})
			return
		}
	}

	// 添加 prompt 字段（可选）
	if prompt != "" {
		if err := writer.WriteField("prompt", prompt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write prompt field"})
			return
		}
	}

	// 添加 response_format 字段
	if err := writer.WriteField("response_format", responseFormat); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write response_format field"})
		return
	}

	if err := writer.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to close writer"})
		return
	}

	// 3. 发送 HTTP 请求
	httpReq, err := http.NewRequest("POST", url, body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	httpReq.Header.Set("Authorization", "Bearer "+h.cfg.Models.MiniMax.APIKey)
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: time.Duration(h.cfg.Models.MiniMax.Timeout)*time.Second + 60} // 音频可能较长
	resp, err := client.Do(httpReq)
	if err != nil {
		logger.Error("Audio Transcription Error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	// 4. 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("MiniMax API error: status %d, body: %s", resp.StatusCode, string(respBody))
		logger.Error("%s", errMsg)
		c.JSON(http.StatusInternalServerError, gin.H{"error": errMsg})
		return
	}

	// 5. 解析 MiniMax 响应并转换为 OpenAI 格式
	var miniMaxResp map[string]interface{}
	if err := json.Unmarshal(respBody, &miniMaxResp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse response: " + err.Error()})
		return
	}

	// 构造 OpenAI 兼容响应
	transcriptionResp := model.TranscriptionResponse{
		Model: modelName,
	}

	if text, ok := miniMaxResp["text"].(string); ok {
		transcriptionResp.Text = text
	}
	if duration, ok := miniMaxResp["duration"].(float64); ok {
		transcriptionResp.Duration = duration
	}
	if lang, ok := miniMaxResp["language"].(string); ok {
		transcriptionResp.Language = lang
	}

	// 记录使用量（按音频时长计费，简化为 1 元/分钟）
	userID, _ := uuid.Parse(userIDStr)
	apiKeyID, _ := uuid.Parse(apiKeyIDStr)

	// 估算费用（音频按字数或时长，这里简化处理）
	_ = transcriptionResp.Duration / 60.0 * 0.1
	billingReq := model.ChatRequest{Model: modelName}
	billingResp := &model.ChatResponse{
		Model: modelName,
		Usage: model.Usage{
			PromptTokens:     len(fileContent) / 100, // 估算
			CompletionTokens: len(transcriptionResp.Text),
			TotalTokens:      len(fileContent)/100 + len(transcriptionResp.Text),
		},
	}

	if err := h.billingService.RecordUsage(userID, apiKeyID, billingReq, billingResp); err != nil {
		logger.Error("Billing Error: %v", err)
	}

	logger.Info("Audio Transcription Response: userID=%s, model=%s, duration=%.2fs, text_len=%d",
		userIDStr, modelName, transcriptionResp.Duration, len(transcriptionResp.Text))

	c.JSON(http.StatusOK, transcriptionResp)
}
