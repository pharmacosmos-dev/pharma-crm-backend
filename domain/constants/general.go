package constants

import "time"

// region General
const (
	// Access token expire time 24 hour
	AccessTokenExpiresInTime time.Duration = 2 * 60 * 24 * time.Minute
	// Refresh token expire time: 30 days
	RefreshTokenExpiresInTime time.Duration = 30 * 24 * time.Hour

	// Context timeouts for reports and other long-running operations
	ContextTimeoutForReports  time.Duration = 1 * time.Minute
	DefaultContextTimeout     time.Duration = 30 * time.Second
	DateTimeTashkent          time.Duration = 5 * time.Hour
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
	HeaderStoreToken                        = "Store-Token"
	HeaderHost                              = "Host"
	DiscountTypePercent                     = "percent"
	DefaultSheetName                        = "List1"
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
	WsEventNoorOrder              = "noor_order"
	WsEventNoorOrderCancel        = "noor_order_cancel"
	WsEventNoorOrderAcceptCourier = "noor_order_accept_courier"
	WsEventTransferSent           = "transfer_sent"
	WsEventTransferChecking       = "transfer_checking"
	WsEventImportCreated          = "import_created"
	WsEventReminderCreated        = "reminder_created"

)

// Payment Types slices
var (
	PaymentAppTypes = []string{PaymentTypeClick, PaymentTypePayme, PaymentTypeAlif, PaymentTypeUzum, PaymentTypeUzumTezkor}
	PaymentTypes    = []string{PaymentTypeCash, PaymentTypeCard, PaymentTypeApp}
)

// Roles slices
var (
	AllAdminRoles      = []string{RoleAdmin, RoleSuperAdmin, RoleFounder, RoleAccountant, RoleDirector, RoleAutoZakaz, RoleManager, RoleRopApteka}
	StoreTargetViewAll = []string{RoleAdmin, RoleSuperAdmin, RoleFounder, RoleDirector, RoleManager}
)

// region Stages
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
	FinishedSaleStages  = []int{SaleStageFinished, SaleStageReturnedFinish}
	PendingSaleStages   = []int{SaleStageNew, SaleStagePending, SaleStageReturning}
	OnlinePendingStages = []int{SaleOnlineStageNew, SaleOnlineStagePending, SaleOnlineStageWaiting}
	SaleOnlineStages    = []int{SaleOnlineStageNew, SaleOnlineStagePending, SaleOnlineStageCanceled, SaleOnlineStageCompleted, SaleOnlineStageWaiting}
	NotFishishedStages  = []int{SaleStageNew, SaleStagePending, SaleStageDrafted, SaleStageOfdWaiting, SaleStageOfdCancelled, SaleStageOfdSent, SaleStagePayWaiting, SaleStageReturning}
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
	SaleStagePayFinished: {
		"en": "Payment finished",
		"ru": "Оплата завершена",
		"uz": "To‘lov yakunlandi",
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

// end region

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

	PaymentTypeCash        = "cash"
	PaymentTypeCard        = "card"
	PaymentTypeApp         = "app"
	PaymentTypeLoyaltyCard = "loyalty_card"
	PaymentTypeClick       = "click"
	PaymentTypePayme       = "payme"
	PaymentTypeUzum        = "uzum"
	PaymentTypeUzumTezkor  = "uzumtezkor"
	PaymentTypeOnlineOrder = "online_order"
	PaymentTypeAlif        = "alif"
	PaymentTypeHumo        = "humo"
	PaymentTypeUzcard      = "uzcard"
	PaymentTypeCARD        = "CARD"
	PaymentTypeCASH        = "CASH"

	// loyalty card transaction types

	LoyaltyCardTransactionIn  = "in"
	LoyaltyCardTransactionOut = "out"

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
	GeneralStatusSentOnec         = "sent-to-1c"
	GeneralStatusFailedSentOnec   = "failed_sent_to_1c"
	GeneralStatusRejection        = "rejection"
	GeneralStatusDeclined         = "DECLINED"
	GeneralStatusTaxFree    = "tax_free"

	// Online sale status
	SaleOnlineStageDefault   = 0
	SaleOnlineStageNew       = 1
	SaleOnlineStagePending   = 2
	SaleOnlineStageCanceled  = -1
	SaleOnlineStageCompleted = 3
	SaleOnlineStageWaiting   = 4

	SaleTypeReturn  = "RETURN"
	SaleTypeSale    = "SALE"
	SaleTypeOnline  = "online"
	SaleTypeOffline = "offline"

	// Service types
	ServiceTypeNoor = "noor"
	ServiceTypeDmed = "dmed"
	ServiceTypeUzum = "uzum"

	ProductMovementImport    = 1
	ProductMovementInventory = 2
)

// Request Path
const (
	// Onec service paths
	OnecPathPrihod   = "/prihod"
	OnecPathRasxod   = "/rasxod"
	OnecPathVozvrat  = "/vozvrat"
	OnecPathZakaz    = "/zakaz"
	OnecPathPerekit  = "/perekit"
	OnecPathInventar = "/inventar"

	// DMED service paths
	DmedPathPrescription        = "/prescriptions"
	DmedPathPatient             = "/patients"
	DmedPathAppointment         = "/appointments"
	DmedRequestActionIssue      = "issue"
	DmedRequestActionCheckIssue = "check-issue"
)

// region Roles
const (
	RoleAdmin          = "ADMIN"
	RoleSuperAdmin     = "SUPERADMIN"
	RoleManager        = "MANAGER"
	RoleAutoZakaz      = "AUTOZAKAZ"
	RoleFounder        = "FOUNDER"
	RoleAccountant     = "ACCOUNTANT"
	RoleDirector       = "DIRECTOR"
	RoleCashier        = "CASHIER"
	RoleHeadOfCashier  = "HEADOFCASHIER"
	RoleFranchise      = "FRANCHISE"
	RoleFranchiseAdmin = "FRANCHISE_ADMIN"
	RoleRopApteka      = "ROP_APTEKA"
)

// movement type
const (
	MovementTypeImport         = "IMPORT"
	MovementTypeSale           = "SALE"
	MovementTypeReturnSale     = "RETURN_SALE"
	MovementTypeReturnSupplier = "RETURN_SUPPLIER"
	MovementTypeTransferOut    = "TRANSFER_OUT"
	MovementTypeTransferIn     = "TRANSFER_IN"
	MovementTypeInventory      = "INVENTORY"
	MovementTypeRepricing      = "REPRICING"
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
	ActionCheckReceipt            = "receipts.check"
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
	PaymeReceiptStatePaid      = 4  // Чек оплачен
	PaymeReceiptStateCancelled = 50 // Чек отменен
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

// end region

// region Alif

const (
	AlifInvalidRequestBodyErrorCode     = 1001 // Неверное тело запроса — невалидный JSON или тип атрибута
	AlifUnauthorizedRequestErrorCode    = 1002 // Неавторизованный запрос — неверный токен мерчанта
	AlifInternalServerErrorCode         = 1003 // Внутренняя/неизвестная ошибка
	AlifNotFoundErrorCode               = 1004 // Не найдено — карта, оплата и т.д.
	AlifInvalidParametersErrorCode      = 1005 // Невалидные параметры — неверная сумма, пустой параметр
	AlifOperationDeclinedErrorCode      = 1007 // Отказано — операция отклонена
	AlifInvalidOtpErrorCode             = 1100 // Неверный OTP
	AlifOtpExpiredErrorCode             = 1101 // Истёк срок действия OTP
	AlifInvalidCardDataErrorCode        = 1102 // Неверные данные карты — неверный номер или срок действия
	AlifCardExpiredErrorCode            = 1103 // Истёк срок действия карты
	AlifSmsInformingNotEnabledErrorCode = 1104 // Не подключено СМС-информирование для карты
	AlifDuplicateRequestErrorCode       = 1105 // Дублирование
	AlifOtpAttemptsExceededErrorCode    = 1106 // Количество попыток ввода OTP исчерпано
	AlifCardBlockedErrorCode            = 1107 // Карта заблокирована
	AlifInsufficientFundsErrorCode      = 1108 // Недостаточно средств на карте
	AlifLimitExceededErrorCode          = 1109 // Превышен лимит
	AlifPhoneNumberMismatchErrorCode    = 1110 // Номер телефона не совпадает с номером телефона СМС-информирования
)

// Alif integration statuses
const (
	AlifStatusSucceeded            = "SUCCEEDED"               // Успешно
	AlifStatusInsufficientFunds    = "INSUFFICIENT_FUNDS"      // Недостаточно средств
	AlifStatusInvalidCard          = "INVALID_CARD"            // Невалидная карта
	AlifStatusReverted             = "REVERTED"                // Отменена
	AlifStatusOtpRequired          = "OTP_REQUIRED"            // Требуется ввод кода подтверждения
	AlifStatusBlockedCard          = "BLOCKED_CARD"            // Заблокированная карта
	AlifStatusExpiredCard          = "EXPIRED_CARD"            // Истек срок действия карты
	AlifStatusSmsNotificationIsOff = "SMS_NOTIFICATION_IS_OFF" // Отключено СМС-информирование
	AlifStatusIncorrectOtp         = "INCORRECT_OTP"           // Неправильный код подтверждения
	AlifStatusExpiredOtp           = "EXPIRED_OTP"             // Просроченный код подтверждения
	AlifStatusPending              = "PENDING"                 // В процессе обработки
	AlifStatusPendingReversal      = "PENDING_REVERSAL"        // Отмена в процессе обработки
	AlifStatusDeclined             = "DECLINED"                // Платеж отклонен
	AlifStatusUnknownError         = "UNKNOWN_ERROR"           // Неизвестная ошибка
)

const (
	AlifPaymentTypeQrShow = "MOBI_SHOW_QR"
	AlifPaymentTypeCard   = "CARD"
	AlifPay               = "alif_pay"
	AlifPayCreatePath     = "/v2/pay"
	AlifPayConfirmPath    = "/v2/confirmPayment"
)

// Transfer logs constants
const (
	TransferTypeMove   = 0
	TransferTypeReturn = 1

	TransferLogStageCreated   = 0
	TransferLogStageSent      = 1
	TransferLogStageReceived  = 2
	TransferLogStageChecking  = 3
	TransferLogStageCompleted = 4
)

// region
const (
	DmedPrescriptionsNotFound      = "errors.prescriptions.not_found"
	DmedPrescriptionsExpired       = "errors.prescriptions.expired"
	DmedPrescriptionsAlreadyIssued = "errors.prescriptions.already_issued"
)
