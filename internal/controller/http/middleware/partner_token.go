package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/config"
)

// PartnerTokenAuth checks the static token of partner APIs: "Authorization: Bearer <token>".
// Tokens come from PARTNER_API_TOKEN, several partners separated by commas.
// With no token configured every request is rejected.
func PartnerTokenAuth(cfg *config.Config) gin.HandlerFunc {
	var tokens []string
	for _, token := range strings.Split(cfg.PartnerApiToken, ",") {
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
