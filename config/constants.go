package config

import (
	"time"
)

const (
	// Access token expire time 24 hour
	AccessTokenExpiresInTime time.Duration = 2 * 60 * 24 * time.Minute
	// Refresh token expire time: 30 days
	RefreshTokenExpiresInTime time.Duration = 30 * 24 * time.Hour

	// Context timeouts for reports and other long-running operations
	ContextTimeoutForReports time.Duration = 1 * time.Minute
	ContextTimeout           time.Duration = 10 * time.Second

	DATE_FORMAT    = "2006-01-02"
	DATE_TIME      = "2006-01-02 15:04:05"
	DATE_1C_FORMAT = "2006-01-02T15:04:05"
	DefaultLimit   = 10
	DefaultOffset  = 0
)

const (
	// region import status

	NEW_IMPORT       = "new"
	PENDING_IMPORT   = "pending"
	COMPLETED_IMPORT = "completed"
	CANCELED_IMPORT  = "canceled"
	WRITEOFF_IMPORT  = "writeoff"

	// end region

	// region cart_item status

	PENDING_CART_ITEM = "pending"
	ACTIVE_CART_ITEM  = "active"
	DELETED_CART_ITEM = "deleted"
	DRAFTED_CART_ITEM = "drafted"
	SOLD_CART_ITEM    = "sold"

	// region product status

	ACTIVE_PRODUCT     = "active"
	INACTIVE_PRODUCT   = "inactive"
	LOW_STOCK_PRODUCT  = "low_stock"
	ZERO_STOCK_PRODUCT = "zero_stock"
	EXPIRED_PRODUCT    = "expired"
	DELETED_PRODUCT    = "deleted"

	// region app payment types

	CASH    = "cash"
	CARD    = "card"
	CLICK   = "click"
	PAYME   = "payme"
	UZUM    = "uzum"
	ALIF    = "alif"
	PERCENT = "percent"

	// region universal status types

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

	// region online status

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

	// region sale type

	SALE_TYPE_RETURN = "RETURN"
	SALE_TYPE_SALE   = "SALE"
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
