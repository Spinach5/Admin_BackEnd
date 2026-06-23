# Book Backend Optimization — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Upgrade the book backend from basic single-image CRUD to production-ready multi-image, paginated, searchable, want-toggle API with database-driven categories.

**Architecture:** Extend existing `book` table with `description`, `condition`, `school_id` columns; add three new tables (`book_images`, `book_categories`, `book_wants`); augment model layer with paginated+search queries and image/want functions; update V1 student handlers and add admin category CRUD.

**Tech Stack:** Go, Gin, sqlx, MySQL, existing project patterns (closure-factory handlers, `*sqlx.DB`-first model functions, `dto.*` response helpers).

## Global Constraints

- Student-facing `/api/v1/books` APIs only — admin `/api/books` stays unchanged
- Simple want toggle — no want-list page, no notifications
- `school_id` column for multi-school pattern consistency
- Separate `book_images` table for multi-image (max 3)
- Database-driven `book_categories` table with admin CRUD
- All existing patterns followed: closure factories for handlers, raw SQL with sqlx, `dto.*` helpers, `middleware.GetStudentUserID(c)` for identity
- No test suite exists in project — manual verification via `go build` + `go run` + curl

---

### Task 1: Database Migration

**Files:**
- Modify: `cmd/migrate/main.go`

**Interfaces:**
- Produces: `book` table with `description`, `condition`, `school_id` columns; `book_images` table; `book_categories` table (seeded with 8 default categories); `book_wants` table

- [ ] **Step 1: Add migration SQL to `cmd/migrate/main.go`**

Add the following blocks after the existing `book` table creation (after line ~200, before the admin account seed section). Each block follows the existing pattern of checking column existence before ALTER, and using `CREATE TABLE IF NOT EXISTS`.

Insert after the `log.Println("  ✓ book")` equivalent section — find the comment `// 插入默认管理员账号` and insert BEFORE it:

```go
	// 书籍表 — 添加 description 列（兼容旧表）
	var bookDescColCount int
	appDB.Get(&bookDescColCount, "SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'book' AND COLUMN_NAME = 'description'")
	if bookDescColCount == 0 {
		appDB.MustExec("ALTER TABLE book ADD COLUMN description TEXT COMMENT '描述/详情'")
		log.Println("  ✓ book.description 列已添加")
	}

	// 书籍表 — 添加 condition 列（兼容旧表）
	var bookCondColCount int
	appDB.Get(&bookCondColCount, "SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'book' AND COLUMN_NAME = 'condition'")
	if bookCondColCount == 0 {
		appDB.MustExec("ALTER TABLE book ADD COLUMN `condition` VARCHAR(20) NOT NULL DEFAULT '几乎全新' COMMENT '新旧程度'")
		log.Println("  ✓ book.condition 列已添加")
	}

	// 书籍表 — 添加 school_id 列（兼容旧表）
	var bookSchoolColCount int
	appDB.Get(&bookSchoolColCount, "SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'book' AND COLUMN_NAME = 'school_id'")
	if bookSchoolColCount == 0 {
		appDB.MustExec("ALTER TABLE book ADD COLUMN school_id VARCHAR(50) NOT NULL DEFAULT 'hbut' COMMENT '学校代码'")
		log.Println("  ✓ book.school_id 列已添加")
	}

	// 书籍图片表
	appDB.MustExec(`
		CREATE TABLE IF NOT EXISTS book_images (
			id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
			book_id INT UNSIGNED NOT NULL,
			image_url VARCHAR(500) NOT NULL,
			sort_order TINYINT UNSIGNED DEFAULT 0,
			FOREIGN KEY (book_id) REFERENCES book(book_id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	log.Println("  ✓ book_images")

	// 书籍种类表
	appDB.MustExec(`
		CREATE TABLE IF NOT EXISTS book_categories (
			id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			school_id VARCHAR(50) NOT NULL DEFAULT 'hbut',
			sort_order INT DEFAULT 0
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	log.Println("  ✓ book_categories")

	// 种子数据：默认书籍种类
	var catCount int
	appDB.Get(&catCount, "SELECT COUNT(*) FROM book_categories")
	if catCount == 0 {
		categories := []struct {
			Name      string
			SortOrder int
		}{
			{"数学", 1}, {"外语", 2}, {"计算机", 3}, {"理工类", 4},
			{"思政类", 5}, {"文学类", 6}, {"经管类", 7}, {"其他", 8},
		}
		for _, c := range categories {
			appDB.MustExec("INSERT INTO book_categories (name, school_id, sort_order) VALUES (?, 'hbut', ?)", c.Name, c.SortOrder)
		}
		log.Println("  ✓ book_categories 种子数据已插入")
	} else {
		log.Println("  book_categories 已有数据，跳过种子")
	}

	// 书籍想要表
	appDB.MustExec(`
		CREATE TABLE IF NOT EXISTS book_wants (
			id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
			book_id INT UNSIGNED NOT NULL,
			user_id INT UNSIGNED NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE KEY uk_book_user (book_id, user_id),
			FOREIGN KEY (book_id) REFERENCES book(book_id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	log.Println("  ✓ book_wants")
```

- [ ] **Step 2: Run migration to verify**

```bash
cd /home/zqw/biancheng/Project/backend && go run ./cmd/migrate
```

Expected: logs showing all new columns/tables created, no errors.

- [ ] **Step 3: Commit**

```bash
git add cmd/migrate/main.go
git commit -m "feat: add book_images, book_categories, book_wants tables and new book columns

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 2: Book Model Extensions

**Files:**
- Modify: `internal/models/book.go`

**Interfaces:**
- Consumes: Updated `book` table schema from Task 1; existing `Book` and `BookWithUser` structs
- Produces:
  - `BookImage` struct: `ID int`, `BookID int`, `ImageURL string`, `SortOrder int`
  - `BookCategory` struct: `ID int`, `Name string`, `SchoolID string`, `SortOrder int`
  - `BookDetail` struct: embeds `BookWithUser`, adds `Description string`, `Condition string`, `Images []BookImage`, `WantCount int`, `IsWanted bool`
  - `GetBooksPaginated(db *sqlx.DB, category, keyword, schoolID string, page, pageSize int) ([]BookWithUser, int, error)`
  - `GetMyBooksPaginated(db *sqlx.DB, userID int, page, pageSize int) ([]BookWithUser, int, error)`
  - `GetBookDetail(db *sqlx.DB, bookID, currentUserID int) (*BookDetail, error)`
  - `CreateBookWithImages(db *sqlx.DB, b *Book, imageURLs []string) error`
  - `UpdateBookWithImages(db *sqlx.DB, b *Book, imageURLs []string) error`
  - `GetBookImages(db *sqlx.DB, bookID int) ([]BookImage, error)`
  - `GetBookImageByID(db *sqlx.DB, imageID int) (*BookImage, error)`
  - `DeleteBookImage(db *sqlx.DB, imageID int) error`
  - `ToggleWant(db *sqlx.DB, bookID, userID int) (wanted bool, wantCount int, err error)`
  - `GetBookCategories(db *sqlx.DB, schoolID string) ([]BookCategory, error)`
  - `CreateBookCategory(db *sqlx.DB, c *BookCategory) error`
  - `UpdateBookCategory(db *sqlx.DB, c *BookCategory) error`
  - `DeleteBookCategory(db *sqlx.DB, id int) error`

- [ ] **Step 1: Append new structs and functions to `internal/models/book.go`**

Add the following code at the end of the file (after line 107, the `GetBooksByUser` function):

```go
// ---- New structs ----

type BookImage struct {
	ID        int    `db:"id" json:"id"`
	BookID    int    `db:"book_id" json:"book_id"`
	ImageURL  string `db:"image_url" json:"url"`
	SortOrder int    `db:"sort_order" json:"sort_order"`
}

type BookCategory struct {
	ID        int    `db:"id" json:"id"`
	Name      string `db:"name" json:"name"`
	SchoolID  string `db:"school_id" json:"school_id"`
	SortOrder int    `db:"sort_order" json:"sort_order"`
}

type BookDetail struct {
	BookWithUser
	Description string      `db:"description" json:"description"`
	Condition   string      `db:"condition" json:"condition"`
	Images      []BookImage `json:"images"`
	WantCount   int         `json:"want_count"`
	IsWanted    bool        `json:"is_wanted"`
}

// ---- Paginated list with search ----

func GetBooksPaginated(db *sqlx.DB, category, keyword, schoolID string, page, pageSize int) ([]BookWithUser, int, error) {
	offset := (page - 1) * pageSize

	var total int
	countSQL := `SELECT COUNT(*) FROM book b WHERE b.status = 'active' AND b.school_id = ?`
	countArgs := []interface{}{schoolID}

	dataSQL := `SELECT b.*, COALESCE(u.nickName, '') AS nickName, COALESCE(u.stuId, '') AS stuId
		FROM book b
		LEFT JOIN users u ON b.user_id = u.id AND u.isDeleted = 0
		WHERE b.status = 'active' AND b.school_id = ?`
	dataArgs := []interface{}{schoolID}

	if category != "" {
		countSQL += " AND b.category = ?"
		countArgs = append(countArgs, category)
		dataSQL += " AND b.category = ?"
		dataArgs = append(dataArgs, category)
	}

	if keyword != "" {
		countSQL += " AND (b.title LIKE CONCAT('%', ?, '%') OR b.isbn LIKE CONCAT('%', ?, '%'))"
		countArgs = append(countArgs, keyword, keyword)
		dataSQL += " AND (b.title LIKE CONCAT('%', ?, '%') OR b.isbn LIKE CONCAT('%', ?, '%'))"
		dataArgs = append(dataArgs, keyword, keyword)
	}

	if err := db.Get(&total, countSQL, countArgs...); err != nil {
		return nil, 0, err
	}

	dataSQL += " ORDER BY b.book_id DESC LIMIT ? OFFSET ?"
	dataArgs = append(dataArgs, pageSize, offset)

	books := make([]BookWithUser, 0)
	if err := db.Select(&books, dataSQL, dataArgs...); err != nil {
		return nil, 0, err
	}

	return books, total, nil
}

func GetMyBooksPaginated(db *sqlx.DB, userID int, page, pageSize int) ([]BookWithUser, int, error) {
	offset := (page - 1) * pageSize

	var total int
	if err := db.Get(&total, "SELECT COUNT(*) FROM book WHERE user_id = ? AND status = 'active'", userID); err != nil {
		return nil, 0, err
	}

	books := make([]BookWithUser, 0)
	err := db.Select(&books, `SELECT b.*, COALESCE(u.nickName, '') AS nickName, COALESCE(u.stuId, '') AS stuId
		FROM book b
		LEFT JOIN users u ON b.user_id = u.id AND u.isDeleted = 0
		WHERE b.user_id = ? AND b.status = 'active'
		ORDER BY b.book_id DESC LIMIT ? OFFSET ?`, userID, pageSize, offset)
	return books, total, err
}

// ---- Detail with images and want info ----

func GetBookDetail(db *sqlx.DB, bookID, currentUserID int) (*BookDetail, error) {
	var detail BookDetail
	err := db.Get(&detail, `SELECT b.*, COALESCE(u.nickName, '') AS nickName, COALESCE(u.stuId, '') AS stuId
		FROM book b
		LEFT JOIN users u ON b.user_id = u.id AND u.isDeleted = 0
		WHERE b.book_id = ?`, bookID)
	if err != nil {
		return nil, err
	}

	images, err := GetBookImages(db, bookID)
	if err != nil {
		return nil, err
	}
	detail.Images = images
	if detail.Images == nil {
		detail.Images = []BookImage{}
	}

	db.Get(&detail.WantCount, "SELECT COUNT(*) FROM book_wants WHERE book_id = ?", bookID)

	var isWanted bool
	db.Get(&isWanted, "SELECT COUNT(*) > 0 FROM book_wants WHERE book_id = ? AND user_id = ?", bookID, currentUserID)
	detail.IsWanted = isWanted

	return &detail, nil
}

// ---- Book images CRUD ----

func GetBookImages(db *sqlx.DB, bookID int) ([]BookImage, error) {
	images := make([]BookImage, 0)
	err := db.Select(&images, "SELECT id, book_id, image_url, sort_order FROM book_images WHERE book_id = ? ORDER BY sort_order", bookID)
	return images, err
}

func GetBookImageByID(db *sqlx.DB, imageID int) (*BookImage, error) {
	var img BookImage
	err := db.Get(&img, "SELECT id, book_id, image_url, sort_order FROM book_images WHERE id = ?", imageID)
	return &img, err
}

func DeleteBookImage(db *sqlx.DB, imageID int) error {
	_, err := db.Exec("DELETE FROM book_images WHERE id = ?", imageID)
	return err
}

func CountBookImages(db *sqlx.DB, bookID int) (int, error) {
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM book_images WHERE book_id = ?", bookID)
	return count, err
}

func InsertBookImage(db *sqlx.DB, bookID int, imageURL string, sortOrder int) error {
	_, err := db.Exec("INSERT INTO book_images (book_id, image_url, sort_order) VALUES (?, ?, ?)", bookID, imageURL, sortOrder)
	return err
}

func ClearBookImages(db *sqlx.DB, bookID int) error {
	_, err := db.Exec("DELETE FROM book_images WHERE book_id = ?", bookID)
	return err
}

// ---- Create / Update with images ----

func CreateBookWithImages(db *sqlx.DB, b *Book, imageURLs []string) error {
	result, err := db.Exec(`INSERT INTO book (title, category, image_url, price, isbn, contact, user_id, status, description, `+"`condition`"+`, school_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		b.Title, b.Category, b.ImageURL, b.Price, b.ISBN, b.Contact, b.UserID, b.Status, b.Description, b.Condition, b.SchoolID)
	if err != nil {
		return err
	}

	bookID, err := result.LastInsertId()
	if err != nil {
		return err
	}

	for i, url := range imageURLs {
		if err := InsertBookImage(db, int(bookID), url, i); err != nil {
			return err
		}
	}

	return nil
}

func UpdateBookWithImages(db *sqlx.DB, b *Book, imageURLs []string) error {
	_, err := db.Exec(`UPDATE book SET title=?, category=?, image_url=?, price=?, isbn=?, contact=?, status=?, description=?, `+"`condition`"+`=?
		WHERE book_id=?`,
		b.Title, b.Category, b.ImageURL, b.Price, b.ISBN, b.Contact, b.Status, b.Description, b.Condition, b.BookID)
	if err != nil {
		return err
	}

	if err := ClearBookImages(db, b.BookID); err != nil {
		return err
	}

	for i, url := range imageURLs {
		if err := InsertBookImage(db, b.BookID, url, i); err != nil {
			return err
		}
	}

	return nil
}

// ---- Want toggle ----

func ToggleWant(db *sqlx.DB, bookID, userID int) (wanted bool, wantCount int, err error) {
	// Try insert first
	_, err = db.Exec("INSERT INTO book_wants (book_id, user_id) VALUES (?, ?)", bookID, userID)
	if err == nil {
		// Insert succeeded — now wanted
		db.Get(&wantCount, "SELECT COUNT(*) FROM book_wants WHERE book_id = ?", bookID)
		return true, wantCount, nil
	}

	// Insert failed (likely duplicate) — delete instead
	db.Exec("DELETE FROM book_wants WHERE book_id = ? AND user_id = ?", bookID, userID)
	db.Get(&wantCount, "SELECT COUNT(*) FROM book_wants WHERE book_id = ?", bookID)
	return false, wantCount, nil
}

// ---- Book categories from DB ----

func GetBookCategories(db *sqlx.DB, schoolID string) ([]BookCategory, error) {
	categories := make([]BookCategory, 0)
	err := db.Select(&categories, "SELECT id, name, school_id, sort_order FROM book_categories WHERE school_id = ? ORDER BY sort_order", schoolID)
	return categories, err
}

func CreateBookCategory(db *sqlx.DB, c *BookCategory) error {
	_, err := db.Exec("INSERT INTO book_categories (name, school_id, sort_order) VALUES (?, ?, ?)", c.Name, c.SchoolID, c.SortOrder)
	return err
}

func UpdateBookCategory(db *sqlx.DB, c *BookCategory) error {
	_, err := db.Exec("UPDATE book_categories SET name=?, sort_order=? WHERE id=?", c.Name, c.SortOrder, c.ID)
	return err
}

func DeleteBookCategory(db *sqlx.DB, id int) error {
	_, err := db.Exec("DELETE FROM book_categories WHERE id = ?", id)
	return err
}
```

Also, update the existing `Book` struct (lines 9-21) to add the new fields:

Change:
```go
type Book struct {
	BookID           int     `db:"book_id" json:"book_id"`
	Title            string  `db:"title" json:"title"`
	Category         *string `db:"category" json:"category"`
	ImageURL         *string `db:"image_url" json:"image_url"`
	Price            *string `db:"price" json:"price"`
	ISBN             *string `db:"isbn" json:"isbn"`
	Contact          *string `db:"contact" json:"contact"`
	UserID           int     `db:"user_id" json:"user_id"`
	Status           string  `db:"status" json:"status"`
	CreateTime       string  `db:"create_time" json:"create_time"`
	StatusChangeTime *string `db:"status_change_time" json:"status_change_time"`
}
```

To:
```go
type Book struct {
	BookID           int     `db:"book_id" json:"book_id"`
	Title            string  `db:"title" json:"title"`
	Category         *string `db:"category" json:"category"`
	ImageURL         *string `db:"image_url" json:"image_url"`
	Price            *string `db:"price" json:"price"`
	ISBN             *string `db:"isbn" json:"isbn"`
	Contact          *string `db:"contact" json:"contact"`
	Description      *string `db:"description" json:"description"`
	Condition        *string `db:"condition" json:"condition"`
	SchoolID         string  `db:"school_id" json:"school_id"`
	UserID           int     `db:"user_id" json:"user_id"`
	Status           string  `db:"status" json:"status"`
	CreateTime       string  `db:"create_time" json:"create_time"`
	StatusChangeTime *string `db:"status_change_time" json:"status_change_time"`
}
```

- [ ] **Step 2: Verify compilation**

```bash
cd /home/zqw/biancheng/Project/backend && go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/models/book.go
git commit -m "feat: add book model extensions — pagination, images, want toggle, DB categories

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 3: V1 Book Handlers (Student)

**Files:**
- Modify: `internal/handlers/v1_book.go`

**Interfaces:**
- Consumes: All model functions from Task 2; existing `saveUploadedImage`, `strPtr`, `ptrStrVal`, `deleteImageFile`, `bookCategories` helpers in `book.go` and `v1_book.go`
- Produces: Updated `V1GetBooks`, `V1GetMyBooks`, `V1GetBookByID`, `V1CreateBook`, `V1UpdateBook`; new `V1UploadBookImage`, `V1DeleteBookImage`, `V1ToggleWant`

- [ ] **Step 1: Rewrite `V1GetBooks` for pagination, search, school_id**

Replace the existing `V1GetBooks` function (currently lines 22-39) with:

```go
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
```

- [ ] **Step 2: Rewrite `V1GetMyBooks` for pagination**

Replace the existing `V1GetMyBooks` function (currently lines 41-52) with:

```go
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
```

- [ ] **Step 3: Rewrite `V1GetBookByID` for detail with images/want info**

Replace the existing `V1GetBookByID` function (currently lines 54-68) with:

```go
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
```

- [ ] **Step 4: Rewrite `V1CreateBook` for new fields + multi-image**

Replace the existing `V1CreateBook` function (currently lines 70-134) with:

```go
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

		if category != "" && !bookCategories[category] {
			dto.BadRequest(c, "无效的书籍种类")
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
```

Add `"strings"` to the import block if not already present.

- [ ] **Step 5: Rewrite `V1UpdateBook` for new fields + image sync**

Replace the existing `V1UpdateBook` function (currently lines 136-203) with:

```go
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
		if category != "" && !bookCategories[category] {
			dto.BadRequest(c, "无效的书籍种类")
			return
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
```

- [ ] **Step 6: Add `V1UploadBookImage` handler**

Append to the end of the file (before the closing, if there's no closing issue — just append):

```go
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
```

- [ ] **Step 7: Add `V1DeleteBookImage` handler**

```go
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
```

- [ ] **Step 8: Add `V1ToggleWant` handler**

```go
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
```

- [ ] **Step 9: Verify compilation**

```bash
cd /home/zqw/biancheng/Project/backend && go build ./...
```

Expected: no errors. Fix any import issues (ensure `"strings"`, `"mime/multipart"` etc. are imported as needed).

- [ ] **Step 10: Commit**

```bash
git add internal/handlers/v1_book.go
git commit -m "feat: update V1 book handlers — pagination, multi-image, want toggle

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 4: Admin Category Handlers

**Files:**
- Modify: `internal/handlers/book.go`

**Interfaces:**
- Consumes: `models.GetBookCategories`, `models.CreateBookCategory`, `models.UpdateBookCategory`, `models.DeleteBookCategory` from Task 2
- Produces: Updated `GetBookCategories` (DB-driven), new `CreateBookCategory`, `UpdateBookCategory`, `DeleteBookCategory`

- [ ] **Step 1: Update `GetBookCategories` to query DB**

Replace the existing `GetBookCategories` function (currently lines 209-214) with:

```go
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
```

Add `"web-backend/internal/models"` to the import block if not already present (it should already be there).

- [ ] **Step 2: Add admin category CRUD handlers**

Append to the end of the file:

```go
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
```

Add `"encoding/json"` is NOT needed since we use `ShouldBindJSON`. The existing imports should already have everything needed. Verify the `"web-backend/internal/models"` import exists.

- [ ] **Step 3: Verify compilation**

```bash
cd /home/zqw/biancheng/Project/backend && go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/handlers/book.go
git commit -m "feat: make book categories DB-driven, add admin CRUD

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 5: Routes

**Files:**
- Modify: `cmd/server/main.go`

**Interfaces:**
- Consumes: All handlers from Tasks 3 and 4
- Produces: New routes registered in the V1 group

- [ ] **Step 1: Add new routes to the V1 student group**

In `cmd/server/main.go`, in the `v1Auth` block (around lines 131-149), add these three routes after the existing book routes:

```go
					v1Auth.POST("/books/upload-image", handlers.V1UploadBookImage())
					v1Auth.DELETE("/books/images/:imageId", handlers.V1DeleteBookImage())
					v1Auth.POST("/books/:id/want", handlers.V1ToggleWant())
```

Add these after line 146 (`v1Auth.DELETE("/books/:id", handlers.V1DeleteBook())`) and before line 147 (`v1Auth.GET("/foods", handlers.V1GetFoods())`).

Also add admin category routes in the `authorized` block. After the existing `authorized.GET("/books/categories", handlers.GetBookCategories())` line (~line 82), add:

```go
				authorized.POST("/books/categories", handlers.CreateBookCategory())
				authorized.PUT("/books/categories/:id", handlers.UpdateBookCategory())
				authorized.DELETE("/books/categories/:id", handlers.DeleteBookCategory())
```

- [ ] **Step 2: Verify compilation**

```bash
cd /home/zqw/biancheng/Project/backend && go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat: register new V1 book routes — upload-image, delete-image, toggle-want

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Verification (Manual)

After all tasks are complete, start the server and verify endpoints:

- [ ] **Start server:**

```bash
cd /home/zqw/biancheng/Project/backend && go run ./cmd/server
```

- [ ] **Test paginated list:**

```bash
curl -s http://localhost:3001/api/v1/books?page=1\&pageSize=5 | jq '.success, .total'
```

Expected: `true` and a number.

- [ ] **Test categories from DB:**

```bash
curl -s http://localhost:3001/api/v1/books/categories | jq '.data'
```

Expected: `["全部", "数学", "外语", "计算机", "理工类", "思政类", "文学类", "经管类", "其他"]`

- [ ] **Test build passes:**

```bash
cd /home/zqw/biancheng/Project/backend && go build -o /dev/null ./cmd/server
```

Expected: exit 0.
