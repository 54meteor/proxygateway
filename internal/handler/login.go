package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Login 登录接口
// 用于 Pure Admin 前端登录
func (h *GatewayHandler) Login(c *gin.Context) {
	// 直接返回成功，允许任意账号登录
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"avatar":      "https://avatars.githubusercontent.com/u/44761321",
			"username":    "admin",
			"nickname":    "管理员",
			"roles":       []string{"admin"},
			"permissions": []string{"*:*:*"},
			"accessToken": "mock-admin-token",
			"refreshToken": "mock-refresh-token",
			"expires":    "2030/10/30 00:00:00",
		},
	})
}

// GetUserInfo 获取用户信息
func (h *GatewayHandler) GetUserInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"avatar":      "https://avatars.githubusercontent.com/u/44761321",
			"username":    "admin",
			"nickname":    "管理员",
			"roles":       []string{"admin"},
			"permissions": []string{"*:*:*"},
		},
	})
}

// GetAsyncRoutes 获取动态路由
func (h *GatewayHandler) GetAsyncRoutes(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": []gin.H{
			{
				"path": "/gateway",
				"name": "Gateway",
				"redirect": "/gateway/dashboard",
				"meta": gin.H{
					"icon":   "ep/gateway",
					"title":  "AI Gateway",
					"rank":   10,
				},
				"children": []gin.H{
					{
						"path": "/gateway/dashboard",
						"name":  "GatewayDashboard",
						"meta":  gin.H{"title": "仪表盘"},
					},
					{
						"path": "/gateway/users",
						"name":  "GatewayUsers",
						"meta":  gin.H{"title": "用户管理"},
					},
					{
						"path": "/gateway/keys",
						"name":  "GatewayKeys",
						"meta":  gin.H{"title": "API Keys"},
					},
					{
						"path": "/gateway/usage",
						"name":  "GatewayUsage",
						"meta":  gin.H{"title": "用量统计"},
					},
				},
			},
		},
	})
}

// RefreshToken 刷新 Token
func (h *GatewayHandler) RefreshToken(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"accessToken": "mock-admin-token",
			"expires":    "2030/10/30 00:00:00",
		},
	})
}
