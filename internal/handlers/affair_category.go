package handlers

import (
	"log"
	"strconv"

	"web-backend/internal/database"
	"web-backend/internal/dto"
	"web-backend/internal/models"

	"github.com/gin-gonic/gin"
)

// GetAffairCategories 获取事务种类列表
// @Summary 获取所有事务种类
// @Tags 事务种类管理
// @Security BearerAuth
// @Produce json
// @Success 200 {object} dto.Response
// @Router /api/affair-categories [get]
func GetAffairCategories() gin.HandlerFunc {
	return func(c *gin.Context) {
		cats, err := models.GetAllAffairCategories(database.DB)
		if err != nil {
			dto.InternalError(c, "获取事务种类列表失败")
			return
		}
		dto.Success(c, cats)
	}
}

// CreateAffairCategory 添加事务种类
// @Summary 添加事务种类
// @Tags 事务种类管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body dto.CreateAffairCategoryRequest true "事务种类信息"
// @Success 200 {object} dto.Response
// @Router /api/affair-categories [post]
func CreateAffairCategory() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.CreateAffairCategoryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			dto.BadRequest(c, "缺少事务种类名称")
			return
		}

		if err := models.CreateAffairCategory(database.DB, req.Name); err != nil {
			log.Printf("添加事务种类失败: %v", err)
			dto.InternalError(c, "添加事务种类失败")
			return
		}

		dto.SuccessMessage(c, "添加事务种类成功")
	}
}

// UpdateAffairCategory 更新事务种类
// @Summary 更新事务种类
// @Tags 事务种类管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "事务种类ID"
// @Param request body dto.UpdateAffairCategoryRequest true "事务种类信息"
// @Success 200 {object} dto.Response
// @Router /api/affair-categories/{id} [put]
func UpdateAffairCategory() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的事务种类ID")
			return
		}

		var req dto.UpdateAffairCategoryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			dto.BadRequest(c, "缺少必要参数")
			return
		}

		if err := models.UpdateAffairCategory(database.DB, id, req.Name); err != nil {
			log.Printf("更新事务种类失败: %v", err)
			dto.InternalError(c, "更新事务种类失败")
			return
		}

		dto.SuccessMessage(c, "更新事务种类成功")
	}
}

// DeleteAffairCategory 删除事务种类
// @Summary 删除事务种类
// @Tags 事务种类管理
// @Security BearerAuth
// @Produce json
// @Param id path int true "事务种类ID"
// @Success 200 {object} dto.Response
// @Router /api/affair-categories/{id} [delete]
func DeleteAffairCategory() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的事务种类ID")
			return
		}

		if err := models.DeleteAffairCategory(database.DB, id); err != nil {
			log.Printf("删除事务种类失败: %v", err)
			dto.InternalError(c, "删除事务种类失败")
			return
		}

		dto.SuccessMessage(c, "删除事务种类成功")
	}
}
