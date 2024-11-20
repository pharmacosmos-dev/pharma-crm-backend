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
	Ok      bool        `json:"ok"`
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
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

const (
	// Success messages
	MsgSuccessCreate = "Successfully created"
	MsgSuccessUpdate = "Successfully updated"
	MsgSuccessDelete = "Successfully deleted"
	MsgSuccessFetch  = "Data fetched successfully"
	// Error messages
	MsgErrInvalidRequest = "Invalid request data"
	MsgErrCreateFailed   = "Failed to create"
	MsgErrUpdateFailed   = "Failed to update"
	MsgErrDeleteFailed   = "Failed to delete"
	MsgErrFetchFailed    = "Failed to fetch data"
	MsgErrInternal       = "Internal server error"
	MsgErrNotFount       = "Information not found"
)

func getOffsetParam(c *gin.Context) (offset int, err error) {
	offsetStr := c.DefaultQuery("offset", "0")
	return strconv.Atoi(offsetStr)
}

func getLimitParam(c *gin.Context) (limit int, err error) {
	limitStr := c.DefaultQuery("limit", "60")
	return strconv.Atoi(limitStr)
}
