package handlers

import (
	"crypto/sha256"
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

// StudentRegister 注册/登录（幂等）
// 已注册 → 验证教务 + 更新密码/昵称 → 返回 token
// 未注册 → 验证教务 + 创建用户 → 返回 token
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

		// 教务系统凭证验证（内部通过云函数 captcha 自动求解滑块）
		if !cfg.SkipEduVerify {
			if err := services.VerifySchoolCredentials(req.SchoolID, req.StuID, req.Password); err != nil {
				log.Printf("教务系统验证失败 stuId=%s schoolId=%s: %v", req.StuID, req.SchoolID, err)
				dto.BadRequest(c, err.Error())
				return
			}
		} else {
			log.Printf("跳过教务系统验证 stuId=%s schoolId=%s (SKIP_EDU_VERIFY=true)", req.StuID, req.SchoolID)
		}

		// 前端发来的是 RSA 密文，先 SHA-256 再 bcrypt（不可逆哈希，非加密）
		sha := sha256.Sum256([]byte(req.Password))
		hashed, err := bcrypt.GenerateFromPassword(sha[:], bcrypt.DefaultCost)
		if err != nil {
			log.Printf("密码哈希失败: %v", err)
			dto.InternalError(c, "服务器错误")
			return
		}
		passwordHash := string(hashed)

		existing, err := models.GetUserByStuIDAndSchoolID(database.DB, req.StuID, req.SchoolID)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("查询用户失败: %v", err)
			dto.InternalError(c, "服务器错误")
			return
		}

		var userID int
		var nickName string

		if err == sql.ErrNoRows {
			// 未注册：创建用户
			user := &models.User{
				StuID:        req.StuID,
				NickName:     req.NickName,
				SchoolID:     req.SchoolID,
				PasswordHash: passwordHash,
			}

			if err := models.CreateUserWithPassword(database.DB, user); err != nil {
				log.Printf("创建用户失败: %v", err)
				dto.InternalError(c, "注册失败")
				return
			}
			userID = user.ID
			nickName = req.NickName
		} else {
			// 已注册：更新密码、昵称（如有变化）
			userID = existing.ID
			nickName = existing.NickName

			if req.NickName != "" && req.NickName != existing.NickName {
				if err := models.UpdateUserNickName(database.DB, userID, req.NickName); err != nil {
					log.Printf("更新昵称失败: %v", err)
				}
				nickName = req.NickName
			}

			if err := models.UpdateUserPassword(database.DB, userID, passwordHash); err != nil {
				log.Printf("更新密码失败: %v", err)
				dto.InternalError(c, "服务器错误")
				return
			}
		}

		models.UpdateUserLastActive(database.DB, userID)

		token, err := services.GenerateToken(userID, "", 0, cfg.JWTSecret, 0)
		if err != nil {
			log.Printf("生成token失败: %v", err)
			dto.InternalError(c, "服务器错误")
			return
		}

		dto.Success(c, gin.H{
			"token": token,
			"user": gin.H{
				"id":       userID,
				"stuId":    req.StuID,
				"nickName": nickName,
				"schoolId": req.SchoolID,
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

// CheckUser 检查用户是否存在 (公开接口)
// @Summary 检查用户是否存在
// @Description 通过学号和学校代码查询用户是否已注册
// @Tags 认证
// @Produce json
// @Param stuId query string true "学号"
// @Param schoolId query string true "学校代码"
// @Success 200 {object} dto.Response
// @Router /api/v1/auth/check-user [get]
func CheckUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		stuID := c.Query("stuId")
		schoolID := c.Query("schoolId")

		if stuID == "" || schoolID == "" {
			dto.BadRequest(c, "请提供 stuId 和 schoolId")
			return
		}

		_, err := models.GetUserByStuIDAndSchoolID(database.DB, stuID, schoolID)
		exists := err == nil

		dto.Success(c, gin.H{
			"stuId":    stuID,
			"schoolId": schoolID,
			"exists":   exists,
		})
	}
}

// StudentHeartbeat 学生心跳检测，更新最后活跃时间
func StudentHeartbeat() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.GetStudentUserID(c)
		if userID == 0 {
			dto.Unauthorized(c, "未找到用户信息")
			return
		}

		if err := models.UpdateUserLastActive(database.DB, userID); err != nil {
			log.Printf("更新学生心跳失败 userId=%d: %v", userID, err)
			dto.InternalError(c, "心跳更新失败")
			return
	}

		dto.SuccessMessage(c, "ok")
	}
}
