package handlers

import (
	"fmt"
	"log"
	"mime/multipart"
	"path/filepath"
	"strconv"
	"strings"

	"web-backend/internal/database"
	"web-backend/internal/dto"
	"web-backend/internal/middleware"
	"web-backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func V1GetBooks() gin.HandlerFunc {
	return func(c *gin.Context) {
		category := c.Query("category")
		var books []models.BookWithUser
		var err error
		if category != "" {
			books, err = models.GetBooksByCategory(database.DB, category)
		} else {
			books, err = models.GetAllBooks(database.DB)
		}
		if err != nil {
			log.Printf("V1获取书籍列表失败: %v", err)
			dto.InternalError(c, "获取书籍列表失败")
			return
		}
		dto.Success(c, books)
	}
}

func V1GetMyBooks() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.GetStudentUserID(c)
		books, err := models.GetBooksByUser(database.DB, userID)
		if err != nil {
			log.Printf("V1获取我的书籍失败: %v", err)
			dto.InternalError(c, "获取我的书籍失败")
			return
		}
		dto.Success(c, books)
	}
}

func V1GetBookByID() gin.HandlerFunc {
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

func V1CreateBook() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.GetStudentUserID(c)

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

		title := c.PostForm("title")
		category := c.PostForm("category")
		price := c.PostForm("price")
		isbn := c.PostForm("isbn")
		contact := c.PostForm("contact")

		if title == "" {
			dto.BadRequest(c, "书名不能为空")
			return
		}

		var imageURL string
		file, err := c.FormFile("image")
		if err == nil {
			url, err := saveUploadedImage(c, file)
			if err != nil {
				dto.BadRequest(c, err.Error())
				return
			}
			imageURL = url
		}

		book := &models.Book{
			Title:    title,
			Category: strPtr(category),
			ImageURL: strPtr(imageURL),
			Price:    strPtr(price),
			ISBN:     strPtr(isbn),
			Contact:  strPtr(contact),
			UserID:   userID,
			Status:   "active",
		}

		if err := models.CreateBook(database.DB, book); err != nil {
			log.Printf("V1添加书籍失败: %v", err)
			dto.InternalError(c, "添加书籍失败")
			return
		}

		dto.SuccessMessage(c, "发布成功")
	}
}

func V1UpdateBook() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.GetStudentUserID(c)
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的书籍ID")
			return
		}

		existing, err := models.GetBookByID(database.DB, id)
		if err != nil {
			dto.Error(c, 404, "书籍不存在")
			return
		}
		if existing.UserID != userID {
			dto.Forbidden(c, "只能编辑自己的书籍")
			return
		}

		title := c.PostForm("title")
		if title == "" {
			dto.BadRequest(c, "书名不能为空")
			return
		}

		imageURL := ptrStrVal(existing.ImageURL)
		file, err := c.FormFile("image")
		if err == nil {
			url, err := saveUploadedImage(c, file)
			if err != nil {
				dto.BadRequest(c, err.Error())
				return
			}
			imageURL = url
		}

		book := &models.Book{
			BookID:   id,
			Title:    title,
			Category: strPtr(c.PostForm("category")),
			ImageURL: strPtr(imageURL),
			Price:    strPtr(c.PostForm("price")),
			ISBN:     strPtr(c.PostForm("isbn")),
			Contact:  strPtr(c.PostForm("contact")),
			Status:   existing.Status,
		}

		if err := models.UpdateBook(database.DB, book); err != nil {
			log.Printf("V1更新书籍失败: %v", err)
			dto.InternalError(c, "更新书籍失败")
			return
		}

		dto.SuccessMessage(c, "更新成功")
	}
}

func V1DeleteBook() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.GetStudentUserID(c)
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的书籍ID")
			return
		}

		if err := models.SoftDeleteBookByUser(database.DB, id, userID); err != nil {
			dto.BadRequest(c, err.Error())
			return
		}

		dto.SuccessMessage(c, "删除成功")
	}
}

func saveUploadedImage(c *gin.Context, file *multipart.FileHeader) (string, error) {
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true}
	if !allowed[ext] {
		return "", fmt.Errorf("不支持的图片格式，仅支持 jpg/jpeg/png/webp")
	}

	if file.Size > 5<<20 {
		return "", fmt.Errorf("图片大小不能超过5MB")
	}

	filename := uuid.New().String() + ext
	dst := filepath.Join("uploads", filename)
	if err := c.SaveUploadedFile(file, dst); err != nil {
		return "", fmt.Errorf("图片上传失败")
	}

	return "/uploads/" + filename, nil
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func ptrStrVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
