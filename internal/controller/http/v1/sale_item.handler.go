package v1

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
)

type SaleItemHander struct {
	*Handler
}

func (h *Handler) NewSaleItemHandler(r *gin.RouterGroup) {
	saleItem := &SaleItemHander{h}
	saleItem.SaleItemRoutes(r)
}

func (h *SaleItemHander) SaleItemRoutes(r *gin.RouterGroup) {
	saleItem := r.Group("/sale_item")
	{
		saleItem.POST("", h.Create)
		saleItem.POST("/multiple", h.MultipleCreate)
		saleItem.GET("/:id", h.Get)
		saleItem.GET("/list", h.List)
		saleItem.PUT("/:id", h.Update)
		saleItem.DELETE("/:id", h.Delete)
	}
}

// Create godoc
// @Summary Create a sale item
// @Description Create a sale item from the request body
// @Tags sale_items
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale_item [post]
func (h *SaleItemHander) Create(c *gin.Context) {
	var (
		body domain.SaleItemRequest
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	if err = h.db.WithContext(c.Request.Context()).
		Table("sale_items").Create(&body).Error; err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}

// MultipleCreate godoc
// @Summary Create a sale item
// @Description Create a sale item from the request body
// @Tags sale_items
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.SaleID true "Sale ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale_item/multiple [post]
func (h *SaleItemHander) MultipleCreate(c *gin.Context) {
	var (
		body domain.SaleID
		res  []domain.CartItemRequest
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	if err = h.db.Where("sale_id = ?", body.SaleID).Table("cart_items").Find(&res).Error; err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	if len(res) > 0 {
		if err = h.db.Table("sale_items").Create(&res).Error; err != nil {
			h.log.Error(fmt.Errorf("err: %v", err))
			handleResponse(c, InternalError, err.Error())
			return
		}
	}
	handleResponse(c, CREATED, nil)
}

// Get godoc
// @Summary Get a sale item
// @Description Get a sale item from the request body
// @Tags sale_items
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "sale item ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale_item/{id} [get]
func (h *SaleItemHander) Get(c *gin.Context) {
	var (
		body domain.SaleItem
		err  error
	)
	if err = h.db.First(&body, "id = ?", c.Param("id")).Error; err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}

// List godoc
// @Summary Get a sale item
// @Description Get a sale item from the request body
// @Tags sale_items
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limmit query int false "Limit"
// @Param offset query int false "Offset"
// @Param sale_id query string false "Sale ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale_item/list [get]
func (h *SaleItemHander) List(c *gin.Context) {
	var (
		body []domain.SaleItem
		err  error
	)
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}

	query := h.db.Limit(limit).Offset(offset)

	if saleID := c.Query("sale_id"); saleID != "" {
		query = query.Where("sale_id = ?", saleID)
	}
	if err = query.Find(&body).Error; err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}

// Update godoc
// @Summary Update a sale item
// @Description Update a sale item from the request body
// @Tags sale_items
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "sale item ID"
// @Param input body domain.SaleItemRequest true "Sale item information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale_item/{id} [put]
func (h *SaleItemHander) Update(c *gin.Context) {
	var (
		body domain.SaleItemRequest
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	if err = h.db.WithContext(c.Request.Context()).
		Where("id = ?", c.Param("id")).
		Table("sale_items").Updates(&body).Error; err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}

// Delete godoc
// @Summary Delete a sale item
// @Description Delete a sale item from the request body
// @Tags sale_items
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "sale item ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale_item/{id} [delete]
func (h *SaleItemHander) Delete(c *gin.Context) {
	if err := h.db.Delete(&domain.SaleItem{}, "id = ?", c.Param("id")).Error; err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, nil)
}
