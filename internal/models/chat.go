package models

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
)

type Conversation struct {
	ID        int    `db:"id" json:"conversation_id"`
	BookID    int    `db:"book_id" json:"book_id"`
	BuyerID   int    `db:"buyer_id" json:"buyer_id"`
	SellerID  int    `db:"seller_id" json:"seller_id"`
	CreatedAt string `db:"created_at" json:"created_at"`
	UpdatedAt string `db:"updated_at" json:"updated_at"`
}

type Message struct {
	ID             int    `db:"id" json:"id"`
	ConversationID int    `db:"conversation_id" json:"conversation_id"`
	SenderID       int    `db:"sender_id" json:"sender_id"`
	Content        string `db:"content" json:"content"`
	CreatedAt      string `db:"created_at" json:"created_at"`
}

// ---- 列表用的聚合结构 ----

type ConversationListItem struct {
	Conversation
	BookTitle string  `db:"book_title" json:"book_title"`
	BookImage *string `db:"book_image" json:"book_image"`
	OtherUserID      int    `db:"other_user_id" json:"-"`
	OtherUserNick    string `db:"other_nick" json:"-"`
	OtherUserStuID   string `db:"other_stu_id" json:"-"`
	OtherUser        struct {
		ID      int    `json:"id"`
		NickName string `json:"nickName"`
		StuID   string `json:"stuId"`
	} `json:"other_user"`
	LastContent  *string `db:"last_content" json:"-"`
	LastSenderID *int    `db:"last_sender_id" json:"-"`
	LastTime     *string `db:"last_time" json:"-"`
	LastMessage  *struct {
		Content   string `json:"content"`
		SenderID  int    `json:"sender_id"`
		CreatedAt string `json:"created_at"`
	} `json:"last_message"`
	UnreadCount int `db:"unread_count" json:"unread_count"`
}

// ---- Conversation CRUD ----

// FindOrCreateConversation 查找或创建会话（幂等）
func FindOrCreateConversation(db *sqlx.DB, bookID, buyerID, sellerID int) (*Conversation, error) {
	var conv Conversation
	err := db.Get(&conv, "SELECT * FROM conversation WHERE book_id = ? AND buyer_id = ? AND seller_id = ?", bookID, buyerID, sellerID)
	if err == nil {
		return &conv, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	result, err := db.Exec("INSERT INTO conversation (book_id, buyer_id, seller_id) VALUES (?, ?, ?)", bookID, buyerID, sellerID)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	conv.ID = int(id)
	conv.BookID = bookID
	conv.BuyerID = buyerID
	conv.SellerID = sellerID
	return &conv, nil
}

// GetConversationByID 获取单个会话
func GetConversationByID(db *sqlx.DB, id int) (*Conversation, error) {
	var conv Conversation
	err := db.Get(&conv, "SELECT * FROM conversation WHERE id = ?", id)
	return &conv, err
}

// GetConversationList 获取用户的所有会话列表（含最新消息预览、对方信息、未读数）
func GetConversationList(db *sqlx.DB, userID int) ([]ConversationListItem, error) {
	items := make([]ConversationListItem, 0)
	err := db.Select(&items, `
		SELECT
			c.*,
			b.title AS book_title,
			b.image_url AS book_image,
			CASE WHEN c.buyer_id = ? THEN c.seller_id ELSE c.buyer_id END AS other_user_id,
			ou.nickName AS other_nick,
			ou.stuId AS other_stu_id,
			lm.content AS last_content,
			lm.sender_id AS last_sender_id,
			lm.created_at AS last_time,
			(SELECT COUNT(*) FROM message m2 WHERE m2.conversation_id = c.id AND m2.sender_id != ? AND m2.created_at > COALESCE((SELECT created_at FROM message m3 WHERE m3.conversation_id = c.id AND m3.sender_id = ? ORDER BY created_at DESC LIMIT 1), '1970-01-01')) AS unread_count
		FROM conversation c
		JOIN book b ON c.book_id = b.book_id
		LEFT JOIN users ou ON ou.id = CASE WHEN c.buyer_id = ? THEN c.seller_id ELSE c.buyer_id END AND ou.isDeleted = 0
		LEFT JOIN message lm ON lm.id = (SELECT m.id FROM message m WHERE m.conversation_id = c.id ORDER BY m.created_at DESC LIMIT 1)
		WHERE c.buyer_id = ? OR c.seller_id = ?
		ORDER BY c.updated_at DESC`,
		userID, userID, userID, userID, userID, userID)
	if err != nil {
		return nil, err
	}

	// 组装嵌套结构
	for i := range items {
		items[i].OtherUser.ID = items[i].OtherUserID
		items[i].OtherUser.NickName = items[i].OtherUserNick
		items[i].OtherUser.StuID = items[i].OtherUserStuID
		if items[i].LastContent != nil {
			items[i].LastMessage = &struct {
				Content   string `json:"content"`
				SenderID  int    `json:"sender_id"`
				CreatedAt string `json:"created_at"`
			}{
				Content:   *items[i].LastContent,
				SenderID:  *items[i].LastSenderID,
				CreatedAt: *items[i].LastTime,
			}
		}
	}
	return items, err
}

// IsConversationParticipant 检查用户是否是会话参与者
func IsConversationParticipant(db *sqlx.DB, conversationID, userID int) (bool, error) {
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM conversation WHERE id = ? AND (buyer_id = ? OR seller_id = ?)", conversationID, userID, userID)
	return count > 0, err
}

// ---- Message CRUD ----

// CreateMessage 发送消息并更新会话 updated_at
func CreateMessage(db *sqlx.DB, msg *Message) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("INSERT INTO message (conversation_id, sender_id, content) VALUES (?, ?, ?)",
		msg.ConversationID, msg.SenderID, msg.Content)
	if err != nil {
		return err
	}

	_, err = tx.Exec("UPDATE conversation SET updated_at = NOW() WHERE id = ?", msg.ConversationID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetMessages 获取会话消息历史（正序分页）
func GetMessages(db *sqlx.DB, conversationID, page, pageSize int) ([]Message, int, error) {
	var total int
	if err := db.Get(&total, "SELECT COUNT(*) FROM message WHERE conversation_id = ?", conversationID); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	msgs := make([]Message, 0)
	err := db.Select(&msgs, "SELECT * FROM message WHERE conversation_id = ? ORDER BY created_at ASC LIMIT ? OFFSET ?",
		conversationID, pageSize, offset)
	return msgs, total, err
}

// UpdateConversationUpdatedAt 更新会话时间
func UpdateConversationUpdatedAt(db *sqlx.DB, conversationID int) error {
	_, err := db.Exec("UPDATE conversation SET updated_at = NOW() WHERE id = ?", conversationID)
	return err
}

// AdminConversation 管理员视角的会话详情
type AdminConversation struct {
	Conversation
	BookTitle    string `db:"book_title" json:"book_title"`
	BuyerNick    string `db:"buyer_nick" json:"buyer_nick"`
	BuyerStuID   string `db:"buyer_stu_id" json:"buyer_stu_id"`
	SellerNick   string `db:"seller_nick" json:"seller_nick"`
	SellerStuID  string `db:"seller_stu_id" json:"seller_stu_id"`
	MessageCount int    `db:"msg_count" json:"message_count"`
	LastContent  *string `db:"last_content" json:"last_content"`
	LastTime     *string `db:"last_time" json:"last_time"`
}

// GetConversationsByUserID 管理员根据用户ID查询其所有会话
func GetConversationsByUserID(db *sqlx.DB, userID int) ([]AdminConversation, error) {
	convs := make([]AdminConversation, 0)
	err := db.Select(&convs, `
		SELECT c.*,
			b.title AS book_title,
			bu.nickName AS buyer_nick, bu.stuId AS buyer_stu_id,
			su.nickName AS seller_nick, su.stuId AS seller_stu_id,
			(SELECT COUNT(*) FROM message m WHERE m.conversation_id = c.id) AS msg_count,
			lm.content AS last_content, lm.created_at AS last_time
		FROM conversation c
		JOIN book b ON c.book_id = b.book_id
		LEFT JOIN users bu ON c.buyer_id = bu.id
		LEFT JOIN users su ON c.seller_id = su.id
		LEFT JOIN message lm ON lm.id = (SELECT m.id FROM message m WHERE m.conversation_id = c.id ORDER BY m.created_at DESC LIMIT 1)
		WHERE c.buyer_id = ? OR c.seller_id = ?
		ORDER BY c.updated_at DESC`, userID, userID)
	return convs, err
}

// GetConversationMessages 获取会话的全部消息（管理员查看）
func GetConversationMessages(db *sqlx.DB, conversationID int) ([]Message, error) {
	msgs := make([]Message, 0)
	err := db.Select(&msgs, "SELECT * FROM message WHERE conversation_id = ? ORDER BY created_at ASC", conversationID)
	return msgs, err
}

// DeleteConversation 管理员删除会话（级联删除消息）
func DeleteConversation(db *sqlx.DB, conversationID int) error {
	_, err := db.Exec("DELETE FROM conversation WHERE id = ?", conversationID)
	return err
}
