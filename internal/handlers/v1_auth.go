package handlers

import (
	"database/sql"
	"log"
	"strconv"

	"web-backend/internal/config"
	"web-backend/internal/database"
	"web-backend/internal/dto"
	"web-backend/internal/middleware"
	"web-backend/internal/models"
	"web-backend/internal/services"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func StudentRegister(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.StudentRegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			dto.BadRequest(c, "请填写所有必填字段")
			return
		}

		if len(req.Password) < 6 {
			dto.BadRequest(c, "密码至少6位")
			return
		}

		_, err := models.GetUserByStuIDAndSchoolID(database.DB, req.StuID, req.SchoolID)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("查询用户失败: %v", err)
			dto.InternalError(c, "服务器错误")
			return
		}
		if err == nil {
			dto.BadRequest(c, "该学号在此学校已注册")
			return
		}

		hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
		if err != nil {
			log.Printf("密码加密失败: %v", err)
			dto.InternalError(c, "服务器错误")
			return
		}

		user := &models.User{
			StuID:        req.StuID,
			NickName:     req.NickName,
			SchoolID:     req.SchoolID,
			PasswordHash: string(hashed),
		}

		if err := models.CreateUserWithPassword(database.DB, user); err != nil {
			log.Printf("创建用户失败: %v", err)
			dto.InternalError(c, "注册失败")
			return
		}

		expireHours := parseExpireHours(cfg)
		token, err := services.GenerateToken(user.ID, "", 0, cfg.JWTSecret, expireHours)
		if err != nil {
			log.Printf("生成token失败: %v", err)
			dto.InternalError(c, "服务器错误")
			return
		}

		dto.Success(c, gin.H{
			"token": token,
			"user": gin.H{
				"id":       user.ID,
				"stuId":    user.StuID,
				"nickName": user.NickName,
				"schoolId": user.SchoolID,
			},
		})
	}
}

func StudentLogin(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.StudentLoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			dto.BadRequest(c, "请填写学号和密码")
			return
		}

		user, err := models.GetUserByStuIDWithPassword(database.DB, req.StuID)
		if err == sql.ErrNoRows {
			dto.Error(c, 200, "学号未注册")
			return
		}
		if err != nil {
			log.Printf("查询用户失败: %v", err)
			dto.InternalError(c, "服务器错误")
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
			dto.Error(c, 200, "密码错误")
			return
		}

		expireHours := parseExpireHours(cfg)
		token, err := services.GenerateToken(user.ID, "", 0, cfg.JWTSecret, expireHours)
		if err != nil {
			log.Printf("生成token失败: %v", err)
			dto.InternalError(c, "服务器错误")
			return
		}

		dto.Success(c, gin.H{
			"token": token,
			"user": gin.H{
				"id":       user.ID,
				"stuId":    user.StuID,
				"nickName": user.NickName,
				"schoolId": user.SchoolID,
			},
		})
	}
}

func StudentMe() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.GetStudentUserID(c)
		user, err := models.GetUserByID(database.DB, userID)
		if err != nil {
			dto.Unauthorized(c, "用户不存在")
			return
		}
		dto.Success(c, gin.H{
			"id":       user.ID,
			"stuId":    user.StuID,
			"nickName": user.NickName,
			"schoolId": user.SchoolID,
		})
	}
}

func parseExpireHours(cfg *config.Config) int {
	if cfg.JWTExpireHours == "" {
		return 24
	}
	h, err := strconv.Atoi(cfg.JWTExpireHours)
	if err != nil || h <= 0 {
		return 24
	}
	return h
}
