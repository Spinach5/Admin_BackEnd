package models

import "github.com/jmoiron/sqlx"

type Food struct {
	ID          int     `db:"id" json:"id"`
	Name        string  `db:"name" json:"name"`
	ShopName    string  `db:"shop_name" json:"shop_name"`
	CanteenName string  `db:"canteen_name" json:"canteen_name"`
	Price       float64 `db:"price" json:"price"`
	Taste       string  `db:"taste" json:"taste"`
	Category    string  `db:"category" json:"category"`
}

func GetAllFoods(db *sqlx.DB) ([]Food, error) {
	var foods []Food
	err := db.Select(&foods, "SELECT id, name, shop_name, canteen_name, price, taste, category FROM foods ORDER BY id")
	return foods, err
}

func GetFoodByID(db *sqlx.DB, id int) (*Food, error) {
	var food Food
	err := db.Get(&food, "SELECT id, name, shop_name, canteen_name, price, taste, category FROM foods WHERE id = ?", id)
	return &food, err
}

func CreateFood(db *sqlx.DB, f *Food) error {
	_, err := db.Exec("INSERT INTO foods (name, shop_name, canteen_name, price, taste, category) VALUES (?, ?, ?, ?, ?, ?)",
		f.Name, f.ShopName, f.CanteenName, f.Price, f.Taste, f.Category)
	return err
}

func UpdateFood(db *sqlx.DB, f *Food) error {
	_, err := db.Exec("UPDATE foods SET name=?, shop_name=?, canteen_name=?, price=?, taste=?, category=? WHERE id=?",
		f.Name, f.ShopName, f.CanteenName, f.Price, f.Taste, f.Category, f.ID)
	return err
}

func DeleteFood(db *sqlx.DB, id int) error {
	_, err := db.Exec("DELETE FROM foods WHERE id = ?", id)
	return err
}

func CreateFoodsBatch(db *sqlx.DB, foods []Food) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO foods (name, shop_name, canteen_name, price, taste, category) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	inserted := 0
	for _, f := range foods {
		_, err := stmt.Exec(f.Name, f.ShopName, f.CanteenName, f.Price, f.Taste, f.Category)
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
