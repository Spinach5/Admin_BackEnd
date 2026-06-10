package handlers

import (
	"log"
	"strconv"

	"web-backend/internal/database"
	"web-backend/internal/dto"
	"web-backend/internal/middleware"
	"web-backend/internal/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// GetUsers 获取用户列表
// @Summary 获取所有用户
// @Tags 管理员管理
// @Security BearerAuth
// @Produce json
// @Success 200 {object} dto.Response
// @Router /api/admin/users [get]
func GetAdmins() gin.HandlerFunc {
	return func(c *gin.Context) {
		users, err := models.GetAllAdmins(database.DB)
		if err != nil {
			dto.InternalError(c, "获取用户列表失败")
			return
		}
		dto.Success(c, users)
	}
}

// GetUserByID 获取单个用户
// @Summary 获取单个用户
// @Tags 管理员管理
// @Security BearerAuth
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} dto.Response
// @Router /api/admin/users/{id} [get]
func GetAdminByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的用户ID")
			return
		}
		user, err := models.GetAdminByID(database.DB, id)
		if err != nil {
			dto.Error(c, 404, "用户不存在")
			return
		}
		dto.Success(c, user)
	}
}

// CreateUser 添加用户
// @Summary 添加用户
// @Tags 管理员管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body dto.CreateAdminRequest true "用户信息"
// @Success 200 {object} dto.Response
// @Router /api/admin/users [post]
func CreateAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.CreateAdminRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			dto.BadRequest(c, "缺少必要参数")
			return
		}

		// 检查账号是否已存在
		if _, err := models.GetAdminByAccount(database.DB, req.Account); err == nil {
			dto.Error(c, 200, "账号已存在，请使用其他账号")
			return
		}

		hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
		if err != nil {
			dto.InternalError(c, "服务器错误")
			return
		}

		if err := models.CreateAdmin(database.DB, req.Account, string(hashed), req.SchoolID, req.IsSuper); err != nil {
			log.Printf("添加用户失败: %v", err)
			dto.InternalError(c, "添加用户失败")
			return
		}

		dto.SuccessMessage(c, "添加用户成功")
	}
}

// UpdateUser 更新用户
// @Summary 更新用户
// @Tags 管理员管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Param request body dto.UpdateAdminRequest true "用户信息"
// @Success 200 {object} dto.Response
// @Router /api/admin/users/{id} [put]
func UpdateAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的用户ID")
			return
		}

		var req dto.UpdateAdminRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			dto.BadRequest(c, "缺少必要参数")
			return
		}

		// 不能修改自己的超级管理员状态
		currentID := middleware.GetCurrentUserID(c)
		if id == currentID && *req.IsSuper != middleware.GetCurrentIsSuper(c) {
			dto.BadRequest(c, "不能修改自己的管理员权限")
			return
		}

		hashed := ""
		if req.Password != "" {
			h, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
			if err != nil {
				dto.InternalError(c, "服务器错误")
				return
			}
			hashed = string(h)
		}

		if err := models.UpdateAdmin(database.DB, id, req.Account, hashed, *req.IsSuper, *req.IsActive); err != nil {
			log.Printf("更新用户失败: %v", err)
			dto.InternalError(c, "更新用户失败")
			return
		}

		dto.SuccessMessage(c, "更新用户成功")
	}
}

// UpdateAdminInfo 更新管理员信息 (不含密码)
func UpdateAdminInfo() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的用户ID")
			return
		}

		var req dto.UpdateAdminInfoRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			dto.BadRequest(c, "缺少必要参数")
			return
		}

		currentID := middleware.GetCurrentUserID(c)
		if id == currentID && *req.IsSuper != middleware.GetCurrentIsSuper(c) {
			dto.BadRequest(c, "不能修改自己的管理员权限")
			return
		}

		if err := models.UpdateAdmin(database.DB, id, req.Account, "", *req.IsSuper, *req.IsActive); err != nil {
			log.Printf("更新管理员失败: %v", err)
			dto.InternalError(c, "更新管理员失败")
			return
		}

		dto.SuccessMessage(c, "更新管理员成功")
	}
}

// DeleteUser 删除用户
// @Summary 删除用户
// @Tags 管理员管理
// @Security BearerAuth
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} dto.Response
// @Router /api/admin/users/{id} [delete]
func DeleteAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的用户ID")
			return
		}

		// 不能删除自己
		if id == middleware.GetCurrentUserID(c) {
			dto.BadRequest(c, "不能删除自己的账户")
			return
		}

		// 检查目标用户是否为超级管理员
		target, err := models.GetAdminByID(database.DB, id)
		if err != nil {
			dto.Error(c, 404, "用户不存在")
			return
		}
		if target.IsSuper == 1 {
			dto.BadRequest(c, "不能删除超级管理员账户")
			return
		}

		if err := models.DeleteAdmin(database.DB, id); err != nil {
			log.Printf("删除用户失败: %v", err)
			dto.InternalError(c, "删除用户失败")
			return
		}

		dto.SuccessMessage(c, "删除用户成功")
	}
}
