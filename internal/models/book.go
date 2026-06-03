package models

import (
	"fmt"

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
	err := db.Select(&books, `SELECT b.*, u.nickName, u.stuId FROM book b
		LEFT JOIN users u ON b.user_id = u.id AND u.isDeleted = 0
		ORDER BY b.book_id`)
	return books, err
}

func GetBookByID(db *sqlx.DB, id int) (*BookWithUser, error) {
	var book BookWithUser
	err := db.Get(&book, `SELECT b.*, u.nickName, u.stuId FROM book b
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

func GetBooksByCategory(db *sqlx.DB, category string) ([]BookWithUser, error) {
	books := make([]BookWithUser, 0)
	err := db.Select(&books, `SELECT b.*, u.nickName, u.stuId FROM book b
		LEFT JOIN users u ON b.user_id = u.id AND u.isDeleted = 0
		WHERE b.category = ? AND b.status = 'active'
		ORDER BY b.book_id DESC`, category)
	return books, err
}

func GetBooksByUser(db *sqlx.DB, userID int) ([]BookWithUser, error) {
	books := make([]BookWithUser, 0)
	err := db.Select(&books, `SELECT b.*, u.nickName, u.stuId FROM book b
		LEFT JOIN users u ON b.user_id = u.id AND u.isDeleted = 0
		WHERE b.user_id = ?
		ORDER BY b.book_id DESC`, userID)
	return books, err
}
