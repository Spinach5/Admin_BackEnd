package handlers

import (
	"database/sql"
	"log"

	"web-backend/internal/config"
	"web-backend/internal/database"
	"web-backend/internal/dto"
	"web-backend/internal/middleware"
	"web-backend/internal/models"
	"web-backend/internal/services"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// Login 用户登录
// @Summary 用户登录
// @Description 使用账号密码登录，返回 JWT token
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "登录信息"
// @Success 200 {object} dto.Response
// @Router /api/auth/login [post]
func Login(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			dto.BadRequest(c, "账号密码不能为空")
			return
		}

		log.Printf("登录请求，账号: %s", req.Account)

		user, err := models.GetUserByAccount(database.DB, req.Account)
		if err == sql.ErrNoRows {
			dto.Error(c, 200, "账号不存在")
			return
		}
		if err != nil {
			log.Printf("查询用户失败: %v", err)
			dto.InternalError(c, "服务器错误")
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
			dto.Error(c, 200, "密码错误")
			return
		}

		if user.IsActive == 1 {
			dto.Error(c, 200, "账号已经登录")
			return
		}

		models.SetUserActive(database.DB, user.Account, 1)

		expireHours := 24
		if v := cfg.JWTExpireHours; v != "" {
			if h, err := parseHours(v); err == nil {
				expireHours = h
			}
		}

		token, err := services.GenerateToken(user.ID, user.Account, user.IsSuper, cfg.JWTSecret, expireHours)
		if err != nil {
			log.Printf("生成 token 失败: %v", err)
			dto.InternalError(c, "服务器错误")
			return
		}

		dto.Success(c, gin.H{
			"token": token,
			"user": gin.H{
				"id":       user.ID,
				"account":  user.Account,
				"is_super": user.IsSuper,
			},
		})
	}
}

// Logout 用户登出
// @Summary 用户登出
// @Description 登出当前用户
// @Tags 认证
// @Security BearerAuth
// @Produce json
// @Success 200 {object} dto.Response
// @Router /api/auth/logout [post]
func Logout() gin.HandlerFunc {
	return func(c *gin.Context) {
		account := middleware.GetCurrentAccount(c)
		if account != "" {
			models.SetUserActive(database.DB, account, 0)
			log.Printf("用户 %s 已登出", account)
		}
		dto.SuccessMessage(c, "已退出登录")
	}
}

// GetMe 获取当前用户信息
// @Summary 获取当前用户
// @Description 获取当前登录用户的详细信息
// @Tags 认证
// @Security BearerAuth
// @Produce json
// @Success 200 {object} dto.Response
// @Router /api/auth/me [get]
func GetMe() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.GetCurrentUserID(c)
		user, err := models.GetUserByID(database.DB, userID)
		if err != nil {
			dto.Unauthorized(c, "用户不存在")
			return
		}
		dto.Success(c, user)
	}
}

// ChangePassword 修改密码
// @Summary 修改密码
// @Description 修改当前用户的登录密码
// @Tags 认证
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body dto.ChangePasswordRequest true "密码信息"
// @Success 200 {object} dto.Response
// @Router /api/auth/change-password [put]
func ChangePassword() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.ChangePasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			dto.BadRequest(c, "请填写完整信息")
			return
		}

		if req.OldPassword == req.NewPassword {
			dto.BadRequest(c, "新密码不能与旧密码相同")
			return
		}

		account := middleware.GetCurrentAccount(c)
		user, err := models.GetUserByAccount(database.DB, account)
		if err != nil {
			dto.InternalError(c, "服务器错误")
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
			dto.BadRequest(c, "原密码错误")
			return
		}

		hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 12)
		if err != nil {
			dto.InternalError(c, "服务器错误")
			return
		}

		if err := models.UpdateUserPassword(database.DB, account, string(hashed)); err != nil {
			dto.InternalError(c, "修改密码失败")
			return
		}

		dto.SuccessMessage(c, "密码修改成功")
	}
}

func parseHours(s string) (int, error) {
	var h int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil
		}
		h = h*10 + int(c-'0')
	}
	return h, nil
}
