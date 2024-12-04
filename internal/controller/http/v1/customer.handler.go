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
		customer.DELETE("/:id", h.Delete)
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
		res  domain.Customer
		err  error
	)
	err = c.ShouldBindJSON(&body)
	if err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	body.Id = uuid.New().String()
	err = h.db.WithContext(c.Request.Context()).
		Table("customers").
		Create(&body).Scan(&res).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, res)
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
// @Param limmit query int false "Limit"
// @Param offset query int false "Offset"
// @Param search query string false "Search"
// @Param store_id query string false "Store ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /customer/list [get]
func (h *CustomerHandler) List(c *gin.Context) {
	var totalAmount int64
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	res := []*domain.Customer{}
	search := fmt.Sprintf("%%%s%%", c.Query("search"))

	// Start building the query
	query := h.db.Model(&domain.Customer{}).
		Limit(limit).
		Offset(offset).
		Count(&totalAmount).
		Order("created_at DESC")

	if search != "" {
		query = query.Where("first_name ILIKE ? OR last_name ILIKE ?", search, search)
	}
	if storeID := c.Query("store_id"); storeID != "" {
		query = query.Where("store_id = ?", storeID)
	}

	if err = query.Find(&res).Error; err != nil {
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
	var err error
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.WithContext(c.Request.Context()).
		Table("customers").
		Where("id = ?", c.Param("id")).
		Updates(&body).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}

// Delete godoc
// @Summary Delete a customer
// @Description Delete a customer from the request body
// @Tags customers
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "customer ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /customer/{id} [delete]
func (h *CustomerHandler) Delete(c *gin.Context) {
	if err := h.db.WithContext(c.Request.Context()).Delete(&domain.Customer{}, "id = ?", c.Param("id")).Error; err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}
