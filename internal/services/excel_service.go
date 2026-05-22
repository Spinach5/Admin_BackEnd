package services

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

type ExcelRow map[string]string

type ExcelResult struct {
	Headers  []string   `json:"headers"`
	Rows     []ExcelRow `json:"rows"`
	Total    int        `json:"total"`
}

func ParseExcel(file io.Reader) (*ExcelResult, error) {
	f, err := excelize.OpenReader(file)
	if err != nil {
		return nil, fmt.Errorf("无法解析 Excel 文件: %v", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("Excel 文件中没有工作表")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("读取工作表失败: %v", err)
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("Excel 文件至少需要包含表头和一行数据")
	}

	headers := make([]string, len(rows[0]))
	for i, h := range rows[0] {
		headers[i] = strings.TrimSpace(h)
	}

	result := &ExcelResult{
		Headers: headers,
		Rows:    make([]ExcelRow, 0),
	}

	for i := 1; i < len(rows); i++ {
		row := make(ExcelRow)
		for j, cell := range rows[i] {
			if j < len(headers) {
				row[headers[j]] = strings.TrimSpace(cell)
			}
		}
		if len(row) > 0 {
			result.Rows = append(result.Rows, row)
		}
	}

	result.Total = len(result.Rows)
	return result, nil
}

func RowToFloat(row ExcelRow, key string) (float64, error) {
	val, ok := row[key]
	if !ok || val == "" {
		return 0, fmt.Errorf("字段 %s 为空", key)
	}
	return strconv.ParseFloat(val, 64)
}
