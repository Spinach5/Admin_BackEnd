package handlers

import (
	"log"
	"strconv"

	"web-backend/internal/database"
	"web-backend/internal/dto"
	"web-backend/internal/models"

	"github.com/gin-gonic/gin"
)

// GetAffairs 获取事务列表
// @Summary 获取所有事务
// @Tags 事务管理
// @Security BearerAuth
// @Produce json
// @Success 200 {object} dto.Response
// @Router /api/affairs [get]
func GetAffairs() gin.HandlerFunc {
	return func(c *gin.Context) {
		affairs, err := models.GetAllAffairs(database.DB)
		if err != nil {
			dto.InternalError(c, "获取事务列表失败")
			return
		}
		dto.Success(c, affairs)
	}
}

// CreateAffair 添加事务
// @Summary 添加事务
// @Tags 事务管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body dto.CreateAffairRequest true "事务信息"
// @Success 200 {object} dto.Response
// @Router /api/affairs [post]
func CreateAffair() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.CreateAffairRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			dto.BadRequest(c, "缺少必要参数")
			return
		}

		affair := &models.Affair{
			Name:     req.Name,
			Category: req.Category,
			Link:     req.Link,
			Details:  req.Details,
			Channel:  req.Channel,
			SchoolID: req.SchoolID,
		}

		if err := models.CreateAffair(database.DB, affair); err != nil {
			log.Printf("添加事务失败: %v", err)
			dto.InternalError(c, "添加事务失败")
			return
		}

		dto.SuccessMessage(c, "添加事务成功")
	}
}

// UpdateAffair 更新事务
// @Summary 更新事务
// @Tags 事务管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "事务ID"
// @Param request body dto.UpdateAffairRequest true "事务信息"
// @Success 200 {object} dto.Response
// @Router /api/affairs/{id} [put]
func UpdateAffair() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的事务ID")
			return
		}

		var req dto.UpdateAffairRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			dto.BadRequest(c, "缺少必要参数")
			return
		}

		affair := &models.Affair{
			ID:       id,
			Name:     req.Name,
			Category: req.Category,
			Link:     req.Link,
			Details:  req.Details,
			Channel:  req.Channel,
			SchoolID: req.SchoolID,
		}

		if err := models.UpdateAffair(database.DB, affair); err != nil {
			log.Printf("更新事务失败: %v", err)
			dto.InternalError(c, "更新事务失败")
			return
		}

		dto.SuccessMessage(c, "更新事务成功")
	}
}

// DeleteAffair 删除事务
// @Summary 删除事务
// @Tags 事务管理
// @Security BearerAuth
// @Produce json
// @Param id path int true "事务ID"
// @Success 200 {object} dto.Response
// @Router /api/affairs/{id} [delete]
func DeleteAffair() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的事务ID")
			return
		}

		if err := models.DeleteAffair(database.DB, id); err != nil {
			log.Printf("删除事务失败: %v", err)
			dto.InternalError(c, "删除事务失败")
			return
		}

		dto.SuccessMessage(c, "删除事务成功")
	}
}
