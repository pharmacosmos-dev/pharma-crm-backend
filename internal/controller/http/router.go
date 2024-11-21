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
func NewRouter(handler *gin.Engine, db *gorm.DB, log *logger.Logger, cfg *config.Config) {

	// Options
	handler.Use(gin.Logger())
	handler.Use(gin.Recovery())

	// Cors Conf
	corConfig := cors.DefaultConfig()
	corConfig.AllowAllOrigins = true
	corConfig.AllowCredentials = true
	corConfig.AllowHeaders = []string{"*"}
	corConfig.AllowBrowserExtensions = true
	corConfig.AllowMethods = []string{"*"}
	handler.Use(cors.New(corConfig))

	// JWTHandler
	jwtHandler := token.JWTHandler{
		Cfg: cfg,
		Log: log,
	}

	controller := v1.NewController(db, cfg, log, jwtHandler)

	// Routers
	handler.GET("/", Ping)
	v1 := handler.Group("/v1")
	v1.Use(middleware.NewAuth(cfg, jwtHandler, db))

	// auth route
	auth := handler.Group("/v1")
	auth.POST("/login", controller.Employee.Login)
	auth.POST("/logout", controller.Employee.Logout)

	// brand route group
	brand := v1.Group("/brand")
	brand.POST("", controller.Brand.Create)
	brand.GET("", controller.Brand.Get)
	brand.GET("/get-list", controller.Brand.List)
	brand.PUT("", controller.Brand.Update)
	brand.DELETE("", controller.Brand.Delete)

	// category route group
	category := v1.Group("/category")
	category.POST("", controller.Category.Create)
	category.GET("", controller.Category.Get)
	category.GET("/get-list", controller.Category.List)
	category.PUT("", controller.Category.Update)
	category.DELETE("", controller.Category.Delete)

	// employee route group
	employee := v1.Group("/employee")
	auth.POST("/employee", controller.Employee.Create)
	employee.GET("", controller.Employee.Get)
	employee.PUT("", controller.Employee.Update)
	employee.GET("/get-list", controller.Employee.List)
	employee.DELETE("", controller.Employee.Delete)

	// product route group
	product := v1.Group("/product")
	product.POST("", controller.Product.Create)
	product.GET("", controller.Product.Get)
	product.PUT("", controller.Product.Update)
	product.GET("/get-list", controller.Product.List)
	product.DELETE("", controller.Product.Delete)
	product.POST("/upload-excel", controller.Product.UploadProduct)

	// role route group
	role := v1.Group("/role")
	auth.POST("/role", controller.Role.Create)
	role.GET("", controller.Role.Get)
	role.PUT("", controller.Role.Update)
	role.GET("/get-list", controller.Role.List)
	role.DELETE("", controller.Role.Delete)

	// store route group
	store := v1.Group("/store")
	store.POST("", controller.Store.Create)
	store.GET("", controller.Store.Get)
	store.PUT("", controller.Store.Update)
	store.GET("/get-list", controller.Store.List)
	store.DELETE("", controller.Store.Delete)

	// customer route group
	customer := v1.Group("/customer")
	customer.POST("", controller.Customer.Create)
	customer.GET("", controller.Customer.Get)
	customer.PUT("", controller.Customer.Update)
	customer.GET("/get-list", controller.Customer.List)
	customer.DELETE("", controller.Customer.Delete)

	// unit route group
	unit := v1.Group("/unit")
	unit.POST("", controller.Unit.Create)
	unit.GET("", controller.Unit.Get)
	unit.PUT("", controller.Unit.Update)
	unit.GET("/get-list", controller.Unit.List)
	unit.DELETE("", controller.Unit.Delete)

	// Swagger Route
	url := ginSwagger.URL("swagger/doc.json")
	handler.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, url))
}

func Ping(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Server is running!!!"})
}
