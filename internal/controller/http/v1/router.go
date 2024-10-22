// Package v1 implements routing paths. Each services in own file.
package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/internal/controller/http/v1/customer"
	"github.com/pharma-crm-backend/internal/controller/http/v1/employee"
	"github.com/pharma-crm-backend/internal/controller/http/v1/product"
	"github.com/pharma-crm-backend/pkg/logger"

	// "github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	// Swagger docs.
	_ "github.com/pharma-crm-backend/docs"
	"github.com/pharma-crm-backend/internal/storage"
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
func NewRouter(handler *gin.Engine, l logger.Interface, s *storage.Storage) {
	// Options
	handler.Use(gin.Logger())
	handler.Use(gin.Recovery())

	// Swagger
	swaggerHandler := ginSwagger.DisablingWrapHandler(swaggerFiles.Handler, "DISABLE_SWAGGER_HTTP_HANDLER")
	handler.GET("/swagger/*any", swaggerHandler)

	// Routers
	handler.GET("/", Ping)
	api := handler.Group("/v1")
	{
		product.NewProductRoutes(api.Group("/product"), s.ProductRepo, l)
		customer.NewCustomerHandler(api.Group("/customer"), s.CustomerRepo, l)
		employee.NewEmployeeHandler(api.Group("/employee"), s.EmployeeRepo, l)

	}
}

func Ping(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Server is running!!!"})
}
