package middleware

// import (
// 	"net/http"
// 	"strings"

// 	jwtg "github.com/dgrijalva/jwt-go"
// 	"github.com/gin-gonic/gin"
// 	"github.com/pharma-crm-backend/config"
// 	"github.com/pharma-crm-backend/domain"
// 	v1 "github.com/pharma-crm-backend/internal/controller/http/v1"
// 	"github.com/pharma-crm-backend/pkg/token"
// 	"gorm.io/gorm"
// )

// type AuhtHandler struct {
// 	*v1.Handler
// }

// func (h *AuhtHandler) NewAuth(cfg *config.Config, jwtHandler token.JWTHandler, db *gorm.DB) gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		allow, err := h.CheckPermission(c)
// 		if err != nil {
// 			v, _ := err.(*jwtg.ValidationError)
// 			if v.Errors == jwtg.ValidationErrorExpired {
// 				h.RequireRefresh(c)
// 			} else {
// 				h.RequirePermission(c)
// 			}
// 		} else if !allow {
// 			h.RequirePermission(c)
// 		}
// 		c.Next()
// 	}
// }

// func (a *AuhtHandler) CheckPermission(c *gin.Context) (bool, error) {
// 	authHeader := c.GetHeader("Authorization")
// 	if authHeader == "" {
// 		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header missing"})
// 		return false, nil
// 	}
// 	// Extract the token
// 	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
// 	if tokenString == "" {
// 		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
// 		return false, nil
// 	}
// 	// Parse and validate JWT token
// 	claims, err := a.JwtHandler.ExtractClaims(tokenString, a.cfg.Secret.SecretKey)
// 	if err != nil {
// 		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
// 		return false, nil
// 	}

// 	var user domain.Employee
// 	if err := a.db.First(&user, "id = ?", claims["user_id"]).Error; err != nil {
// 		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
// 		return false, nil
// 	}
// 	if user.RoleId != claims["role_id"] {
// 		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
// 		return false, nil
// 	}
// 	c.Set("user_id", claims["user_id"])
// 	c.Set("role_id", claims["role_id"])

// 	return true, nil
// }

// // RequireRefresh aborts request with 401 status
// func (a *AuhtHandler) RequireRefresh(c *gin.Context) {
// 	c.JSON(http.StatusUnauthorized, v1.Response{
// 		Ok:      false,
// 		Code:    http.StatusUnauthorized,
// 		Message: "Unauthorized",
// 		Data:    "Token is expired",
// 	})
// 	c.AbortWithStatus(401)
// }

// // RequirePermission aborts request with 403 status
// func (a *AuhtHandler) RequirePermission(c *gin.Context) {
// 	c.AbortWithStatus(403)
// }
