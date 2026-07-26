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

		log.Printf("V1GetMaterials: semester=%s, className=%s", semester, className)

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
			log.Printf("学生查询成功: %d 本教材", len(details))
			resp := make([]models.MaterialResponse, 0, len(details))
			for _, d := range details {
				r := d.Material.ToMaterialResponse()
				r.Semester = semester
				r.Classes = []string{className}
				resp = append(resp, r)
			}
			dto.SuccessWithTotal(c, resp, len(resp))
			return
		}

		// 仅按学期查询
		list, err := models.GetMaterialsBySemester(database.DB, semester)
		if err != nil {
			log.Printf("学生查询教材失败(sem=%s): %v", semester, err)
			dto.InternalError(c, "查询教材失败")
			return
		}
		log.Printf("学生查询成功(sem=%s): %d 本教材", semester, len(list))
		resp := make([]models.MaterialResponse, 0, len(list))
		for _, m := range list {
			resp = append(resp, m.ToMaterialResponse())
		}
		dto.SuccessWithTotal(c, resp, len(resp))
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
		resp := m.ToMaterialResponse()
		var semesters []string
		err = database.DB.Select(&semesters, `SELECT DISTINCT p.semester FROM package_books pb
			JOIN book_packages p ON pb.package_id = p.package_id
			WHERE pb.book_id = ?`, m.BookID)
		if err == nil && len(semesters) > 0 {
			resp.Semester = semesters[0]
		}
		var classes []string
		err = database.DB.Select(&classes, `SELECT DISTINCT c.class_name FROM package_books pb
			JOIN book_packages p ON pb.package_id = p.package_id
			JOIN class_packages cp ON p.package_id = cp.package_id
			JOIN classes c ON cp.class_id = c.class_id
			WHERE pb.book_id = ? ORDER BY c.class_name`, m.BookID)
		if err == nil {
			resp.Classes = classes
		}
		dto.Success(c, resp)
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
