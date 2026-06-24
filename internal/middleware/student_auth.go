package middleware

import (
	"strings"

	"web-backend/internal/config"
	"web-backend/internal/database"
	"web-backend/internal/dto"
	"web-backend/internal/models"
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

		// 检查账户是否被冻结
		frozen, err := models.IsUserFrozen(database.DB, claims.UserID)
		if err == nil && frozen {
			dto.Forbidden(c, "账户已被冻结，请联系管理员")
			c.Abort()
			return
		}

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
