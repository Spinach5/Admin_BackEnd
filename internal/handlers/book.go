package handlers

import (
	"log"
	"strconv"

	"web-backend/internal/database"
	"web-backend/internal/dto"
	"web-backend/internal/models"

	"github.com/gin-gonic/gin"
)

// GetBooks 获取书籍列表
func GetBooks() gin.HandlerFunc {
	return func(c *gin.Context) {
		books, err := models.GetAllBooks(database.DB)
		if err != nil {
			log.Printf("获取书籍列表失败: %v", err)
			dto.InternalError(c, "获取书籍列表失败")
			return
		}
		dto.Success(c, books)
	}
}

// GetBookByID 获取单本书籍
func GetBookByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的书籍ID")
			return
		}
		book, err := models.GetBookByID(database.DB, id)
		if err != nil {
			dto.Error(c, 404, "书籍不存在")
			return
		}
		dto.Success(c, book)
	}
}

// CreateBook 添加书籍
func CreateBook() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.CreateBookRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			dto.BadRequest(c, "缺少必要参数")
			return
		}

		book := &models.Book{
			Title:    req.Title,
			Category: req.Category,
			ImageURL: req.ImageURL,
			Price:    req.Price,
			ISBN:     req.ISBN,
			Contact:  req.Contact,
			UserID:   req.UserID,
			Status:   req.Status,
		}

		if err := models.CreateBook(database.DB, book); err != nil {
			log.Printf("添加书籍失败: %v", err)
			dto.InternalError(c, "添加书籍失败")
			return
		}

		dto.SuccessMessage(c, "添加书籍成功")
	}
}

// UpdateBook 更新书籍
func UpdateBook() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的书籍ID")
			return
		}

		var req dto.UpdateBookRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			dto.BadRequest(c, "缺少必要参数")
			return
		}

		book := &models.Book{
			BookID:   id,
			Title:    req.Title,
			Category: req.Category,
			ImageURL: req.ImageURL,
			Price:    req.Price,
			ISBN:     req.ISBN,
			Contact:  req.Contact,
			Status:   ptrVal(req.Status, "active"),
		}

		if err := models.UpdateBook(database.DB, book); err != nil {
			log.Printf("更新书籍失败: %v", err)
			dto.InternalError(c, "更新书籍失败")
			return
		}

		dto.SuccessMessage(c, "更新书籍成功")
	}
}

// DeleteBook 删除书籍
func DeleteBook() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的书籍ID")
			return
		}

		if err := models.DeleteBook(database.DB, id); err != nil {
			log.Printf("删除书籍失败: %v", err)
			dto.InternalError(c, "删除书籍失败")
			return
		}

		dto.SuccessMessage(c, "删除书籍成功")
	}
}

func ptrVal(s *string, def string) string {
	if s == nil {
		return def
	}
	return *s
}
