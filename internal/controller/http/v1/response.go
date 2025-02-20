package v1

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func getPaginationParams(c *gin.Context) (limit, offset int, err error) {
	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")

	// Parse the limit parameter
	limit, err = strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		return 10, 0, nil // Default to 10 if invalid
	}
	// Parse the offset parameter
	offset, err = strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		return limit, 0, nil // Default to 0 if invalid
	}

	return limit, offset, nil
}

// Status Response struct for http response
type Status struct {
	Code        int    `json:"code"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

var (
	OK = Status{
		Code:        200,
		Status:      "OK",
		Description: "The request has succeeded.",
	}
	CREATED = Status{
		Code:        201,
		Status:      "CREATED",
		Description: "The request has succeeded, and a new resource has been created as a result.",
	}
	NoContent = Status{
		Code:        204,
		Status:      "NO CONTENT",
		Description: "The server successfully processed the request, but there is no content to send in the response.",
	}
	ResetContent = Status{
		Code:        205,
		Status:      "RESET CONTENT",
		Description: "The server successfully processed the request, and the client should reset the view or clear the form used for the request.",
	}
	PartialContent = Status{
		Code:        206,
		Status:      "PARTIAL CONTENT",
		Description: "The server is delivering only part of the resource due to a range header sent by the client.",
	}
	BadRequest = Status{
		Code:        400,
		Status:      "ERROR",
		Description: "The server could not understand the request due to invalid syntax.",
	}
	UNAUTHORIZED = Status{
		Code:        401,
		Status:      "ERROR",
		Description: "The client must authenticate itself to get the requested response.",
	}
	FORBIDDEN = Status{
		Code:        403,
		Status:      "ERROR",
		Description: "The client does not have access rights to the content; authentication will not help.",
	}
	NotFound = Status{
		Code:        404,
		Status:      "ERROR",
		Description: "The server could not find the requested resource.",
	}
	NotAcceptable = Status{
		Code:        406,
		Status:      "ERROR",
		Description: "The server cannot produce a response matching the list of acceptable values defined in the request's headers.",
	}
	CONFLICT = Status{
		Code:        409,
		Status:      "ERROR",
		Description: "The request conflicts with the current state of the server, such as edit conflicts in a database.",
	}
	UnprocessableEntity = Status{
		Code:        422,
		Status:      "UNPROCESSABLE_ENTITY",
		Description: "The server understands the request but cannot process it due to semantic errors.",
	}
	TooManyRequests = Status{
		Code:        429,
		Status:      "TOO_MANY_REQUESTS",
		Description: "The user has sent too many requests in a given amount of time",
	}
	InternalError = Status{
		Code:        500,
		Status:      "ERROR",
		Description: "The server encountered an unexpected condition that prevented it from fulfilling the request.",
	}
	BadGateway = Status{
		Code:        502,
		Status:      "ERROR",
		Description: "The server, acting as a gateway or proxy, received an invalid response from the upstream server.",
	}
)
