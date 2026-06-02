package middleware

import (
	"bytes"
	"encoding/json"
	"io"

	"web-backend/internal/database"
	"web-backend/internal/dto"
	"web-backend/internal/models"

	"github.com/gin-gonic/gin"
)

func V1Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := c.GetRawData()
		if err != nil {
			dto.BadRequest(c, "无法读取请求数据")
			c.Abort()
			return
		}

		var req dto.V1BaseRequest
		if err := json.Unmarshal(body, &req); err != nil || req.ID == 0 || req.StuID == "" || req.SchoolID == "" {
			dto.BadRequest(c, "缺少身份参数 (id, stuId, schoolId)")
			c.Abort()
			return
		}

		user, err := models.GetUserByID(database.DB, req.ID)
		if err != nil || user.IsDeleted == 1 {
			dto.Unauthorized(c, "用户不存在或已注销")
			c.Abort()
			return
		}

		if user.StuID != req.StuID || user.SchoolID != req.SchoolID {
			dto.Unauthorized(c, "身份信息不匹配")
			c.Abort()
			return
		}

		c.Set("v1_user_id", req.ID)
		c.Set("v1_stu_id", req.StuID)
		c.Set("v1_school_id", req.SchoolID)

		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
		c.Next()
	}
}
