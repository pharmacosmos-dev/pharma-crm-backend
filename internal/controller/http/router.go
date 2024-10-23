// Package v1 implements routing paths. Each services in own file.
package http

import (
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/pharma-crm-backend/config"
	v1 "github.com/pharma-crm-backend/internal/controller/http/v1"
	"github.com/pharma-crm-backend/internal/services"
	"github.com/pharma-crm-backend/internal/storage/repo"
	"github.com/pharma-crm-backend/pkg/logger"

	// "github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

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
func NewRouter(handler *gin.Engine, db *sqlx.DB, log *logger.Logger, cfg *config.Config) {

	// Options
	handler.Use(gin.Logger())
	handler.Use(gin.Recovery())

	// Repositories
	productRepo := repo.NewProductRepository(db, log)
	customerRepo := repo.NewCustomerRepository(db, log)
	employeeRepo := repo.NewEmployeeRepository(db, log)

	// Services
	customerService := services.NewCustomerService(customerRepo, cfg, log)
	productService := services.NewProductService(productRepo, cfg, log)
	employeeService := services.NewEmployeeService(employeeRepo, cfg, log)

	// Swagger
	swaggerHandler := ginSwagger.DisablingWrapHandler(swaggerFiles.Handler, "DISABLE_SWAGGER_HTTP_HANDLER")
	handler.GET("/swagger/*any", swaggerHandler)

	// Routers
	handler.GET("/", Ping)
	api := handler.Group("/v1")

	v1.NewProductRoutes(api.Group("/product"), productService, log)
	v1.NewCustomerHandler(api.Group("/customer"), customerService, log)
	v1.NewEmployeeHandler(api.Group("/employee"), employeeService, log)

}

func Ping(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Server is running!!!"})
}
