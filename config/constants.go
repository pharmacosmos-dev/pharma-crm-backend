package config

import "time"

const (
	// Access token expire time 24 hour
	AccessTokenExpiresInTime time.Duration = 2 * 60 * 24 * time.Minute
	// Refresh token expire time: 30 days
	RefreshTokenExpiresInTime time.Duration = 30 * 24 * time.Hour

	DATE_FORMAT      = "2006-01-02"
	DATE_TIME_FORMAT = "2006-01-02 15:04:05"
	DefaultLimit     = 10
	DefaultOffset    = 0
)

const (
	// region IMPORT status

	NEW_IMPORT       = "new"
	PENDING_IMPORT   = "pending"
	COMPLETED_IMPORT = "completed"
	CANCELED_IMPORT  = "canceled"
	WRITEOFF_IMPORT  = "writeoff"

	// region CART ITEM status

	PENDING_CART_ITEM = "pending"
	ACTIVE_CART_ITEM  = "active"
	DELETED_CART_ITEM = "deleted"
	DRAFTED_CART_ITEM = "drafted"
	SOLD_CART_ITEM    = "sold"

	// region PRODUCT status

	ACTIVE_PRODUCT     = "active"
	INACTIVE_PRODUCT   = "inactive"
	LOW_STOCK_PRODUCT  = "low_stock"
	ZERO_STOCK_PRODUCT = "zero_stock"
	EXPIRED_PRODUCT    = "expired"
	DELETED_PRODUCT    = "deleted"

	// region APP payment types

	CLICK = "click"
	PAYME = "payme"
	UZUM  = "uzum"
	CASH  = "cash"
	CARD  = "card"

	// region Universal status types

	NEW       = "new"
	PENDING   = "pending"
	COMPLETED = "completed"
	CANCELED  = "canceled"
	DONE      = "done"
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
)
