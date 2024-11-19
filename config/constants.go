package config

import "time"

const (
	// Access token expire time 24 hour
	AccessTokenExpiresInTime time.Duration = 1 * 60 * 24 * time.Minute
)
