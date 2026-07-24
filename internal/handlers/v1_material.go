package handlers

import (
	"log"
	"strconv"
	"strings"

	"web-backend/internal/database"
	"web-backend/internal/dto"
	"web-backend/internal/models"

	"github.com/gin-gonic/gin"
)

// V1GetMaterials 学生端查询教材
// @Summary 学生查询教材
// @Description 学生只能查询；按学期/班级筛选
// @Tags 教材管理
// @Security BearerAuth
// @Produce json
// @Param semester query string true "学期"
// @Param class_name query string false "班级名称"
// @Success 200 {object} dto.Response
// @Router /api/v1/materials [get]
func V1GetMaterials() gin.HandlerFunc {
	return func(c *gin.Context) {
		semester := strings.TrimSpace(c.Query("semester"))
		className := strings.TrimSpace(c.Query("class_name"))

		if semester == "" {
			dto.BadRequest(c, "请指定学期参数 (semester)")
			return
		}

		if className != "" {
			// 按班级 + 学期查询
			details, err := models.GetMaterialsByClassAndSemester(database.DB, className, semester)
			if err != nil {
				log.Printf("学生查询教材失败(class=%s,sem=%s): %v", className, semester, err)
				dto.InternalError(c, "查询教材失败")
				return
			}
			dto.SuccessWithTotal(c, details, len(details))
			return
		}

		// 仅按学期查询
		list, err := models.GetMaterialsBySemester(database.DB, semester)
		if err != nil {
			log.Printf("学生查询教材失败(sem=%s): %v", semester, err)
			dto.InternalError(c, "查询教材失败")
			return
		}
		dto.SuccessWithTotal(c, list, len(list))
	}
}

// V1GetMaterialByID 学生端查询单本教材
// @Summary 学生查询单本教材
// @Tags 教材管理
// @Security BearerAuth
// @Produce json
// @Param id path int true "教材ID"
// @Success 200 {object} dto.Response
// @Router /api/v1/materials/{id} [get]
func V1GetMaterialByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的教材ID")
			return
		}
		m, err := models.GetMaterialByID(database.DB, id)
		if err != nil {
			dto.Error(c, 404, "教材不存在")
			return
		}
		dto.Success(c, m)
	}
}

// V1GetClasses 学生端查询班级列表
// @Summary 学生查询班级列表（用于筛选教材）
// @Tags 教材管理
// @Security BearerAuth
// @Produce json
// @Success 200 {object} dto.Response
// @Router /api/v1/classes [get]
func V1GetClasses() gin.HandlerFunc {
	return func(c *gin.Context) {
		classes, err := models.GetAllClasses(database.DB)
		if err != nil {
			log.Printf("查询班级列表失败: %v", err)
			dto.InternalError(c, "查询班级列表失败")
			return
		}
		dto.SuccessWithTotal(c, classes, len(classes))
	}
}
