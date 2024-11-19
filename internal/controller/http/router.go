// Package v1 implements routing paths. Each services in own file.
package http

import (
	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/internal/controller/http/middleware"
	v1 "github.com/pharma-crm-backend/internal/controller/http/v1"
	"github.com/pharma-crm-backend/pkg/logger"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"

	// Swagger docs.
	_ "github.com/pharma-crm-backend/docs"
)

// Swagger spec:
// @title       Pharma CRM API
// @version     1.0
// @description     This is a sample server celler server.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @securityDefinitions.basic  BasicAuth
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Bearer token
// @type apiKey
// @BasePath    /v1

// NewRouter -.
func NewRouter(handler *gin.Engine, db *gorm.DB, log *logger.Logger, cfg *config.Config) {

	// Options
	handler.Use(gin.Logger())
	handler.Use(gin.Recovery())

	controller := v1.NewController(db, cfg, log)

	// Swagger
	swaggerHandler := ginSwagger.DisablingWrapHandler(swaggerFiles.Handler, "DISABLE_SWAGGER_HTTP_HANDLER")
	handler.GET("/docs/*any", swaggerHandler)

	// Routers
	handler.GET("/", Ping)
	api := handler.Group("/v1")
	api.Use(middleware.AuthMiddleware())

	// brand route group
	brand := api.Group("/brand")
	brand.POST("", controller.Brand.Create)
	brand.GET("", controller.Brand.Get)
	brand.GET("/get-list", controller.Brand.List)
	brand.PUT("", controller.Brand.Update)
	brand.DELETE("", controller.Brand.Delete)

	// category route group
	category := api.Group("/category")
	category.POST("", controller.Category.Create)
	category.GET("", controller.Category.Get)
	category.GET("/get-list", controller.Category.List)
	category.PUT("", controller.Category.Update)
	category.DELETE("", controller.Category.Delete)

}

func Ping(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Server is running!!!"})
}
