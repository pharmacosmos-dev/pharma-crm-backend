package v1

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
)

type AutoOrderHandler struct {
	*Handler
}

func (h *Handler) NewAutoOrderHandler(r *gin.RouterGroup) {
	autoOrder := &AutoOrderHandler{h}
	autoOrder.AutoOrderRoutes(r)
}

func (h *AutoOrderHandler) AutoOrderRoutes(r *gin.RouterGroup) {
	autoOrder := r.Group("/auto-order")
	{
		autoOrder.POST("", h.Create)
		autoOrder.POST("/confirm", h.Confirm)
		// autoOrder.GET("/:id", h.Get)
		autoOrder.GET("/list", h.List)
		// autoOrder.PUT("/:id", h.Update)
		// autoOrder.DELETE("/:id", h.Delete)
	}
	autoOrderDetail := r.Group("auto-order-detail")
	{
		autoOrderDetail.GET("/list", h.AutoOrderDetailList)
		autoOrderDetail.PUT("/change-quantity/:id", h.ChangeAdjustedOrder)
	}
}

// CreateAutoOrder godoc
// @Summary Create auto order
// @Description Create auto order
// @Security     BearerAuth
// @Tags 	auto_orders
// @Accept 	json
// @Produce json
// @Param 	input body 	domain.AutoOrderRequest true "Auto order information"
// @Success 201 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /auto-order 	[post]
func (h *AutoOrderHandler) Create(c *gin.Context) {
	var (
		body domain.AutoOrderRequest
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	body.Id = uuid.New().String()
	body.Status = config.NEW

	err = tx.Table("auto_orders").Create(&body).Error
	if err != nil {
		tx.Rollback()
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	autoOrderDetails, err := h.storage.GenerateAutoOrderDetail(c.Request.Context(), body.StoreId, body.IntervalDay)
	if err != nil {
		tx.Rollback()
		h.log.Error(err)
		handleResponse(c, InternalError, "Failed to generate auto order for the store")
		return
	}

	for i := range autoOrderDetails {
		autoOrderDetails[i].AutoOrderId = body.Id
	}

	err = tx.
		Table("auto_order_details").
		Create(&autoOrderDetails).Error
	if err != nil {
		tx.Rollback()
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	if err = tx.Commit().Error; err != nil {
		tx.Rollback()
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, CREATED, "CREATED")
}

// ConfirmAutoOrder godoc
// @Summary Confirm auto order
// @Description Confirm auto order
// @Tags 	auto_orders
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	input body 	[]domain.AutoOrderConfirm true "Auto order information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /auto-order/confirm [post]
func (h *AutoOrderHandler) Confirm(c *gin.Context) {
	var (
		body          []domain.AutoOrderConfirm
		imports       []domain.ImportRequest
		importDetails []domain.ImportDetailRequest
		err           error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	if len(body) > 0 {
		for _, item := range body {
			importID := uuid.New().String()

			// Mahsulot narxini olish
			var product struct {
				RetailPrice  float64 `gorm:"retail_price" json:"retail_price"`
				MaterialCode int     `gorm:"material_code" json:"material_code"`
			}

			err := h.db.Raw(`select retail_price, material_code from products where id = ?`, item.ProductId).Scan(&product).Error
			if err != nil {
				h.log.Error(err)
				handleResponse(c, BadRequest, "Product not found or error occurred while fetching price")
				return
			}

			imports = append(imports, domain.ImportRequest{
				Id:             importID,
				StoreID:        item.StoreId,
				Status:         config.NEW_IMPORT,
				ImportDate:     time.Now().Format("2006-01-02 15:04:05"),
				DocumentNumber: utils.GenerateDocumentNumber(),
			})
			importDetails = append(importDetails, domain.ImportDetailRequest{
				ImportID:       importID,
				ProductID:      &item.ProductId,
				ReceivedCount:  int(item.AdjustedOrder),
				ReceivedAmount: product.RetailPrice * float64(item.AdjustedOrder),
			})
		}
	}
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	err = tx.Table("imports").Create(&imports).Error
	if err != nil {
		tx.Rollback()
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = tx.Table("import_details").Create(&importDetails).Error
	if err != nil {
		tx.Rollback()
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	if err = tx.Commit().Error; err != nil {
		tx.Rollback()
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, "CONFIRMED")
}

// ListAutoOrder godoc
// @Summary List auto orders
// @Description List auto orders
// @Tags 	auto_orders
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	limit query int false "Limit"
// @Param 	offset query int false "Offset"
// @Param 	store_id query string false "Store ID"
// @Param 	status query string false "Status"
// @Param 	auto_order_date query string false "Date"
// @Param 	search query string false "Search"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /auto-order/list [get]
func (h *AutoOrderHandler) List(c *gin.Context) {
	var (
		autoOrders []domain.AutoOrder
		err        error
		totalCount int64
		storeID    = c.Query("store_id")
		search     = c.Query("search")
		status     = c.Query("status")
		date       = c.Query("auto_order_date")
	)
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	query := h.db.
		Model(&domain.AutoOrder{}).
		Select(`auto_orders.*, 
		SUM(aod.adjusted_order_quantity) AS adjusted_order_quantity,
		SUM(aod.response_order_quantity) AS response_order_quantity`).
		Preload("Store").
		Joins("LEFT JOIN auto_order_details aod ON auto_orders.id = aod.auto_order_id").
		Joins("JOIN stores s ON auto_orders.store_id = s.id")

	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		query = query.Where("CAST(auto_orders.public_id AS TEXT) LIKE ? OR s.name ILIKE ?", search, search)
	}

	if storeID != "" {
		query = query.Where("auto_orders.store_id = ?", storeID)
	}

	if status != "" {
		query = query.Where("auto_orders.status = ?", status)
	}

	if date != "" {
		if _, err := time.Parse("2006-01-02", date); err != nil {
			handleResponse(c, BadRequest, "Invalid date format")
			return
		}
		query = query.Where("auto_orders.auto_order_date::date = ?", date)
	}

	err = query.
		Group("auto_orders.id").
		Offset(offset).Limit(limit).
		Find(&autoOrders).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	result := utils.ListResponse(autoOrders, totalCount, limit, offset)
	handleResponse(c, OK, result)
}

// AutoOrderDetailList godoc
// @Summary List auto order details
// @Description List auto order details
// @Tags auto_order_details
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param store_id query string false "Store ID"
// @Param auto_order_id query string false "Auto Order ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 401 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /auto-order-detail/list [get]
func (h *AutoOrderHandler) AutoOrderDetailList(c *gin.Context) {
	var (
		autoOrderDetails []*domain.AutoOrderDetail
		totalCount       int64
		autoOrderID      = c.Query("auto_order_id")
		storeID          = c.Query("store_id")
	)
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	query := h.db.
		Model(&domain.AutoOrderDetail{}).
		Preload("AutoOrder")
	if storeID != "" {
		query = query.Where("store_id = ?", storeID)
	}
	if autoOrderID != "" {
		query = query.Where("auto_order_id = ?", autoOrderID)
	}

	err = query.
		Count(&totalCount).
		Offset(offset).Limit(limit).
		Order("created_at DESC").
		Find(&autoOrderDetails).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	result := utils.ListResponse(autoOrderDetails, totalCount, limit, offset)
	handleResponse(c, OK, result)
}

// ChangeAdjustedOrder godoc
// @Summary Change adjusted order quantity
// @Description Change adjusted order quantity
// @Tags auto_order_details
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	id 	path string true "ID"
// @Param 	input body  domain.AdjustedOrderQuantity true "Adjusted Order Quantity"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /auto-order-detail/change-quantity/{id} [put]
func (h *AutoOrderHandler) ChangeAdjustedOrder(c *gin.Context) {
	var (
		body domain.AdjustedOrderQuantity
		err  error
		id   = c.Param("id")
	)
	if id == "" || id == "undefined" {
		handleResponse(c, BadRequest, "ID is required")
		return
	}
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.
		Model(&domain.AutoOrderDetail{}).
		Where("id = ?", id).
		Update("adjusted_order_quantity", body.AdjustedOrderQuantity).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	handleResponse(c, OK, "UPDATED")
}
