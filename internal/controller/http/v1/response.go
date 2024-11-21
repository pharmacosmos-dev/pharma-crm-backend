package v1

import (
	"fmt"
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

func getPaginationParams(c *gin.Context) (limit, offset int, err error) {
	// Default values for limit and offset
	const defaultLimit = 20
	const defaultOffset = 0

	// Parse the limit parameter
	limitStr := c.DefaultQuery("limit", strconv.Itoa(defaultLimit))
	limit, err = strconv.Atoi(limitStr)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid limit parameter: %w", err)
	}
	// Parse the offset parameter
	offsetStr := c.DefaultQuery("offset", strconv.Itoa(defaultOffset))
	offset, err = strconv.Atoi(offsetStr)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid offset parameter: %w", err)
	}

	return limit, offset, nil
}
