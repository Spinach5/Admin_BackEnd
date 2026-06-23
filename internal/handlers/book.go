package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
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
	"golang.org/x/crypto/bcrypt"
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
		fixBookImageURLs(books)
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
		if book.ImageURL != nil {
			book.ImageURL = strPtr(ToAbsoluteURL(ptrStrVal(book.ImageURL)))
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
		schoolID := c.Query("school_id")
		if schoolID == "" {
			schoolID = "hbut"
		}

		categories, err := models.GetBookCategories(database.DB, schoolID)
		if err != nil {
			log.Printf("获取书籍种类失败: %v", err)
			dto.InternalError(c, "获取书籍种类失败")
			return
		}

		// Build string list with "全部" prepended for frontend filter bar
		names := make([]string, 0, len(categories)+1)
		names = append(names, "全部")
		for _, cat := range categories {
			names = append(names, cat.Name)
		}

		dto.Success(c, names)
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

// ---- Image URL resolution ----

var BaseURL string

// SetBaseURL sets the base URL used to convert relative image paths to absolute URLs.
func SetBaseURL(url string) {
	BaseURL = strings.TrimRight(url, "/")
}

// ToAbsoluteURL converts a relative path (e.g. /uploads/foo.jpg) to an absolute URL
// when BaseURL is configured. If the path is already absolute or BaseURL is empty,
// the path is returned unchanged.
func ToAbsoluteURL(path string) string {
	if BaseURL == "" || path == "" {
		return path
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return BaseURL + path
}

// fixBookImageURLs converts relative image URLs in a book list to absolute URLs.
func fixBookImageURLs(books []models.BookWithUser) {
	for i := range books {
		if books[i].ImageURL != nil {
			books[i].ImageURL = strPtr(ToAbsoluteURL(ptrStrVal(books[i].ImageURL)))
		}
	}
}

// fixBookDetailImageURLs converts relative image URLs in a book detail to absolute URLs.
func fixBookDetailImageURLs(detail *models.BookDetail) {
	if detail.ImageURL != nil {
		detail.ImageURL = strPtr(ToAbsoluteURL(ptrStrVal(detail.ImageURL)))
	}
	for i := range detail.Images {
		detail.Images[i].ImageURL = ToAbsoluteURL(detail.Images[i].ImageURL)
	}
}

// fixClubImageURLs converts relative image URLs in a club list to absolute URLs.
func fixClubImageURLs(clubs []models.ClubWithPrincipal) {
	for i := range clubs {
		if clubs[i].ImageURL != nil {
			clubs[i].ImageURL = strPtr(ToAbsoluteURL(ptrStrVal(clubs[i].ImageURL)))
		}
	}
}

// CreateBookCategory 添加书籍种类 (admin)
func CreateBookCategory() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name      string `json:"name"`
			SchoolID  string `json:"school_id"`
			SortOrder int    `json:"sort_order"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			dto.BadRequest(c, "参数错误")
			return
		}
		if req.Name == "" {
			dto.BadRequest(c, "种类名称不能为空")
			return
		}
		if req.SchoolID == "" {
			req.SchoolID = "hbut"
		}

		cat := &models.BookCategory{
			Name:      req.Name,
			SchoolID:  req.SchoolID,
			SortOrder: req.SortOrder,
		}
		if err := models.CreateBookCategory(database.DB, cat); err != nil {
			log.Printf("添加书籍种类失败: %v", err)
			dto.InternalError(c, "添加书籍种类失败")
			return
		}
		dto.SuccessMessage(c, "添加种类成功")
	}
}

// UpdateBookCategory 更新书籍种类 (admin)
func UpdateBookCategory() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的种类ID")
			return
		}

		var req struct {
			Name      string `json:"name"`
			SortOrder int    `json:"sort_order"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			dto.BadRequest(c, "参数错误")
			return
		}
		if req.Name == "" {
			dto.BadRequest(c, "种类名称不能为空")
			return
		}

		cat := &models.BookCategory{
			ID:        id,
			Name:      req.Name,
			SortOrder: req.SortOrder,
		}
		if err := models.UpdateBookCategory(database.DB, cat); err != nil {
			log.Printf("更新书籍种类失败: %v", err)
			dto.InternalError(c, "更新书籍种类失败")
			return
		}
		dto.SuccessMessage(c, "更新种类成功")
	}
}

// DeleteBookCategory 删除书籍种类 (admin)
func DeleteBookCategory() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的种类ID")
			return
		}

		if err := models.DeleteBookCategory(database.DB, id); err != nil {
			log.Printf("删除书籍种类失败: %v", err)
			dto.InternalError(c, "删除书籍种类失败")
			return
		}
		dto.SuccessMessage(c, "删除种类成功")
	}
}

// ---- 本地身份验证 ----

// verifyBookCredentials 本地验证用户身份（学号+学校+密码）
// 前端传来的 password 是 RSA 加密后的密文，与注册时相同的 SHA-256 → bcrypt 流程
func verifyBookCredentials(schoolID, stuID, password string) error {
	// 1. 数据合理性验证
	if schoolID == "" || stuID == "" || password == "" {
		return fmt.Errorf("缺少身份验证参数 (school_id, stu_id, password)")
	}
	if len(schoolID) > 50 {
		return fmt.Errorf("学校ID格式不正确")
	}
	if len(stuID) > 30 {
		return fmt.Errorf("学号格式不正确")
	}

	// 2. 查询用户是否存在
	user, err := models.GetUserByStuIDAndSchoolIDWithPassword(database.DB, stuID, schoolID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("账户不存在，请先注册")
		}
		log.Printf("验证凭证查询用户失败: %v", err)
		return fmt.Errorf("服务器错误")
	}

	// 3. 验证密码：RSA 密文 → SHA-256 → bcrypt 比对
	sha := sha256.Sum256([]byte(password))
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(hex.EncodeToString(sha[:]))); err != nil {
		return fmt.Errorf("密码错误")
	}

	return nil
}
