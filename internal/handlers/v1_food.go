package handlers

import (
	"log"

	"web-backend/internal/database"
	"web-backend/internal/dto"
	"web-backend/internal/middleware"
	"web-backend/internal/models"

	"github.com/gin-gonic/gin"
)

// V1GetFoods 学生获取食物列表 (JWT StudentAuth, 按学校过滤)
// @Summary 获取食物列表
// @Tags 美食-V1
// @Security BearerAuth
// @Produce json
// @Param category query string false "分类"
// @Param taste query string false "口味"
// @Param canteen_name query string false "食堂名称"
// @Param shop_name query string false "店铺名称"
// @Param search query string false "搜索关键词（匹配名称）"
// @Success 200 {object} dto.Response
// @Router /api/v1/foods [get]
func V1GetFoods() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.GetStudentUserID(c)
		if userID == 0 {
			dto.Unauthorized(c, "未登录")
			return
		}

		user, err := models.GetUserByID(database.DB, userID)
		if err != nil {
			log.Printf("V1获取用户信息失败 userId=%d: %v", userID, err)
			dto.Unauthorized(c, "用户不存在")
			return
		}

		category := c.Query("category")
		taste := c.Query("taste")
		canteenName := c.Query("canteen_name")
		shopName := c.Query("shop_name")
		search := c.Query("search")

		foods, err := models.GetFoodsBySchoolWithFilters(
			database.DB,
			user.SchoolID,
			category,
			taste,
			canteenName,
			shopName,
			search,
		)
		if err != nil {
			log.Printf("V1获取食物列表失败 schoolId=%s: %v", user.SchoolID, err)
			dto.InternalError(c, "获取食物列表失败")
			return
		}

		dto.Success(c, foods)
	}
}

// V1GetFoodFilters 学生获取食物筛选选项（分类、口味、食堂、店铺）
// @Summary 获取食物筛选选项
// @Tags 美食-V1
// @Security BearerAuth
// @Produce json
// @Success 200 {object} dto.Response
// @Router /api/v1/foods/filters [get]
func V1GetFoodFilters() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.GetStudentUserID(c)
		if userID == 0 {
			dto.Unauthorized(c, "未登录")
			return
		}

		user, err := models.GetUserByID(database.DB, userID)
		if err != nil {
			log.Printf("V1获取用户信息失败 userId=%d: %v", userID, err)
			dto.Unauthorized(c, "用户不存在")
			return
		}

		categories, err := models.GetDistinctCategoriesBySchool(database.DB, user.SchoolID)
		if err != nil {
			log.Printf("V1获取食物分类失败 schoolId=%s: %v", user.SchoolID, err)
			dto.InternalError(c, "获取筛选选项失败")
			return
		}

		tastes, err := models.GetDistinctTastesBySchool(database.DB, user.SchoolID)
		if err != nil {
			log.Printf("V1获取食物口味失败 schoolId=%s: %v", user.SchoolID, err)
			dto.InternalError(c, "获取筛选选项失败")
			return
		}

		canteens, err := models.GetDistinctCanteensBySchool(database.DB, user.SchoolID)
		if err != nil {
			log.Printf("V1获取食堂列表失败 schoolId=%s: %v", user.SchoolID, err)
			dto.InternalError(c, "获取筛选选项失败")
			return
		}

		shops, err := models.GetDistinctShopsBySchool(database.DB, user.SchoolID)
		if err != nil {
			log.Printf("V1获取店铺列表失败 schoolId=%s: %v", user.SchoolID, err)
			dto.InternalError(c, "获取筛选选项失败")
			return
		}

		dto.Success(c, gin.H{
			"categories": categories,
			"tastes":     tastes,
			"canteens":   canteens,
			"shops":      shops,
		})
	}
}
