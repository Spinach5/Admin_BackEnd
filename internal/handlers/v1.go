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

// V1GetBooks 获取书籍列表 (全量)
func V1GetBooks() gin.HandlerFunc {
	return func(c *gin.Context) {
		books, err := models.GetAllBooks(database.DB)
		if err != nil {
			log.Printf("V1获取书籍列表失败: %v", err)
			dto.InternalError(c, "获取书籍列表失败")
			return
		}
		dto.Success(c, books)
	}
}

// V1AddBook 添加书籍 (最多5本活跃)
func V1AddBook() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.V1AddBookRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			dto.BadRequest(c, "缺少必要参数")
			return
		}

		userID := c.GetInt("v1_user_id")

		count, err := models.CountActiveBooksByUser(database.DB, userID)
		if err != nil {
			log.Printf("V1查询书籍数量失败: %v", err)
			dto.InternalError(c, "添加书籍失败")
			return
		}
		if count >= 5 {
			dto.BadRequest(c, "已达发布上限（5本）")
			return
		}

		book := &models.Book{
			Title:    req.Title,
			Category: req.Category,
			ImageURL: req.ImageURL,
			Price:    req.Price,
			ISBN:     req.ISBN,
			Contact:  req.Contact,
			UserID:   userID,
			Status:   "active",
		}

		if err := models.CreateBook(database.DB, book); err != nil {
			log.Printf("V1添加书籍失败: %v", err)
			dto.InternalError(c, "添加书籍失败")
			return
		}

		dto.SuccessMessage(c, "添加书籍成功")
	}
}

// V1DeleteBook 软删除书籍 (只能删自己的)
func V1DeleteBook() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.V1DeleteBookRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			dto.BadRequest(c, "缺少必要参数")
			return
		}

		userID := c.GetInt("v1_user_id")

		if err := models.SoftDeleteBookByUser(database.DB, req.BookID, userID); err != nil {
			dto.BadRequest(c, err.Error())
			return
		}

		dto.SuccessMessage(c, "删除书籍成功")
	}
}
