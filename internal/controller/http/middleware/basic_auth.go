package middleware

import "github.com/golanguzb70/middleware/gin/basicauth"

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

func BasicAuth1C() basicauth.Config {
	return basicauth.Config{
		Users: []basicauth.User{
			{
				UserName: "pharma1c",
				Password: "b2kOigr7",
			},
		},
		RequireAuthForAll: true,
	}
}
