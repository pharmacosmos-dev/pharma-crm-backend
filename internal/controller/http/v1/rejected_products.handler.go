package v1

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/xuri/excelize/v2"
)

type RejectedProductsHandler struct {
	*Handler
}

func (h *Handler) NewRejectedProductsHandler(r *gin.RouterGroup) {
	rejectedProducts := &RejectedProductsHandler{h}
	rejectedProducts.RejectedProductsRoutes(r)
}

func (h *RejectedProductsHandler) RejectedProductsRoutes(r *gin.RouterGroup) {
	rejectedProducts := r.Group("/rejected-products")
	{
		rejectedProducts.POST("", h.Create)
		rejectedProducts.GET("/products", h.GetListOfProducts)
		//rejectedProducts.GET("/:id", h.Get)
		rejectedProducts.GET("/list", h.List)
		rejectedProducts.POST("/export-excel", h.ExportRejectedProducts)
		//rejectedProducts.PUT("/:id", h.Update)
		//rejectedProducts.DELETE("", h.Delete)
	}
}

// godoc Create
// @Summary Create a rejected product
// @Description Create a rejected product
// @Tags rejected-products
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param body body domain.RejectedProductRequest true "Rejected product request"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /rejected-products [post]
func (h *RejectedProductsHandler) Create(c *gin.Context) {
	var body domain.RejectedProductRequest

	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	userId, ok := c.Get("user_id")
	if !ok {
		h.log.Warn("Error on getting user id from context")
		handleResponse(c, BadRequest, "User not authorized")
		return
	}
	// get creator id from set header
	body.CreatedBy = userId.(string)
	if body.StoreID == "" {
		handleResponse(c, BadRequest, "store_id required")
		return
	}

	if err := h.service.CreateRejectedProduct(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, "CREATED")
}

// godoc GetListOfProducts
// @Summary Get list of products
// @Description Get list of products
// @Tags rejected-products
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param   limit query int false "Limit"
// @Param   offset query int false "Offset"
// @Param   search query string false "Search"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /rejected-products/products [get]
func (h *RejectedProductsHandler) GetListOfProducts(c *gin.Context) {
	var req domain.RejectedProductQueryParam
	if err := c.ShouldBindQuery(&req); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	req.Limit, req.Offset = defaultLimitOffset(req.Limit, req.Offset)
	products, err := h.service.SearchProductsByName(req.Search, req.Limit, req.Offset)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, products)
}

// List RejectedProducts godoc
// @Summary Get rejected products list
// @Description Get rejected products grouped by store_id and product (id or name) with total count per group
// @Tags rejected-products
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param store_id query string false "Store ID"
// @Param product_id query string false "Product ID"
// @Param search query string false "Product Name"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /rejected-products/list [get]
func (h *RejectedProductsHandler) List(c *gin.Context) {
	var (
		param domain.RejectedProductQueryParam
		err   error
	)

	// Bind query parameters
	if err = c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	// Default limit/offset
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	// Query database
	rejectedProducts, totalCount, err := h.service.ListRejectedProducts(&param)
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}

	result := utils.ListResponse(rejectedProducts, totalCount, param.Limit, param.Offset)
	handleResponse(c, OK, result)
}

// ExportRejectedProducts godoc
// @Summary Export Rejected Products to Excel
// @Description Export grouped rejected products to Excel
// @Tags rejected-products
// @Security BearerAuth
// @Produce json
// @Param	order 	query string false "Order"
// @Param   search 	query string false "Search"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param start_date query string false "Start Date"
// @Param end_date query string false "End Date"
// @Param store_id query string false "Store ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /rejected-products/export-excel [POST]
func (h *RejectedProductsHandler) ExportRejectedProducts(c *gin.Context) {
	var param domain.RejectedProductQueryParam
	if err := c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, "Invalid query parameters")
		return
	}
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	// get grouped rejected products
	res, _, err := h.service.ListRejectedProducts(&param)
	if err != nil {
		handleResponse(c, InternalError, "Can't get rejected products data")
		return
	}

	// create excel file
	f := excelize.NewFile()
	sheet := "RejectedProducts"
	f.SetSheetName("Sheet1", sheet)

	// set headers
	headers := []string{"ID", "Название продукта", "Количество", "Название магазина", "Создатель"}
	err = setExcelHeaders(f, sheet, headers)
	if err != nil {
		h.log.Error("Failed to create style:", err)
		handleResponse(c, InternalError, "Error on giving style to excel")
		return
	}

	// fill rows
	for i, val := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheet, "A"+row, i+1)
		f.SetCellValue(sheet, "B"+row, val.ProductName)
		f.SetCellValue(sheet, "C"+row, val.Count)
		f.SetCellValue(sheet, "D"+row, val.StoreName)
		f.SetCellValue(sheet, "E"+row, val.CreatedBy)
	}

	// save excel
	saveExcelToUploads(c, f, *h.log, "rejected_products")
}
