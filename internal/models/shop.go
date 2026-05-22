package models

import "github.com/jmoiron/sqlx"

type Shop struct {
	ID          int     `db:"id" json:"id"`
	Name        string  `db:"name" json:"name"`
	CanteenName string  `db:"canteen_name" json:"canteen_name"`
	Rating      float64 `db:"rating" json:"rating"`
	Comment     string  `db:"comment" json:"comment"`
	Min         float64 `db:"min" json:"min"`
	Max         float64 `db:"max" json:"max"`
}

func GetAllShops(db *sqlx.DB) ([]Shop, error) {
	var shops []Shop
	err := db.Select(&shops, "SELECT id, name, canteen_name, rating, comment, min, max FROM shops ORDER BY id")
	return shops, err
}

func GetShopByID(db *sqlx.DB, id int) (*Shop, error) {
	var shop Shop
	err := db.Get(&shop, "SELECT id, name, canteen_name, rating, comment, min, max FROM shops WHERE id = ?", id)
	return &shop, err
}

func CreateShop(db *sqlx.DB, s *Shop) error {
	_, err := db.Exec("INSERT INTO shops (name, canteen_name, rating, comment, min, max) VALUES (?, ?, ?, ?, ?, ?)",
		s.Name, s.CanteenName, s.Rating, s.Comment, s.Min, s.Max)
	return err
}

func UpdateShop(db *sqlx.DB, s *Shop) error {
	_, err := db.Exec("UPDATE shops SET name=?, canteen_name=?, rating=?, comment=?, min=?, max=? WHERE id=?",
		s.Name, s.CanteenName, s.Rating, s.Comment, s.Min, s.Max, s.ID)
	return err
}

func DeleteShop(db *sqlx.DB, id int) error {
	_, err := db.Exec("DELETE FROM shops WHERE id = ?", id)
	return err
}

func CreateShopsBatch(db *sqlx.DB, shops []Shop) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO shops (name, canteen_name, rating, comment, min, max) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	inserted := 0
	for _, s := range shops {
		_, err := stmt.Exec(s.Name, s.CanteenName, s.Rating, s.Comment, s.Min, s.Max)
		if err != nil {
			continue
		}
		inserted++
	}

	if err := tx.Commit(); err != nil {
		return inserted, err
	}
	return inserted, nil
}
