package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/config"
)

// NoorTokenAuth checks the static token Noor sends to /v1/noor/*: "Authorization: Bearer <token>".
// The token is NOOR_API_TOKEN, the same one we send to Noor's API, so changing it affects both directions.
// The token has no expiry; revoke it by changing the env and recreating the container.
// With no token configured every request is rejected.
func NoorTokenAuth(cfg *config.Config) gin.HandlerFunc {
	var tokens []string
	for _, token := range strings.Split(cfg.Integration.NoorApiToken, ",") {
		if token = strings.TrimSpace(token); token != "" {
			tokens = append(tokens, token)
		}
	}

	return func(ctx *gin.Context) {
		token := strings.TrimSpace(strings.TrimPrefix(ctx.GetHeader("Authorization"), "Bearer "))
		for _, allowed := range tokens {
			if subtle.ConstantTimeCompare([]byte(token), []byte(allowed)) == 1 {
				ctx.Next()
				return
			}
		}

		ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
	}
}
