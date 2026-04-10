// Package router 路由配置包
// 设置所有 HTTP 路由和中间件
package router

import (
	"ai-gateway/internal/admin"
	"ai-gateway/internal/handler"
	"ai-gateway/internal/middleware"
	"ai-gateway/internal/service/auth"

	"github.com/gin-gonic/gin"
)

// Setup 设置路由
// 配置所有 HTTP 端点和中间件
func Setup(
	r *gin.Engine,
	gatewayHandler *handler.GatewayHandler,
	authService *auth.AuthService,
	adminHandler *admin.AdminHandler,
) {
	// 全局中间件 - CORS
	r.Use(middleware.CORSMiddleware())

	// 公开接口
	// 健康检查
	r.GET("/health", gatewayHandler.HealthCheck)

	// 登录接口（Pure Admin）
	r.POST("/login", gatewayHandler.Login)
	r.POST("/refreshToken", gatewayHandler.RefreshToken)
	r.GET("/get-async-routes", gatewayHandler.GetAsyncRoutes)

	// 管理后台 API（Pure Admin 前端调用）
	adminGroup := r.Group("/admin")
	{
		adminGroup.GET("/", adminHandler.DashboardAPI)
		adminGroup.GET("/users", adminHandler.UsersAPI)
		adminGroup.GET("/keys", adminHandler.KeysAPI)
		adminGroup.GET("/usage", adminHandler.UsageAPI)

		// 管理 API
		adminGroup.POST("/api/user/recharge", adminHandler.Recharge)
		adminGroup.POST("/api/user/reset", adminHandler.ResetBalance)
		adminGroup.POST("/api/user/create", adminHandler.CreateUserAPI)
		adminGroup.POST("/api/user/update", adminHandler.UpdateUserAPI)
		adminGroup.POST("/api/user/delete", adminHandler.DeleteUserAPI)
		adminGroup.POST("/api/key/create", adminHandler.CreateKeyAPI)
		adminGroup.POST("/api/key/reset", adminHandler.ResetKeyAPI)
		adminGroup.POST("/api/key/toggle", adminHandler.ToggleKey)
		adminGroup.POST("/api/key/delete", adminHandler.DeleteKey)

		// 模型管理 API（统一为 POST 风格，注意顺序：具体路径在前，动态路径在后）
		adminGroup.GET("/models", adminHandler.ListModelsAPI)
		adminGroup.POST("/models", adminHandler.CreateModelAPI)
		adminGroup.POST("/models/update", adminHandler.UpdateModelAPI)
		adminGroup.POST("/models/delete", adminHandler.DeleteModelAPI)
		adminGroup.GET("/models/:id", adminHandler.GetModelAPI)
		adminGroup.GET("/models/:id/pricing", adminHandler.GetModelPricingAPI)
		adminGroup.POST("/models/:id/pricing/update", adminHandler.UpdateModelPricingAPI)
	}

	// 旧版管理后台 HTML（保留）
	r.GET("/admin-dashboard", adminHandler.Dashboard)
	r.GET("/admin-users", adminHandler.Users)
	r.GET("/admin-keys", adminHandler.Keys)
	r.GET("/admin-usage", adminHandler.Usage)

	// 调试接口（仅开发环境使用）
	r.POST("/debug/init", gatewayHandler.InitUser)           // 初始化测试用户
	r.GET("/debug/keys", gatewayHandler.DebugListAllKeys)    // 列出所有 Keys
	r.GET("/debug/check", gatewayHandler.DebugCheckKey)      // 检查指定 Key

	// API v1 路由组
	v1 := r.Group("/v1")
	{
		// 公开接口
		v1.GET("/models", gatewayHandler.ListModels) // 模型列表

		// 需要认证的接口
		protected := v1.Group("")
		protected.Use(middleware.APIKeyAuth(authService))
		{
			// 聊天接口（兼容 OpenAI）
			protected.POST("/chat/completions", gatewayHandler.ChatComplete)
				// 文本补全接口（兼容 OpenAI）
				protected.POST("/completions", gatewayHandler.Completions)
			// Embeddings 接口（兼容 OpenAI）
			protected.POST("/embeddings", gatewayHandler.Embeddings)
			// Images 接口（兼容 OpenAI）
			protected.POST("/images/generations", gatewayHandler.Images)
			// Audio Transcription 接口（兼容 OpenAI）
			protected.POST("/audio/transcriptions", gatewayHandler.AudioTranscriptions)
			// 用户用量查询
			protected.GET("/usage", gatewayHandler.GetUserUsage)
			// 创建 API Key
			protected.GET("/me/balance", gatewayHandler.GetMyBalance)
			protected.GET("/me/usage", gatewayHandler.GetMyUsage)
			protected.POST("/keys", gatewayHandler.CreateAPIKey)
		}
	}
}
