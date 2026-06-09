package models

import "github.com/jmoiron/sqlx"

type Club struct {
	ID           int     `db:"id" json:"id"`
	Name         string  `db:"name" json:"name"`
	Introduction *string `db:"introduction" json:"introduction"`
	Activities   *string `db:"activities" json:"activities"`
	Category     *string `db:"category" json:"category"`
	ImageURL     *string `db:"image_url" json:"image_url"`
	Nature       int     `db:"nature" json:"nature"`
	Contact      *string `db:"contact" json:"contact"`
}

func GetAllClubs(db *sqlx.DB) ([]Club, error) {
	clubs := make([]Club, 0)
	err := db.Select(&clubs, "SELECT id, name, introduction, activities, category, image_url, nature, contact FROM clubs ORDER BY id")
	return clubs, err
}

func GetClubByID(db *sqlx.DB, id int) (*Club, error) {
	var club Club
	err := db.Get(&club, "SELECT id, name, introduction, activities, category, image_url, nature, contact FROM clubs WHERE id = ?", id)
	return &club, err
}

func CreateClub(db *sqlx.DB, c *Club) error {
	_, err := db.Exec("INSERT INTO clubs (name, introduction, activities, category, image_url, nature, contact) VALUES (?, ?, ?, ?, ?, ?, ?)",
		c.Name, c.Introduction, c.Activities, c.Category, c.ImageURL, c.Nature, c.Contact)
	return err
}

func UpdateClub(db *sqlx.DB, c *Club) error {
	_, err := db.Exec("UPDATE clubs SET name=?, introduction=?, activities=?, category=?, image_url=?, nature=?, contact=? WHERE id=?",
		c.Name, c.Introduction, c.Activities, c.Category, c.ImageURL, c.Nature, c.Contact, c.ID)
	return err
}

func DeleteClub(db *sqlx.DB, id int) error {
	_, err := db.Exec("DELETE FROM clubs WHERE id = ?", id)
	return err
}
