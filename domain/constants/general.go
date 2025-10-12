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
	TimeOnlyDateFormat                      = "2006-01-02"
	DateTimeFormat                          = "2006-01-02 15:04:05"
	DateTimeFormatRFC3339                   = "2006-01-02T15:04:05"
	DefaultLimit                            = 10
	DefaultOffset                           = 0
	SumsToTiyns                             = 100
	NoValue                                 = ""
	HeaderContentType                       = "Content-Type"
	HeaderAccept                            = "Accept"
	ContentTypeJson                         = "application/json"
	ContentTypeFormUrlEncoded               = "application/x-www-form-urlencoded"
	AuthBasic                               = "Basic"
	AuthBearer                              = "Bearer"
	HeaderXAuth                             = "X-Auth"
	HeaderAuth                              = "Auth"
	HeaderHost                              = "Host"
	DiscountTypePercent                     = "percent"
	// Languages
	LanguageRu    = "ru"
	LanguageUz    = "uz"
	LanguageEn    = "en"
	LanguageKiril = "cr"

	// Validation
	DefaultValidationErrKey     = "default"
	MaxFileSizeValidationErrKey = "max_file_size"
	MimeTypeValidationErrKey    = "mime_type"

	// WebSocket events
	WsEventNoorOrder = "noor_order"
)

// Payment Types slices
var (
	PaymentAppTypes = []string{PaymentTypeClick, PaymentTypePayme, PaymentTypeAlif, PaymentTypeUzum}
	PaymentTypes    = []string{PaymentTypeCash, PaymentTypeCard, PaymentTypeApp}
)

// Roles slices
var (
	AllAdminRoles = []string{RoleAdmin, RoleSuperAdmin, RoleFounder, RoleAccountant, RoleDirector, RoleAutoZakaz, RoleManager}
)

// Sale stages
const (
	SaleStageNew            = 1
	SaleStagePending        = 2
	SaleStageDrafted        = 3
	SaleStageOfdWaiting     = 4
	SaleStageOfdCancelled   = 5
	SaleStageOfdSent        = 6
	SaleStagePayWaiting     = 7
	SaleStagePayFinished    = 8
	SaleStageFinished       = 9
	SaleStageReturning      = 10
	SaleStageReturnedFinish = 11
)

var (
	FinishedSaleStages = []int{SaleStageFinished, SaleStageReturnedFinish}
	PendingSaleStages  = []int{SaleStageNew, SaleStagePending, SaleStageReturning}
)

var SaleStages = map[int]map[string]string{
	SaleStageNew: {
		"en": "New",
		"ru": "Новый",
		"uz": "Yangi",
	},
	SaleStagePending: {
		"en": "Pending",
		"ru": "В ожидании",
		"uz": "Kutilmoqda",
	},
	SaleStageDrafted: {
		"en": "Drafted",
		"ru": "Черновик",
		"uz": "Qoralama",
	},
	SaleStageOfdWaiting: {
		"en": "OFD waiting",
		"ru": "Ожидание ОФД",
		"uz": "OFD kutilmoqda",
	},
	SaleStageOfdCancelled: {
		"en": "OFD cancelled",
		"ru": "Отменено ОФД",
		"uz": "OFD bekor qilindi",
	},
	SaleStageOfdSent: {
		"en": "OFD sent",
		"ru": "Отправлено ОФД",
		"uz": "OFD yuborildi",
	},
	SaleStagePayWaiting: {
		"en": "Payment waiting",
		"ru": "Ожидание оплаты",
		"uz": "To‘lov kutilmoqda",
	},
	SaleStageFinished: {
		"en": "Finished",
		"ru": "Завершено",
		"uz": "Yakunlandi",
	},
	SaleStageReturning: {
		"en": "Returning",
		"ru": "Возврат",
		"uz": "Qaytarilmoqda",
	},
	SaleStageReturnedFinish: {
		"en": "Return finished",
		"ru": "Возврат завершён",
		"uz": "Qaytarish yakunlandi",
	},
}

const (

	// company
	PharmaCosmos = "Pharma Cosmos"

	// product status

	ProductStatusActive    = "active"
	ProductStatusInactive  = "inactive"
	ProductStatusLowStock  = "low_stock"
	ProductStatusZeroStock = "zero_stock"
	ProductStatusExpired   = "expired"
	ProductStatusDeleted   = "deleted"

	// payment types

	PaymentTypeCash   = "cash"
	PaymentTypeCard   = "card"
	PaymentTypeApp    = "app"
	PaymentTypeClick  = "click"
	PaymentTypePayme  = "payme"
	PaymentTypeUzum   = "uzum"
	PaymentTypeAlif   = "alif"
	PaymentTypeHumo   = "humo"
	PaymentTypeUzcard = "uzcard"

	// Universal status types

	GeneralStatusNew        = "new"
	GeneralStatusPending    = "pending"
	GeneralStatusProcessing = "processing"
	GeneralStatusCompleted  = "completed"
	GeneralStatusCanceled   = "canceled"
	GeneralStatusDone       = "done"
	GeneralStatusDeleted    = "deleted"
	GeneralStatusActive     = "active"
	GeneralStatusInactive   = "inactive"
	GeneralStatusConfirmed  = "confirmed"
	GeneralStatusSent       = "sent"
	GeneralStatusChecking   = "checking"
	GeneralStatusDrafted    = "drafted"
	GeneralStatusWriteOff   = "writeoff"
	GeneralStatusSentOnec   = "sent-to-1c"
	GeneralStatusDeclined   = "DECLINED"
	GeneralStatusTaxFree    = "tax_free"

	// Online sale status
	SaleOnlineStageDefault   = 0
	SaleOnlineStageNew       = 1
	SaleOnlineStagePending   = 2
	SaleOnlineStageCanceled  = -1
	SaleOnlineStageCompleted = 2

	SaleTypeReturn  = "RETURN"
	SaleTypeSale    = "SALE"
	SaleTypeOnline  = "online"
	SaleTypeOffline = "offline"

	// Service types
	ServiceTypeNoor = "noor"
	ServiceTypeDmed = "dmed"
)

// Request Path
const (
	// Onec service paths
	OnecPathPrihod  = "/prihod"
	OnecPathRasxod  = "/rasxod"
	OnecPathVozvrat = "/vozvrat"
	OnecPathZakaz   = "/zakaz"

	// DMED service paths
	DmedPathPrescription        = "/prescriptions"
	DmedPathPatient             = "/patients"
	DmedPathAppointment         = "/appointments"
	DmedRequestActionIssue      = "issue"
	DmedRequestActionCheckIssue = "check-issue"
)

// region Roles
const (
	RoleAdmin         = "ADMIN"
	RoleSuperAdmin    = "SUPERADMIN"
	RoleManager       = "MANAGER"
	RoleAutoZakaz     = "AUTOZAKAZ"
	RoleFounder       = "FOUNDER"
	RoleAccountant    = "ACCOUNTANT"
	RoleDirector      = "DIRECTOR"
	RoleCashier       = "CASHIER"
	RoleHeadOfCashier = "HEADOFCASHIER"
)

// end region

// region Payme
const (
	ActionCheckPerformTransaction = "CheckPerformTransaction"
	ActionCreateTransaction       = "CreateTransaction"
	ActionCheckTransaction        = "CheckTransaction"
	ActionCancelTransaction       = "CancelTransaction"
	ActionPerformTransaction      = "PerformTransaction"
	ActionCreateCard              = "cards.create"
	ActionGetVerifyCode           = "cards.get_verify_code"
	ActionVerify                  = "cards.verify"
	ActionCheck                   = "cards.check"
	ActionCardsRemove             = "cards.remove"
	ActionCreateReceipt           = "receipts.create"
	ActionPayReceipt              = "receipts.pay"
	ActionSetFiscalData           = "receipts.set_fiscal_data"
	ActionCancelReceipt           = "receipts.cancel"
	ActionConfirmHold             = "receipts.confirm_hold"
)

const (
	PaymeMethodNotPostErrorCode           = -32300
	PaymeJSONParseErrorCode               = -32700
	PaymeMissingRequiredFieldsErrorCode   = -32600
	PaymeMethodNotFoundErrorCode          = -32601
	PaymeInsufficientFundsErrorCode       = -31630
	PaymeCardNotFoundError                = -31400
	PaymeInsufficientPrivilegesErrorCode  = -32504
	PaymeSystemErrorCode                  = -32400
	PaymeInvalidAmountErrorCode           = -31001
	PaymeTransactionNotFoundErrorCode     = -31003
	PaymeCannotCancelTransactionErrorCode = -31007
	PaymeCannotPerformOperationErrorCode  = -31008
	PaymeServerNotOperationalErrorCode    = -31100
	PaymeTemporarilyUnavailableErrorCode  = -31625
	PaymeUserInputErrorsCode              = -31050
	PaymeIncorrectOTPError                = -31103
	PaymeOTPExpiredErrorCode              = -31101
	PaymeInvalidExpiryDateErrorCode       = -31300
)

const (
	PaymeTransactionStateCreated              = 1
	PaymeTransactionStateFinished             = 2
	PaymeTransactionStateCancelled            = -1
	PaymeTransactionStateCancelledAfterFinish = -2
)

const (
	CheckPerformTransaction = "CheckPerformTransaction"
	CreateTransaction       = "CreateTransaction"
	PerformTransaction      = "PerformTransaction"
	CancelTransaction       = "CancelTransaction"
)

const (
	PaymeRecipientNotFoundOrInactive = 1  // One or more recipients have not been found or are inactive in Pay me Business.
	PaymeDebitOperationError         = 2  // An error occurred when performing a debit transaction at the processing center.
	PaymeTransactionError            = 3  // Transaction execution error.
	PaymeTransactionTimeout          = 4  // The transaction was canceled due to a timeout.
	PaymeRefund                      = 5  // Refund of money.
	PaymeUnknownError                = 10 // Unknown error.
)

// end region

// region Click

// methods
const (
	ActionClickPassCreate       = "click_pass"
	ActionClickPassCheck        = "click_pass_check"
	ActionClickPassCancel       = "click_pass_cancel"
	ActionClickPassConfirm      = "click_pass_confirm"
	ActionClickPassConfirmation = "click_pass_confirmation"
)

const (
	ClickPassCreatePath = "/click_pass/payment"
	ClickPassCheckPath  = "/payment/status/"
	ClickPassCancelPath = "/payment/reversal/"
	
)

const (
	ClickNotPaidErrorCode      = -1 // < 0 // Error (details in error_note)
	ClickPaymentCreateCode     = 0  // Payment created
	ClickPaymentProcessingCode = 1  // Processing
	ClickPaymentSucceedCode    = 2  // Payment successful
)
