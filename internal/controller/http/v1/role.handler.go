package v1

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
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
		role.DELETE("/multiple/delete", h.MultipleDelete)
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
// @Success 201 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /role [post]
func (h *RoleHandler) Create(c *gin.Context) {
	var (
		body            domain.RoleRequest
		rolePermissions []domain.RolePermission
		err             error
	)
	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error("ERROR on binding json: ", err.Error())
		handleResponse(c, BadRequest, err.Error())
		return
	}

	body.Id = uuid.New().String()
	body.PublicId = utils.GenerateRandomCode()

	err = h.db.
		WithContext(c.Request.Context()).
		Table("roles").
		Create(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	for _, i := range body.Permissions {
		rolePermissions = append(rolePermissions, domain.RolePermission{
			ID:           uuid.New().String(),
			RoleID:       body.Id,
			PermissionID: i.PermissionId,
			IsActive:     i.IsActive,
			CreatedAt:    nil,
			UpdatedAt:    nil,
		})
		if len(i.ChildIds) > 0 {
			for _, j := range i.ChildIds {
				rolePermissions = append(rolePermissions, domain.RolePermission{
					ID:           uuid.New().String(),
					RoleID:       body.Id,
					PermissionID: j,
					IsActive:     i.IsActive,
					CreatedAt:    nil,
					UpdatedAt:    nil,
				})
			}
		}
	}
	err = h.db.
		WithContext(c.Request.Context()).
		Create(&rolePermissions).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, "CREATED")
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
	roleID := c.Param("id")
	var role domain.Role
	err := h.db.First(&role, "id = ?", roleID).Error
	if err != nil {
		h.log.Error(err.Error())
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, role)
}

// List godoc
// @Summary Get a role
// @Description Get a role from the request body
// @Tags roles
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param search query string false "Search"
// @Param status query string false "Status (1 -> active, 0 -> inactive 2 -> deleted)"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /role/list [get]
func (h *RoleHandler) List(c *gin.Context) {
	status := c.Query("status")
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	res := []*domain.Role{}
	var totalCount int64
	if status == "" {
		status = "1"
	}
	q := h.db.Model(&domain.Role{}).Where("status = ?", status)
	if search := c.Query("search"); search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		q = q.Where("name ILIKE ? OR description ILIKE ?", search, search)
	}

	err = q.
		Count(&totalCount).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&res).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	data := utils.ListResponse(res, totalCount, limit, offset)
	handleResponse(c, OK, data)
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
	var id = c.Param("id")
	err := h.db.
		WithContext(c.Request.Context()).
		Delete(&domain.Role{}, "id = ?", id).Error
	if err != nil {
		h.log.Error(err.Error())
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}

// MultipleDelete godoc
// @Summary Delete all roles
// @Description Delete all roles from the request body
// @Tags roles
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	body body    []string  true "role IDs"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /role/multiple/delete [delete]
func (h *RoleHandler) MultipleDelete(c *gin.Context) {
	var (
		ids []string
		err error
	)
	if err = c.ShouldBindJSON(&ids); err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.Table("roles").Where("id IN (?)", ids).Updates(map[string]interface{}{"status": 2}).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")

}
