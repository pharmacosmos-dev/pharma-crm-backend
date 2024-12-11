package v1

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
	"gorm.io/gorm"
)

type SaleHandler struct {
	*Handler
}

func (h *Handler) NewSaleHandler(r *gin.RouterGroup) {
	sale := &SaleHandler{h}
	sale.SaleRoutes(r)
}

func (h *SaleHandler) SaleRoutes(r *gin.RouterGroup) {
	sale := r.Group("/sale")
	{
		sale.POST("", h.Create)
		sale.GET("/:id", h.Get)
		sale.GET("/list", h.List)
		sale.PUT("/:id", h.Update)
		sale.DELETE("/:id", h.Delete)
		sale.POST("/final", h.FinalSale)
		sale.GET("/check", h.CheckSale)
	}
}

// Create godoc
// @Summary Create a sale
// @Description Create a sale from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.SaleRequest true "Sale information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale [post]
func (h *SaleHandler) Create(c *gin.Context) {
	var (
		body domain.SaleRequest
		res  domain.Sale
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	body.ID = uuid.New().String()
	body.SaleNumber = utils.GenerateCode()
	err = h.db.WithContext(c.Request.Context()).
		Table("sales").
		Create(&body).Scan(&res).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, CREATED, res)
}

// Get godoc
// @Summary Get a sale
// @Description Get a sale from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "sale ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/{id} [get]
func (h *SaleHandler) Get(c *gin.Context) {
	var res domain.Sale
	if err := h.db.First(&res, "id = ?", c.Param("id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, OK, nil)
			return
		}
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}

// List godoc
// @Summary Get a sale
// @Description Get a sale from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limmit query int false "Limit"
// @Param offset query int false "Offset"
// @Param employee_id query string false "Employee ID"
// @Param cash_box_id query string false "Cash Box ID"
// @Param start_date query string false "Start Date"
// @Param end_date query string false "End Date"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/list [get]
func (h *SaleHandler) List(c *gin.Context) {
	var totalAmount int64
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	res := []domain.Sale{}
	query := h.db.Model(&domain.Sale{})
	if employeeID := c.Query("employee_id"); employeeID != "" {
		query = query.Where("employee_id = ?", employeeID)
	}
	if cashBoxID := c.Query("cash_box_id"); cashBoxID != "" {
		query = query.Where("cash_box_id = ?", cashBoxID)
	}
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	if startDate != "" && endDate != "" {
		query = query.Where("created_at BETWEEN ? AND ?", startDate, endDate)
	}
	err = query.Where("status = ?", "completed").
		Count(&totalAmount).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&res).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	// Group sales by date using a slice
	type GroupedSales struct {
		Date  string        `json:"date"`
		Sales []domain.Sale `json:"sales"`
	}
	groupedData := []GroupedSales{}

	groupedSalesMap := make(map[string]*GroupedSales)
	for _, sale := range res {
		date := sale.CreatedAt.Format("2006-01-02") // Format date as YYYY-MM-DD
		if group, exists := groupedSalesMap[date]; exists {
			group.Sales = append(group.Sales, sale)
		} else {
			newGroup := &GroupedSales{
				Date:  date,
				Sales: []domain.Sale{sale},
			}
			groupedSalesMap[date] = newGroup
			groupedData = append(groupedData, *newGroup)
		}
	}

	data := utils.ListResponse(groupedData, totalAmount, limit, offset)
	handleResponse(c, OK, data)
}

// Update godoc
// @Summary Update a sale
// @Description Update a sale from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "sale ID"
// @Param input body domain.SaleUpdateRequest true "Sale information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/{id} [put]
func (h *SaleHandler) Update(c *gin.Context) {
	var (
		body domain.SaleUpdateRequest
		res  []domain.CartItemRequest
		id   = c.Param("id")
	)
	err := c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.WithContext(c.Request.Context()).
		Table("sales").
		Where("id = ?", id).
		Updates(&body).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}

	err = h.db.Where("sale_id = ?", id).
		Table("cart_items").Find(&res).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	if len(res) > 0 {
		err = h.db.Table("sale_items").Create(&res).Error
		if err != nil {
			h.log.Error(fmt.Errorf("err: %v", err))
			handleResponse(c, InternalError, err.Error())
			return
		}
		for _, item := range res {
			err = h.db.Table("products").Where("id = ?", item.ProductID).
				UpdateColumn("quantity", gorm.Expr("quantity - ?", item.Quantity)).Error
			if err != nil {
				h.log.Error(fmt.Errorf("err: %v", err))
				handleResponse(c, InternalError, err.Error())
				return
			}
		}

	}
	err = h.db.Delete(&domain.CartItem{}, "sale_id = ?", id).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}

// Delete godoc
// @Summary Delete a sale
// @Description Delete a sale from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "sale ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/{id} [delete]
func (h *SaleHandler) Delete(c *gin.Context) {
	var id = c.Param("id")
	err := h.db.Delete(&domain.Sale{}, "id = ?", id).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, nil)
}

// FinalSale
// @Summary Final Sale
// @Description Final Sale from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.FinalSale true "Sale information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/final [post]
func (h *SaleHandler) FinalSale(c *gin.Context) {
	var (
		body         domain.FinalSale
		res          []domain.CartItemRequest
		productCount int
	)
	err := c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}

	err = h.db.Where("sale_id = ?", body.SaleID).
		Table("cart_items").Find(&res).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	if len(res) > 0 {
		err = h.db.Table("sale_items").Create(&res).Error
		if err != nil {
			h.log.Error(fmt.Errorf("err: %v", err))
			handleResponse(c, InternalError, err.Error())
			return
		}
		for _, item := range res {
			productCount += item.Quantity
			err = h.db.Table("products").Where("id = ?", item.ProductID).
				UpdateColumn("quantity", gorm.Expr("quantity - ?", item.Quantity)).Error
			if err != nil {
				h.log.Error(fmt.Errorf("err: %v", err))
				handleResponse(c, InternalError, err.Error())
				return
			}
		}

	}
	err = h.db.WithContext(c.Request.Context()).
		Table("sales").
		Where("id = ?", body.SaleID).
		Updates(map[string]interface{}{
			"total_amount":  body.TotalAmount,
			"product_count": productCount,
			"status":        "completed"}).Error

	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}

	err = h.db.Delete(&domain.CartItem{}, "sale_id = ?", body.SaleID).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, body)
}

// CheckSale
// @Summary Check Sale
// @Description Check Sale from the request body
// @Tags sales
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /sale/check [get]
func (h *SaleHandler) CheckSale(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	var res domain.Sale
	err := h.db.First(&res, "employee_id = ? AND status = ?", userID, "pending").Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleResponse(c, OK, nil)
			return
		}
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}
