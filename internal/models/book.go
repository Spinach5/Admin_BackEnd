package models

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
)

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

type BookWithUser struct {
	Book
	NickName string `db:"nickName" json:"nickName"`
	StuID    string `db:"stuId" json:"stuId"`
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
	_, err := db.Exec(`INSERT INTO book (title, category, image_url, price, isbn, contact, user_id, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		b.Title, b.Category, b.ImageURL, b.Price, b.ISBN, b.Contact, b.UserID, b.Status)
	return err
}

func UpdateBook(db *sqlx.DB, b *Book) error {
	_, err := db.Exec(`UPDATE book SET title=?, category=?, image_url=?, price=?, isbn=?, contact=?, status=?
		WHERE book_id=?`,
		b.Title, b.Category, b.ImageURL, b.Price, b.ISBN, b.Contact, b.Status, b.BookID)
	return err
}

func DeleteBook(db *sqlx.DB, id int) error {
	_, err := db.Exec("DELETE FROM book WHERE book_id = ?", id)
	return err
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

	result, err := tx.Exec(`INSERT INTO book (title, category, image_url, price, isbn, contact, user_id, status, description, `+"`condition`"+`, school_id)
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

	_, err = tx.Exec(`UPDATE book SET title=?, category=?, image_url=?, price=?, isbn=?, contact=?, status=?, description=?, `+"`condition`"+`=?
		WHERE book_id=?`,
		b.Title, b.Category, b.ImageURL, b.Price, b.ISBN, b.Contact, b.Status, b.Description, b.Condition, b.BookID)
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
	if err == nil {
		return false
	}
	// mysql driver returns *mysql.MySQLError for server errors
	if mysqlErr, ok := err.(interface{ Number() uint16 }); ok {
		return mysqlErr.Number() == 1062
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

func DeleteBookCategory(db *sqlx.DB, id int) error {
	_, err := db.Exec("DELETE FROM book_categories WHERE id = ?", id)
	return err
}
