// Package middleware HTTP 中间件包
// 提供请求日志、认证、CORS 等中间件
package middleware

import (
	"time"

	"ai-gateway/internal/logger"

	"github.com/gin-gonic/gin"
)

// RequestLogger 请求日志中间件
// 记录每个请求的方法、路径、状态码、耗时、客户端IP
// 返回格式: METHOD PATH STATUS LATENCY CLIENT_IP
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 记录开始时间
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method
		
		// 执行后续处理
		c.Next()
		
		// 计算耗时
		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		
		// 根据状态码选择日志级别
		if status >= 400 {
			logger.Warn("%s %s %d %v %s",
				method, path, status, latency, clientIP)
		} else {
			logger.Info("%s %s %d %v %s",
				method, path, status, latency, clientIP)
		}
	}
}

// Recovery Panic 恢复中间件
// 捕获 panic 并返回 500 错误，防止服务崩溃
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 记录 panic 信息
				logger.Error("Panic recovered: %v", err)
				// 返回 500 错误
				c.AbortWithStatusJSON(500, gin.H{
					"error": "Internal server error",
				})
			}
		}()
		c.Next()
	}
}
