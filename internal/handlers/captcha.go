package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"web-backend/internal/dto"

	"github.com/gin-gonic/gin"
)

type captchaSolveReq struct {
	ShadeImage  string `json:"shadeImage"`
	CutoutImage string `json:"cutoutImage"`
}

// CaptchaSolve 内部端点，转发到本地 Python slider-solver 微服务
func CaptchaSolve() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req captchaSolveReq
		if err := c.ShouldBindJSON(&req); err != nil || req.ShadeImage == "" || req.CutoutImage == "" {
			dto.BadRequest(c, "缺少 shadeImage 或 cutoutImage")
			return
		}

		body, _ := json.Marshal(req)
		resp, err := http.Post("http://127.0.0.1:5001/solve", "application/json", bytes.NewReader(body))
		if err != nil {
			log.Printf("[CaptchaSolve] 调用求解服务失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"x": 0})
			return
		}
		defer resp.Body.Close()

		data, _ := io.ReadAll(resp.Body)
		var result struct {
			X     int    `json:"x"`
			Error string `json:"error"`
		}
		if json.Unmarshal(data, &result) != nil || result.Error != "" {
			log.Printf("[CaptchaSolve] 求解失败: %s", string(data))
			c.JSON(http.StatusInternalServerError, gin.H{"x": 0})
			return
		}

		c.JSON(http.StatusOK, gin.H{"x": result.X})
	}
}
