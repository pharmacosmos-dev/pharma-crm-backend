package v1

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/xuri/excelize/v2"
)

type UzumTezkorProductHandler struct {
	*Handler
}

func (h *Handler) NewUzumTezkorProductHandler(r *gin.RouterGroup) {
	umtkproduct := &UzumTezkorProductHandler{h}
	umtkproduct.UzumTezkorProductRoutes(r)
}

func (h *UzumTezkorProductHandler) UzumTezkorProductRoutes(r *gin.RouterGroup) {
	umtkproduct := r.Group("/uzumtezkor-products")
	{
		umtkproduct.GET("/list", h.List)
		umtkproduct.POST("/create", h.Create)
		umtkproduct.PUT("/update-price", h.UpdatePrice)
		umtkproduct.POST("/upload-product-price-excel", h.UploadExcel)
	}
}


// Create godoc
// @Summary		Create new online product price record (CRM)
// @Tags		UzumTezkor Products
// @Security	BearerAuth
// @Accept		json
// @Produce		json
// @Param		body body domain.CreateOnlinePriceRequest true "material_code, type, retail_price"
// @Success		201 {object} v1.Response
// @Failure		400 {object} v1.Response
// @Failure		500 {object} v1.Response
// @Router		/uzumtezkor-products/create [post]
func (h *UzumTezkorProductHandler) Create(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user == nil {
		handleServiceResponse(c, UNAUTHORIZED, domain.UnauthorizedError)
		return
	}

	var req domain.CreateOnlinePriceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	if err := h.service.CreateOnlinePrice(ctx, &req, user.UserId); err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, CREATED, "CREATED")
}

// UpdatePrice godoc
// @Summary		Update online product price by material_code (CRM)
// @Tags		UzumTezkor Products
// @Security	BearerAuth
// @Accept		json
// @Produce		json
// @Param		body body domain.UpdateOnlinePriceRequest true "material_code and new retail_price"
// @Success		200 {object} v1.Response
// @Failure		400 {object} v1.Response
// @Failure		500 {object} v1.Response
// @Router		/uzumtezkor-products/update-price [put]
func (h *UzumTezkorProductHandler) UpdatePrice(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user == nil {
		handleServiceResponse(c, UNAUTHORIZED, domain.UnauthorizedError)
		return
	}

	var req domain.UpdateOnlinePriceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	if err := h.service.UpdateOnlinePriceByMaterialCode(ctx, &req, user.UserId); err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, "UPDATED")
}

// UploadExcel godoc
// @Summary		Bulk update online prices from Excel (CRM)
// @Tags		UzumTezkor Products
// @Security	BearerAuth
// @Accept		multipart/form-data
// @Produce		json
// @Param		file  formData  file    true   "Excel: col A=product name, col B=material_code, col C=retail_price"
// @Param		type  query     string  false  "Platform type (default: uzum)"
// @Success		200 {object} v1.Response
// @Failure		400 {object} v1.Response
// @Failure		500 {object} v1.Response
// @Router		/uzumtezkor-products/upload-product-price-excel [post]
func (h *UzumTezkorProductHandler) UploadExcel(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user == nil {
		handleResponse(c, UNAUTHORIZED, domain.UnauthorizedError)
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		handleResponse(c, BadRequest, "file is required")
		return
	}

	f, err := fileHeader.Open()
	if err != nil {
		handleResponse(c, BadRequest, "could not open file")
		return
	}
	defer f.Close()

	xlsx, err := excelize.OpenReader(f)
	if err != nil {
		handleResponse(c, BadRequest, "invalid excel file")
		return
	}

	sheetName := xlsx.GetSheetName(0)
	rows, err := xlsx.GetRows(sheetName)
	if err != nil {
		handleResponse(c, BadRequest, "could not read excel rows")
		return
	}

	productType := c.Query("type")
	if productType == "" {
		productType = "uzum"
	}

	var items []domain.UzumTezKorProductRepriceItem
	for i, row := range rows {
		if i == 0 {
			continue
		}
		if len(row) < 3 {
			continue
		}
		materialCode := strings.TrimSpace(row[1])
		retailPrice := parseFloat(row[2])
		if materialCode == "" || retailPrice <= 0 {
			continue
		}
		items = append(items, domain.UzumTezKorProductRepriceItem{
			MaterialCode: materialCode,
			RetailPrice:  retailPrice,
		})
	}

	if len(items) == 0 {
		handleResponse(c, BadRequest, "no valid rows found in excel")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	updated, notFound, err := h.service.BulkUpdateOnlinePriceFromExcel(ctx, items, productType, user.UserId)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, gin.H{
		"updated":   updated,
		"not_found": notFound,
	})
}

// List godoc
// @Summary		List UzumTezkor product price history (CRM)
// @Tags		UzumTezkor Products
// @Security	BearerAuth
// @Produce		json
// @Param		type          query string false "Platform type (uzum, yandex_eda)"
// @Param		product_id    query string false "Product ID"
// @Param		material_code query string false "Material code"
// @Param		limit         query int    false "Limit"
// @Param		offset        query int    false "Offset"
// @Success		200 {object} v1.Response
// @Router		/uzumtezkor-products/list [get]
func (h *UzumTezkorProductHandler) List(c *gin.Context) {
	var params domain.UzumTezkorProductQueryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	result, total, err := h.service.GetOnlineProducts(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, result, total)
}
