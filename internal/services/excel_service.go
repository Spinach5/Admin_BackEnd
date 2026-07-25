package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

type ExcelRow map[string]string

type ExcelResult struct {
	Headers []string   `json:"headers"`
	Rows    []ExcelRow `json:"rows"`
	Total   int        `json:"total"`
}

const parseTimeout = 30 * time.Second

func ParseExcel(file io.Reader) (*ExcelResult, error) {
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, file); err != nil {
		return nil, fmt.Errorf("读取文件失败: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), parseTimeout)
	defer cancel()

	done := make(chan error, 1)
	var result *ExcelResult

	go func() {
		f, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
		if err != nil {
			done <- fmt.Errorf("无法解析 Excel 文件: %v", err)
			return
		}
		defer f.Close()

		sheets := f.GetSheetList()
		if len(sheets) == 0 {
			done <- fmt.Errorf("Excel 文件中没有工作表")
			return
		}

		rows, err := f.GetRows(sheets[0])
		if err != nil {
			done <- fmt.Errorf("读取工作表失败: %v", err)
			return
		}

		if len(rows) < 2 {
			done <- fmt.Errorf("Excel 文件至少需要包含表头和一行数据")
			return
		}

		headers := make([]string, len(rows[0]))
		for i, h := range rows[0] {
			headers[i] = strings.TrimSpace(h)
		}

		result = &ExcelResult{
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
		done <- nil
	}()

	select {
	case err := <-done:
		if err != nil {
			return nil, err
		}
		return result, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("Excel 解析超时（超过 %ds），请检查文件是否过大或格式是否正确", int(parseTimeout.Seconds()))
	}
}

func RowToFloat(row ExcelRow, key string) (float64, error) {
	val, ok := row[key]
	if !ok || val == "" {
		return 0, fmt.Errorf("字段 %s 为空", key)
	}
	return strconv.ParseFloat(val, 64)
}
