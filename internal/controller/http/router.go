// Package v1 implements routing paths. Each services in own file.
package http

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/config"
	_ "github.com/pharma-crm-backend/docs"
	"github.com/pharma-crm-backend/internal/controller/http/middleware"
	v1 "github.com/pharma-crm-backend/internal/controller/http/v1"
	"github.com/pharma-crm-backend/pkg/logger"
	"github.com/pharma-crm-backend/pkg/token"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

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
func NewRouter(ginEngine *gin.Engine, db *gorm.DB, log *logger.Logger, cfg *config.Config) {

	// Basic Auth
	basicAuth := middleware.BasicAuth()
	ginEngine.Use(basicAuth.Middleware)

	// Options
	ginEngine.Use(gin.Logger())
	ginEngine.Use(gin.Recovery())

	// CORS Configuration
	corConfig := cors.DefaultConfig()
	corConfig.AllowAllOrigins = true
	corConfig.AllowCredentials = true
	corConfig.AllowHeaders = []string{"*"}
	corConfig.AllowBrowserExtensions = true
	corConfig.AllowMethods = []string{"*"}
	ginEngine.Use(cors.New(corConfig))

	// JWTHandler
	jwtHandler := token.JWTHandler{
		Cfg: cfg,
		Log: log,
	}

	// Handlers
	handler := v1.NewHandler(cfg, db, log, &jwtHandler)
	handler.InitRoutes(ginEngine)

	// PING
	ginEngine.GET("/", Ping)

	// Swagger Route
	url := ginSwagger.URL("swagger/doc.json")
	ginEngine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, url))
}

func Ping(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Server is running!!!"})
}
