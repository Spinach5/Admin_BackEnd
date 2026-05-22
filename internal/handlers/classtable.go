package handlers

import (
	"log"

	"web-backend/internal/dto"

	"github.com/gin-gonic/gin"
)

// GetClasstable 获取课表
// @Summary 获取课表
// @Description 通过超星平台获取课表数据
// @Tags 课表
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body dto.ClasstableRequest true "课表查询参数"
// @Success 200 {object} dto.Response
// @Router /api/classtable [post]
func GetClasstable() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.ClasstableRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			dto.BadRequest(c, "请填写完整信息")
			return
		}

		// 课表查询代理 - 保留接口，具体实现依赖外部API
		log.Printf("课表查询请求: username=%s, year=%s, semester=%s", req.Username, req.Year, req.Semester)

		dto.Success(c, gin.H{
			"note":   "课表查询功能需要对接超星API，请联系管理员配置",
			"params": req,
		})
	}
}
