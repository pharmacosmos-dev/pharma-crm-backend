package v1

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

type CustomerHandler struct {
	*Handler
}

func (h *Handler) NewCustomerHandler(r *gin.RouterGroup) {
	customer := &CustomerHandler{h}
	customer.CustomerRoutes(r)
}

func (h *CustomerHandler) CustomerRoutes(r *gin.RouterGroup) {
	customer := r.Group("/customer")
	{
		customer.POST("", h.Create)
		customer.GET("/:id", h.Get)
		customer.GET("/list", h.List)
		customer.PUT("/:id", h.Update)
		customer.DELETE("/soft-delete", h.SoftDelete)
		customer.DELETE("/hard-delete", h.HardDelete)
	}
}

// Create godoc
// @Summary Create a customer
// @Description Create a customer from the request body
// @Tags customers
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param customer body domain.CustomerRequest true "Customer information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /customer [post]
func (h *CustomerHandler) Create(c *gin.Context) {
	var (
		body domain.CustomerRequest
		err  error
	)
	err = c.ShouldBindJSON(&body)
	if err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	createdBy, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}

	body.Id = uuid.New().String()
	body.PublicId = utils.GenerateRandomCode()
	body.CreatedBy = cast.ToString(createdBy)
	err = h.db.WithContext(c.Request.Context()).
		Table("customers").
		Create(&body).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, body)
}

// Get godoc
// @Summary Get a customer
// @Description Get a customer from the request body
// @Tags customers
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "customer ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /customer/{id} [get]
func (h *CustomerHandler) Get(c *gin.Context) {
	var res domain.Customer
	if err := h.db.First(&res, "id = ?", c.Param("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleResponse(c, NotFound, nil)
			return
		}
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}

// List godoc
// @Summary Get a customer
// @Description Get a customer from the request body
// @Tags customers
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param search query string false "Search"
// @Param store_id query string false "Store ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /customer/list [get]
func (h *CustomerHandler) List(c *gin.Context) {
	var totalAmount int64
	var search = c.Query("search")
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	res := []*domain.Customer{}

	// Start building the query
	query := h.db.
		Model(&domain.Customer{}).
		Preload("Store").
		Select("customers.*, sales.created_at AS sale_date, sales.total_amount AS sale_amount").
		Joins("LEFT JOIN sales ON sales.customer_id = customers.id")

	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		query = query.Where("first_name ILIKE ? OR last_name ILIKE ? OR CAST(public_id AS TEXT) ILIKE ? OR ? = ANY(phone)",
			search, search, search, strings.Trim(search, "%"))
	}
	if storeID := c.Query("store_id"); storeID != "" {
		query = query.Where("store_id = ?", storeID)
	}
	err = query.
		Count(&totalAmount).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&res).Error
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	result := utils.ListResponse(res, totalAmount, limit, offset)

	handleResponse(c, OK, result)
}

// Update godoc
// @Summary Update a customer
// @Description Update a customer from the request body
// @Tags customers
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "customer ID"
// @Param customer body domain.CustomerRequest true "Customer information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /customer/{id} [put]
func (h *CustomerHandler) Update(c *gin.Context) {
	var body domain.CustomerRequest
	var id = c.Param("id")
	var err error
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.WithContext(c.Request.Context()).
		Table("customers").
		Where("id = ?", id).
		Updates(&body).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err.Error()))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}

// SoftDelete godoc
// @Summary Soft delete a customer
// @Description Soft delete a customer from the request body
// @Tags customers
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param ids body []string true "customer IDs"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /customer/soft-delete [delete]
func (h *CustomerHandler) SoftDelete(c *gin.Context) {
	var ids []string
	err := c.ShouldBindJSON(&ids)
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err.Error()))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.
		WithContext(c.Request.Context()).
		Where("id IN (?)", ids).
		Updates(map[string]interface{}{
			"is_active": false,
			"status":    0}).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}

// HardDelete godoc
// @Summary Hard delete a customer
// @Description Hard delete a customer from the request body
// @Tags customers
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param ids body []string true "customer IDs"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /customer/hard-delete [delete]
func (h *CustomerHandler) HardDelete(c *gin.Context) {
	var ids []string
	err := c.ShouldBindJSON(&ids)
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err.Error()))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.
		WithContext(c.Request.Context()).
		Where("id IN (?)", ids).
		Delete(&domain.Customer{}).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err.Error()))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}
