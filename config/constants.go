package config

import "time"

const (
	// Access token expire time 24 hour
	AccessTokenExpiresInTime time.Duration = 2 * 60 * 24 * time.Minute
	// Refresh token expire time: 30 days
	RefreshTokenExpiresInTime time.Duration = 30 * 24 * time.Hour

	DATE_FORMAT      = "2006-01-02"
	DATE_TIME_FORMAT = "2006-01-02 15:04:05"
)

const (
	// IMPORT status
	NEW_IMPORT       = "new"
	PENDING_IMPORT   = "pending"
	COMPLETED_IMPORT = "completed"
	CANCELED_IMPORT  = "canceled"
	WRITEOFF_IMPORT  = "writeoff"

	// CART ITEM status
	PENDING_CART_ITEM = "pending"
	ACTIVE_CART_ITEM  = "active"
	DELETED_CART_ITEM = "deleted"
	DRAFTED_CART_ITEM = "drafted"
	SOLD_CART_ITEM    = "sold"

	// PRODUCT status
	ACTIVE_PRODUCT     = "active"
	INACTIVE_PRODUCT   = "inactive"
	LOW_STOCK_PRODUCT  = "low_stock"
	ZERO_STOCK_PRODUCT = "zero_stock"
	EXPIRED_PRODUCT    = "expired"
	DELETED_PRODUCT    = "deleted"

	// APP payment types
	CLICK = "click"
	PAYME = "payme"
	UZUM  = "uzum"
)
