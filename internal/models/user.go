package models

import (
	"time"

	"github.com/jmoiron/sqlx"
)

type User struct {
	ID        int       `db:"id" json:"id"`
	Account   string    `db:"account" json:"account"`
	Password  string    `db:"password" json:"-"`
	IsSuper   int       `db:"is_super" json:"is_super"`
	IsActive  int       `db:"is_active" json:"is_active"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

func GetUserByAccount(db *sqlx.DB, account string) (*User, error) {
	var user User
	err := db.Get(&user, "SELECT id, account, password, is_super, is_active, created_at, updated_at FROM users WHERE account = ?", account)
	return &user, err
}

func GetUserByID(db *sqlx.DB, id int) (*User, error) {
	var user User
	err := db.Get(&user, "SELECT id, account, is_super, is_active, created_at, updated_at FROM users WHERE id = ?", id)
	return &user, err
}

func GetAllUsers(db *sqlx.DB) ([]User, error) {
	var users []User
	err := db.Select(&users, "SELECT id, account, is_super, is_active, created_at, updated_at FROM users ORDER BY id")
	return users, err
}

func CreateUser(db *sqlx.DB, account, password string, isSuper int) error {
	_, err := db.Exec("INSERT INTO users (account, password, is_super) VALUES (?, ?, ?)", account, password, isSuper)
	return err
}

func UpdateUser(db *sqlx.DB, id int, account, password string, isSuper, isActive int) error {
	if password != "" {
		_, err := db.Exec("UPDATE users SET account=?, password=?, is_super=?, is_active=? WHERE id=?",
			account, password, isSuper, isActive, id)
		return err
	}
	_, err := db.Exec("UPDATE users SET account=?, is_super=?, is_active=? WHERE id=?",
		account, isSuper, isActive, id)
	return err
}

func DeleteUser(db *sqlx.DB, id int) error {
	_, err := db.Exec("DELETE FROM users WHERE id = ?", id)
	return err
}

func UpdateUserPassword(db *sqlx.DB, account, password string) error {
	_, err := db.Exec("UPDATE users SET password=? WHERE account=?", password, account)
	return err
}

func SetUserActive(db *sqlx.DB, account string, active int) error {
	_, err := db.Exec("UPDATE users SET is_active = ? WHERE account = ?", active, account)
	return err
}
