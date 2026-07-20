package v1

import (
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/internal/controller/http/middleware"
	"github.com/pharma-crm-backend/internal/controller/ws"
	"github.com/pharma-crm-backend/internal/services"
	"github.com/pharma-crm-backend/pkg/logger"
	"github.com/pharma-crm-backend/pkg/token"
	"github.com/pharma-crm-backend/pkg/utils"
	"gorm.io/gorm"
)

type Handler struct {
	log             *logger.Logger
	db              *gorm.DB
	cfg             *config.Config
	JwtHandler      *token.JWTHandler
	service         *services.Services
	validator       *utils.Validator
	ordersToMutexes sync.Map
	hub             *ws.Hub
}

func NewHandler(
	cfg *config.Config,
	db *gorm.DB,
	log *logger.Logger,
	jwt *token.JWTHandler,
	service *services.Services,
	validator *utils.Validator,
	hub *ws.Hub,
) *Handler {

	return &Handler{
		cfg:             cfg,
		db:              db,
		log:             log,
		JwtHandler:      jwt,
		service:         service,
		validator:       validator,
		ordersToMutexes: sync.Map{},
		hub:             hub,
	}
}

func (h *Handler) InitRoutes(r *gin.Engine) {
	v1 := r.Group("/v1")
	// Auth Middleware
	bearerAuth := middleware.NewAuthMiddleware(h.cfg, h.JwtHandler, h.db)
	v1.Use(bearerAuth.NewAuth())

	// Route Group for Public APIs
	public := r.Group("/v1")

	// Route Group for 1C APIs
	v1c := r.Group("/v1")
	// Auth Middleware for 1C
    v1c.Use(bearerAuth.Check1CAuth())

	// Route Group for External APIs
	external := r.Group("/v1")
	// Basic Auth Middleware for External APIs
	basicAuth := middleware.ExternalBasicAuth(h.cfg)
	external.Use(basicAuth.Middleware)

	// Route Group for Partner APIs
	partner := r.Group("/uzum")
	// Auth Middleware for Partner
	partnerAuth := middleware.NewAuthMiddleware(h.cfg, h.JwtHandler, h.db)
	partner.Use(partnerAuth.CheckOAuthToken())

	// Handlers
	{
		h.NewAuthHandler(public)
		h.NewAutoOrderHandler(v1)
		h.NewBrandController(v1)
		h.NewCategoryHander(v1)
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
		h.NewDraftHandler(v1)
		h.NewPaymentTypeHandler(v1)
		h.NewPermissionHandler(v1)
		h.NewSalePaymentHandler(v1)
		h.NewImportHandler(v1)
		h.NewProductOnecHandler(v1c)
		h.NewShiftHandler(v1)
		h.NewProducerHandler(v1)
		h.NewDashboardHandler(v1)
		h.NewDiscountCardHandler(v1)
		h.NewHelperHandler(public)
		h.NewCompanyHandler(v1)
		h.NewFinanceCategoryHandler(v1)
		h.NewFinanceOperationHandler(v1)
		h.NewProductBonusHandler(v1)
		h.NewInventoryHandler(v1)
		h.NewReportHandler(v1)
		h.NewWriteOffHandler(v1)
		h.NewReturnHandler(v1)
		h.NewTransferHandler(v1)
		h.NewExpenseHandler(v1)
		h.NewRepricingHandler(v1)
		h.NewRejectedProductsHandler(v1)
		h.NewUzumTezkorProductHandler(v1)
		h.NewLoyaltyCardHandler(v1)
		h.NewLogHandler(v1)
		h.NewPartnerHandler(v1)
		h.NewOstatokHandler(public)
		h.NewStoreTargetHandler(v1)
		h.NewReminderHandler(v1)
		// handler for external apis
		h.NewNoorHandler(external)
		// handler for partner auth apis
		h.NewPartnerAuthHandler(r.Group(""))
		// handler for uzum apis
		h.NewUzumHandler(partner)
	}
}

// Default handler response body
type Response struct {
	Ok      bool   `json:"ok"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Count   int64  `json:"count,omitempty"`
	Data    any    `json:"data"`
}

// handleResponse to send consistent JSON responses
func handleResponse(c *gin.Context, status Status, data any, count ...int64) {
	var responseCount int64

	if len(count) > 0 {
		responseCount = count[0]
	}

	c.JSON(status.Code, Response{
		Ok:      status.Code >= 200 && status.Code < 400, // true for 2xx status codes
		Code:    status.Code,
		Message: status.Description,
		Data:    data,
		Count:   responseCount,
	})
}

// handleServiceResponse handles responses from service layer
// If error is domain.Error, it extracts the proper status code and message
// Otherwise returns success response with data
func handleServiceResponse(c *gin.Context, data any, err error) {
	if err != nil {
		// Check if error is domain.Error type
		if domainErr, ok := err.(*domain.Error); ok {
			// Map domain error code to Status
			status := mapErrorCodeToStatus(domainErr.Code)
			handleResponse(c, status, domainErr.Message)
			return
		}
		// For unknown errors, return internal server error
		handleResponse(c, InternalError, err.Error())
		return
	}

	// Success response
	handleResponse(c, OK, data)
}

// mapErrorCodeToStatus maps HTTP status codes to Status objects
func mapErrorCodeToStatus(code int) Status {
	switch code {
	case 202:
		return Accepted
	case 207:
		return MultiStatus
	case 400:
		return BadRequest
	case 401:
		return UNAUTHORIZED
	case 403:
		return FORBIDDEN
	case 404:
		return NotFound
	case 406:
		return NotAcceptable
	case 409:
		return CONFLICT
	case 422:
		return UnprocessableEntity
	case 429:
		return TooManyRequests
	case 500:
		return InternalError
	case 502:
		return BadGateway
	default:
		return InternalError
	}
}

// Integration handler error response body
type IntegrationErrorResponse struct {
	Message string `json:"message"`
}

// handle response for noor
func handleResponseNoor(c *gin.Context, statusCode int, data any) {
	if statusCode >= 200 && statusCode < 400 {
		// Success: send data as-is
		c.JSON(statusCode, data)
		return
	}

	switch data.(type) {
	case map[string]any, []any:
		c.JSON(statusCode, data)
		return
	}

	// Error: wrap in {"message": "..."}
	var errMsg string
	switch v := data.(type) {
	case string:
		errMsg = v
	case error:
		errMsg = v.Error()
	default:
		errMsg = "Internal server error"
	}

	c.JSON(statusCode, IntegrationErrorResponse{Message: errMsg})
}

// default limit and offset
func defaultLimitOffset(limit, offset int) (int, int) {
	if limit == 0 {
		limit = constants.DefaultLimit
	}
	if offset == 0 {
		offset = constants.DefaultOffset
	}
	return limit, offset
}
