package handlers

import (
	"log"
	"strconv"

	"web-backend/internal/database"
	"web-backend/internal/dto"
	"web-backend/internal/models"

	"github.com/gin-gonic/gin"
)

// GetFoods 获取食物列表
// @Summary 获取所有食物
// @Tags 食物管理
// @Security BearerAuth
// @Produce json
// @Success 200 {object} dto.Response
// @Router /api/foods [get]
func GetFoods() gin.HandlerFunc {
	return func(c *gin.Context) {
		foods, err := models.GetAllFoods(database.DB)
		if err != nil {
			dto.InternalError(c, "获取食物列表失败")
			return
		}
		dto.Success(c, foods)
	}
}

// CreateFood 添加食物
// @Summary 添加食物
// @Tags 食物管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body dto.CreateFoodRequest true "食物信息"
// @Success 200 {object} dto.Response
// @Router /api/foods [post]
func CreateFood() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.CreateFoodRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			dto.BadRequest(c, "缺少必要参数")
			return
		}

		food := &models.Food{
			Name:        req.Name,
			ShopName:    req.ShopName,
			CanteenName: req.CanteenName,
			Price:       req.Price,
			Taste:       req.Taste,
			Category:    req.Category,
		}

		if err := models.CreateFood(database.DB, food); err != nil {
			log.Printf("添加食物失败: %v", err)
			dto.InternalError(c, "添加食物失败")
			return
		}

		dto.SuccessMessage(c, "添加食物成功")
	}
}

// UpdateFood 更新食物
// @Summary 更新食物
// @Tags 食物管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "食物ID"
// @Param request body dto.UpdateFoodRequest true "食物信息"
// @Success 200 {object} dto.Response
// @Router /api/foods/{id} [put]
func UpdateFood() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的食物ID")
			return
		}

		var req dto.UpdateFoodRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			dto.BadRequest(c, "缺少必要参数")
			return
		}

		food := &models.Food{
			ID:          id,
			Name:        req.Name,
			ShopName:    req.ShopName,
			CanteenName: req.CanteenName,
			Price:       req.Price,
			Taste:       req.Taste,
			Category:    req.Category,
		}

		if err := models.UpdateFood(database.DB, food); err != nil {
			log.Printf("更新食物失败: %v", err)
			dto.InternalError(c, "更新食物失败")
			return
		}

		dto.SuccessMessage(c, "更新食物成功")
	}
}

// DeleteFood 删除食物
// @Summary 删除食物
// @Tags 食物管理
// @Security BearerAuth
// @Produce json
// @Param id path int true "食物ID"
// @Success 200 {object} dto.Response
// @Router /api/foods/{id} [delete]
func DeleteFood() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的食物ID")
			return
		}

		if err := models.DeleteFood(database.DB, id); err != nil {
			log.Printf("删除食物失败: %v", err)
			dto.InternalError(c, "删除食物失败")
			return
		}

		dto.SuccessMessage(c, "删除食物成功")
	}
}
