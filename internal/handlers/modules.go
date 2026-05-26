package handlers

import (
	"web-backend/internal/dto"

	"github.com/gin-gonic/gin"
)

// GetModules 获取模块列表
// @Summary 获取模块列表
// @Description 返回前端导航模块列表
// @Tags 系统
// @Security BearerAuth
// @Produce json
// @Success 200 {object} dto.Response
// @Router /api/modules [get]
func GetModules() gin.HandlerFunc {
	return func(c *gin.Context) {
		modules := []gin.H{
			{"name": "管理员列表", "path": "/admin", "icon": "people"},
			{"name": "餐厅列表", "path": "/shops", "icon": "store"},
			{"name": "食物列表", "path": "/foods", "icon": "restaurant"},
			{"name": "事务列表", "path": "/affairs", "icon": "assignment"},
			{"name": "事务种类", "path": "/affair-categories", "icon": "category"},

		}
		dto.Success(c, modules)
	}
}
