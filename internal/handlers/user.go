package handlers

import (
	"log"
	"strconv"

	"web-backend/internal/database"
	"web-backend/internal/dto"
	"web-backend/internal/models"

	"github.com/gin-gonic/gin"
)

// GetUsers 获取普通用户列表
func GetUsers() gin.HandlerFunc {
	return func(c *gin.Context) {
		users, err := models.GetAllUsers(database.DB)
		if err != nil {
			log.Printf("获取用户列表失败: %v", err)
			dto.InternalError(c, "获取用户列表失败")
			return
		}
		dto.Success(c, users)
	}
}

// GetUserByID 获取单个用户
func GetUserByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的用户ID")
			return
		}
		user, err := models.GetUserByID(database.DB, id)
		if err != nil {
			dto.Error(c, 404, "用户不存在")
			return
		}
		dto.Success(c, user)
	}
}

// CreateUser 添加普通用户
func CreateUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.CreateUserRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			dto.BadRequest(c, "缺少必要参数")
			return
		}

		if _, err := models.GetUserByStuID(database.DB, req.StuID); err == nil {
			dto.Error(c, 200, "学号已存在")
			return
		}

		user := &models.User{
			StuID:    req.StuID,
			NickName: req.NickName,
			SchoolID: req.SchoolID,
		}

		if err := models.CreateUser(database.DB, user); err != nil {
			log.Printf("添加用户失败: %v", err)
			dto.InternalError(c, "添加用户失败")
			return
		}

		dto.SuccessMessage(c, "添加用户成功")
	}
}

// UpdateUser 更新普通用户
func UpdateUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的用户ID")
			return
		}

		var req dto.UpdateUserRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			dto.BadRequest(c, "缺少必要参数")
			return
		}

		user := &models.User{
			ID:       id,
			StuID:    req.StuID,
			NickName: req.NickName,
			SchoolID: req.SchoolID,
		}

		if err := models.UpdateUser(database.DB, user); err != nil {
			log.Printf("更新用户失败: %v", err)
			dto.InternalError(c, "更新用户失败")
			return
		}

		dto.SuccessMessage(c, "更新用户成功")
	}
}

// SoftDeleteUser 软删除用户
func SoftDeleteUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的用户ID")
			return
		}

		if err := models.SoftDeleteUser(database.DB, id); err != nil {
			log.Printf("软删除用户失败: %v", err)
			dto.InternalError(c, "删除用户失败")
			return
		}

		dto.SuccessMessage(c, "删除用户成功")
	}
}

// HardDeleteUser 硬删除用户
func HardDeleteUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的用户ID")
			return
		}

		if err := models.HardDeleteUser(database.DB, id); err != nil {
			log.Printf("硬删除用户失败: %v", err)
			dto.InternalError(c, "删除用户失败")
			return
		}

		dto.SuccessMessage(c, "删除用户成功")
	}
}

// SetUserFrozen 冻结/解冻用户
func SetUserFrozen() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的用户ID")
			return
		}

		var req struct {
			Frozen bool `json:"frozen"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			dto.BadRequest(c, "参数错误")
			return
		}

		if err := models.SetUserFrozen(database.DB, id, req.Frozen); err != nil {
			log.Printf("设置冻结状态失败: %v", err)
			dto.InternalError(c, "操作失败")
			return
		}

		msg := "已解冻"
		if req.Frozen {
			msg = "已冻结"
		}
		dto.SuccessMessage(c, msg)
	}
}
