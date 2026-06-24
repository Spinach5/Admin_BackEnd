package handlers

import (
	"log"
	"strconv"

	"web-backend/internal/database"
	"web-backend/internal/dto"
	"web-backend/internal/middleware"
	"web-backend/internal/models"

	"github.com/gin-gonic/gin"
)

// V1CreateConversation 发起/获取会话（幂等）
func V1CreateConversation() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.GetStudentUserID(c)

		var req struct {
			BookID int `json:"book_id"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || req.BookID == 0 {
			dto.BadRequest(c, "请提供有效的 book_id")
			return
		}

		// 查书籍，获取 seller_id
		book, err := models.GetBookByID(database.DB, req.BookID)
		if err != nil {
			dto.Error(c, 404, "书籍不存在")
			return
		}
		if book.UserID == userID {
			dto.BadRequest(c, "不能和自己聊天")
			return
		}

		conv, err := models.FindOrCreateConversation(database.DB, req.BookID, userID, book.UserID)
		if err != nil {
			log.Printf("创建会话失败: %v", err)
			dto.InternalError(c, "创建会话失败")
			return
		}

		// 查询对方用户信息
		otherID := book.UserID
		other, err := models.GetUserByID(database.DB, otherID)
		if err != nil {
			log.Printf("查询用户失败: %v", err)
			dto.InternalError(c, "查询用户失败")
			return
		}

		dto.Success(c, gin.H{
			"conversation_id": conv.ID,
			"book_id":         book.BookID,
			"book_title":      book.Title,
			"book_image":      book.ImageURL,
			"other_user": gin.H{
				"id":       other.ID,
				"nickName": other.NickName,
				"stuId":    other.StuID,
			},
			"created_at": conv.CreatedAt,
		})
	}
}

// V1GetConversations 我的会话列表
func V1GetConversations() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.GetStudentUserID(c)

		items, err := models.GetConversationList(database.DB, userID)
		if err != nil {
			log.Printf("获取会话列表失败: %v", err)
			dto.InternalError(c, "获取会话列表失败")
			return
		}
		if items == nil {
			items = make([]models.ConversationListItem, 0)
		}

		dto.Success(c, items)
	}
}

// V1GetMessages 获取消息历史
func V1GetMessages() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.GetStudentUserID(c)
		convID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的会话ID")
			return
		}

		ok, err := models.IsConversationParticipant(database.DB, convID, userID)
		if err != nil || !ok {
			dto.Forbidden(c, "无权访问此会话")
			return
		}

		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
		if page < 1 {
			page = 1
		}
		if pageSize < 1 || pageSize > 100 {
			pageSize = 20
		}

		msgs, total, err := models.GetMessages(database.DB, convID, page, pageSize)
		if err != nil {
			log.Printf("获取消息失败: %v", err)
			dto.InternalError(c, "获取消息失败")
			return
		}
		if msgs == nil {
			msgs = make([]models.Message, 0)
		}

		dto.SuccessWithTotal(c, msgs, total)
	}
}

// V1SendMessage 发送消息
func V1SendMessage() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.GetStudentUserID(c)
		convID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的会话ID")
			return
		}

		ok, err := models.IsConversationParticipant(database.DB, convID, userID)
		if err != nil || !ok {
			dto.Forbidden(c, "无权在此会话发消息")
			return
		}

		var req struct {
			Content string `json:"content"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || req.Content == "" {
			dto.BadRequest(c, "消息内容不能为空")
			return
		}

		msg := &models.Message{
			ConversationID: convID,
			SenderID:       userID,
			Content:        req.Content,
		}

		if err := models.CreateMessage(database.DB, msg); err != nil {
			log.Printf("发送消息失败: %v", err)
			dto.InternalError(c, "发送消息失败")
			return
		}

		dto.Success(c, gin.H{
			"id":              msg.ID,
			"conversation_id": msg.ConversationID,
			"sender_id":       msg.SenderID,
			"content":         msg.Content,
			"created_at":      msg.CreatedAt,
		})
	}
}

// ---- 管理员聊天接口 ----

// AdminGetConversationsByUser 管理员根据用户ID查询聊天记录
func AdminGetConversationsByUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := strconv.Atoi(c.Param("userId"))
		if err != nil {
			dto.BadRequest(c, "无效的用户ID")
			return
		}

		convs, err := models.GetConversationsByUserID(database.DB, userID)
		if err != nil {
			log.Printf("管理员查询会话失败: %v", err)
			dto.InternalError(c, "查询失败")
			return
		}
		if convs == nil {
			convs = make([]models.AdminConversation, 0)
		}

		dto.Success(c, convs)
	}
}

// AdminGetConversationMessages 管理员查看会话消息
func AdminGetConversationMessages() gin.HandlerFunc {
	return func(c *gin.Context) {
		convID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的会话ID")
			return
		}

		msgs, err := models.GetConversationMessages(database.DB, convID)
		if err != nil {
			log.Printf("管理员查询消息失败: %v", err)
			dto.InternalError(c, "查询失败")
			return
		}
		if msgs == nil {
			msgs = make([]models.Message, 0)
		}

		dto.Success(c, msgs)
	}
}

// AdminDeleteConversation 管理员删除会话
func AdminDeleteConversation() gin.HandlerFunc {
	return func(c *gin.Context) {
		convID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的会话ID")
			return
		}

		if err := models.DeleteConversation(database.DB, convID); err != nil {
			log.Printf("管理员删除会话失败: %v", err)
			dto.InternalError(c, "删除失败")
			return
		}

		dto.SuccessMessage(c, "删除成功")
	}
}

