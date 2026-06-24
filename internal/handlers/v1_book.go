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

		fixBookImageURLs(books)
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

		fixBookImageURLs(books)
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

		fixBookDetailImageURLs(detail)
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

		// Accept JSON or form data
		var title, author, publisher, coverURL, category, price, isbn, contact, description, condition, schoolID string
		var isDelivery int
		var bookType int16
		var imageURLs []string

		contentType := c.ContentType()
		if contentType == "application/json" {
			var req struct {
				Name        string   `json:"name"`
				Author      string   `json:"author"`
				Publisher   string   `json:"publisher"`
				CoverURL    string   `json:"cover_url"`
				Category    string   `json:"category"`
				Price       string   `json:"price"`
				ISBN        string   `json:"isbn"`
				Contact     string   `json:"contact"`
				Description string   `json:"description"`
				Condition   string   `json:"condition"`
				SchoolID    string   `json:"school_id"`
				IsDelivery  int      `json:"is_delivery"`
				BookType    int16    `json:"book_type"`
				Images      []string `json:"images"`
				StuID       string   `json:"stu_id"`
				Password    string   `json:"password"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				dto.BadRequest(c, "参数错误")
				return
			}
			if err := verifyBookCredentials(req.SchoolID, req.StuID, req.Password); err != nil {
				dto.BadRequest(c, err.Error())
				return
			}
			title = strings.TrimSpace(req.Name)
			author = req.Author
			publisher = req.Publisher
			coverURL = req.CoverURL
			category = req.Category
			price = req.Price
			isbn = req.ISBN
			contact = req.Contact
			description = req.Description
			condition = req.Condition
			schoolID = req.SchoolID
			isDelivery = req.IsDelivery
			bookType = req.BookType
			imageURLs = req.Images
		} else {
			// 表单数据：从表单字段提取凭证
			if err := verifyBookCredentials(
				c.PostForm("school_id"),
				c.PostForm("stu_id"),
				c.PostForm("password"),
			); err != nil {
				dto.BadRequest(c, err.Error())
				return
			}
			title = strings.TrimSpace(c.PostForm("title"))
			author = c.PostForm("author")
			publisher = c.PostForm("publisher")
			coverURL = c.PostForm("cover_url")
			category = c.PostForm("category")
			price = c.PostForm("price")
			isbn = c.PostForm("isbn")
			contact = c.PostForm("contact")
			description = c.PostForm("description")
			condition = c.PostForm("condition")
			schoolID = c.PostForm("school_id")
			if c.PostForm("is_delivery") == "1" {
				isDelivery = 1
			}
			bookTypeStr := c.PostForm("book_type")
			bookType = 1
			if bookTypeStr == "2" {
				bookType = 2
			}
			imageURLsRaw := c.PostForm("image_urls")
			if imageURLsRaw != "" {
				for _, u := range strings.Split(imageURLsRaw, ",") {
					u = strings.TrimSpace(u)
					if u != "" {
						imageURLs = append(imageURLs, u)
					}
				}
			}
		}

		if schoolID == "" {
			schoolID = "hbut"
		}

		if title == "" {
			dto.BadRequest(c, "书名不能为空")
			return
		}

		if len(imageURLs) > 3 {
			dto.BadRequest(c, "最多上传3张图片")
			return
		}

		// Also support single legacy file upload (form data only)
		var firstImageURL string
		if contentType != "application/json" {
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

		// book_type 默认值
		if bookType == 0 {
			bookType = 1
		}

		book := &models.Book{
			Title:       title,
			Author:      strPtr(author),
			Publisher:   strPtr(publisher),
			CoverURL:    strPtr(coverURL),
			Category:    strPtr(category),
			ImageURL:    strPtr(firstImageURL),
			Price:       strPtr(price),
			ISBN:        strPtr(isbn),
			Contact:     strPtr(contact),
			Description: strPtr(description),
			Condition:   strPtr(condition),
			SchoolID:    schoolID,
			IsDelivery:  isDelivery,
			BookType:    bookType,
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

		var title, author, publisher, coverURL, category, price, isbn, contact, description, condition string
		var isDelivery int
		var bookType int16
		var imageURLs []string

		contentType := c.ContentType()
		if contentType == "application/json" {
			var req struct {
				Name        string   `json:"name"`
				Author      string   `json:"author"`
				Publisher   string   `json:"publisher"`
				CoverURL    string   `json:"cover_url"`
				Category    string   `json:"category"`
				Price       string   `json:"price"`
				ISBN        string   `json:"isbn"`
				Contact     string   `json:"contact"`
				Description string   `json:"description"`
				Condition   string   `json:"condition"`
				IsDelivery  int      `json:"is_delivery"`
				BookType    int16    `json:"book_type"`
				Images      []string `json:"images"`
				SchoolID    string   `json:"school_id"`
				StuID       string   `json:"stu_id"`
				Password    string   `json:"password"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				dto.BadRequest(c, "参数错误")
				return
			}
			if err := verifyBookCredentials(req.SchoolID, req.StuID, req.Password); err != nil {
				dto.BadRequest(c, err.Error())
				return
			}
			title = strings.TrimSpace(req.Name)
			author = req.Author
			publisher = req.Publisher
			coverURL = req.CoverURL
			category = req.Category
			price = req.Price
			isbn = req.ISBN
			contact = req.Contact
			description = req.Description
			condition = req.Condition
			isDelivery = req.IsDelivery
			bookType = req.BookType
			imageURLs = req.Images
		} else {
			// 表单数据：从表单字段提取凭证
			if err := verifyBookCredentials(
				c.PostForm("school_id"),
				c.PostForm("stu_id"),
				c.PostForm("password"),
			); err != nil {
				dto.BadRequest(c, err.Error())
				return
			}
			title = strings.TrimSpace(c.PostForm("title"))
			author = c.PostForm("author")
			publisher = c.PostForm("publisher")
			coverURL = c.PostForm("cover_url")
			category = c.PostForm("category")
			price = c.PostForm("price")
			isbn = c.PostForm("isbn")
			contact = c.PostForm("contact")
			description = c.PostForm("description")
			condition = c.PostForm("condition")
			if c.PostForm("is_delivery") != "" {
				if c.PostForm("is_delivery") == "1" {
					isDelivery = 1
				} else {
					isDelivery = 0
				}
			} else {
				isDelivery = existing.IsDelivery
			}
			bookTypeStr := c.PostForm("book_type")
			if bookTypeStr != "" {
				if bookTypeStr == "2" {
					bookType = 2
				} else {
					bookType = 1
				}
			} else {
				bookType = existing.BookType
			}
			imageURLsRaw := c.PostForm("image_urls")
			if imageURLsRaw != "" {
				for _, u := range strings.Split(imageURLsRaw, ",") {
					u = strings.TrimSpace(u)
					if u != "" {
						imageURLs = append(imageURLs, u)
					}
				}
			}
		}

		if title == "" {
			dto.BadRequest(c, "书名不能为空")
			return
		}

		// Fallbacks to existing values
		if author == "" {
			author = ptrStrVal(existing.Author)
		}
		if publisher == "" {
			publisher = ptrStrVal(existing.Publisher)
		}
		if coverURL == "" {
			coverURL = ptrStrVal(existing.CoverURL)
		}
		if category == "" {
			category = ptrStrVal(existing.Category)
		}
		if condition == "" {
			condition = ptrStrVal(existing.Condition)
		}
		if description == "" {
			description = ptrStrVal(existing.Description)
		}

		// book_type 默认值
		if bookType == 0 {
			bookType = existing.BookType
			if bookType == 0 {
				bookType = 1
			}
		}

		// If no images submitted, preserve existing
		if len(imageURLs) == 0 {
			existingImages, _ := models.GetBookImages(database.DB, id)
			for _, img := range existingImages {
				imageURLs = append(imageURLs, img.ImageURL)
			}
		}

		if len(imageURLs) > 3 {
			dto.BadRequest(c, "最多上传3张图片")
			return
		}

		// Legacy single file upload (form data only)
		if contentType != "application/json" {
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
		}

		if len(imageURLs) > 3 {
			dto.BadRequest(c, "最多上传3张图片")
			return
		}

		firstImageURL := ""
		if len(imageURLs) > 0 {
			firstImageURL = imageURLs[0]
		}

		book := &models.Book{
			BookID:      id,
			Title:       title,
			Author:      strPtr(author),
			Publisher:   strPtr(publisher),
			CoverURL:    strPtr(coverURL),
			Category:    strPtr(category),
			ImageURL:    strPtr(firstImageURL),
			Price:       strPtr(price),
			ISBN:        strPtr(isbn),
			Contact:     strPtr(contact),
			Description: strPtr(description),
			Condition:   strPtr(condition),
			IsDelivery:  isDelivery,
			BookType:    bookType,
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

		// 验证身份凭证（优先 JSON body，其次 query 参数）
		var schoolID, stuID, password string
		if c.ContentType() == "application/json" {
			var creds struct {
				SchoolID string `json:"school_id"`
				StuID    string `json:"stu_id"`
				Password string `json:"password"`
			}
			if err := c.ShouldBindJSON(&creds); err != nil {
				dto.BadRequest(c, "参数错误")
				return
			}
			schoolID, stuID, password = creds.SchoolID, creds.StuID, creds.Password
		} else {
			schoolID = c.Query("school_id")
			stuID = c.Query("stu_id")
			password = c.Query("password")
		}
		if err := verifyBookCredentials(schoolID, stuID, password); err != nil {
			dto.BadRequest(c, err.Error())
			return
		}

		// 硬删除：删除数据库记录和磁盘图片文件
		imageURLs, err := models.HardDeleteBookByUser(database.DB, id, userID)
		if err != nil {
			dto.BadRequest(c, err.Error())
			return
		}

		// 清理磁盘上的图片文件
		for _, url := range imageURLs {
			deleteImageFile(url)
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

		log.Printf("V1上传图片: name=%s, size=%d, header=%v", file.Filename, file.Size, file.Header)

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
		// Extension missing or unknown — try MIME type from Content-Type header
		ext = extByMIME(file.Header.Get("Content-Type"))
		if !allowed[ext] {
			return "", fmt.Errorf("不支持的图片格式，仅支持 jpg/jpeg/png/webp")
		}
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

	return ToAbsoluteURL("/uploads/" + filename), nil
}

func extByMIME(mime string) string {
	switch mime {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	default:
		return ""
	}
}
