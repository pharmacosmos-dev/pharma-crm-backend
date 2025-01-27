package v1

import (
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
		autoOrder.POST("/confirm", h.Confirm)
		// autoOrder.GET("/:id", h.Get)
		autoOrder.GET("/list", h.List)
		// autoOrder.PUT("/:id", h.Update)
		// autoOrder.DELETE("/:id", h.Delete)
	}
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
				ImportID:            importID,
				ProductID:           &item.ProductId,
				ReceivedCount:       int(item.AdjustedOrder),
				ReceivedAmount:      product.RetailPrice * float64(item.AdjustedOrder),
				ProductMaterialCode: product.MaterialCode,
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
	)
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	autoOrders, totalCount, err = h.storage.ListAutoOrder(c.Request.Context(), limit, offset, storeID, search)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
	}
	result := utils.ListResponse(autoOrders, totalCount, limit, offset)
	handleResponse(c, OK, result)
}
