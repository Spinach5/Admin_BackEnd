package handlers

import (
	"log"

	"web-backend/internal/database"
	"web-backend/internal/dto"
	"web-backend/internal/models"

	"github.com/gin-gonic/gin"
)

// V1GetFoods 获取食物列表 (按 schoolId 过滤)
func V1GetFoods() gin.HandlerFunc {
	return func(c *gin.Context) {
		schoolID := c.GetString("v1_school_id")
		foods, err := models.GetFoodsBySchool(database.DB, schoolID)
		if err != nil {
			log.Printf("V1获取食物列表失败: %v", err)
			dto.InternalError(c, "获取食物列表失败")
			return
		}
		dto.Success(c, foods)
	}
}

// V1GetShops 获取店铺列表 (按 schoolId 过滤)
func V1GetShops() gin.HandlerFunc {
	return func(c *gin.Context) {
		schoolID := c.GetString("v1_school_id")
		shops, err := models.GetShopsBySchool(database.DB, schoolID)
		if err != nil {
			log.Printf("V1获取店铺列表失败: %v", err)
			dto.InternalError(c, "获取店铺列表失败")
			return
		}
		dto.Success(c, shops)
	}
}

// V1GetAffairs 获取事务列表 (按 schoolId 过滤)
func V1GetAffairs() gin.HandlerFunc {
	return func(c *gin.Context) {
		schoolID := c.GetString("v1_school_id")
		affairs, err := models.GetAffairsBySchool(database.DB, schoolID)
		if err != nil {
			log.Printf("V1获取事务列表失败: %v", err)
			dto.InternalError(c, "获取事务列表失败")
			return
		}
		dto.Success(c, affairs)
	}
}
