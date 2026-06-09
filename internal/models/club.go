package models

import (
	"errors"
	"slices"

	"github.com/jmoiron/sqlx"
)

type Club struct {
	ID           int     `db:"id" json:"id"`
	Name         string  `db:"name" json:"name"`
	Introduction *string `db:"introduction" json:"introduction"`
	Activities   *string `db:"activities" json:"activities"`
	Category     *string `db:"category" json:"category"`
	ImageURL     *string `db:"image_url" json:"image_url"`
	Nature       int     `db:"nature" json:"nature"`
	SchoolId     string  `db:"schoolId" json:"schoolId"`
	Contact      *string `db:"contact" json:"contact"`
	PrincipalID  *int    `db:"principal_id" json:"principal_id"`
}

// ClubWithPrincipal 查询时带负责人的 nickName
type ClubWithPrincipal struct {
	ID            int     `db:"id" json:"id"`
	Name          string  `db:"name" json:"name"`
	Introduction  *string `db:"introduction" json:"introduction"`
	Activities    *string `db:"activities" json:"activities"`
	Category      *string `db:"category" json:"category"`
	ImageURL      *string `db:"image_url" json:"image_url"`
	Nature        int     `db:"nature" json:"nature"`
	SchoolId      string  `db:"schoolId" json:"schoolId"`
	Contact       *string `db:"contact" json:"contact"`
	PrincipalID   *int    `db:"principal_id" json:"principal_id"`
	PrincipalName string  `db:"principal_name" json:"principal_name"`
}

var allowedCategories = []string{
	"学术科技类",
	"创新创业类",
	"文化艺术类",
	"体育活动类",
	"志愿公益类",
	"思想政治类",
	"其他",
}

// validateClub 校验俱乐部数据的合法性
func validateClub(c *Club) error {
	if c.Nature < 0 || c.Nature > 2 {
		return errors.New("nature must be between 0 and 2")
	}
	if c.Name == "" {
		return errors.New("name cannot be empty")
	}
	if c.Category != nil {
		if !slices.Contains(allowedCategories, *c.Category) {
			return errors.New("invalid category: must be one of [学术科技类, 创新创业类, 文化艺术类, 体育活动类, 志愿公益类, 思想政治类]")
		}
	}
	return nil
}

func GetAllClubs(db *sqlx.DB) ([]ClubWithPrincipal, error) {
	clubs := make([]ClubWithPrincipal, 0)
	err := db.Select(&clubs, `SELECT c.id, c.name, c.introduction, c.activities, c.category, c.image_url, c.schoolId, c.nature, c.contact, c.principal_id,
		COALESCE(u.nickName, '') AS principal_name
		FROM clubs c
		LEFT JOIN users u ON c.principal_id = u.id
		ORDER BY c.id`)
	return clubs, err
}

func GetClubByID(db *sqlx.DB, id int) (*ClubWithPrincipal, error) {
	var club ClubWithPrincipal
	err := db.Get(&club, `SELECT c.id, c.name, c.introduction, c.activities, c.category, c.image_url, c.schoolId, c.nature, c.contact, c.principal_id,
		COALESCE(u.nickName, '') AS principal_name
		FROM clubs c
		LEFT JOIN users u ON c.principal_id = u.id
		WHERE c.id = ?`, id)
	return &club, err
}

func CreateClub(db *sqlx.DB, c *Club) error {
	if err := validateClub(c); err != nil {
		return err
	}

	_, err := db.Exec("INSERT INTO clubs (name, introduction, activities, category, image_url, schoolId, nature, contact, principal_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		c.Name, c.Introduction, c.Activities, c.Category, c.ImageURL, c.SchoolId, c.Nature, c.Contact, c.PrincipalID)
	return err
}

func UpdateClub(db *sqlx.DB, c *Club) error {
	if err := validateClub(c); err != nil {
		return err
	}

	_, err := db.Exec("UPDATE clubs SET name=?, introduction=?, activities=?, category=?, image_url=?, schoolId=?, nature=?, contact=?, principal_id=? WHERE id=?",
		c.Name, c.Introduction, c.Activities, c.Category, c.ImageURL, c.SchoolId, c.Nature, c.Contact, c.PrincipalID, c.ID)
	return err
}

func DeleteClub(db *sqlx.DB, id int) error {
	_, err := db.Exec("DELETE FROM clubs WHERE id = ?", id)
	return err
}
