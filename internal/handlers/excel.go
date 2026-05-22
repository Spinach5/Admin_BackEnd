package handlers

import (
	"fmt"
	"log"

	"web-backend/internal/database"
	"web-backend/internal/dto"
	"web-backend/internal/models"
	"web-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// ImportExcel 导入 Excel 数据
// @Summary Excel 导入
// @Description 上传 Excel 文件并导入到指定表
// @Tags 系统
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param table query string true "目标表: shops, foods, affairs"
// @Param file formData file true "Excel 文件"
// @Success 200 {object} dto.Response
// @Router /api/excel/import [post]
func ImportExcel() gin.HandlerFunc {
	return func(c *gin.Context) {
		table := c.Query("table")
		if table == "" {
			dto.BadRequest(c, "请指定目标表 (table=shops|foods|affairs)")
			return
		}

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

		var inserted int
		switch table {
		case "shops":
			inserted, err = importShops(result)
		case "foods":
			inserted, err = importFoods(result)
		case "affairs":
			inserted, err = importAffairs(result)
		default:
			dto.BadRequest(c, "不支持的目标表: "+table)
			return
		}

		if err != nil {
			log.Printf("Excel 导入失败: %v", err)
			dto.InternalError(c, "导入失败: "+err.Error())
			return
		}

		dto.SuccessMessage(c, fmt.Sprintf("成功导入 %d 条记录", inserted))
	}
}

func importShops(result *services.ExcelResult) (int, error) {
	var shops []models.Shop
	for _, row := range result.Rows {
		rating, _ := services.RowToFloat(row, "rating")
		min, _ := services.RowToFloat(row, "min")
		max, _ := services.RowToFloat(row, "max")

		shops = append(shops, models.Shop{
			Name:        row["name"],
			CanteenName: row["canteen_name"],
			Rating:      rating,
			Comment:     row["comment"],
			Min:         min,
			Max:         max,
		})
	}
	return models.CreateShopsBatch(database.DB, shops)
}

func importFoods(result *services.ExcelResult) (int, error) {
	var foods []models.Food
	for _, row := range result.Rows {
		price, _ := services.RowToFloat(row, "price")

		foods = append(foods, models.Food{
			Name:        row["name"],
			ShopName:    row["shop_name"],
			CanteenName: row["canteen_name"],
			Price:       price,
			Taste:       row["taste"],
			Category:    row["category"],
		})
	}
	return models.CreateFoodsBatch(database.DB, foods)
}

func importAffairs(result *services.ExcelResult) (int, error) {
	var affairs []models.Affair
	for _, row := range result.Rows {
		affairs = append(affairs, models.Affair{
			Name:     row["affair_name"],
			Category: row["affair_category"],
			Link:     row["link"],
			Details:  row["details"],
			Channel:  row["channel"],
		})
	}
	return models.CreateAffairsBatch(database.DB, affairs)
}

// PreviewExcel 预览 Excel 数据
// @Summary Excel 预览
// @Description 上传 Excel 文件预览解析结果，不实际导入
// @Tags 系统
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Excel 文件"
// @Success 200 {object} dto.Response
// @Router /api/excel/preview [post]
func PreviewExcel() gin.HandlerFunc {
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

		dto.Success(c, gin.H{
			"headers": result.Headers,
			"rows":    result.Rows,
			"total":   result.Total,
		})
	}
}
