package middleware

import (
	"strings"

	"web-backend/internal/config"
	"web-backend/internal/dto"
	"web-backend/internal/services"

	"github.com/gin-gonic/gin"
)

func StudentAuth(cfg *config.Config) gin.HandlerFunc {
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

		c.Set("student_user_id", claims.UserID)

		// 令牌永不过期，仅在用户主动注销时失效
		// 心跳端点 /api/v1/auth/heartbeat 仍可用于更新 last_active_at（用于统计/监控）

		c.Next()
	}
}

func GetStudentUserID(c *gin.Context) int {
	id, _ := c.Get("student_user_id")
	if id == nil {
		return 0
	}
	return id.(int)
}
