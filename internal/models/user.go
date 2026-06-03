package models

import "github.com/jmoiron/sqlx"

type User struct {
	ID        int    `db:"id" json:"id"`
	StuID     string `db:"stuId" json:"stuId"`
	NickName  string `db:"nickName" json:"nickName"`
	SchoolID  string `db:"schoolId" json:"schoolId"`
	PasswordHash string `db:"password_hash" json:"-"`
	CreatedAt string `db:"createdAt" json:"createdAt"`
	IsDeleted int    `db:"isDeleted" json:"isDeleted"`
}

func GetAllUsers(db *sqlx.DB) ([]User, error) {
	users := make([]User, 0)
	err := db.Select(&users, "SELECT id, stuId, nickName, schoolId, createdAt, isDeleted FROM users WHERE isDeleted = 0 ORDER BY id")
	return users, err
}

func GetUserByID(db *sqlx.DB, id int) (*User, error) {
	var user User
	err := db.Get(&user, "SELECT id, stuId, nickName, schoolId, createdAt, isDeleted FROM users WHERE id = ?", id)
	return &user, err
}

func GetUserByStuID(db *sqlx.DB, stuID string) (*User, error) {
	var user User
	err := db.Get(&user, "SELECT id, stuId, nickName, schoolId, createdAt, isDeleted FROM users WHERE stuId = ? AND isDeleted = 0", stuID)
	return &user, err
}

func CreateUser(db *sqlx.DB, u *User) error {
	_, err := db.Exec("INSERT INTO users (stuId, nickName, schoolId) VALUES (?, ?, ?)",
		u.StuID, u.NickName, u.SchoolID)
	return err
}

func CreateUserWithPassword(db *sqlx.DB, u *User) error {
	result, err := db.Exec(
		"INSERT INTO users (stuId, nickName, schoolId, password_hash) VALUES (?, ?, ?, ?)",
		u.StuID, u.NickName, u.SchoolID, u.PasswordHash,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	u.ID = int(id)
	return nil
}

func GetUserByStuIDWithPassword(db *sqlx.DB, stuID string) (*User, error) {
	var user User
	err := db.Get(&user,
		"SELECT id, stuId, nickName, schoolId, password_hash, createdAt, isDeleted FROM users WHERE stuId = ? AND isDeleted = 0",
		stuID,
	)
	return &user, err
}

func UpdateUser(db *sqlx.DB, u *User) error {
	_, err := db.Exec("UPDATE users SET stuId=?, nickName=?, schoolId=? WHERE id=? AND isDeleted=0",
		u.StuID, u.NickName, u.SchoolID, u.ID)
	return err
}

func SoftDeleteUser(db *sqlx.DB, id int) error {
	_, err := db.Exec("UPDATE users SET isDeleted = 1 WHERE id = ?", id)
	return err
}

func HardDeleteUser(db *sqlx.DB, id int) error {
	_, err := db.Exec("DELETE FROM users WHERE id = ?", id)
	return err
}
