package constants

// region ERROR keys
var (
	BadRequestError              = "bad.request"
	DMEDError                    = "dmed.error"
	AcceptedCountError           = "warning.accepted_count.is.null"
	NotFoundError                = "not.found"
	InternalServerError          = "internal.server.error"
	InvalidRequestBodyError      = "invalid.request.body"
	UnauthorizedError            = "user.not.authorized"
	ForbiddinError               = "forbidden"
	InvalidQueryError            = "invalid.query.param"
	NotEnoughProductError        = "not.enough.product"
	ConflictError                = "conflict"
	ResourceNotFoundError        = "resource.not.found"
	RateLimitExceededError       = "rate.limit.exceeded"
	AlreadyExistsError           = "already.exists"
	DependencyFailedError        = "dependency.failed"
	AlreadyCompletedError        = "already.completed"
	SaleIsClosedError            = "sale.is.closed"
	PaymentTypeRequiredError     = "payment.type.required"
	InvalidPaymentTypeError      = "invalid.payment.type"
	InvalidSaleAmount            = "invalid.sale.amount"
	SerialOrMarkingRequiredError = "serial.or.marking.required"
)

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// // Error makes it compatible with the `error` interface.
// func (e *Error) Error() string {
// 	return e.Message
// }

// // NewError creates a new Error instance with an optional message
// func NewError(code int, message ...string) *Error {
// 	err := &Error{
// 		Code:    code,
// 		Message: utils.StatusMessage(code),
// 	}
// 	if len(message) > 0 {
// 		err.Message = message[0]
// 	}
// 	return err
// }
