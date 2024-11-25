package v1

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
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
	var body domain.CustomerRequest
	var res domain.Customer
	if err := c.ShouldBindJSON(&body); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	body.Id = uuid.New().String()
	if err := h.db.WithContext(c.Request.Context()).Create(&body).Scan(&res).Error; err != nil {
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
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, nil)
			return
		}
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
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /customer/list [get]
func (h *CustomerHandler) List(c *gin.Context) {
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	res := []*domain.Customer{}
	search := fmt.Sprintf("%%%s%%", c.Query("search"))
	if err := h.db.Limit(limit).Offset(offset).Where("name ILIKE ?", search).Find(&res).Error; err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}

// Update godoc
// @Summary Update a customer
// @Description Update a customer from the request body
// @Tags customers
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "customer ID"
// @Param customer body domain.CategoryRequest true "Customer information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /customer/{id} [put]
func (h *CustomerHandler) Update(c *gin.Context) {
	var body domain.Customer
	if err := h.db.WithContext(c.Request.Context()).Model(&body).Where("id = ?", c.Param("id")).Updates(&body).Error; err != nil {
		h.log.Error(err)
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
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}
