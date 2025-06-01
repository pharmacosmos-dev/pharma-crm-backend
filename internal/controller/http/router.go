// Package v1 implements routing paths. Each services in own file.
package http

import (
	"fmt"

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

	// middleware
	option.Gin.Use(customCORSMiddleware())
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
	allowedOrigins := map[string]bool{
		"https://pharma.gofurov.me": true,
		"https://tpharma.noor.uz":   true,
		"https://pharma.noor.uz":    true,
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		fmt.Println("Incoming Origin:", origin)

		// Always set CORS headers
		c.Writer.Header().Set("Vary", "Origin")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, Platform-Type")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")
		c.Writer.Header().Set("Access-Control-Max-Age", "3600")

		if allowedOrigins[origin] {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			fmt.Println("❌ Blocked or missing origin:", origin)
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
