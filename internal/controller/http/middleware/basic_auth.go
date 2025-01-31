package middleware

import (
	"github.com/golanguzb70/middleware/gin/basicauth"
	"github.com/pharma-crm-backend/config"
)

func BasicAuth() basicauth.Config {
	// This configuration checks for all incoming requests for authentication
	return basicauth.Config{
		Users: []basicauth.User{
			{
				UserName: "pharma",
				Password: "54321",
			},
		},
		RestrictedUrls: []string{
			"/swagger/*",
		},
	}
}

func ExternalBasicAuth(cfg *config.Config) basicauth.Config {
	return basicauth.Config{
		Users: []basicauth.User{
			{
				UserName: cfg.ExternalAPIUsername,
				Password: cfg.ExternalAPIPassword,
			},
		},
		RestrictedUrls: []string{
			"/v1/external/*",
		},
	}
}
