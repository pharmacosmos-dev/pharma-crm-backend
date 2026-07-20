package domain

import (
	"fmt"
	"net/http"
)

// region ERROR keys
var (
	MultiStatus = NewError(http.StatusMultiStatus, "duplicate")

	// 400 – Bad Request (mijoz noto‘g‘ri so‘rov yuborgan)
	BadRequestError              = NewError(http.StatusBadRequest, "bad.request")
	InvalidRequestBodyError      = NewError(http.StatusBadRequest, "invalid.request.body")
	InvalidTimeFormatError       = NewError(http.StatusBadRequest, "invalid.time.format")
	InvalidQueryError            = NewError(http.StatusBadRequest, "invalid.query.param")
	InvalidPhoneError            = NewError(http.StatusBadRequest, "invalid.phone.format")
	PaymentTypeRequiredError     = NewError(http.StatusBadRequest, "payment.type.required")
	InvalidPaymentTypeError      = NewError(http.StatusBadRequest, "invalid.payment.type")
	InvalidSaleAmount            = NewError(http.StatusBadRequest, "invalid.sale.amount")
	SerialOrMarkingRequiredError = NewError(http.StatusBadRequest, "serial.or.marking.required")
	AcceptedCountError           = NewError(http.StatusBadRequest, "accepted.count.is.null")
	AcceptedCountMismatchError   = NewError(http.StatusBadRequest, "accepted.count.not.equal.scanned.count")
	RejectedCountMismatchError   = NewError(http.StatusBadRequest, "rejected.count.not.equal.rejected.items.count")
	ScannedCountZeroError        = NewError(http.StatusBadRequest, "scanned.count.is.zero")
	DuplicateError               = NewError(http.StatusBadRequest, "duplicate")
	IncorrectOTPError            = NewError(http.StatusBadRequest, "incorrect.otp")
	IncorrectCardExpiryDateError = NewError(http.StatusBadRequest, "incorrect.expiry.date")
	IncorrectCardError           = NewError(http.StatusBadRequest, "incorrect.card")
	AlifInvalidRequestBodyError  = NewError(http.StatusBadRequest, "alif.invalid.request.body")
	AlifInvalidParametersError   = NewError(http.StatusBadRequest, "alif.invalid.parameters")
	AlifPaymentRevertedError     = NewError(http.StatusBadRequest, "REVERTED")
	AlifPaymentDeclinedError     = NewError(http.StatusBadRequest, "DECLINED")
	UserIdIsRequiredError        = NewError(http.StatusBadRequest, "user.id.is.required")
	FiscalSignRequiredError      = NewError(http.StatusBadRequest, "fiscal.sign.required")
	StoreIdsRequiredError        = NewError(http.StatusBadRequest, "store_ids.required")
	ReminderDateRangeError       = NewError(http.StatusBadRequest, "reminder.to_date.must.be.after.from_date")
	ReminderExpiredDateError     = NewError(http.StatusBadRequest, "reminder.to_date.must.be.in.future")

	// 401 – Unauthorized (token noto‘g‘ri yoki mavjud emas)
	UnauthorizedError     = NewError(http.StatusUnauthorized, "user.not.authorized")
	AlifUnauthorizedError = NewError(http.StatusUnauthorized, "alif.unauthorized")

	// 402 - PaymentRequired (mablag' yetarli emas)
	InsufficientFunds = NewError(http.StatusPaymentRequired, "insufficient.funds")

	// 403 – Forbidden (ruxsat yo‘q)
	ForbiddinError = NewError(http.StatusForbidden, "forbidden")

	// 404 – Not Found (resurs topilmadi)
	NotFoundError         = NewError(http.StatusNotFound, "not.found")
	ResourceNotFoundError = NewError(http.StatusNotFound, "resource.not.found")
	CardNotFoundError     = NewError(http.StatusNotFound, "card.not.found")
	PrescriptionNotFound  = NewError(http.StatusNotFound, "prescription.not.found")

	// 408 - Timeout (vaqt yetmadi)
	OTPExpiredError          = NewError(http.StatusRequestTimeout, "otp.expired")
	PrescriptionExpiredError = NewError(http.StatusRequestTimeout, "prescription.expired")

	// 409 – Conflict (mavjud ma'lumot bilan to‘qnashuv)
	CanNotCreateYourselfError            = NewError(http.StatusConflict, "you.can.not.create.yourself")
	ConflictError                        = NewError(http.StatusConflict, "conflict")
	NotEnoughProductError                = NewError(http.StatusConflict, "not.enough.product")
	AlreadyExistsError                   = NewError(http.StatusConflict, "already.exists")
	AlreadyCompletedError                = NewError(http.StatusConflict, "already.completed")
	AlreadySentError                     = NewError(http.StatusConflict, "already.sent")
	SaleIsClosedError                    = NewError(http.StatusConflict, "sale.is.closed")
	PrescriptionsAlreadyIssued           = NewError(http.StatusConflict, "prescriptions.already.issued")
	AlreadyUpdatedError                  = NewError(http.StatusConflict, "already.updated")
	NoOpenCashboxError                   = NewError(http.StatusConflict, "no.open.cashbox")
	AlreadyAcceptedError                 = NewError(http.StatusConflict, "already.accepted")
	DuplicatePhoneError                  = NewError(http.StatusConflict, "duplicate.phone.error")
	DuplicateLoyaltyCardError            = NewError(http.StatusConflict, "duplicate.loyalty.card.error")
	AlreadyHaveOpenCashboxOperationError = NewError(http.StatusConflict, "already.have.open.cashbox.operation")
	UnfinishedTransferOrReturnError      = NewError(http.StatusConflict, "unfinished.transfer.or.return")
	ActiveInventoryError                 = NewError(http.StatusConflict, "active.inventory.exists")

	// 202 – Accepted (so’rov qabul qilindi, tolov noaniq)
	PaymePendingError = NewError(http.StatusAccepted, "payme.payment.pending")
	
	// 424 – Failed Dependency (tashqi tizimga bog’liq xatolik)
	DependencyFailedError    = NewError(http.StatusFailedDependency, "dependency.failed")
	PaymeNotOperationalError = NewError(http.StatusFailedDependency, "payme.not.operational")
	ClickNotOperationalError = NewError(http.StatusFailedDependency, "click.not.operational")

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

type NotAdditionError struct {
	Code int
	Data any
}

func (e *NotAdditionError) Error() string {
	return "not.addition.error"
}

func NewNotAdditionError(code int, data any) *NotAdditionError {
	return &NotAdditionError{
		Code: code,
		Data: data,
	}
}

type DmedError struct {
	StatusCode int
	Body       []byte
}

func (e *DmedError) Error() string {
	return fmt.Sprintf(
		"dmed request failed: status=%d body=%s",
		e.StatusCode,
		string(e.Body),
	)
}
