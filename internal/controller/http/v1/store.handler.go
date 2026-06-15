package v1

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/helper"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type StoreHandler struct {
	*Handler
}

func (h *Handler) NewStoreHandler(r *gin.RouterGroup) {
	store := &StoreHandler{h}
	store.StoreRoutes(r)
}

func (h *StoreHandler) StoreRoutes(r *gin.RouterGroup) {
	store := r.Group("/store")
	{
		store.POST("", h.Create)
		store.GET("/:id", h.Get)
		store.GET("/list", h.FetchStores)
		store.GET("/export-excel", h.ExportExcel)
		store.PUT("/:id", h.Update)
		store.PUT("/online-order", h.UpdateOnlineOrder)
		store.DELETE("/:id", h.Delete)
	}
}

// Create godoc
// @Summary Create a store
// @Description Create a store from the request body
// @Tags stores
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.StoreRequest true "Store information"
// @Success 201 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /store [post]
func (h *StoreHandler) Create(c *gin.Context) {
	var (
		body domain.StoreRequest
	)
	// bind request body
	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Errorf("could not bind store create request body: %v", err)
		handleServiceResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}
	// validate phone number
	if body.Phone != nil && !utils.IsValidPhone(*body.Phone) {
		handleResponse(c, BadRequest, "Invalid phone number")
		return
	}

	// generate store id
	body.Id = uuid.New().String()
	// create new store info
	err := h.db.
		WithContext(c.Request.Context()).
		Table("stores").
		Create(&body).Error
	if err != nil {
		h.log.Errorf("could not create store: %v", err)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
		return
	}
	handleResponse(c, CREATED, body)
}

// Get godoc
// @Summary Get a store
// @Description Get a store from the request body
// @Tags stores
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "store ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /store/{id} [get]
func (h *StoreHandler) Get(c *gin.Context) {
	var res domain.Store
	err := h.db.Take(&res, "id = ?", c.Param("id")).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleServiceResponse(c, NotFound, domain.NotFoundError)
			return
		}
		h.log.Errorf("could not get store by id: %v", err)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
		return
	}
	handleResponse(c, OK, res)
}

// List godoc
// @Summary Get a store
// @Description Get a store from the request body
// @Tags stores
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param search query string false "Search"
// @Param product_id query string false "Product ID"
// @Param is_franchise query bool false "is_franchise"
// @Param is_online query bool false "Filter by online order stores"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /store/list [get]
func (h *StoreHandler) FetchStores(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.StoreQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		h.log.Errorf("could not bind query params for store list: %v", err)
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.DefaultContextTimeout)
	defer cancel()

	if !helper.IsAdmin(user) {
		switch user.Role {
		case constants.RoleFranchise:
			params.CompanyIds, _ = h.service.GetCompanyIds(ctx, true)
			params.CompanyId = ""
		case constants.RoleFranchiseAdmin:
			params.CompanyId = user.CompanyId
		default:
			params.CompanyId = user.CompanyId
			params.StoreId = user.StoreId
		}
	}

	if len(user.StoreIds) > 0 {
		params.StoreIds = user.StoreIds
		params.CompanyId = ""
		params.StoreId = ""
	}

	res, totalCount, ids, err := h.service.GetStores(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	handleResponse(c, OK, map[string]interface{}{
		"_meta": utils.Meta{
			TotalCount:  totalCount,
			PerPage:     params.Limit,
			CurrentPage: (params.Offset / params.Limit) + 1,
			PageCount:   int((totalCount + int64(params.Limit) - 1) / int64(params.Limit)),
		},
		"data": res,
		"ids":  ids,
	})
}

// List godoc
// @Summary Get a store export in Excel format
// @Description Get a store export in Excel format
// @Tags stores
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param search query string false "Search"
// @Param product_id query string false "Product ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /store/export-excel [get]
func (h *StoreHandler) ExportExcel(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.StoreQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		h.log.Errorf("could not bind query params for store list: %v", err)
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	// check if employee is not admin or superadmin
	if !helper.IsAdmin(user) {
		params.CompanyId = user.CompanyId
	}

	res, _, _, err := h.service.GetStores(c.Request.Context(), &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	// Create excel file
	f := excelize.NewFile()
	sheetName := "List"
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"ID", "Наименование", "Полная Наименование", "Телефон", "Режим работы", "Адрес", "Координаты", "Дата создания"}

	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Color: "000000",
		},
	})
	if err != nil {
		h.log.Error("Failed to create style:", err)
		handleResponse(c, InternalError, "Error on giving style to excel")
		return
	}

	for i, h := range headers {
		col := string(rune('A'+i)) + "1"
		f.SetCellValue(sheetName, col, h)
		f.SetCellStyle(sheetName, col, col, headerStyle)
	}

	// Ma'lumotlarni qo'shish
	for i, r := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, r.StoreCode)
		f.SetCellValue(sheetName, "B"+row, r.Name)
		f.SetCellValue(sheetName, "C"+row, r.DetailedName)
		f.SetCellValue(sheetName, "D"+row, r.Phone)
		f.SetCellValue(sheetName, "E"+row, r.WorkHours)
		f.SetCellValue(sheetName, "F"+row, r.Address)
		f.SetCellValue(sheetName, "G"+row, r.Coordinates.ToSinglePointWKT())
		f.SetCellValue(sheetName, "H"+row, r.CreatedAt.Add(constants.DateTimeTashkent).Format(constants.DateTimeFormat))
	}

	// Faylni uploads/ papkasiga UUID bilan saqlash
	fileName := "Apteka_" + time.Now().Add(time.Hour*5).Format("2006-01-02_15-04-05") + ".xlsx"
	filePath := filepath.Join("uploads", fileName)

	// uploads/ papkasi mavjud bo‘lmasa, yaratish
	if _, err := os.Stat("uploads"); os.IsNotExist(err) {
		err := os.Mkdir("uploads", os.ModePerm)
		if err != nil {
			h.log.Error("Failed to create uploads directory:", err)
			handleResponse(c, InternalError, "Failed to create uploads folder")
			return
		}
	}

	// Faylni diskka yozish
	if err := f.SaveAs(filePath); err != nil {
		h.log.Errorf("Failed to save Excel file: %v", err)
		handleResponse(c, InternalError, "Failed to save Excel file")
		return
	}

	// Foydalanuvchiga file path yoki URLni qaytarish
	handleResponse(c, OK, gin.H{
		"file_name": fileName,
	})
}

// Update godoc
// @Summary Update a store
// @Description Update a store from the request body
// @Tags stores
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "store ID"
// @Param store body domain.StoreRequest true "Store information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /store/{id} [put]
func (h *StoreHandler) Update(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, UNAUTHORIZED, domain.UnauthorizedError)
		return
	}

	var (
		body domain.StoreUpdateRequest
		id   = c.Param("id")
	)
	// bind request body
	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Errorf("could not bind store update request body: %v", err)
		handleServiceResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.DefaultContextTimeout)
	defer cancel()

	// validate phone number
	if body.Phone != nil && !utils.IsValidPhone(*body.Phone) {
		handleServiceResponse(c, BadRequest, domain.InvalidPhoneError)
		return
	}

	body.UpdatedBy = user.UserId
	// update store info
	err := h.db.WithContext(ctx).
		Model(&domain.Store{}).
		Where("id = ?", id).
		Updates(&body).Error
	if err != nil {
		h.log.Errorf("could not update store: %v", err)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
		return
	}
	handleResponse(c, OK, body)
}

// UpdateOnlineOrder godoc
// @Summary Update online order status for multiple stores
// @Description Set is_online_order field for given store IDs
// @Tags stores
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.UpdateOnlineOrderRequest true "Store IDs and online order flag"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /store/online-order [put]
func (h *StoreHandler) UpdateOnlineOrder(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, UNAUTHORIZED, domain.UnauthorizedError)
		return
	}

	var body domain.UpdateOnlineOrderRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Errorf("could not bind update online order request body: %v", err)
		handleServiceResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}

	err := h.db.
		WithContext(c.Request.Context()).
		Model(&domain.Store{}).
		Where("id IN ?", body.StoreIds).
		Update("is_online_order", body.IsOnlineOrder).Error
	if err != nil {
		h.log.Errorf("could not update is_online_order for stores: %v", err)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
		return
	}

	handleResponse(c, OK, "updated")
}

// Delete godoc
// @Summary Delete a store
// @Description Delete a store from the request body
// @Tags 		 stores
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "store ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /store/{id} [delete]
func (h *StoreHandler) Delete(c *gin.Context) {
	var id = c.Param("id")
	err := h.db.
		WithContext(c.Request.Context()).
		Model(&domain.Store{}).
		Where("id = ?", id).
		Update("is_active", false).
		Update("deleted_at", time.Now()).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}
