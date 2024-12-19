package v1

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"gorm.io/gorm"
)

type UnitTypeHandler struct {
	*Handler
}

func (h *Handler) NewUnitHandler(r *gin.RouterGroup) {
	unit := &UnitTypeHandler{h}
	unit.UnitRoutes(r)
}

func (h *UnitTypeHandler) UnitRoutes(r *gin.RouterGroup) {
	unit := r.Group("/unit-types")
	{
		unit.POST("", h.Create)
		unit.GET("/:id", h.Get)
		unit.GET("/list", h.List)
		unit.PUT("/:id", h.Update)
		unit.DELETE("/:id", h.Delete)
	}
}

// Create godoc
// @Summary Create a unit
// @Description Create a unit types from the request body
// @Tags units
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param unit body domain.UnitTypeRequest true "Unit information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /unit-types [post]
func (h *UnitTypeHandler) Create(c *gin.Context) {
	var (
		body domain.UnitTypeRequest
		err  error
	)
	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	body.Id = uuid.New().String()
	err = h.db.
		WithContext(c.Request.Context()).
		Table("unit_types").
		Create(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, body)
}

// Get godoc
// @Summary Get a unit types
// @Description Get a unit from the request body
// @Tags units
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "unit ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /unit-types/{id} [get]
func (h *UnitTypeHandler) Get(c *gin.Context) {
	var res domain.UnitType
	err := h.db.First(&res, "id = ?", c.Param("id")).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleResponse(c, NotFound, "Unit not found")
			return
		}
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}

// List godoc
// @Summary Get a unit
// @Description Get a unit from the request body
// @Tags units
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /unit-types/list [get]
func (h *UnitTypeHandler) List(c *gin.Context) {
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	res := []*domain.UnitType{}
	err = h.db.
		Limit(limit).
		Offset(offset).
		Find(&res).Error
	if err != nil {
		h.log.Error(fmt.Errorf("error fetching unit types: %w", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}

// Update godoc
// @Summary Update a unit
// @Description Update a unit from the request body
// @Tags units
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "unit ID"
// @Param unit body domain.UnitTypeRequest true "Unit information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /unit-types/{id} [put]
func (h *UnitTypeHandler) Update(c *gin.Context) {
	var (
		body domain.UnitTypeRequest
	)
	id := c.Param("id")
	err := h.db.
		WithContext(c.Request.Context()).
		Table("unit_types").
		Where("id = ?", id).
		Updates(&body).Error
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}

// Delete godoc
// @Summary Delete a unit
// @Description Delete a unit from the request body
// @Tags units
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "unit ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /unit-types/{id} [delete]
func (h *UnitTypeHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	err := h.db.
		WithContext(c.Request.Context()).
		Delete(&domain.UnitType{}, "id = ?", id).Error
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}
