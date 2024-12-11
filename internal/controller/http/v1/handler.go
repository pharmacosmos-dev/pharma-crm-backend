package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/internal/controller/http/middleware"
	"github.com/pharma-crm-backend/pkg/logger"
	"github.com/pharma-crm-backend/pkg/token"
	"gorm.io/gorm"
)

type Handler struct {
	log        *logger.Logger
	db         *gorm.DB
	cfg        *config.Config
	JwtHandler *token.JWTHandler
}

func NewHandler(cfg *config.Config, db *gorm.DB, log *logger.Logger, jwt *token.JWTHandler) *Handler {
	return &Handler{
		cfg:        cfg,
		db:         db,
		log:        log,
		JwtHandler: jwt,
	}
}

func (h *Handler) InitRoutes(r *gin.Engine) {
	v1 := r.Group("/v1")
	// Route Group for Public APIs
	public := r.Group("/v1")
	// Route Group for 1C APIs
	v1c := r.Group("/v1")
	// Auth Middleware
	bearerAuth := middleware.NewAuthMiddleware(h.cfg, h.JwtHandler, h.db)
	v1.Use(bearerAuth.NewAuth())

	// Basic Auth Middleware
	basicAuth1C := middleware.BasicAuth1C()
	v1c.Use(basicAuth1C.Middleware)
	{
		h.NewAuthHandler(public)
		h.NewBrandController(v1)
		h.NewCategoryController(v1)
		h.NewCustomerHandler(v1)
		h.NewProductHandler(v1)
		h.NewStoreHandler(v1)
		h.NewRoleHandler(v1)
		h.NewUnitHandler(v1)
		h.NewEmployeeHandler(v1)
		h.NewUploadHandler(public)
		h.NewCashBoxHandler(v1)
		h.NewCashBoxOperationHandler(v1)
		h.NewCartItemHandler(v1)
		h.NewSaleHandler(v1)
		h.NewSaleItemHandler(v1)
		h.NewDraftHandler(v1)
		h.NewPaymentTypeHandler(v1)
		h.NewPermissionHandler(v1)
		h.NewSalePaymentHandler(v1)
		h.NewProduct1cHandler(v1c)
	}
}

type Response struct {
	Ok      bool        `json:"ok"`
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// handleResponse to send consistent JSON responses
func handleResponse(c *gin.Context, status Status, data interface{}) {
	c.JSON(status.Code, Response{
		Ok:      status.Code >= 200 && status.Code < 400, // true for 2xx status codes
		Code:    status.Code,
		Message: status.Description,
		Data:    data,
	})
}
