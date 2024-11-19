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
	// _ "github.com/pharma-crm-backend/docs"
)

// @title           Swagger PHARMA API
// @version         1.0
// @description     This is a sample server celler server.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @BasePath  /v1

// @securityDefinitions.basic  BasicAuth
// @securityDefinitions.apikey BearerAuth
// @securityDefinitions.apikey BearerAuth

// @in header
// @name Authorization

// @externalDocs.description  OpenAPI
// @externalDocs.url          https://swagger.io/resources/open-api/

// NewRouter -.
func NewRouter(handler *gin.Engine, db *gorm.DB, log *logger.Logger, cfg *config.Config) {

	// Options
	handler.Use(gin.Logger())
	handler.Use(gin.Recovery())

	controller := v1.NewController(db, cfg, log)

	// Routers
	handler.GET("/", Ping)
	api := handler.Group("/v1")
	auth := handler.Group("/v1")
	api.Use(middleware.AuthMiddleware())

	// auth route
	auth.POST("/login", controller.Employee.Login)
	auth.POST("/logout", controller.Employee.Logout)

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

	// employee route group
	employee := api.Group("/employee")
	employee.POST("", controller.Employee.Create)
	employee.GET("", controller.Employee.Get)
	employee.PUT("", controller.Employee.Update)
	employee.GET("/get-list", controller.Employee.List)
	employee.DELETE("", controller.Employee.Delete)

	// product route group
	product := api.Group("/product")
	product.POST("", controller.Product.Create)
	product.GET("", controller.Product.Get)
	product.PUT("", controller.Product.Update)
	product.GET("/get-list", controller.Product.List)
	product.DELETE("", controller.Product.Delete)

	// role route group
	role := api.Group("/role")
	role.POST("", controller.Role.Create)
	role.GET("", controller.Role.Get)
	role.PUT("", controller.Role.Update)
	role.GET("/get-list", controller.Role.List)
	role.DELETE("", controller.Role.Delete)

	// store route group
	store := api.Group("/store")
	store.POST("", controller.Store.Create)
	store.GET("", controller.Store.Get)
	store.PUT("", controller.Store.Update)
	store.GET("/get-list", controller.Store.List)
	store.DELETE("", controller.Store.Delete)

	// customer route group
	customer := api.Group("/customer")
	customer.POST("", controller.Customer.Create)
	customer.GET("", controller.Customer.Get)
	customer.PUT("", controller.Customer.Update)
	customer.GET("/get-list", controller.Customer.List)
	customer.DELETE("", controller.Customer.Delete)

	// unit route group
	unit := api.Group("/unit")
	unit.POST("", controller.Unit.Create)
	unit.GET("", controller.Unit.Get)
	unit.PUT("", controller.Unit.Update)
	unit.GET("/get-list", controller.Unit.List)
	unit.DELETE("", controller.Unit.Delete)

	// Swagger
	handler.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

func Ping(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Server is running!!!"})
}
