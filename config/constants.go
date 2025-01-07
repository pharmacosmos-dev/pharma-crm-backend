package config

import "time"

const (
	// Access token expire time 24 hour
	AccessTokenExpiresInTime time.Duration = 1 * 60 * 24 * time.Minute
	// Refresh token expire time: 30 days
	RefreshTokenExpiresInTime time.Duration = 30 * 24 * time.Hour
)

const (
	// IMPORT status
	NEW_IMPORT       = "new"
	PENDING_IMPORT   = "pending"
	COMPLETED_IMPORT = "completed"
	CANCELED_IMPORT  = "canceled"

	// CART ITEM status
	PENDING_CART_ITEM = "pending"
	ACTIVE_CART_ITEM  = "active"
	DELETED_CART_ITEM = "deleted"
	DRAFTED_CART_ITEM = "drafted"
	SOLD_CART_ITEM    = "sold"
)
