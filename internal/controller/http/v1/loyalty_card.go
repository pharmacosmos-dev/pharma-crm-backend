package v1

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/xuri/excelize/v2"
)

type LoyaltyCardHandler struct {
	*Handler
}

func (h *Handler) NewLoyaltyCardHandler(r *gin.RouterGroup) {
	loyaltyCardHandler := &LoyaltyCardHandler{h}
	loyaltyCardHandler.LoyaltyCardRoutes(r)
}

func (h *LoyaltyCardHandler) LoyaltyCardRoutes(r *gin.RouterGroup) {
	loyaltyCard := r.Group("/loyalty_card")
	{
		loyaltyCard.POST("", h.Create)
		loyaltyCard.GET("/dashboard", h.GetDashboard)
		loyaltyCard.GET("/top", h.GetTopCustomers)
		loyaltyCard.GET("/export-excel/", h.ExportLoyaltyCardsExcel)
		// loyaltyCard.GET("/:id", h.Get)
		// loyaltyCard.GET("/list", h.List)
		// loyaltyCard.PUT("/:id", h.Update)
		// loyaltyCard.DELETE("/:id", h.Delete)
	}
}

// Create godoc
// @Summary Create Loyalty Card
// @Description create Loyalty Card
// @Tags loyalty_card
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param loyalty_card body domain.LoyaltyCardCreateRequest true "Loyalty Card info"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /loyalty_card [post]
func (h *LoyaltyCardHandler) Create(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}
	var req domain.LoyaltyCardCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleResponse(c, BadRequest, fmt.Sprintf("Invalid request: %s", err.Error()))
		return
	}

	if !(req.VirtualLoyaltyCardNeeded || *req.LoyaltyCardBarcode != "") {
		handleResponse(c, BadRequest, "Either virtual_loyalty_card_needed must be true or loyalty_card_barcode must be provided")
		return
	}

	req.LoyaltyCardCreatedBy = user.UserId
	customer, err := h.service.CreateLoyaltyCard(&req)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	handleResponse(c, CREATED, customer)
}

// GetDashboard godoc
// @Summary Get Loyalty Card Dashboard Statistics
// @Description Returns loyalty card statistics including total cashback, card counts, and distribution by level
// @Tags loyalty_card
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param from_date query string false "Start date for new cards filter" example:"2024-01-01"
// @Param to_date query string false "End date for new cards filter" example:"2024-12-31"
// @Param is_loyalty query bool false "Filter customers by loyalty card status (true=has card, null=no filter)"
// @Param limit query int false "Number of customers to return" default:10
// @Param offset query int false "Offset for pagination" default:0
// @Success 200 {object} v1.Response{data=domain.LoyaltyCardDashboard}
// @Failure 401 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /loyalty_card/dashboard [get]
func (h *LoyaltyCardHandler) GetDashboard(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var req domain.LoyaltyCardDashboardRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		handleResponse(c, BadRequest, fmt.Sprintf("Invalid query parameters: %s", err.Error()))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	req.Limit, req.Offset = defaultLimitOffset(req.Limit, req.Offset)

	dashboard, err := h.service.GetLoyaltyCardDashboard(ctx, &req)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	handleResponse(c, OK, dashboard)
}

// GetTopCustomers godoc
// @Summary Get Top Loyalty Card Customers
// @Description Returns top customers by cashback earned, with optional date filtering
// @Tags loyalty_card
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Number of customers to return" default:10
// @Param offset query int false "Offset for pagination" default:0
// @Param from_date query string false "Start date for sales filter" example:"2024-01-01"
// @Param to_date query string false "End date for sales filter" example:"2024-12-31"
// @Success 200 {object} v1.Response{data=[]domain.LoyaltyCardTopCustomer}
// @Failure 401 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /loyalty_card/top [get]
func (h *LoyaltyCardHandler) GetTopCustomers(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var req domain.LoyaltyCardTopRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		handleResponse(c, BadRequest, fmt.Sprintf("Invalid query parameters: %s", err.Error()))
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	req.Limit, req.Offset = defaultLimitOffset(req.Limit, req.Offset)

	customers, count, err := h.service.GetLoyaltyCardTopCustomers(ctx, &req)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	result := utils.ListResponse(customers, count, req.Limit, req.Offset)

	handleResponse(c, OK, result)
}

// @Summary      Download loyalty card list as Excel
// @Description  Export filtered loyalty card list to an Excel file
// @Tags         loyalty_card
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param 		 limit query int false "Number of customers to return" default:10
// @Param 		 offset query int false "Offset for pagination" default:0
// @Param        from_date      query     string   false "Start date for filtering cards"
// @Param        to_date        query     string   false "End date for filtering cards"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /loyalty_card/export-excel [get]
func (h *LoyaltyCardHandler) ExportLoyaltyCardsExcel(c *gin.Context) {
		user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}
	var params domain.LoyaltyCardTopRequest
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, nil, domain.InvalidQueryError)
		return
	}
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	// if !helper.IsAdmin(user) {
	// 	params.CompanyId = user.CompanyId
	// }

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()
	// get loyalty card top list data
	res, _, err := h.service.GetLoyaltyCardTopCustomers(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	// Excel fayl yaratish
	f := excelize.NewFile()
	sheetName := constants.DefaultSheetName
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"ID", "ФИО", "Телефон", "Штрихкод карты клиента", "Уровень карты лояльности", "Общая сумма покупок", "Общая сумма полученного кешбэка"}

	err = setExcelHeaders(f, sheetName, headers)
	if err != nil {
		h.log.Error("Failed to create style:", err)
		handleResponse(c, InternalError, "Error on giving style to excel")
		return
	}

	// Ma'lumotlarni qo'shish
	for i, lytcard := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, lytcard.PublicID)
		f.SetCellValue(sheetName, "B"+row, lytcard.FullName)
		f.SetCellValue(sheetName, "C"+row, lytcard.Phone)
		f.SetCellValue(sheetName, "D"+row, lytcard.LoyaltyCardBarcode)
		f.SetCellValue(sheetName, "E"+row, lytcard.LoyaltyCardLevelName)
		f.SetCellValue(sheetName, "F"+row, lytcard.TotalSpent)
		f.SetCellValue(sheetName, "G"+row, lytcard.TotalCashbackEarned)
	}
	saveExcelToUploads(c, f, *h.log, "Loyalty_Cards")
}
