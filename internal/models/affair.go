package models

import (
	"time"

	"github.com/jmoiron/sqlx"
)

type Affair struct {
	ID        int       `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Category  string    `db:"category" json:"category"`
	Link      string    `db:"link" json:"link"`
	Details   string    `db:"details" json:"details"`
	Channel   string    `db:"channel" json:"channel"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

func GetAllAffairs(db *sqlx.DB) ([]Affair, error) {
	var affairs []Affair
	err := db.Select(&affairs, "SELECT id, name, category, link, details, channel, created_at FROM affairs ORDER BY id")
	return affairs, err
}

func GetAffairByID(db *sqlx.DB, id int) (*Affair, error) {
	var affair Affair
	err := db.Get(&affair, "SELECT id, name, category, link, details, channel, created_at FROM affairs WHERE id = ?", id)
	return &affair, err
}

func CreateAffair(db *sqlx.DB, a *Affair) error {
	_, err := db.Exec("INSERT INTO affairs (name, category, link, details, channel) VALUES (?, ?, ?, ?, ?)",
		a.Name, a.Category, a.Link, a.Details, a.Channel)
	return err
}

func UpdateAffair(db *sqlx.DB, a *Affair) error {
	_, err := db.Exec("UPDATE affairs SET name=?, category=?, link=?, details=?, channel=? WHERE id=?",
		a.Name, a.Category, a.Link, a.Details, a.Channel, a.ID)
	return err
}

func DeleteAffair(db *sqlx.DB, id int) error {
	_, err := db.Exec("DELETE FROM affairs WHERE id = ?", id)
	return err
}

func CreateAffairsBatch(db *sqlx.DB, affairs []Affair) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO affairs (name, category, link, details, channel) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	inserted := 0
	for _, a := range affairs {
		_, err := stmt.Exec(a.Name, a.Category, a.Link, a.Details, a.Channel)
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
