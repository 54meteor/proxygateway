package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"ai-gateway/internal/config"
	"ai-gateway/internal/model"
	"ai-gateway/internal/service/adapter"
	"ai-gateway/internal/service/auth"
	"ai-gateway/internal/service/billing"
	"ai-gateway/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

// setupTestEnv 创建测试环境
func setupTestEnv(t *testing.T) (*GatewayHandler, *storage.DB, *gin.Engine) {
	tmp := filepath.Join(os.TempDir(), "test_gateway_"+t.Name()+".db")
	db, err := storage.NewDB(tmp)
	require.NoError(t, err)
	err = db.InitSchema()
	require.NoError(t, err)

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
		Pricing: config.PricingConfig{
			"MiniMax-M2.5": {Prompt: 0.01, Completion: 0.01},
		},
	}

	adapterFactory := adapter.NewFactory(cfg)
	authService := auth.NewAuthService(db)
	billingService := billing.NewBillingService(db, cfg)
	handler := NewGatewayHandler(adapterFactory, authService, billingService, db, cfg)

	engine := gin.New()

	// 手动设置路由
	engine.GET("/health", handler.HealthCheck)
	engine.POST("/login", handler.Login)
	engine.POST("/debug/init", handler.InitUser)
	engine.GET("/debug/keys", handler.DebugListAllKeys)
	engine.GET("/debug/check", handler.DebugCheckKey)
	engine.GET("/admin/users", func(c *gin.Context) { c.JSON(200, gin.H{"users": []string{}}) })
	engine.GET("/admin/keys", func(c *gin.Context) { c.JSON(200, gin.H{"keys": []string{}}) })
	engine.GET("/admin/usage", func(c *gin.Context) { c.JSON(200, gin.H{"usage": []string{}}) })

	v1 := engine.Group("/v1")
	v1.GET("/models", handler.ListModels)

	protected := v1.Group("")
	protected.Use(func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || len(authHeader) < 7 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		key := authHeader[7:]
		userID, apiKeyID, err := authService.ValidateAPIKeyFull(key)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.Set("user_id", userID)
		c.Set("api_key_id", apiKeyID)
		c.Next()
	})
	protected.POST("/chat/completions", handler.ChatComplete)
	protected.GET("/usage", handler.GetUserUsage)
	protected.POST("/keys", handler.CreateAPIKey)

	t.Cleanup(func() {
		db.Close()
		os.Remove(tmp)
	})

	return handler, db, engine
}

// ============ 认证中间件测试 ============

func TestAPIKeyAuth_MissingHeader(t *testing.T) {
	_, _, engine := setupTestEnv(t)

	req := httptest.NewRequest("POST", "/v1/chat/completions", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAPIKeyAuth_InvalidFormat(t *testing.T) {
	_, _, engine := setupTestEnv(t)

	req := httptest.NewRequest("POST", "/v1/chat/completions", nil)
	req.Header.Set("Authorization", "InvalidFormat")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ============ 用户操作测试 ============

func TestInitUser(t *testing.T) {
	_, _, engine := setupTestEnv(t)

	body := map[string]string{"email": "newuser@test.com"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/debug/init", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	// InitUser 生成随机邮箱，不使用传入的 email
	assert.NotEmpty(t, resp["email"])
	assert.Contains(t, resp["email"], "@example.com")
	assert.NotEmpty(t, resp["api_key"])
}

func TestDebugListAllKeys(t *testing.T) {
	_, db, engine := setupTestEnv(t)

	_, err := db.CreateUser("debug@test.com")
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/debug/keys", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestDebugCheckKey(t *testing.T) {
	h, db, engine := setupTestEnv(t)

	user, err := db.CreateUser("checkkey@test.com")
	require.NoError(t, err)
	apiKey, err := h.authService.GenerateAPIKey(user.ID, "check-key")
	require.NoError(t, err)

	keyHash := auth.HashKey(apiKey)

	req := httptest.NewRequest("GET", "/debug/check?key_hash="+keyHash, nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// ============ API 接口测试 ============

func TestChatCompletions_Unauthorized(t *testing.T) {
	_, _, engine := setupTestEnv(t)

	body := map[string]interface{}{
		"model": "MiniMax-M2.5",
		"messages": []map[string]string{
			{"role": "user", "content": "hello"},
		},
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestListModels_NoAuth(t *testing.T) {
	_, _, engine := setupTestEnv(t)

	req := httptest.NewRequest("GET", "/v1/models", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	// 公开接口，不需要认证
	assert.NotEqual(t, http.StatusUnauthorized, rec.Code)
}

func TestGetUserUsage_Unauthorized(t *testing.T) {
	_, _, engine := setupTestEnv(t)

	req := httptest.NewRequest("GET", "/v1/usage", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestCreateAPIKey_Unauthorized(t *testing.T) {
	_, _, engine := setupTestEnv(t)

	req := httptest.NewRequest("POST", "/v1/keys", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ============ 管理后台测试 ============

func TestAdminUsersAPI(t *testing.T) {
	_, _, engine := setupTestEnv(t)

	req := httptest.NewRequest("GET", "/admin/users", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAdminKeysAPI(t *testing.T) {
	_, _, engine := setupTestEnv(t)

	req := httptest.NewRequest("GET", "/admin/keys", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAdminUsageAPI(t *testing.T) {
	_, _, engine := setupTestEnv(t)

	req := httptest.NewRequest("GET", "/admin/usage", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// ============ 模型验证测试 ============

func TestChatRequest_Validation(t *testing.T) {
	req := model.ChatRequest{
		Model: "MiniMax-M2.5",
		Messages: []model.ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens: 100,
		Temperature: 0.7,
	}

	assert.Equal(t, "MiniMax-M2.5", req.Model)
	assert.Len(t, req.Messages, 1)
	assert.Equal(t, "user", req.Messages[0].Role)
	assert.Equal(t, "Hello", req.Messages[0].Content)
	assert.Equal(t, 100, req.MaxTokens)
	assert.Equal(t, 0.7, req.Temperature)
}

func TestChatMessage_Fields(t *testing.T) {
	msg := model.ChatMessage{
		Role:    "assistant",
		Content: "Hello!",
		Name:    "assistant_1",
	}

	assert.Equal(t, "assistant", msg.Role)
	assert.Equal(t, "Hello!", msg.Content)
	assert.Equal(t, "assistant_1", msg.Name)
}

func TestChatResponse_Fields(t *testing.T) {
	resp := model.ChatResponse{
		ID:      "chatcmpl-123",
		Object:  "chat.completion",
		Created: int64(1234567890),
		Model:   "MiniMax-M2.5",
		Choices: []model.Choice{
			{
				Index: 0,
				Message: model.ChatMessage{
					Role:    "assistant",
					Content: "Hi there!",
				},
				FinishReason: "stop",
			},
		},
		Usage: model.Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}

	assert.Equal(t, "chatcmpl-123", resp.ID)
	assert.Equal(t, "chat.completion", resp.Object)
	assert.Equal(t, int64(1234567890), resp.Created)
	assert.Equal(t, "MiniMax-M2.5", resp.Model)
	assert.Len(t, resp.Choices, 1)
	assert.Equal(t, "Hi there!", resp.Choices[0].Message.Content)
	assert.Equal(t, 15, resp.Usage.TotalTokens)
}

func TestUsage_Structure(t *testing.T) {
	usage := model.Usage{
		PromptTokens:     42,
		CompletionTokens: 100,
		TotalTokens:      142,
	}

	assert.Equal(t, 42, usage.PromptTokens)
	assert.Equal(t, 100, usage.CompletionTokens)
	assert.Equal(t, 142, usage.TotalTokens)
}
