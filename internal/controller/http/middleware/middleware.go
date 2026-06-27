package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/pkg/etc"
	"github.com/pharma-crm-backend/pkg/token"
	"gorm.io/gorm"
)

type MiddlewareHandler struct {
	db         *gorm.DB
	cfg        *config.Config
	JwtHandler *token.JWTHandler
}

// Response struct for API responses
type Response struct {
	Ok      bool   `json:"ok"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func NewAuthMiddleware(cfg *config.Config, jwtHandler *token.JWTHandler, db *gorm.DB) *MiddlewareHandler {
	return &MiddlewareHandler{
		cfg:        cfg,
		db:         db,
		JwtHandler: jwtHandler,
	}
}

// NewAuth creates a new authentication middleware
func (m *MiddlewareHandler) NewAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		allow, err := m.CheckPermission(c)
		if err != nil {
			// Handle JWT-related errors
			if errors.Is(err, jwtv5.ErrTokenExpired) {
				m.RequireRefresh(c)
				return
			}
			m.RequirePermission(c)
			return
		}
		if !allow {
			m.RequirePermission(c)
			return
		}
		c.Next()
	}
}

// CheckPermission checks user permissions
func (a *MiddlewareHandler) CheckPermission(c *gin.Context) (bool, error) {
	if c.Request.Method == "OPTIONS" {
		c.AbortWithStatus(http.StatusNoContent)
		return false, nil
	}

	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header missing"})
		return false, nil
	}
	// Extract the token
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
		return false, nil
	}
	// Parse and validate JWT token
	claims, err := a.JwtHandler.ExtractClaims(tokenString, a.cfg.Secret.SecretKey)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
		return false, nil
	}

	c.Set("user_id", claims["user_id"])
	c.Set("company_id", claims["company_id"])
	c.Set("store_id", claims["store_id"])
	c.Set("store_ids", claims["store_ids"])
	c.Set("role", claims["role"])

	return true, nil
}

// RequireRefresh aborts request with 401 status
func (a *MiddlewareHandler) RequireRefresh(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, Response{
		Ok:      false,
		Code:    http.StatusUnauthorized,
		Message: "Unauthorized",
		Data:    "Token is expired",
	})
	c.AbortWithStatus(401)
}

// RequirePermission aborts request with 403 status
func (a *MiddlewareHandler) RequirePermission(c *gin.Context) {
	c.AbortWithStatus(403)
}

// CheckOAuthToken validates OAuth2 Bearer tokens and user tokens
func (a *MiddlewareHandler) CheckOAuthToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip OPTIONS requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header missing"})
			return
		}

		// Extract the Bearer token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "" || tokenString == authHeader {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format, expected: Bearer <token>"})
			return
		}

		// Parse and validate JWT token
		claims, err := a.JwtHandler.ExtractClaims(tokenString, a.cfg.Secret.SecretKey)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		// Check token type - support both OAuth2 client_credentials and user tokens
		tokenType, exists := claims["token_type"]
		if exists && tokenType == "client_credentials" {
			// OAuth2 client credentials token
			clientId, ok := claims["client_id"].(string)
			if !ok {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token: missing client_id"})
				return
			}

			// Extract scopes if present
			scopes := []string{}
			if scopesRaw, ok := claims["scopes"].([]interface{}); ok {
				for _, s := range scopesRaw {
					if scope, ok := s.(string); ok {
						scopes = append(scopes, scope)
					}
				}
			}

			// Set context for OAuth client
			c.Set("client_id", clientId)
			c.Set("scopes", scopes)
			c.Set("token_type", "client_credentials")
		} else {
			// Regular user token
			c.Set("user_id", claims["user_id"])
			c.Set("company_id", claims["company_id"])
			c.Set("store_id", claims["store_id"])
			c.Set("store_ids", claims["store_ids"])
			c.Set("role", claims["role"])
			c.Set("token_type", "user")
		}

		c.Next()
	}
}

// Check1CAuth checks 1C authorization
func (a *MiddlewareHandler) Check1CAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header missing"})
			return
		}
		// Extract the token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			return
		}

		password, err := etc.Decrypt(tokenString, a.cfg.HashKey)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			return
		}
		if password != a.cfg.OnecPassword {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set("password", password)
		c.Next()
	}
}
