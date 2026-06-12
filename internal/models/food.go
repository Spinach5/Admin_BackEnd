package models

import "github.com/jmoiron/sqlx"

type Food struct {
	ID          int     `db:"id" json:"id"`
	Name        string  `db:"name" json:"name"`
	ShopName    string  `db:"shop_name" json:"shop_name"`
	CanteenName string  `db:"canteen_name" json:"canteen_name"`
	SchoolID    string  `db:"school_id" json:"school_id"`
	Price       float64 `db:"price" json:"price"`
	Taste       string  `db:"taste" json:"taste"`
	Category    string  `db:"category" json:"category"`
}

func GetFoodsBySchool(db *sqlx.DB, schoolID string) ([]Food, error) {
	foods := make([]Food, 0)
	err := db.Select(&foods, "SELECT id, name, shop_name, canteen_name, school_id, price, taste, category FROM foods WHERE school_id = ? ORDER BY id", schoolID)
	return foods, err
}

func GetAllFoods(db *sqlx.DB) ([]Food, error) {
	foods := make([]Food, 0)
	err := db.Select(&foods, "SELECT id, name, shop_name, canteen_name, school_id, price, taste, category FROM foods ORDER BY id")
	return foods, err
}

func GetFoodByID(db *sqlx.DB, id int) (*Food, error) {
	var food Food
	err := db.Get(&food, "SELECT id, name, shop_name, canteen_name, school_id, price, taste, category FROM foods WHERE id = ?", id)
	return &food, err
}

func CreateFood(db *sqlx.DB, f *Food) error {
	_, err := db.Exec("INSERT INTO foods (name, shop_name, canteen_name, school_id, price, taste, category) VALUES (?, ?, ?, ?, ?, ?, ?)",
		f.Name, f.ShopName, f.CanteenName, f.SchoolID, f.Price, f.Taste, f.Category)
	return err
}

func UpdateFood(db *sqlx.DB, f *Food) error {
	_, err := db.Exec("UPDATE foods SET name=?, shop_name=?, canteen_name=?, school_id=?, price=?, taste=?, category=? WHERE id=?",
		f.Name, f.ShopName, f.CanteenName, f.SchoolID, f.Price, f.Taste, f.Category, f.ID)
	return err
}

func DeleteFood(db *sqlx.DB, id int) error {
	_, err := db.Exec("DELETE FROM foods WHERE id = ?", id)
	return err
}

// GetFoodsBySchoolWithFilters 按学校ID和可选过滤条件查询食物
func GetFoodsBySchoolWithFilters(db *sqlx.DB, schoolID, category, taste, canteenName, shopName, search string) ([]Food, error) {
	query := "SELECT id, name, shop_name, canteen_name, school_id, price, taste, category FROM foods WHERE school_id = ?"
	args := []interface{}{schoolID}

	if category != "" {
		query += " AND category = ?"
		args = append(args, category)
	}
	if taste != "" {
		query += " AND taste = ?"
		args = append(args, taste)
	}
	if canteenName != "" {
		query += " AND canteen_name = ?"
		args = append(args, canteenName)
	}
	if shopName != "" {
		query += " AND shop_name = ?"
		args = append(args, shopName)
	}
	if search != "" {
		query += " AND name LIKE ?"
		args = append(args, "%"+search+"%")
	}

	query += " ORDER BY id"

	foods := make([]Food, 0)
	err := db.Select(&foods, query, args...)
	return foods, err
}

// GetDistinctCategoriesBySchool 获取某学校下的所有食物分类
func GetDistinctCategoriesBySchool(db *sqlx.DB, schoolID string) ([]string, error) {
	var categories []string
	err := db.Select(&categories, "SELECT DISTINCT category FROM foods WHERE school_id = ? AND category != '' ORDER BY category", schoolID)
	return categories, err
}

// GetDistinctTastesBySchool 获取某学校下的所有食物口味
func GetDistinctTastesBySchool(db *sqlx.DB, schoolID string) ([]string, error) {
	var tastes []string
	err := db.Select(&tastes, "SELECT DISTINCT taste FROM foods WHERE school_id = ? AND taste != '' ORDER BY taste", schoolID)
	return tastes, err
}

// GetDistinctCanteensBySchool 获取某学校下的所有食堂名称
func GetDistinctCanteensBySchool(db *sqlx.DB, schoolID string) ([]string, error) {
	var canteens []string
	err := db.Select(&canteens, "SELECT DISTINCT canteen_name FROM foods WHERE school_id = ? AND canteen_name != '' ORDER BY canteen_name", schoolID)
	return canteens, err
}

// GetDistinctShopsBySchool 获取某学校下的所有店铺名称
func GetDistinctShopsBySchool(db *sqlx.DB, schoolID string) ([]string, error) {
	var shops []string
	err := db.Select(&shops, "SELECT DISTINCT shop_name FROM foods WHERE school_id = ? AND shop_name != '' ORDER BY shop_name", schoolID)
	return shops, err
}

func CreateFoodsBatch(db *sqlx.DB, foods []Food) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO foods (name, shop_name, canteen_name, school_id, price, taste, category) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	inserted := 0
	for _, f := range foods {
		_, err := stmt.Exec(f.Name, f.ShopName, f.CanteenName, f.SchoolID, f.Price, f.Taste, f.Category)
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
