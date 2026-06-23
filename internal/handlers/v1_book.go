package handlers

import (
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"os"
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
		keyword := c.Query("keyword")
		schoolID := c.Query("school_id")
		if schoolID == "" {
			schoolID = "hbut"
		}

		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
		if page < 1 {
			page = 1
		}
		if pageSize < 1 || pageSize > 100 {
			pageSize = 20
		}

		books, total, err := models.GetBooksPaginated(database.DB, category, keyword, schoolID, page, pageSize)
		if err != nil {
			log.Printf("V1获取书籍列表失败: %v", err)
			dto.InternalError(c, "获取书籍列表失败")
			return
		}

		dto.SuccessWithTotal(c, books, total)
	}
}

func V1GetMyBooks() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.GetStudentUserID(c)

		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
		if page < 1 {
			page = 1
		}
		if pageSize < 1 || pageSize > 100 {
			pageSize = 20
		}

		books, total, err := models.GetMyBooksPaginated(database.DB, userID, page, pageSize)
		if err != nil {
			log.Printf("V1获取我的书籍失败: %v", err)
			dto.InternalError(c, "获取我的书籍失败")
			return
		}

		dto.SuccessWithTotal(c, books, total)
	}
}

func V1GetBookByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的书籍ID")
			return
		}

		currentUserID := middleware.GetStudentUserID(c)

		detail, err := models.GetBookDetail(database.DB, id, currentUserID)
		if err != nil {
			dto.Error(c, 404, "书籍不存在")
			return
		}

		dto.Success(c, detail)
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

		title := strings.TrimSpace(c.PostForm("title"))
		category := c.PostForm("category")
		price := c.PostForm("price")
		isbn := c.PostForm("isbn")
		contact := c.PostForm("contact")
		description := c.PostForm("description")
		condition := c.PostForm("condition")
		schoolID := c.PostForm("school_id")
		if schoolID == "" {
			schoolID = "hbut"
		}

		if title == "" {
			dto.BadRequest(c, "书名不能为空")
			return
		}

		// Parse comma-separated image URLs
		imageURLsRaw := c.PostForm("image_urls")
		var imageURLs []string
		if imageURLsRaw != "" {
			for _, u := range strings.Split(imageURLsRaw, ",") {
				u = strings.TrimSpace(u)
				if u != "" {
					imageURLs = append(imageURLs, u)
				}
			}
		}
		if len(imageURLs) > 3 {
			dto.BadRequest(c, "最多上传3张图片")
			return
		}

		// Also support single legacy file upload
		var firstImageURL string
		file, err := c.FormFile("image")
		if err == nil {
			url, err := saveUploadedImage(c, file)
			if err != nil {
				log.Printf("V1图片上传失败: %v", err)
				dto.BadRequest(c, err.Error())
				return
			}
			firstImageURL = url
			imageURLs = append([]string{url}, imageURLs...)
		} else if err != http.ErrMissingFile {
			log.Printf("V1读取上传文件失败: %v", err)
		}

		if len(imageURLs) > 3 {
			dto.BadRequest(c, "最多上传3张图片")
			return
		}

		if len(imageURLs) > 0 {
			firstImageURL = imageURLs[0]
		}

		if condition == "" {
			condition = "几乎全新"
		}

		book := &models.Book{
			Title:       title,
			Category:    strPtr(category),
			ImageURL:    strPtr(firstImageURL),
			Price:       strPtr(price),
			ISBN:        strPtr(isbn),
			Contact:     strPtr(contact),
			Description: strPtr(description),
			Condition:   strPtr(condition),
			SchoolID:    schoolID,
			UserID:      userID,
			Status:      "active",
		}

		if err := models.CreateBookWithImages(database.DB, book, imageURLs); err != nil {
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

		title := strings.TrimSpace(c.PostForm("title"))
		if title == "" {
			dto.BadRequest(c, "书名不能为空")
			return
		}

		category := c.PostForm("category")
		if category == "" {
			category = ptrStrVal(existing.Category)
		}

		// Parse image URLs
		imageURLsRaw := c.PostForm("image_urls")
		var imageURLs []string
		if imageURLsRaw != "" {
			for _, u := range strings.Split(imageURLsRaw, ",") {
				u = strings.TrimSpace(u)
				if u != "" {
					imageURLs = append(imageURLs, u)
				}
			}
		}
		if len(imageURLs) > 3 {
			dto.BadRequest(c, "最多上传3张图片")
			return
		}

		// If no new image_urls submitted, preserve existing images
		if imageURLsRaw == "" {
			existingImages, _ := models.GetBookImages(database.DB, id)
			for _, img := range existingImages {
				imageURLs = append(imageURLs, img.ImageURL)
			}
		}

		// Legacy single file upload
		file, err := c.FormFile("image")
		if err == nil {
			url, err := saveUploadedImage(c, file)
			if err != nil {
				log.Printf("V1图片上传失败: %v", err)
				dto.BadRequest(c, err.Error())
				return
			}
			imageURLs = append([]string{url}, imageURLs...)
		} else if err != http.ErrMissingFile {
			log.Printf("V1读取上传文件失败: %v", err)
		}

		if len(imageURLs) > 3 {
			dto.BadRequest(c, "最多上传3张图片")
			return
		}

		firstImageURL := ""
		if len(imageURLs) > 0 {
			firstImageURL = imageURLs[0]
		}

		condition := c.PostForm("condition")
		if condition == "" {
			condition = ptrStrVal(existing.Condition)
		}
		description := c.PostForm("description")
		if description == "" {
			description = ptrStrVal(existing.Description)
		}

		book := &models.Book{
			BookID:      id,
			Title:       title,
			Category:    strPtr(category),
			ImageURL:    strPtr(firstImageURL),
			Price:       strPtr(c.PostForm("price")),
			ISBN:        strPtr(c.PostForm("isbn")),
			Contact:     strPtr(c.PostForm("contact")),
			Description: strPtr(description),
			Condition:   strPtr(condition),
			Status:      existing.Status,
		}

		if err := models.UpdateBookWithImages(database.DB, book, imageURLs); err != nil {
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

// V1UploadBookImage 独立上传书籍图片
func V1UploadBookImage() gin.HandlerFunc {
	return func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			dto.BadRequest(c, "请选择图片文件")
			return
		}

		url, err := saveUploadedImage(c, file)
		if err != nil {
			log.Printf("V1上传书籍图片失败: %v", err)
			dto.BadRequest(c, err.Error())
			return
		}

		dto.Success(c, gin.H{"url": url})
	}
}

// V1DeleteBookImage 删除单张书籍图片
func V1DeleteBookImage() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.GetStudentUserID(c)
		imageID, err := strconv.Atoi(c.Param("imageId"))
		if err != nil {
			dto.BadRequest(c, "无效的图片ID")
			return
		}

		img, err := models.GetBookImageByID(database.DB, imageID)
		if err != nil {
			dto.Error(c, 404, "图片不存在")
			return
		}

		// Verify ownership through book
		book, err := models.GetBookByID(database.DB, img.BookID)
		if err != nil {
			dto.Error(c, 404, "书籍不存在")
			return
		}
		if book.UserID != userID {
			dto.Forbidden(c, "只能删除自己的图片")
			return
		}

		// Delete file from disk
		deleteImageFile(img.ImageURL)

		// Delete DB record
		if err := models.DeleteBookImage(database.DB, imageID); err != nil {
			log.Printf("V1删除书籍图片失败: %v", err)
			dto.InternalError(c, "删除图片失败")
			return
		}

		dto.SuccessMessage(c, "删除成功")
	}
}

// V1ToggleWant 切换想要状态
func V1ToggleWant() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.GetStudentUserID(c)
		bookID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的书籍ID")
			return
		}

		// Verify book exists and is active
		book, err := models.GetBookByID(database.DB, bookID)
		if err != nil {
			dto.Error(c, 404, "书籍不存在")
			return
		}
		if book.Status != "active" {
			dto.BadRequest(c, "该书籍已下架")
			return
		}

		wanted, wantCount, err := models.ToggleWant(database.DB, bookID, userID)
		if err != nil {
			log.Printf("V1切换想要失败: %v", err)
			dto.InternalError(c, "操作失败")
			return
		}

		dto.Success(c, gin.H{
			"wanted":     wanted,
			"want_count": wantCount,
		})
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

	dir := uploadDir()
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("创建上传目录失败: %w", err)
		}
	}

	filename := uuid.New().String() + ext
	dst := filepath.Join(dir, filename)
	if err := c.SaveUploadedFile(file, dst); err != nil {
		return "", fmt.Errorf("保存文件失败: %w", err)
	}

	return "/uploads/" + filename, nil
}
