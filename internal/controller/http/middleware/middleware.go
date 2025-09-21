package middleware

import (
	"net/http"
	"strings"

	jwtg "github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
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
			if jwtErr, ok := err.(*jwtg.ValidationError); ok && jwtErr.Errors == jwtg.ValidationErrorExpired {
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
	// var user domain.Employee
	// if err := a.db.First(&user, "id = ?", claims["user_id"]).Error; err != nil {
	// 	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
	// 	return false, nil
	// }
	// if user.RoleId != claims["role_id"] {
	// 	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
	// 	return false, nil
	// }
	c.Set("user_id", claims["user_id"])
	c.Set("company_id", claims["company_id"])

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
