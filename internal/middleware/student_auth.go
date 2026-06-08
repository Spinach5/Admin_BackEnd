package middleware

import (
	"database/sql"
	"log"
	"strings"
	"time"

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

		// 心跳超时检测：超过 5 分钟未活跃则拒绝请求
		lastActive, err := models.GetUserLastActive(database.DB, claims.UserID)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("查询用户活跃时间失败 userId=%d: %v", claims.UserID, err)
		}
		if lastActive != "" {
			t, parseErr := time.Parse("2006-01-02 15:04:05", lastActive[:19])
			if parseErr == nil && time.Since(t) > 5*time.Minute {
				dto.Unauthorized(c, "会话已过期，请重新登录")
				c.Abort()
				return
			}
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
