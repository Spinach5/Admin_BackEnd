package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"web-backend/internal/config"
	"web-backend/internal/dto"

	"github.com/gin-gonic/gin"
)

type captchaSolveReq struct {
	ShadeImage  string `json:"shadeImage"`
	CutoutImage string `json:"cutoutImage"`
}

// CaptchaSolve 滑块缺口距离计算，转发到 Python captcha 微服务
func CaptchaSolve(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req captchaSolveReq
		if err := c.ShouldBindJSON(&req); err != nil || req.ShadeImage == "" || req.CutoutImage == "" {
			dto.BadRequest(c, "缺少 shadeImage 或 cutoutImage")
			return
		}

		body, _ := json.Marshal(req)
		serviceURL := cfg.CaptchaServiceURL
		if !strings.HasPrefix(serviceURL, "http://") && !strings.HasPrefix(serviceURL, "https://") {
			serviceURL = "http://" + serviceURL
		}
		url := serviceURL + "/solve"
		resp, err := http.Post(url, "application/json", bytes.NewReader(body))
		if err != nil {
			log.Printf("[CaptchaSolve] 调用求解服务失败: %v", err)
			dto.InternalError(c, "验证码服务不可用")
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
			dto.InternalError(c, "滑块计算失败")
			return
		}

		dto.Success(c, gin.H{"x": result.X})
	}
}
