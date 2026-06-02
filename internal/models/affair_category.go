package models

import (
	"time"

	"github.com/jmoiron/sqlx"
)

type AffairCategory struct {
	ID        int       `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

func GetAllAffairCategories(db *sqlx.DB) ([]AffairCategory, error) {
	cats := make([]AffairCategory, 0)
	err := db.Select(&cats, "SELECT id, name, created_at FROM affair_categories ORDER BY id")
	return cats, err
}

func GetAffairCategoryByID(db *sqlx.DB, id int) (*AffairCategory, error) {
	var cat AffairCategory
	err := db.Get(&cat, "SELECT id, name, created_at FROM affair_categories WHERE id = ?", id)
	return &cat, err
}

func CreateAffairCategory(db *sqlx.DB, name string) error {
	_, err := db.Exec("INSERT INTO affair_categories (name) VALUES (?)", name)
	return err
}

func UpdateAffairCategory(db *sqlx.DB, id int, name string) error {
	_, err := db.Exec("UPDATE affair_categories SET name=? WHERE id=?", name, id)
	return err
}

func DeleteAffairCategory(db *sqlx.DB, id int) error {
	_, err := db.Exec("DELETE FROM affair_categories WHERE id = ?", id)
	return err
}
