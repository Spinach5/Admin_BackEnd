package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

	"web-backend/internal/database"
	"web-backend/internal/dto"
	"web-backend/internal/models"
	"web-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// GetSemesters 获取学期列表（用于下拉选择）
// @Summary 获取学期列表
// @Description 获取所有已存在的学期列表，用于前端下拉选择
// @Tags 教材管理
// @Security BearerAuth
// @Produce json
// @Success 200 {object} dto.Response
// @Router /api/materials/semesters [get]
func GetSemesters() gin.HandlerFunc {
	return func(c *gin.Context) {
		semesters, err := models.GetSemesters(database.DB)
		if err != nil {
			log.Printf("获取学期列表失败: %v", err)
			dto.InternalError(c, "获取学期列表失败")
			return
		}
		dto.Success(c, semesters)
	}
}

// GetMaterials 查询教材列表
// @Summary 查询教材列表
// @Description 管理员查询教材；支持按学期、班级、关键字筛选
// @Tags 教材管理
// @Security BearerAuth
// @Produce json
// @Param semester query string false "学期"
// @Param class_name query string false "班级名称"
// @Param keyword query string false "关键字（ISBN/书名/作者/出版社）"
// @Success 200 {object} dto.Response
// @Router /api/materials [get]
func GetMaterials() gin.HandlerFunc {
	return func(c *gin.Context) {
		semester := strings.TrimSpace(c.Query("semester"))
		className := strings.TrimSpace(c.Query("class_name"))
		keyword := strings.TrimSpace(c.Query("keyword"))

		// 1. 按 班级 + 学期 查询
		if className != "" && semester != "" {
			log.Printf("按班级+学期查询教材: className=%s, semester=%s", className, semester)
			details, err := models.GetMaterialsByClassAndSemester(database.DB, className, semester)
			if err != nil {
				log.Printf("查询教材失败(class=%s,sem=%s): %v", className, semester, err)
				dto.InternalError(c, "查询教材失败: "+err.Error())
				return
			}
			log.Printf("查询成功，共 %d 条记录", len(details))
			resp := make([]models.MaterialResponse, 0, len(details))
			for _, d := range details {
				r := d.Material.ToMaterialResponse()
				r.Semester = semester
				resp = append(resp, r)
			}
			dto.SuccessWithTotal(c, resp, len(resp))
			return
		}

		// 2. 仅按学期查询（含关联班级）
		if semester != "" {
			list, err := models.GetMaterialsBySemester(database.DB, semester)
			if err != nil {
				log.Printf("查询教材失败(sem=%s): %v", semester, err)
				dto.InternalError(c, "查询教材失败")
				return
			}
			resp := make([]models.MaterialResponse, 0, len(list))
			for _, m := range list {
				resp = append(resp, m.ToMaterialResponse())
			}
			dto.SuccessWithTotal(c, resp, len(resp))
			return
		}

		// 3. 仅按班级查询
		if className != "" {
			list, err := models.GetMaterialsByClass(database.DB, className)
			if err != nil {
				log.Printf("查询教材失败(class=%s): %v", className, err)
				dto.InternalError(c, "查询教材失败")
				return
			}
			resp := make([]models.MaterialResponse, 0, len(list))
			for _, m := range list {
				resp = append(resp, m.ToMaterialResponse())
			}
			dto.SuccessWithTotal(c, resp, len(resp))
			return
		}

		// 4. 关键字搜索（不限定学期）
		if keyword != "" {
			list, err := models.SearchMaterialsWithClasses(database.DB, keyword)
			if err != nil {
				log.Printf("搜索教材失败: %v", err)
				dto.InternalError(c, "搜索教材失败")
				return
			}
			resp := make([]models.MaterialResponse, 0, len(list))
			for _, m := range list {
				resp = append(resp, m.ToMaterialResponse())
			}
			dto.SuccessWithTotal(c, resp, len(resp))
			return
		}

		// 4. 全部教材
		list, err := models.GetAllMaterialsWithClasses(database.DB)
		if err != nil {
			log.Printf("查询教材失败: %v", err)
			dto.InternalError(c, "查询教材失败")
			return
		}
		resp := make([]models.MaterialResponse, 0, len(list))
		for _, m := range list {
			resp = append(resp, m.ToMaterialResponse())
		}
		dto.SuccessWithTotal(c, resp, len(resp))
	}
}

// CreateMaterial 添加教材
// @Summary 添加教材（管理员）
// @Tags 教材管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body dto.CreateMaterialRequest true "教材信息"
// @Success 200 {object} dto.Response
// @Router /api/materials [post]
func CreateMaterial() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.CreateMaterialRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			dto.BadRequest(c, "缺少必要参数")
			return
		}

		m := &models.Material{
			ISBN:      models.CleanISBN(req.ISBN),
			Title:     req.Title,
			Author:    sql.NullString{String: req.Author, Valid: req.Author != ""},
			Publisher: sql.NullString{String: req.Publisher, Valid: req.Publisher != ""},
			Price:     sql.NullFloat64{Float64: req.Price, Valid: req.Price > 0},
			ExtraInfo: sql.NullString{String: req.ExtraInfo, Valid: req.ExtraInfo != ""},
		}

		if err := models.CreateMaterial(database.DB, m); err != nil {
			log.Printf("添加教材失败: %v", err)
			dto.InternalError(c, "添加教材失败")
			return
		}

		// 若同时指定了学期+班级，则将其关联到对应教材包
		if req.Semester != "" && len(req.ClassNames) > 0 {
			rows := []models.MaterialImportRow{{
				CourseName: "",
				ISBN:       req.ISBN,
				Title:      req.Title,
				Publisher:  req.Publisher,
				Author:     req.Author,
				Price:      req.Price,
				ClassNames: req.ClassNames,
				Department: "",
				ExtraInfo:  req.ExtraInfo,
			}}
			if _, err := models.ImportMaterialsFromRows(database.DB, rows, req.Semester, req.AcademicYear, c.Request.Context()); err != nil {
				log.Printf("教材已添加但关联班级失败: %v", err)
				dto.Success(c, gin.H{
					"book_id": m.BookID,
					"warning": "教材已添加，但关联班级/教材包失败: " + err.Error(),
				})
				return
			}
		}

		dto.Success(c, gin.H{"book_id": m.BookID})
	}
}

// UpdateMaterial 修改教材
// @Summary 修改教材（管理员）
// @Tags 教材管理
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "教材ID"
// @Param request body dto.UpdateMaterialRequest true "教材信息"
// @Success 200 {object} dto.Response
// @Router /api/materials/{id} [put]
func UpdateMaterial() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的教材ID")
			return
		}

		var req dto.UpdateMaterialRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			dto.BadRequest(c, "缺少必要参数")
			return
		}

		m := &models.Material{
			BookID:    id,
			ISBN:      models.CleanISBN(req.ISBN),
			Title:     req.Title,
			Author:    sql.NullString{String: req.Author, Valid: req.Author != ""},
			Publisher: sql.NullString{String: req.Publisher, Valid: req.Publisher != ""},
			Price:     sql.NullFloat64{Float64: req.Price, Valid: req.Price > 0},
			ExtraInfo: sql.NullString{String: req.ExtraInfo, Valid: req.ExtraInfo != ""},
		}

		if err := models.UpdateMaterial(database.DB, m); err != nil {
			log.Printf("修改教材失败: %v", err)
			dto.InternalError(c, "修改教材失败")
			return
		}

		dto.SuccessMessage(c, "修改成功")
	}
}

// DeleteMaterial 删除教材
// @Summary 删除教材（管理员）
// @Tags 教材管理
// @Security BearerAuth
// @Produce json
// @Param id path int true "教材ID"
// @Success 200 {object} dto.Response
// @Router /api/materials/{id} [delete]
func DeleteMaterial() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			dto.BadRequest(c, "无效的教材ID")
			return
		}

		if err := models.DeleteMaterial(database.DB, id); err != nil {
			log.Printf("删除教材失败: %v", err)
			dto.InternalError(c, "删除教材失败")
			return
		}

		dto.SuccessMessage(c, "删除成功")
	}
}

// ImportMaterialsExcel Excel 批量导入教材
// @Summary Excel 导入教材（管理员）
// @Description 表头: 课程名称 标准书号 教材名称 出版社 作者 估定价 折扣 折后价 班级信息 院系 备注; 需通过 query 传递 semester
// @Tags 教材管理
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param semester query string true "学期，如 2024-2025-1"
// @Param academic_year query string false "学年，如 2024-2025"
// @Param file formData file true "Excel 文件"
// @Success 200 {object} dto.Response
// @Router /api/materials/import [post]
const maxExcelFileSize = 10 << 20

func ImportMaterialsExcel() gin.HandlerFunc {
	return func(c *gin.Context) {
		semester := strings.TrimSpace(c.Query("semester"))
		if semester == "" {
			dto.BadRequest(c, "请通过 query 参数指定学期 (semester)")
			return
		}
		academicYear := strings.TrimSpace(c.Query("academic_year"))

		file, fileHeader, err := c.Request.FormFile("file")
		if err != nil {
			dto.BadRequest(c, "请上传 Excel 文件")
			return
		}
		defer file.Close()

		if fileHeader.Size > maxExcelFileSize {
			dto.BadRequest(c, fmt.Sprintf("文件大小不能超过 %dMB", maxExcelFileSize>>20))
			return
		}

		log.Printf("开始导入教材 Excel: filename=%s, size=%d, semester=%s", fileHeader.Filename, fileHeader.Size, semester)

		result, err := services.ParseExcel(file)
		if err != nil {
			dto.BadRequest(c, err.Error())
			return
		}

		rows, parseErr := convertMaterialRows(result)
		if parseErr != nil {
			log.Printf("教材 Excel 解析行数据失败: %v", parseErr)
			dto.BadRequest(c, parseErr.Error())
			return
		}

		log.Printf("Excel 解析完成，共 %d 条数据待导入", len(rows))

		inserted, err := models.ImportMaterialsFromRows(database.DB, rows, semester, academicYear, c.Request.Context())
		if err != nil {
			log.Printf("Excel 导入教材失败: %v", err)
			dto.InternalError(c, "导入失败: "+err.Error())
			return
		}

		log.Printf("教材 Excel 导入成功: 总计 %d 条，成功导入 %d 条", len(rows), inserted)

		dto.Success(c, gin.H{
			"inserted": inserted,
			"total":    len(rows),
			"message":  fmt.Sprintf("成功导入 %d 条教材记录", inserted),
		})
	}
}

// PreviewMaterialsExcel 预览 Excel 解析结果（不写入数据库）
// @Summary 预览 Excel 教材数据
// @Tags 教材管理
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Excel 文件"
// @Success 200 {object} dto.Response
// @Router /api/materials/preview [post]
func PreviewMaterialsExcel() gin.HandlerFunc {
	return func(c *gin.Context) {
		file, _, err := c.Request.FormFile("file")
		if err != nil {
			dto.BadRequest(c, "请上传 Excel 文件")
			return
		}
		defer file.Close()

		result, err := services.ParseExcel(file)
		if err != nil {
			dto.BadRequest(c, err.Error())
			return
		}

		rows, parseErr := convertMaterialRows(result)
		if parseErr != nil {
			dto.BadRequest(c, parseErr.Error())
			return
		}

		dto.Success(c, gin.H{
			"headers": result.Headers,
			"rows":    rows,
			"total":   len(rows),
		})
	}
}

// convertMaterialRows 将 Excel 解析结果转换为 MaterialImportRow
func convertMaterialRows(result *services.ExcelResult) ([]models.MaterialImportRow, error) {
	rows := make([]models.MaterialImportRow, 0, len(result.Rows))
	for _, r := range result.Rows {
		// 兼容多种表头写法（带/不带空格、繁简差异）
		get := func(keys ...string) string {
			for _, k := range keys {
				if v, ok := r[k]; ok && v != "" {
					return v
				}
			}
			return ""
		}

		isbn := models.CleanISBN(get("标准书号", "书号", "ISBN", "isbn"))
		title := get("教材名称", "书名", "教材名")
		if isbn == "" || title == "" {
			// 跳过无效行
			continue
		}

		priceStr := get("估定价", "定价", "价格", "折后价")
		price, _ := strconv.ParseFloat(strings.TrimSpace(priceStr), 64)

		// 班级信息可能含多个班级（逗号/分号/中文逗号分隔）
		classField := get("班级信息", "班级")
		classNames := splitClassNames(classField)

		// 备注包含课程名称 + 备注 + 折扣信息
		courseName := get("课程名称", "课程")
		discount := get("折扣")
		finalPrice := get("折后价")
		remark := get("备注")
		extraParts := []string{}
		if courseName != "" {
			extraParts = append(extraParts, "课程:"+courseName)
		}
		if discount != "" {
			extraParts = append(extraParts, "折扣:"+discount)
		}
		if finalPrice != "" {
			extraParts = append(extraParts, "折后价:"+finalPrice)
		}
		if remark != "" {
			extraParts = append(extraParts, remark)
		}

		rows = append(rows, models.MaterialImportRow{
			CourseName: courseName,
			ISBN:       isbn,
			Title:      title,
			Publisher:  get("出版社"),
			Author:     get("作者"),
			Price:      price,
			ClassNames: classNames,
			Department: get("院系", "系别"),
			ExtraInfo:  strings.Join(extraParts, "；"),
		})
	}
	return rows, nil
}

// splitClassNames 拆分班级信息字符串
func splitClassNames(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	// 替换常见分隔符为统一分隔符
	s = strings.ReplaceAll(s, "，", ",")
	s = strings.ReplaceAll(s, "；", ";")
	s = strings.ReplaceAll(s, "、", ",")
	s = strings.ReplaceAll(s, "/", ",")
	s = strings.ReplaceAll(s, "\n", ",")

	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
