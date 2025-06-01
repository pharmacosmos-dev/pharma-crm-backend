// Package v1 implements routing paths. Each services in own file.
package http

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/config"
	_ "github.com/pharma-crm-backend/docs"
	"github.com/pharma-crm-backend/internal/controller/http/middleware"
	v1 "github.com/pharma-crm-backend/internal/controller/http/v1"
	"github.com/pharma-crm-backend/internal/services"
	"github.com/pharma-crm-backend/pkg/logger"
	"github.com/pharma-crm-backend/pkg/token"
	"github.com/pharma-crm-backend/pkg/utils"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

type Options struct {
	Gin     *gin.Engine
	Db      *gorm.DB
	Log     *logger.Logger
	Cfg     *config.Config
	Service *services.Services
}

// @title Pharma API docs
// @version 1.0
// @description This is a sample server caller server.
// @termsOfService  http://swagger.io/terms/

// @BasePath  /v1

// @securityDefinitions.basic  BasicAuth
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// NewRouter -.
func NewRouter(option Options) {

	// Basic Auth
	basicAuth := middleware.BasicAuth()

	// Method 1: Using gin-contrib/cors package (Recommended)
	option.Gin.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "https://tpharma.noor.uz"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length", "X-Total-Count"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	option.Gin.Use(gin.Logger())
	option.Gin.Use(gin.Recovery())
	option.Gin.Use(basicAuth.BasicAuthMiddleware)
	// JWTHandler
	jwtHandler := token.JWTHandler{
		Cfg: option.Cfg,
		Log: option.Log,
	}
	// validation
	validator := utils.NewValidator(option.Log)

	// Handlers
	handler := v1.NewHandler(option.Cfg, option.Db, option.Log, &jwtHandler, option.Service, validator)
	handler.InitRoutes(option.Gin)

	// PING
	option.Gin.GET("/", Ping)

	// Swagger Route
	url := ginSwagger.URL("swagger/doc.json")
	option.Gin.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, url))
}

func Ping(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Server is running!!!"})
}

// custom corse middleware
func customCORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		allowedOrigins := map[string]bool{
			"https://tpharma.noor.uz":   true,
			"https://pharma.noor.uz":    true,
			"https://pharma.gofurov.me": true,
		}

		if allowedOrigins[origin] {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
