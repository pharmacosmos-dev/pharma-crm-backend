package config

import "time"

const (
	// Access token expire time 24 hour
	AccessTokenExpiresInTime time.Duration = 1 * 60 * 24 * time.Minute
	// Refresh token expire time: 30 days
	RefreshTokenExpiresInTime time.Duration = 30 * 24 * time.Hour
)
