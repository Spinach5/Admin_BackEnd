package middleware

import (
	"strings"

	"web-backend/internal/config"
	"web-backend/internal/dto"
	"web-backend/internal/services"

	"github.com/gin-gonic/gin"
)

func Auth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			dto.Unauthorized(c, "未提供认证令牌")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			dto.Unauthorized(c, "认证格式错误")
			c.Abort()
			return
		}

		claims, err := services.ParseToken(parts[1], cfg.JWTSecret)
		if err != nil {
			dto.Unauthorized(c, "令牌无效或已过期")
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("account", claims.Account)
		c.Set("is_super", claims.IsSuper)

		c.Next()
	}
}

func RequireSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		isSuper, _ := c.Get("is_super")
		if isSuper == nil || isSuper.(int) != 1 {
			dto.Forbidden(c, "权限不足：仅超级管理员可执行此操作")
			c.Abort()
			return
		}
		c.Next()
	}
}

func GetCurrentAccount(c *gin.Context) string {
	account, _ := c.Get("account")
	if account == nil {
		return ""
	}
	return account.(string)
}

func GetCurrentUserID(c *gin.Context) int {
	id, _ := c.Get("user_id")
	if id == nil {
		return 0
	}
	return id.(int)
}

func GetCurrentIsSuper(c *gin.Context) int {
	isSuper, _ := c.Get("is_super")
	if isSuper == nil {
		return 0
	}
	return isSuper.(int)
}
