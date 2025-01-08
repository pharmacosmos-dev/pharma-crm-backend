package v1

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
	"gorm.io/gorm"
)

type PermissionHandler struct {
	*Handler
}

func (h *Handler) NewPermissionHandler(r *gin.RouterGroup) {
	permission := &PermissionHandler{h}
	permission.PermissionRoutes(r)
}

func (h *PermissionHandler) PermissionRoutes(r *gin.RouterGroup) {
	permission := r.Group("/permission")
	{
		permission.POST("", h.Create)
		permission.GET("/:id", h.Get)
		permission.GET("/list", h.List)
		permission.PUT("/:id", h.Update)
		permission.DELETE("/:id", h.Delete)
		permission.GET("/role/:role_id", h.GetPermissionsByRoleID)
		permission.GET("/list-parents", h.ListParents)
	}
}

// Create doc
// @Summary Create Permission
// @Description Create Permission
// @Tags Permission
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.PermissionRequest true "Permission information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /permission [post]
func (h *PermissionHandler) Create(c *gin.Context) {
	var body domain.PermissionRequest
	err := c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	// Generate a new UUID for the record
	body.Id = uuid.New().String()
	body.Method = utils.StringArray(body.Method)
	if body.ParentId == nil {
		body.Key = body.EntityName
	}
	err = h.db.WithContext(c.Request.Context()).
		Table("permissions").
		Create(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, CREATED, body)
}

// Get doc
// @Summary Get Permission
// @Description Get Permission
// @Tags Permission
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Permission ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /permission/{id} [get]
func (h *PermissionHandler) Get(c *gin.Context) {
	var res domain.Permission
	var id = c.Param("id")
	err := h.db.First(&res, "id = ?", id).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}

// List doc
// @Summary List Permission
// @Description List Permission
// @Tags Permission
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param role_id query string false "Role ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /permission/list [get]
func (h *PermissionHandler) List(c *gin.Context) {
	var (
		res    []domain.MainPermission
		roleID = c.Param("role_id")
	)

	// Start the query
	query := h.db.Table("permissions").
		Where("parent_id IS NULL").
		Preload("Permissions.Children")
	// Conditionally add role filtering if role_id is provided
	if roleID != "" {
		query = query.Preload("Permissions", func(db *gorm.DB) *gorm.DB {
			return db.Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id AND role_permissions.role_id = ?", roleID)
		}).Select("permissions.*, COALESCE(role_permissions.is_active, false) AS is_active")
	} else {
		query = query.Preload("Permissions")
	}

	// Execute the query
	err := query.Find(&res).Error

	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}

// Update doc
// @Summary Update Permission
// @Description Update Permission
// @Tags Permission
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Permission ID"
// @Param input body domain.PermissionRequest true "Permission information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /permission/{id} [put]
func (h *PermissionHandler) Update(c *gin.Context) {
	var body domain.PermissionRequest
	err := c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.WithContext(c.Request.Context()).
		Table("permissions").
		Where("id = ?", c.Param("id")).
		Updates(&body).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}

// Delete doc
// @Summary Delete Permission
// @Description Delete Permission
// @Tags Permission
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Permission ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /permission/{id} [delete]
func (h *PermissionHandler) Delete(c *gin.Context) {
	var id = c.Param("id")
	err := h.db.Delete(&domain.Permission{}, "id = ?", id).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, nil)
}

// GetPermissionsByRoleID doc
// @Summary Get Permissions by Role ID
// @Description Get Permissions by Role ID
// @Tags Permission
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param role_id path string true "Role ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /permission/role/{role_id} [get]
func (h *PermissionHandler) GetPermissionsByRoleID(c *gin.Context) {
	var (
		res    []domain.Permission
		roleID = c.Param("role_id")
	)

	err := h.db.
		Table("permissions").
		Select("permissions.*, role_permissions.is_active as is_active").
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ?", roleID).Find(&res).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}

// ListParents doc
// @Summary List Permission
// @Description List Permission
// @Tags Permission
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /permission/list-parents [GET]
func (h *PermissionHandler) ListParents(c *gin.Context) {
	var res []domain.Permission
	err := h.db.Find(&res, "parent_id IS NULL").Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}
