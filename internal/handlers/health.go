package handlers

import (
	"web-backend/internal/database"
	"web-backend/internal/dto"

	"github.com/gin-gonic/gin"
)

// HealthCheck 健康检查
// @Summary 健康检查
// @Description 检查服务器和数据库连接状态
// @Tags 系统
// @Produce json
// @Success 200 {object} dto.Response
// @Router /api/health [get]
func HealthCheck(c *gin.Context) {
	if err := database.HealthCheck(); err != nil {
		dto.Error(c, 503, "数据库连接失败")
		return
	}
	dto.SuccessMessage(c, "服务运行正常")
}
