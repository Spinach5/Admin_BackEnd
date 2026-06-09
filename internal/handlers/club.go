package handlers

import (
	"log"
	"strconv"

	"web-backend/internal/database"
	"web-backend/internal/dto"
	"web-backend/internal/models"

	"github.com/gin-gonic/gin"
)

// --- 管理员接口 ---

// GetClubs 获取社团列表
func GetClubs() gin.HandlerFunc {
	return func(c *gin.Context) {
		clubs, err := models.GetAllClubs(database.DB)
		if err != nil {
			log.Printf("获取社团列表失败: %v", err)
			dto.InternalError(c, "获取社团列表失败")
			return
		}
		dto.Success(c, clubs)
	}
}

// GetClubByID 获取单个社团
func GetClubByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的社团ID")
			return
		}

		club, err := models.GetClubByID(database.DB, id)
		if err != nil {
			log.Printf("获取社团详情失败 id=%d: %v", id, err)
			dto.Error(c, 200, "社团不存在")
			return
		}
		dto.Success(c, club)
	}
}

// CreateClub 添加社团
func CreateClub() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.CreateClubRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			dto.BadRequest(c, "缺少必要参数")
			return
		}

		club := &models.Club{
			Name:         req.Name,
			Introduction: req.Introduction,
			Activities:   req.Activities,
			Category:     req.Category,
			ImageURL:     req.ImageURL,
			Nature:       req.Nature,
			Contact:      req.Contact,
			PrincipalID:  req.PrincipalID,
		}

		if err := models.CreateClub(database.DB, club); err != nil {
			log.Printf("添加社团失败: %v", err)
			dto.InternalError(c, "添加社团失败")
			return
		}

		dto.SuccessMessage(c, "添加社团成功")
	}
}

// UpdateClub 更新社团
func UpdateClub() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的社团ID")
			return
		}

		var req dto.UpdateClubRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			dto.BadRequest(c, "缺少必要参数")
			return
		}

		club := &models.Club{
			ID:           id,
			Name:         req.Name,
			Introduction: req.Introduction,
			Activities:   req.Activities,
			Category:     req.Category,
			ImageURL:     req.ImageURL,
			Nature:       req.Nature,
			Contact:      req.Contact,
			PrincipalID:  req.PrincipalID,
		}

		if err := models.UpdateClub(database.DB, club); err != nil {
			log.Printf("更新社团失败: %v", err)
			dto.InternalError(c, "更新社团失败")
			return
		}

		dto.SuccessMessage(c, "更新社团成功")
	}
}

// DeleteClub 删除社团
func DeleteClub() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的社团ID")
			return
		}

		if err := models.DeleteClub(database.DB, id); err != nil {
			log.Printf("删除社团失败: %v", err)
			dto.InternalError(c, "删除社团失败")
			return
		}

		dto.SuccessMessage(c, "删除社团成功")
	}
}

// --- V1 学生接口 (只读) ---

// V1GetClubs 学生获取社团列表
func V1GetClubs() gin.HandlerFunc {
	return func(c *gin.Context) {
		clubs, err := models.GetAllClubs(database.DB)
		if err != nil {
			log.Printf("获取社团列表失败: %v", err)
			dto.InternalError(c, "获取社团列表失败")
			return
		}
		dto.Success(c, clubs)
	}
}

// V1GetClubByID 学生获取单个社团
func V1GetClubByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的社团ID")
			return
		}

		club, err := models.GetClubByID(database.DB, id)
		if err != nil {
			log.Printf("获取社团详情失败 id=%d: %v", id, err)
			dto.Error(c, 200, "社团不存在")
			return
		}
		dto.Success(c, club)
	}
}
