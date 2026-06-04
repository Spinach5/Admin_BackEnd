package handlers

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"web-backend/internal/database"
	"web-backend/internal/dto"
	"web-backend/internal/middleware"
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

// CreateBook 添加书籍 (admin)
func CreateBook() gin.HandlerFunc {
	return func(c *gin.Context) {
		title := c.PostForm("title")
		category := c.PostForm("category")
		price := c.PostForm("price")
		isbn := c.PostForm("isbn")
		contact := c.PostForm("contact")
		status := c.PostForm("status")

		if title == "" {
			dto.BadRequest(c, "书名不能为空")
			return
		}
		if status == "" {
			status = "active"
		}

		if category != "" && !bookCategories[category] {
			dto.BadRequest(c, "无效的书籍种类")
			return
		}

		var imageURL string
		file, err := c.FormFile("image")
		if err == nil {
			url, err := saveUploadedImage(c, file)
			if err != nil {
				log.Printf("图片上传失败: %v", err)
				dto.BadRequest(c, err.Error())
				return
			}
			imageURL = url
		} else if err != http.ErrMissingFile {
			log.Printf("读取上传文件失败: %v", err)
		}

		userID := middleware.GetCurrentUserID(c)

		book := &models.Book{
			Title:    title,
			Category: strPtr(category),
			ImageURL: strPtr(imageURL),
			Price:    strPtr(price),
			ISBN:     strPtr(isbn),
			Contact:  strPtr(contact),
			UserID:   userID,
			Status:   status,
		}

		if err := models.CreateBook(database.DB, book); err != nil {
			log.Printf("添加书籍失败: %v", err)
			dto.InternalError(c, "添加书籍失败")
			return
		}

		dto.SuccessMessage(c, "添加书籍成功")
	}
}

// UpdateBook 更新书籍 (admin)
func UpdateBook() gin.HandlerFunc {
	return func(c *gin.Context) {
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

		title := c.PostForm("title")
		if title == "" {
			dto.BadRequest(c, "书名不能为空")
			return
		}

		category := c.PostForm("category")
		if category != "" && !bookCategories[category] {
			dto.BadRequest(c, "无效的书籍种类")
			return
		}

		imageURL := ptrStrVal(existing.ImageURL)
		file, err := c.FormFile("image")
		if err == nil {
			url, err := saveUploadedImage(c, file)
			if err != nil {
				log.Printf("图片上传失败: %v", err)
				dto.BadRequest(c, err.Error())
				return
			}
			imageURL = url
		} else if err != http.ErrMissingFile {
			log.Printf("读取上传文件失败: %v", err)
		}

		if c.PostForm("delete_image") == "true" {
			deleteImageFile(imageURL)
			imageURL = ""
		}

		book := &models.Book{
			BookID:   id,
			Title:    title,
			Category: strPtr(category),
			ImageURL: strPtr(imageURL),
			Price:    strPtr(c.PostForm("price")),
			ISBN:     strPtr(c.PostForm("isbn")),
			Contact:  strPtr(c.PostForm("contact")),
			Status:   existing.Status,
		}

		stat := c.PostForm("status")
		if stat != "" {
			book.Status = stat
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

var BookCategoryList = []string{"数学", "外语", "计算机", "理工类", "思政类", "文学类", "经管类", "其他"}

var bookCategories = func() map[string]bool {
	m := make(map[string]bool, len(BookCategoryList))
	for _, c := range BookCategoryList {
		m[c] = true
	}
	return m
}()

// GetBookCategories 获取书籍种类列表
func GetBookCategories() gin.HandlerFunc {
	return func(c *gin.Context) {
		dto.Success(c, BookCategoryList)
	}
}

func uploadDir() string {
	dir := os.Getenv("UPLOAD_DIR")
	if dir == "" {
		dir = "./uploads"
	}
	return dir
}

func deleteImageFile(imageURL string) {
	if imageURL == "" {
		return
	}
	filename := filepath.Base(imageURL)
	dst := filepath.Join(uploadDir(), filename)
	if err := os.Remove(dst); err != nil {
		log.Printf("删除图片文件失败 %s: %v", dst, err)
	}
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
