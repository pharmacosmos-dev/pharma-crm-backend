package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
)

type RoleHandler struct {
	*Handler
}

func (h *Handler) NewRoleHandler(r *gin.RouterGroup) {
	role := &RoleHandler{h}
	role.RoleRoutes(r)
}

func (h *RoleHandler) RoleRoutes(r *gin.RouterGroup) {
	role := r.Group("/role")
	{
		role.POST("", h.Create)
		role.GET("/:id", h.Get)
		role.GET("/list", h.List)
		role.PUT("/:id", h.Update)
		role.DELETE("/:id", h.Delete)
	}
}

// Create godoc
// @Summary Create a role
// @Description Create a role from the request body
// @Tags roles
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	role body domain.RoleRequest true "Role information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /role [post]
func (h *RoleHandler) Create(c *gin.Context) {
	var (
		body domain.RoleRequest
		err  error
	)
	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	body.Id = uuid.New().String()
	err = h.db.WithContext(c.Request.Context()).Model(&domain.Role{}).Create(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, body)
}

// Get godoc
// @Summary Get a role
// @Description Get a role from the request body
// @Tags roles
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "role ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /role/{id} [get]
func (h *RoleHandler) Get(c *gin.Context) {
	var res domain.Role
	if err := h.db.Model(&domain.Role{}).First(&res, "id = ?", c.Param("id")).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}

// List godoc
// @Summary Get a role
// @Description Get a role from the request body
// @Tags roles
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limmit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /role/list [get]
func (h *RoleHandler) List(c *gin.Context) {
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	res := []*domain.Role{}
	if err := h.db.Limit(limit).Offset(offset).Find(&res).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}

// Update godoc
// @Summary Update a role
// @Description Update a role from the request body
// @Tags roles
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "role ID"
// @Param role body domain.RoleRequest true "Role information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /role/{id} [put]
func (h *RoleHandler) Update(c *gin.Context) {
	var (
		body domain.RoleRequest
		err  error
	)
	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.WithContext(c.Request.Context()).
		Model(&domain.Role{}).
		Where("id = ?", c.Param("id")).
		Updates(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}

// Delete godoc
// @Summary Delete a role
// @Description Delete a role from the request body
// @Tags roles
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "role ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /role/{id} [delete]
func (h *RoleHandler) Delete(c *gin.Context) {
	if err := h.db.WithContext(c.Request.Context()).Delete(&domain.Role{}, "id = ?", c.Param("id")).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}
