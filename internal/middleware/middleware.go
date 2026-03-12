// Package middleware HTTP 中间件包
package middleware

import (
	"net/http"
	"strings"

	"ai-gateway/internal/service/auth"

	"github.com/gin-gonic/gin"
)

// APIKeyAuth API Key 认证中间件
// 从 Authorization Header 中提取 API Key 并验证
// 验证通过后将 user_id 和 api_key_id 存入 Context
func APIKeyAuth(authService *auth.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 Header 获取 API Key
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		// 解析格式: Bearer <api_key>
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			c.Abort()
			return
		}

		apiKey := parts[1]

		// 验证 Key，获取 userID 和 apiKeyID
		userID, apiKeyID, err := authService.ValidateAPIKeyFull(apiKey)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid API key"})
			c.Abort()
			return
		}

		// 将 userID 和 apiKeyID 存入 Context，供后续处理器使用
		c.Set("user_id", userID)
		c.Set("api_key_id", apiKeyID)
		c.Set("api_key", apiKey)

		c.Next()
	}
}

// CORSMiddleware CORS 中间件
// 允许跨域请求
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置允许的响应头
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		// 处理预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// RateLimitMiddleware 限流中间件 (简化版)
// TODO: 实现 Redis 分布式限流
func RateLimitMiddleware(requestsPerMinute int) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 暂未实现
		c.Next()
	}
}
