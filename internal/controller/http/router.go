// Package v1 implements routing paths. Each services in own file.
package http

import (
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
	Gin  *gin.Engine
	Db   *gorm.DB
	Log  *logger.Logger
	Cfg  *config.Config
	Strg *services.Services
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
	option.Gin.Use(basicAuth.Middleware)

	// Options
	option.Gin.Use(gin.Logger())
	option.Gin.Use(gin.Recovery())

	// CORS Configuration
	corConfig := cors.DefaultConfig()
	corConfig.AllowAllOrigins = true
	corConfig.AllowCredentials = true
	corConfig.AllowHeaders = []string{"*"}
	corConfig.AllowBrowserExtensions = true
	corConfig.AllowMethods = []string{"*"}
	option.Gin.Use(cors.New(corConfig))

	// JWTHandler
	jwtHandler := token.JWTHandler{
		Cfg: option.Cfg,
		Log: option.Log,
	}
	// validation
	validator := utils.NewValidator(option.Log)

	// Handlers
	handler := v1.NewHandler(option.Cfg, option.Db, option.Log, &jwtHandler, option.Strg, validator)
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
