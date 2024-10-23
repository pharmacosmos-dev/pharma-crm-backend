package http

import (
	"github.com/gin-gonic/gin"
)

type response struct {
	Error      string `json:"error" example:"message"`
	StatusCode int    `json:"status_code" example:"200"`
	Message    string `json:"message" example:"message"`
}

func ErrorResponse(c *gin.Context, code int, msg string, err error) {
	c.AbortWithStatusJSON(code, response{Error: err.Error(), Message: msg, StatusCode: code})
}
