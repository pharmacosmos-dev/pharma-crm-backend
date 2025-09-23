package constants

import "time"

// region general
const (
	// Access token expire time 24 hour
	AccessTokenExpiresInTime time.Duration = 2 * 60 * 24 * time.Minute
	// Refresh token expire time: 30 days
	RefreshTokenExpiresInTime time.Duration = 30 * 24 * time.Hour

	// Context timeouts for reports and other long-running operations
	ContextTimeoutForReports  time.Duration = 1 * time.Minute
	DefaultContextTimeout     time.Duration = 30 * time.Second
	DATE_FORMAT                             = "2006-01-02"
	DATE_TIME                               = "2006-01-02 15:04:05"
	DATE_1C_FORMAT                          = "2006-01-02T15:04:05"
	DefaultLimit                            = 10
	DefaultOffset                           = 0
	ContentTypeJson                         = "application/json"
	ContentTypeFormUrlEncoded               = "application/x-www-form-urlencoded"
	AuthBasic                               = "Basic"
	AuthBearer                              = "Bearer"
	HeaderXAuth                             = "X-Auth"
	HeaderHost                              = "Host"

	// WebSocket events
	WsEventNoorOrder = "noor_order"
)

const (
	// region status
	NEW_IMPORT       = "new"
	PENDING_IMPORT   = "pending"
	COMPLETED_IMPORT = "completed"
	CANCELED_IMPORT  = "canceled"
	WRITEOFF_IMPORT  = "writeoff"

	// company
	PHARMA_COSMOS = "Pharma Cosmos"

	// cart item status
	PENDING_CART_ITEM = "pending"
	ACTIVE_CART_ITEM  = "active"
	DELETED_CART_ITEM = "deleted"
	DRAFTED_CART_ITEM = "drafted"
	SOLD_CART_ITEM    = "sold"

	// product status

	ACTIVE_PRODUCT     = "active"
	INACTIVE_PRODUCT   = "inactive"
	LOW_STOCK_PRODUCT  = "low_stock"
	ZERO_STOCK_PRODUCT = "zero_stock"
	EXPIRED_PRODUCT    = "expired"
	DELETED_PRODUCT    = "deleted"

	// payment types

	CASH    = "cash"
	CARD    = "card"
	CLICK   = "click"
	PAYME   = "payme"
	UZUM    = "uzum"
	ALIF    = "alif"
	PERCENT = "percent"

	// universal status types

	NEW        = "new"
	PENDING    = "pending"
	PROCESSING = "processing"
	COMPLETED  = "completed"
	CANCELED   = "canceled"
	DONE       = "done"
	DELETED    = "deleted"
	ACTIVE     = "active"
	INACTIVE   = "inactive"
	CONFIRMED  = "confirmed"
	SENT       = "sent"
	CHECKING   = "checking"

	// online sale status

	ONLINE_STATUS_DEFAULT   = 0
	ONLINE_STATUS_NEW       = 1
	ONLINE_STATUS_PENDING   = 2
	ONLINE_STATUS_CANCELED  = -1
	ONLINE_STATUS_COMPLETED = 2

	// order type
	SALE_TYPE_ONLINE  = "online"
	SALE_TYPE_OFFLINE = "offline"

	// Service types
	NOOR = "noor"

	SALE_TYPE_RETURN = "RETURN"
	SALE_TYPE_SALE   = "SALE"
)

// Onec Request Path
const (
	OnecPathPrihod  = "/prihod"
	OnecPathRasxod  = "/rasxod"
	OnecPathVozvrat = "/vozvrat"
	OnecPathZakaz   = "/zakaz"
)

// region languages

const (
	LanguageRu    = "ru"
	LanguageUz    = "uz"
	LanguageEn    = "en"
	LanguageKiril = "kiril"
)

const (
	DefaultValidationErrKey     = "default"
	MaxFileSizeValidationErrKey = "max_file_size"
	MimeTypeValidationErrKey    = "mime_type"
)

// region role types
const (
	ADMIN      = "ADMIN"
	SUPERADMIN = "SUPERADMIN"
	MANAGER    = "MANAGER"
	AUTOZAKAZ  = "AUTOZAKAZ"
	FOUNDER    = "FOUNDER"
	ACCOUNTANT = "ACCOUNTANT"
	DIRECTOR   = "DIRECTOR"
)

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// region error types
var (
	BadRequestError          = "bad.request"
	DMEDError                = "dmed.error"
	AcceptedCountError       = "warning.accepted_count.is.null"
	NotFoundError            = "not.found"
	InternalServerError      = "internal.server.error"
	InvalidRequestBodyError  = "invalid.request.body"
	UnauthorizedError        = "user.not.authorized"
	ForbiddinError           = "forbidden"
	InvalidQueryError        = "invalid.query.param"
	NotEnoughProductError    = "not.enough.product"
	ConflictError            = "conflict"
	ResourceNotFoundError    = "resource.not.found"
	RateLimitExceededError   = "rate.limit.exceeded"
	AlreadyExistsError       = "already.exists"
	DependencyFailedError    = "dependency.failed"
	AlreadyCompletedError    = "already.completed"
	PaymentTypeRequiredError = "payment.type.required"
)

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
