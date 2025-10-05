package domain

import (
	"net/http"
)

// region ERROR keys
var (
	// 400 – Bad Request (mijoz noto‘g‘ri so‘rov yuborgan)
	BadRequestError              = NewError(http.StatusBadRequest, "bad.request")
	InvalidRequestBodyError      = NewError(http.StatusBadRequest, "invalid.request.body")
	InvalidQueryError            = NewError(http.StatusBadRequest, "invalid.query.param")
	PaymentTypeRequiredError     = NewError(http.StatusBadRequest, "payment.type.required")
	InvalidPaymentTypeError      = NewError(http.StatusBadRequest, "invalid.payment.type")
	InvalidSaleAmount            = NewError(http.StatusBadRequest, "invalid.sale.amount")
	SerialOrMarkingRequiredError = NewError(http.StatusBadRequest, "serial.or.marking.required")
	AcceptedCountError           = NewError(http.StatusBadRequest, "accepted.count.is.null")

	// 401 – Unauthorized (token noto‘g‘ri yoki mavjud emas)
	UnauthorizedError = NewError(http.StatusUnauthorized, "user.not.authorized")

	// 403 – Forbidden (ruxsat yo‘q)
	ForbiddinError = NewError(http.StatusForbidden, "forbidden")

	// 404 – Not Found (resurs topilmadi)
	NotFoundError         = NewError(http.StatusNotFound, "not.found")
	ResourceNotFoundError = NewError(http.StatusNotFound, "resource.not.found")

	// 409 – Conflict (mavjud ma'lumot bilan to‘qnashuv)
	ConflictError         = NewError(http.StatusConflict, "conflict")
	NotEnoughProductError = NewError(http.StatusConflict, "not.enough.product")
	AlreadyExistsError    = NewError(http.StatusConflict, "already.exists")
	AlreadyCompletedError = NewError(http.StatusConflict, "already.completed")
	SaleIsClosedError     = NewError(http.StatusConflict, "sale.is.closed")

	// 424 – Failed Dependency (tashqi tizimga bog‘liq xatolik)
	DependencyFailedError = NewError(http.StatusFailedDependency, "dependency.failed")

	// 429 – Too Many Requests (rate limit)
	RateLimitExceededError = NewError(http.StatusTooManyRequests, "rate.limit.exceeded")

	// 500 – Internal Server Error (backenddagi xatoliklar)
	InternalServerError = NewError(http.StatusInternalServerError, "internal.server.error")
	DMEDError           = NewError(http.StatusInternalServerError, "dmed.error")
)

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// // Error makes it compatible with the `error` interface.
func (e *Error) Error() string {
	return e.Message
}

// NewError creates a new Error instance with an optional message
func NewError(code int, message string) *Error {
	err := &Error{
		Code:    code,
		Message: message,
	}

	return err
}
