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
		permission.DELETE("/delete", h.Delete)
		permission.GET("/role/:role_id", h.GetPermissionsByRoleID)
		permission.GET("/list-parents", h.ListParents)
		permission.GET("/filter", h.ListPermissions)
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
	if body.ParentId == nil && body.Key == "" {
		body.Key = body.Route
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
		h.log.Error(err)
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
		roleID = c.Query("role_id")
	)

	// Base query for parent permissions
	query := h.db.Table("permissions").
		Where("permissions.parent_id IS NULL").
		Preload("Permissions", func(db *gorm.DB) *gorm.DB {
			// Preload Permissions (children of MainPermission)
			if roleID != "" {
				return db.Select(`
						permissions.*, 
						COALESCE(role_permissions.is_active, false) AS is_active
					`).
					Joins(`
						LEFT JOIN role_permissions 
						ON role_permissions.permission_id = permissions.id 
						AND role_permissions.role_id = ?
					`, roleID).
					Preload("Children", func(childDB *gorm.DB) *gorm.DB {
						// Preload Children of Permissions
						return childDB.Select(`
								permissions.*, 
								COALESCE(role_permissions.is_active, false) AS is_active
							`).
							Joins(`
								LEFT JOIN role_permissions 
								ON role_permissions.permission_id = permissions.id 
								AND role_permissions.role_id = ?
							`, roleID)
					})
			}
			// Default Preload without role_id
			return db.Preload("Children")
		})

	// Execute the query
	err := query.Debug().Find(&res).Error

	if err != nil {
		h.log.Error(err)
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
// @Accept 	json
// @Produce json
// @Param 	ids body []string true "Permission IDs"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /permission/delete [delete]
func (h *PermissionHandler) Delete(c *gin.Context) {
	var ids []string
	// bind the request body
	err := c.ShouldBindJSON(&ids)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// delete the permissions
	err = h.db.Delete(&domain.Permission{}, "id IN (?)", ids).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
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
	err := h.db.Raw(`
	WITH parent_permissions AS (
		SELECT 
			id, 
			CAST(name || ' MODULE' AS VARCHAR) AS name, 
			route, 
			type, 
			parent_id, 
			description, 
			key, 
			method, 
			created_at, 
			updated_at
		FROM permissions
		WHERE type = 'MODULE'
	)
	SELECT id, name, route, type, parent_id, description, key, method, created_at, updated_at
	FROM parent_permissions
	UNION ALL
	SELECT id, name, route, type, parent_id, description, key, method, created_at, updated_at
	FROM permissions
	WHERE parent_id IN (SELECT id FROM parent_permissions);
	`).Scan(&res).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}

// ListPermissions doc
// @Summary List Permission
// @Description List Permission
// @Tags Permission
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param 	search query string false "Search"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /permission/filter [GET]
func (h *PermissionHandler) ListPermissions(c *gin.Context) {
	var (
		allPermissions []domain.Permission
		search         = c.Query("search")
	)
	// build query
	query := `SELECT id, route, type, name, description, parent_id, method, created_at, updated_at FROM permissions `
	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		query += fmt.Sprintf(" WHERE name ILIKE '%s' OR description ILIKE '%s'", search, search)
	}
	// Get all permissions
	err := h.db.Raw(query).Scan(&allPermissions).Error

	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	// 2. Rekursiv ravishda daraxt tuzilishini yaratish
	permissionTree := buildPermissionTree(allPermissions, "")

	c.JSON(200, permissionTree)
}

// buildPermissionTree doc
func buildPermissionTree(all []domain.Permission, parentID string) []domain.Permission {
	var tree []domain.Permission
	for _, perm := range all {
		if perm.ParentId == parentID {
			// Rekursiv chaqirib, `children` larni qo‘shish
			perm.Children = buildPermissionTree(all, perm.Id)
			tree = append(tree, perm)
		}
	}
	return tree
}
