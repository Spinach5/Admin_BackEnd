package dto

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Message string      `json:"message,omitempty"`
	Total   int         `json:"total,omitempty"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{Success: true, Data: data})
}

func SuccessWithTotal(c *gin.Context, data interface{}, total int) {
	c.JSON(http.StatusOK, Response{Success: true, Data: data, Total: total})
}

func SuccessMessage(c *gin.Context, message string) {
	c.JSON(http.StatusOK, Response{Success: true, Message: message})
}

func Error(c *gin.Context, status int, message string) {
	c.JSON(status, Response{Success: false, Message: message})
}

func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, message)
}

func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, message)
}

func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, message)
}

func InternalError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, message)
}
