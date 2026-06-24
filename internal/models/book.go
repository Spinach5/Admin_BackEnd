package models

import (
	"errors"
	"fmt"
	"log"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

type Book struct {
	BookID           int     `db:"book_id" json:"book_id"`
	Title            string  `db:"title" json:"title"`
	Author           *string `db:"author" json:"author"`
	Publisher        *string `db:"publisher" json:"publisher"`
	Category         *string `db:"category" json:"category"`
	ImageURL         *string `db:"image_url" json:"image_url"`
	Price            *string `db:"price" json:"price"`
	ISBN             *string `db:"isbn" json:"isbn"`
	Contact          *string `db:"contact" json:"contact"`
	Description      *string `db:"description" json:"description"`
	Condition        *string `db:"condition" json:"condition"`
	SchoolID         string  `db:"school_id" json:"school_id"`
	IsDelivery       int     `db:"is_delivery" json:"is_delivery"`
	BookType         int16   `db:"book_type" json:"book_type"`
	UserID           int     `db:"user_id" json:"user_id"`
	Status           string  `db:"status" json:"status"`
	CreateTime       string  `db:"create_time" json:"create_time"`
	StatusChangeTime *string `db:"status_change_time" json:"status_change_time"`
}

type BookWithUser struct {
	Book
	NickName   string `db:"nickName" json:"nickName"`
	StuID      string `db:"stuId" json:"stuId"`
	IsDelivery int    `db:"is_delivery" json:"is_delivery"`
}

type BookWithUserAndImages struct {
	BookWithUser
	Images []BookImage `json:"images"`
}

func GetAllBooks(db *sqlx.DB) ([]BookWithUser, error) {
	books := make([]BookWithUser, 0)
	err := db.Select(&books, `SELECT b.*, COALESCE(u.nickName, '') AS nickName, COALESCE(u.stuId, '') AS stuId FROM book b
		LEFT JOIN users u ON b.user_id = u.id AND u.isDeleted = 0
		ORDER BY b.book_id`)
	return books, err
}

func GetBookByID(db *sqlx.DB, id int) (*BookWithUser, error) {
	var book BookWithUser
	err := db.Get(&book, `SELECT b.*, COALESCE(u.nickName, '') AS nickName, COALESCE(u.stuId, '') AS stuId FROM book b
		LEFT JOIN users u ON b.user_id = u.id AND u.isDeleted = 0
		WHERE b.book_id = ?`, id)
	return &book, err
}

func CreateBook(db *sqlx.DB, b *Book) error {
	_, err := db.Exec(`INSERT INTO book (title, author, publisher, category, image_url, price, isbn, contact, user_id, status, book_type, school_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		b.Title, b.Author, b.Publisher, b.Category, b.ImageURL, b.Price, b.ISBN, b.Contact, b.UserID, b.Status, b.BookType, b.SchoolID)
	return err
}

func UpdateBook(db *sqlx.DB, b *Book) error {
	_, err := db.Exec(`UPDATE book SET title=?, author=?, publisher=?, category=?, image_url=?, price=?, isbn=?, contact=?, status=?, book_type=?, school_id=?
		WHERE book_id=?`,
		b.Title, b.Author, b.Publisher, b.Category, b.ImageURL, b.Price, b.ISBN, b.Contact, b.Status, b.BookType, b.SchoolID, b.BookID)
	return err
}

func DeleteBook(db *sqlx.DB, id int) error {
	_, err := db.Exec("DELETE FROM book WHERE book_id = ?", id)
	return err
}

// HardDeleteBookByUser 硬删除书籍及其关联数据，返回图片 URL 列表用于清理磁盘文件
func HardDeleteBookByUser(db *sqlx.DB, bookID, userID int) ([]string, error) {
	var imageURLs []string

	tx, err := db.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 验证所有权
	var ownerID int
	if err := tx.Get(&ownerID, "SELECT user_id FROM book WHERE book_id = ?", bookID); err != nil {
		return nil, fmt.Errorf("书籍不存在")
	}
	if ownerID != userID {
		return nil, fmt.Errorf("只能删除自己的书籍")
	}

	// 获取图片 URL 用于后续清理磁盘文件
	if err := tx.Select(&imageURLs, "SELECT image_url FROM book_images WHERE book_id = ?", bookID); err != nil {
		return nil, err
	}

	// 删除关联数据
	tx.Exec("DELETE FROM book_images WHERE book_id = ?", bookID)
	tx.Exec("DELETE FROM book_wants WHERE book_id = ?", bookID)

	// 删除书籍本身
	if _, err := tx.Exec("DELETE FROM book WHERE book_id = ? AND user_id = ?", bookID, userID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return imageURLs, nil
}

func CountActiveBooksByUser(db *sqlx.DB, userID int) (int, error) {
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM book WHERE user_id = ? AND status = 'active'", userID)
	return count, err
}

func SoftDeleteBookByUser(db *sqlx.DB, bookID, userID int) error {
	result, err := db.Exec("UPDATE book SET status = 'deleted' WHERE book_id = ? AND user_id = ?", bookID, userID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("只能删除自己的书籍")
	}
	return nil
}

func GetAllActiveBooks(db *sqlx.DB) ([]BookWithUser, error) {
	books := make([]BookWithUser, 0)
	err := db.Select(&books, `SELECT b.*, COALESCE(u.nickName, '') AS nickName, COALESCE(u.stuId, '') AS stuId FROM book b
		LEFT JOIN users u ON b.user_id = u.id AND u.isDeleted = 0
		WHERE b.status = 'active'
		ORDER BY b.book_id`)
	return books, err
}

func GetBooksByCategory(db *sqlx.DB, category string) ([]BookWithUser, error) {
	books := make([]BookWithUser, 0)
	err := db.Select(&books, `SELECT b.*, COALESCE(u.nickName, '') AS nickName, COALESCE(u.stuId, '') AS stuId FROM book b
		LEFT JOIN users u ON b.user_id = u.id AND u.isDeleted = 0
		WHERE b.category = ? AND b.status = 'active'
		ORDER BY b.book_id DESC`, category)
	return books, err
}

func GetBooksByUser(db *sqlx.DB, userID int) ([]BookWithUser, error) {
	books := make([]BookWithUser, 0)
	err := db.Select(&books, `SELECT b.*, COALESCE(u.nickName, '') AS nickName, COALESCE(u.stuId, '') AS stuId FROM book b
		LEFT JOIN users u ON b.user_id = u.id AND u.isDeleted = 0
		WHERE b.user_id = ? AND b.status = 'active'
		ORDER BY b.book_id DESC`, userID)
	return books, err
}

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

type BookCategoryWithCount struct {
	BookCategory
	BookCount int `db:"book_count" json:"book_count"`
}

type BookDetail struct {
	BookWithUser
	Images    []BookImage `json:"images"`
	WantCount int         `json:"want_count"`
	IsWanted  bool        `json:"is_wanted"`
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

	var wantCount int
	if err := db.Get(&wantCount, "SELECT COUNT(*) FROM book_wants WHERE book_id = ?", bookID); err != nil {
		log.Printf("GetBookDetail: want_count query failed for book %d: %v", bookID, err)
	}
	detail.WantCount = wantCount

	var isWanted bool
	if err := db.Get(&isWanted, "SELECT COUNT(*) > 0 FROM book_wants WHERE book_id = ? AND user_id = ?", bookID, currentUserID); err != nil {
		log.Printf("GetBookDetail: is_wanted query failed for book %d user %d: %v", bookID, currentUserID, err)
	}
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
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	result, err := tx.Exec(`INSERT INTO book (title, author, publisher, category, image_url, price, isbn, contact, user_id, status, description, `+"`condition`"+`, school_id, is_delivery, book_type)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		b.Title, b.Author, b.Publisher, b.Category, b.ImageURL, b.Price, b.ISBN, b.Contact, b.UserID, b.Status, b.Description, b.Condition, b.SchoolID, b.IsDelivery, b.BookType)
	if err != nil {
		return err
	}

	bookID, err := result.LastInsertId()
	if err != nil {
		return err
	}

	for i, url := range imageURLs {
		if _, err := tx.Exec("INSERT INTO book_images (book_id, image_url, sort_order) VALUES (?, ?, ?)", bookID, url, i); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func UpdateBookWithImages(db *sqlx.DB, b *Book, imageURLs []string) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`UPDATE book SET title=?, author=?, publisher=?, category=?, image_url=?, price=?, isbn=?, contact=?, status=?, description=?, `+"`condition`"+`=?, is_delivery=?, book_type=?
		WHERE book_id=?`,
		b.Title, b.Author, b.Publisher, b.Category, b.ImageURL, b.Price, b.ISBN, b.Contact, b.Status, b.Description, b.Condition, b.IsDelivery, b.BookType, b.BookID)
	if err != nil {
		return err
	}

	if _, err := tx.Exec("DELETE FROM book_images WHERE book_id = ?", b.BookID); err != nil {
		return err
	}

	for i, url := range imageURLs {
		if _, err := tx.Exec("INSERT INTO book_images (book_id, image_url, sort_order) VALUES (?, ?, ?)", b.BookID, url, i); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ---- Want toggle ----

func ToggleWant(db *sqlx.DB, bookID, userID int) (wanted bool, wantCount int, err error) {
	_, err = db.Exec("INSERT INTO book_wants (book_id, user_id) VALUES (?, ?)", bookID, userID)
	if err == nil {
		db.Get(&wantCount, "SELECT COUNT(*) FROM book_wants WHERE book_id = ?", bookID)
		return true, wantCount, nil
	}

	// Check for duplicate key error (MySQL errno 1062)
	if isDuplicateKeyError(err) {
		db.Exec("DELETE FROM book_wants WHERE book_id = ? AND user_id = ?", bookID, userID)
		db.Get(&wantCount, "SELECT COUNT(*) FROM book_wants WHERE book_id = ?", bookID)
		return false, wantCount, nil
	}

	return false, 0, err
}

func isDuplicateKeyError(err error) bool {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == 1062
	}
	return false
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

// UpdateBookCategoryWithSync 更新分类名称并同步更新所有关联书籍的 category 字段
func UpdateBookCategoryWithSync(db *sqlx.DB, categoryID int, newName string, sortOrder int) error {
	// 先查出旧名称
	var oldCat BookCategory
	if err := db.Get(&oldCat, "SELECT id, name, school_id, sort_order FROM book_categories WHERE id = ?", categoryID); err != nil {
		return fmt.Errorf("分类不存在")
	}

	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 更新分类名称
	if _, err := tx.Exec("UPDATE book_categories SET name=?, sort_order=? WHERE id=?", newName, sortOrder, categoryID); err != nil {
		return err
	}

	// 同步更新 books 表中的 category 字符串
	if oldCat.Name != newName {
		if _, err := tx.Exec("UPDATE book SET category = ? WHERE category = ?", newName, oldCat.Name); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// DeleteBookCategory 删除分类（仅删除分类本身，不删除关联书籍）
func DeleteBookCategory(db *sqlx.DB, id int) error {
	_, err := db.Exec("DELETE FROM book_categories WHERE id = ?", id)
	return err
}

// GetCategoriesWithBookCount 获取分类列表并统计每个分类下的活跃书籍数量
func GetCategoriesWithBookCount(db *sqlx.DB, schoolID string) ([]BookCategoryWithCount, error) {
	categories := make([]BookCategoryWithCount, 0)
	err := db.Select(&categories, `SELECT c.id, c.name, c.school_id, c.sort_order,
		COALESCE((SELECT COUNT(*) FROM book b WHERE b.category = c.name COLLATE utf8mb4_unicode_ci AND b.status = 'active'), 0) AS book_count
		FROM book_categories c
		WHERE c.school_id = ?
		ORDER BY c.sort_order`, schoolID)
	return categories, err
}

// CountBooksByCategoryName 按分类名统计该分类下的活跃书籍数量
func CountBooksByCategoryName(db *sqlx.DB, categoryName string) (int, error) {
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM book WHERE category = ? AND status = 'active'", categoryName)
	return count, err
}

// DeleteBookCategoryCascade 删除分类，同时删除该分类下的所有书籍（含图片记录）
// 返回被删除书籍的图片 URL 列表，用于清理磁盘文件
func DeleteBookCategoryCascade(db *sqlx.DB, categoryID int) ([]string, error) {
	// 先查出分类名称
	var cat BookCategory
	if err := db.Get(&cat, "SELECT id, name, school_id, sort_order FROM book_categories WHERE id = ?", categoryID); err != nil {
		return nil, fmt.Errorf("分类不存在")
	}

	tx, err := db.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 获取该分类下所有书籍的图片 URL
	var imageURLs []string
	if err := tx.Select(&imageURLs, `SELECT bi.image_url FROM book_images bi
		INNER JOIN book b ON bi.book_id = b.book_id
		WHERE b.category = ?`, cat.Name); err != nil {
		return nil, err
	}

	// 删除书籍的关联数据
	tx.Exec("DELETE FROM book_wants WHERE book_id IN (SELECT book_id FROM book WHERE category = ?)", cat.Name)
	tx.Exec("DELETE FROM book_images WHERE book_id IN (SELECT book_id FROM book WHERE category = ?)", cat.Name)

	// 删除该分类下的所有书籍
	if _, err := tx.Exec("DELETE FROM book WHERE category = ?", cat.Name); err != nil {
		return nil, err
	}

	// 删除分类
	if _, err := tx.Exec("DELETE FROM book_categories WHERE id = ?", categoryID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return imageURLs, nil
}

// GetBooksImagesByIDs 批量获取书籍图片，返回 map[bookID][]BookImage
func GetBooksImagesByIDs(db *sqlx.DB, bookIDs []int) (map[int][]BookImage, error) {
	if len(bookIDs) == 0 {
		return make(map[int][]BookImage), nil
	}

	query, args, err := sqlx.In("SELECT id, book_id, image_url, sort_order FROM book_images WHERE book_id IN (?) ORDER BY sort_order", bookIDs)
	if err != nil {
		return nil, err
	}
	query = db.Rebind(query)

	images := make([]BookImage, 0)
	if err := db.Select(&images, query, args...); err != nil {
		return nil, err
	}

	result := make(map[int][]BookImage, len(bookIDs))
	for _, img := range images {
		result[img.BookID] = append(result[img.BookID], img)
	}

	// Ensure every bookID has at least an empty slice
	for _, id := range bookIDs {
		if _, ok := result[id]; !ok {
			result[id] = []BookImage{}
		}
	}

	return result, nil
}
