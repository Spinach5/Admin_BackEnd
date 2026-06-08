package models

import (
	"time"

	"github.com/jmoiron/sqlx"
)

type Admin struct {
	ID        int       `db:"id" json:"id"`
	Account   string    `db:"account" json:"account"`
	Password  string    `db:"password" json:"-"`
	SchoolID  string    `db:"schoolId" json:"schoolId"`
	IsSuper   int       `db:"is_super" json:"is_super"`
	IsActive  int       `db:"is_active" json:"is_active"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

func GetAdminByAccount(db *sqlx.DB, account string) (*Admin, error) {
	var admin Admin
	err := db.Get(&admin, "SELECT id, account, password, schoolId, is_super, is_active, created_at, updated_at FROM admins WHERE account = ?", account)
	return &admin, err
}

func GetAdminByID(db *sqlx.DB, id int) (*Admin, error) {
	var admin Admin
	err := db.Get(&admin, "SELECT id, account, schoolId, is_super, is_active, created_at, updated_at FROM admins WHERE id = ?", id)
	return &admin, err
}

func GetAllAdmins(db *sqlx.DB) ([]Admin, error) {
	admins := make([]Admin, 0)
	err := db.Select(&admins, "SELECT id, account, schoolId, is_super, is_active, created_at, updated_at FROM admins ORDER BY id")
	return admins, err
}

func CreateAdmin(db *sqlx.DB, account, password string, isSuper int) error {
	_, err := db.Exec("INSERT INTO admins (account, password, is_super) VALUES (?, ?, ?)", account, password, isSuper)
	return err
}

func UpdateAdmin(db *sqlx.DB, id int, account, password string, isSuper, isActive int) error {
	if password != "" {
		_, err := db.Exec("UPDATE admins SET account=?, password=?, is_super=?, is_active=? WHERE id=?",
			account, password, isSuper, isActive, id)
		return err
	}
	_, err := db.Exec("UPDATE admins SET account=?, is_super=?, is_active=? WHERE id=?",
		account, isSuper, isActive, id)
	return err
}

func DeleteAdmin(db *sqlx.DB, id int) error {
	_, err := db.Exec("DELETE FROM admins WHERE id = ?", id)
	return err
}

func UpdateAdminPassword(db *sqlx.DB, account, password string) error {
	_, err := db.Exec("UPDATE admins SET password=? WHERE account=?", password, account)
	return err
}

func SetAdminActive(db *sqlx.DB, account string, active int) error {
	_, err := db.Exec("UPDATE admins SET is_active = ? WHERE account = ?", active, account)
	return err
}

func UpdateAdminLastActive(db *sqlx.DB, account string) error {
	_, err := db.Exec("UPDATE admins SET last_active_at = NOW() WHERE account = ?", account)
	return err
}

func GetAdminLastActive(db *sqlx.DB, account string) (string, error) {
	var lastActive string
	err := db.Get(&lastActive, "SELECT last_active_at FROM admins WHERE account = ?", account)
	return lastActive, err
}

func CleanStaleAdminSessions(db *sqlx.DB, timeoutMinutes int) error {
	_, err := db.Exec(
		"UPDATE admins SET is_active = 0 WHERE is_active = 1 AND last_active_at IS NOT NULL AND last_active_at < DATE_SUB(NOW(), INTERVAL ? MINUTE)",
		timeoutMinutes,
	)
	return err
}
