package middleware

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/golanguzb70/middleware/gin/basicauth"
	"github.com/pharma-crm-backend/config"

	"github.com/gin-gonic/gin"
)

func BasicAuth() Config {
	// This configuration checks for all incoming requests for authentication
	return Config{
		Users: []User{
			{
				UserName: "pharma",
				Password: "54321",
			},
		},
		RestrictedUrls: []string{
			"/swagger/docs/*",
		},
	}
}

func BasicAuthUzum() Config {
	// This configuration is for Uzum swagger docs authentication
	return Config{
		Users: []User{
			{
				UserName: "uzum@inter",
				Password: "Uzum@092uz",
			},
		},
		RestrictedUrls: []string{
			"/uzum-docs/*",
		},
	}
}

func ExternalBasicAuth(cfg *config.Config) basicauth.Config {
	return basicauth.Config{
		Users: []basicauth.User{
			{
				UserName: cfg.ExternalApiUsername,
				Password: cfg.ExternalApiPassword,
			},
		},
		RestrictedUrls: []string{
			"/v1/external/*",
		},
	}
}

// method for checking authorization
func (cfg *Config) BasicAuthMiddleware(ctx *gin.Context) {
	var (
		authRequired = cfg.RequireAuthForAll
		url          = ctx.Request.URL.Path
		method       = ctx.Request.Method
		authHeader   = ctx.GetHeader("Authorization")
	)

	if ctx.Request.Method == "OPTIONS" {
		ctx.AbortWithStatus(http.StatusNoContent)
		return
	}

	if strings.Contains(strings.Join(cfg.RestrictedMethods, ","), method) {
		authRequired = true
	}

	if !authRequired && len(cfg.RestrictedUrls) > 0 {
		for _, e := range cfg.RestrictedUrls {
			if strings.Contains(e, "*") && strings.Contains(url, strings.TrimSuffix(e, "/*")) {
				authRequired = true
				break
			} else if strings.Contains(e, "{") && string(e[:strings.LastIndex(e, "/")]) == string(url[:strings.LastIndex(url, "/")]) {
				authRequired = true
				break
			} else if e == url {
				authRequired = true
			}
		}
	}

	if authRequired {
		for _, u := range cfg.Users {
			if authHeader == "" {
				ctx.Header("WWW-Authenticate", "Basic realm=Authorization Required")
				ctx.AbortWithStatus(http.StatusUnauthorized)
				return
			}

			credentials := strings.SplitN(authHeader, " ", 2)
			if len(credentials) != 2 {
				ctx.Header("WWW-Authenticate", "Basic realm=Authorization Required")
				ctx.AbortWithStatus(http.StatusUnauthorized)
				return
			}

			decodedCredentials, err := base64.StdEncoding.DecodeString(credentials[1])
			if err != nil {
				ctx.Header("WWW-Authenticate", "Basic realm=Authorization Required")
				ctx.AbortWithStatus(http.StatusUnauthorized)
				return
			}

			credentialsPair := strings.SplitN(string(decodedCredentials), ":", 2)
			if len(credentialsPair) != 2 {
				ctx.Header("WWW-Authenticate", "Basic realm=Authorization Required")
				ctx.AbortWithStatus(http.StatusUnauthorized)
				return
			}

			if credentialsPair[0] != u.UserName || credentialsPair[1] != u.Password {
				ctx.Header("WWW-Authenticate", "Basic realm=Authorization Required")
				ctx.AbortWithStatus(http.StatusUnauthorized)
				return
			}
		}
	}
	ctx.Next()
}
