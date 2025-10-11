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
	ContentTypeJson                         = "application/json"
	ContentTypeFormUrlEncoded               = "application/x-www-form-urlencoded"
	AuthBasic                               = "Basic"
	AuthBearer                              = "Bearer"
	HeaderXAuth                             = "X-Auth"
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
	SaleStageFinished       = 8
	SaleStageReturning      = 9
	SaleStageReturnedFinish = 10
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
	GeneralStatusSentOnec   = ""
	GeneralStatusDeclined   = ""
	GeneralStatusTaxFree    = ""

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
