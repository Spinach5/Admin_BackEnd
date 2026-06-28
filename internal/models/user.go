package models

import "github.com/jmoiron/sqlx"

type User struct {
	ID        int    `db:"id" json:"id"`
	StuID     string `db:"stuId" json:"stuId"`
	NickName  string `db:"nickName" json:"nickName"`
	SchoolID  string `db:"schoolId" json:"schoolId"`
	// PasswordHash is intentionally omitted from non-auth queries (GetAllUsers, GetUserByID, etc.).
	// Only auth-specific functions (GetUserByStuIDWithPassword, CreateUserWithPassword) include it.
	PasswordHash string `db:"password_hash" json:"-"`
	CreatedAt string `db:"createdAt" json:"createdAt"`
	IsDeleted int    `db:"isDeleted" json:"isDeleted"`
	IsFrozen  int    `db:"is_frozen" json:"is_frozen"`
}

func GetAllUsers(db *sqlx.DB) ([]User, error) {
	users := make([]User, 0)
	err := db.Select(&users, "SELECT id, stuId, nickName, schoolId, createdAt, isDeleted, is_frozen FROM users WHERE isDeleted = 0 ORDER BY id")
	return users, err
}

func GetUserByID(db *sqlx.DB, id int) (*User, error) {
	var user User
	err := db.Get(&user, "SELECT id, stuId, nickName, schoolId, createdAt, isDeleted, is_frozen FROM users WHERE id = ?", id)
	return &user, err
}

func GetUserByStuID(db *sqlx.DB, stuID string) (*User, error) {
	var user User
	err := db.Get(&user, "SELECT id, stuId, nickName, schoolId, createdAt, isDeleted, is_frozen FROM users WHERE stuId = ? AND isDeleted = 0", stuID)
	return &user, err
}

func GetUserByStuIDAndSchoolID(db *sqlx.DB, stuID, schoolID string) (*User, error) {
	var user User
	err := db.Get(&user, "SELECT id, stuId, nickName, schoolId, createdAt, isDeleted, is_frozen FROM users WHERE stuId = ? AND schoolId = ? AND isDeleted = 0", stuID, schoolID)
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
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	u.ID = int(id)
	return nil
}

func GetUserByStuIDAndSchoolIDWithPassword(db *sqlx.DB, stuID, schoolID string) (*User, error) {
	var user User
	err := db.Get(&user,
		"SELECT id, stuId, nickName, schoolId, password_hash, createdAt, isDeleted, is_frozen FROM users WHERE stuId = ? AND schoolId = ? AND isDeleted = 0",
		stuID, schoolID,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func GetUserByStuIDWithPassword(db *sqlx.DB, stuID string) (*User, error) {
	var user User
	err := db.Get(&user,
		"SELECT id, stuId, nickName, schoolId, password_hash, createdAt, isDeleted, is_frozen FROM users WHERE stuId = ? AND isDeleted = 0",
		stuID,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func UpdateUser(db *sqlx.DB, u *User) error {
	_, err := db.Exec("UPDATE users SET stuId=?, nickName=?, schoolId=? WHERE id=? AND isDeleted=0",
		u.StuID, u.NickName, u.SchoolID, u.ID)
	return err
}

func UpdateUserPassword(db *sqlx.DB, userID int, passwordHash string) error {
	_, err := db.Exec("UPDATE users SET password_hash = ? WHERE id = ? AND isDeleted = 0", passwordHash, userID)
	return err
}

func UpdateUserNickName(db *sqlx.DB, userID int, nickName string) error {
	_, err := db.Exec("UPDATE users SET nickName = ? WHERE id = ? AND isDeleted = 0", nickName, userID)
	return err
}

func SoftDeleteUser(db *sqlx.DB, id int) error {
	_, err := db.Exec("UPDATE users SET isDeleted = 1 WHERE id = ?", id)
	return err
}

func HardDeleteUser(db *sqlx.DB, id int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. 删除该用户发送的所有消息（message.sender_id 无外键约束）
	if _, err := tx.Exec("DELETE FROM message WHERE sender_id = ?", id); err != nil {
		return err
	}

	// 2. 删除该用户参与的所有会话（conversation.buyer_id/seller_id 无外键约束）
	//    关联的 message 会通过 ON DELETE CASCADE 自动删除
	if _, err := tx.Exec("DELETE FROM conversation WHERE buyer_id = ? OR seller_id = ?", id, id); err != nil {
		return err
	}

	// 3. 删除该用户发布的书籍（book.user_id 无外键约束）
	//    关联的 book_images、book_wants 会通过 ON DELETE CASCADE 自动删除
	if _, err := tx.Exec("DELETE FROM book WHERE user_id = ?", id); err != nil {
		return err
	}

	// 4. 硬删除用户（clubs.principal_id → SET NULL, purchase.buyer_id → CASCADE, book_wants.user_id → CASCADE 由数据库FK自动处理）
	if _, err := tx.Exec("DELETE FROM users WHERE id = ?", id); err != nil {
		return err
	}

	return tx.Commit()
}

func UpdateUserLastActive(db *sqlx.DB, userID int) error {
	_, err := db.Exec("UPDATE users SET last_active_at = NOW() WHERE id = ?", userID)
	return err
}

func SetUserFrozen(db *sqlx.DB, userID int, frozen bool) error {
	v := 0
	if frozen {
		v = 1
	}
	_, err := db.Exec("UPDATE users SET is_frozen = ? WHERE id = ?", v, userID)
	return err
}

func IsUserFrozen(db *sqlx.DB, userID int) (bool, error) {
	var v int
	err := db.Get(&v, "SELECT is_frozen FROM users WHERE id = ?", userID)
	return v == 1, err
}

func GetUserLastActive(db *sqlx.DB, userID int) (string, error) {
	var lastActive string
	err := db.Get(&lastActive, "SELECT last_active_at FROM users WHERE id = ? AND isDeleted = 0", userID)
	return lastActive, err
}
