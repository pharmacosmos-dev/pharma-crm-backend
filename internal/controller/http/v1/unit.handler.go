package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"gorm.io/gorm"
)

type UnitHandler struct {
	*Handler
}

func (h *Handler) NewUnitHandler(r *gin.RouterGroup) {
	unit := &UnitHandler{h}
	unit.UnitRoutes(r)
}

func (h *UnitHandler) UnitRoutes(r *gin.RouterGroup) {
	unit := r.Group("/unit")
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
// @Description Create a unit from the request body
// @Tags units
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param unit body domain.Unit true "Unit information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /unit [post]
func (h *UnitHandler) Create(c *gin.Context) {
	var (
		body domain.Unit
		res  domain.Unit
		err  error
	)
	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	body.Id = uuid.New().String()
	if err := h.db.WithContext(c.Request.Context()).Model(&res).Create(&body).Scan(&res).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, res)
}

// Get godoc
// @Summary Get a unit
// @Description Get a unit from the request body
// @Tags units
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "unit ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /unit/{id} [get]
func (h *UnitHandler) Get(c *gin.Context) {
	var res domain.Unit
	if err := h.db.First(&res, "id = ?", c.Param("id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, nil)
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
// @Router /unit/list [get]
func (h *UnitHandler) List(c *gin.Context) {
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	res := []*domain.Unit{}
	err = h.db.Limit(limit).Offset(offset).Find(&res).Error
	if err != nil {
		h.log.Error(err)
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
// @Param unit body domain.Unit true "Unit information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /unit/{id} [put]
func (h *UnitHandler) Update(c *gin.Context) {
	var (
		body domain.Unit
		res  domain.Unit
	)
	if err := h.db.WithContext(c.Request.Context()).Model(&res).Where("id = ?", c.Param("id")).Updates(&body).Error; err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
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
// @Router /unit/{id} [delete]
func (h *UnitHandler) Delete(c *gin.Context) {
	if err := h.db.WithContext(c.Request.Context()).Delete(&domain.Unit{}, "id = ?", c.Param("id")).Error; err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}
