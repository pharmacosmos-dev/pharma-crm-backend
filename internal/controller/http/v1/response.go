package v1

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// RequestBody defines the structure for the incoming requests
type RequestBody[T any] struct {
	Data T `json:"data"`
}

type Response struct {
	Ok      bool        `json:"ok" example:"true"`
	Code    int         `json:"code" example:"200"`
	Message string      `json:"message" example:"message"`
	Data    interface{} `json:"data" example:"data"`
}

// handleResponse to send consistent JSON responses
func handleResponse(c *gin.Context, statusCode int, message string, data interface{}) {
	c.JSON(statusCode, Response{
		Ok:      statusCode >= http.StatusOK && statusCode < http.StatusBadRequest, // true for 2xx status codes
		Code:    statusCode,
		Message: message,
		Data:    data,
	})
}

// Success messages
const (
	MsgSuccessCreate = "Successfully created"
	MsgSuccessUpdate = "Successfully updated"
	MsgSuccessDelete = "Successfully deleted"
	MsgSuccessFetch  = "Data fetched successfully"
)

// Error messages
const (
	MsgErrInvalidRequest = "Invalid request data"
	MsgErrCreateFailed   = "Failed to create"
	MsgErrUpdateFailed   = "Failed to update"
	MsgErrDeleteFailed   = "Failed to delete"
	MsgErrFetchFailed    = "Failed to fetch data"
	MsgErrInternal       = "Internal server error"
)

func getOffsetParam(c *gin.Context) (offset int, err error) {
	offsetStr := c.DefaultQuery("offset", "0")
	return strconv.Atoi(offsetStr)
}

func getLimitParam(c *gin.Context) (limit int, err error) {
	limitStr := c.DefaultQuery("limit", "60")
	return strconv.Atoi(limitStr)
}
