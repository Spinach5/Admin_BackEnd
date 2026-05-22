package handlers

import (
	"log"
	"strconv"

	"web-backend/internal/database"
	"web-backend/internal/dto"
	"web-backend/internal/models"

	"github.com/gin-gonic/gin"
)

// GetShops 获取餐厅列表
// @Summary 获取所有餐厅
// @Tags 餐厅管理
// @Security BearerAuth
// @Produce json
// @Success 200 {object} dto.Response
// @Router /api/shops [get]
func GetShops() gin.HandlerFunc {
	return func(c *gin.Context) {
		shops, err := models.GetAllShops(database.DB)
		if err != nil {
			dto.InternalError(c, "获取餐厅列表失败")
			return
		}
		dto.Success(c, shops)
	}
}

// CreateShop 添加餐厅
// @Summary 添加餐厅
// @Tags 餐厅管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body dto.CreateShopRequest true "餐厅信息"
// @Success 200 {object} dto.Response
// @Router /api/shops [post]
func CreateShop() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.CreateShopRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			dto.BadRequest(c, "缺少必要参数")
			return
		}

		shop := &models.Shop{
			Name:        req.Name,
			CanteenName: req.CanteenName,
			Rating:      req.Rating,
			Comment:     req.Comment,
			Min:         req.Min,
			Max:         req.Max,
		}

		if err := models.CreateShop(database.DB, shop); err != nil {
			log.Printf("添加餐厅失败: %v", err)
			dto.InternalError(c, "添加餐厅失败")
			return
		}

		dto.SuccessMessage(c, "添加餐厅成功")
	}
}

// UpdateShop 更新餐厅
// @Summary 更新餐厅
// @Tags 餐厅管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "餐厅ID"
// @Param request body dto.UpdateShopRequest true "餐厅信息"
// @Success 200 {object} dto.Response
// @Router /api/shops/{id} [put]
func UpdateShop() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的餐厅ID")
			return
		}

		var req dto.UpdateShopRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			dto.BadRequest(c, "缺少必要参数")
			return
		}

		shop := &models.Shop{
			ID:          id,
			Name:        req.Name,
			CanteenName: req.CanteenName,
			Rating:      req.Rating,
			Comment:     req.Comment,
			Min:         req.Min,
			Max:         req.Max,
		}

		if err := models.UpdateShop(database.DB, shop); err != nil {
			log.Printf("更新餐厅失败: %v", err)
			dto.InternalError(c, "更新餐厅失败")
			return
		}

		dto.SuccessMessage(c, "更新餐厅成功")
	}
}

// DeleteShop 删除餐厅
// @Summary 删除餐厅
// @Tags 餐厅管理
// @Security BearerAuth
// @Produce json
// @Param id path int true "餐厅ID"
// @Success 200 {object} dto.Response
// @Router /api/shops/{id} [delete]
func DeleteShop() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的餐厅ID")
			return
		}

		if err := models.DeleteShop(database.DB, id); err != nil {
			log.Printf("删除餐厅失败: %v", err)
			dto.InternalError(c, "删除餐厅失败")
			return
		}

		dto.SuccessMessage(c, "删除餐厅成功")
	}
}
