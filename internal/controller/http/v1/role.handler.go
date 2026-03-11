package v1

import (
	"encoding/json"
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
		role.GET("/list-with-permissions", h.ListRoleWithPermissions)
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

	err = h.db.
		WithContext(c.Request.Context()).
		Table("roles").
		Create(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	if len(body.Permissions) > 0 {
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
						IsActive:     true,
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
	var (
		search = c.Query("search")
		status = c.Query("status")
	)
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
	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		q = q.Where("CAST(public_id AS TEXT) ILIKE ? OR name ILIKE ? OR description ILIKE ?", search, search, search)
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
// @Param role body domain.RoleUpdateRequest true "Role information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /role/{id} [put]
func (h *RoleHandler) Update(c *gin.Context) {
	var (
		body            domain.RoleUpdateRequest
		rolePermissions []domain.RolePermission
		id              = c.Param("id")
		err             error
	)
	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.WithContext(c.Request.Context()).
		Table("roles").
		Where("id = ?", id).
		Updates(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	err = h.db.
		WithContext(c.Request.Context()).
		Delete(&domain.RolePermission{}, "role_id = ?", id).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	if len(body.Permissions) > 0 {
		for _, perm := range body.Permissions {
			rolePermissions = append(rolePermissions, domain.RolePermission{
				PermissionID: perm.PermissionId,
				RoleID:       id,
				IsActive:     perm.IsActive,
			})
			if len(perm.ChildIds) > 0 {
				for _, j := range perm.ChildIds {
					rolePermissions = append(rolePermissions, domain.RolePermission{
						RoleID:       id,
						PermissionID: j,
						IsActive:     true,
					})
				}
			}
		}

		err = h.db.
			WithContext(c.Request.Context()).
			Table("role_permissions").
			Create(&rolePermissions).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
	}

	handleResponse(c, OK, "UPDATED")
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
	err := h.db.WithContext(c.Request.Context()).
		Delete(&domain.RolePermission{}, "role_id = ?", id).Error
	if err != nil {
		h.log.Error(err.Error())
		handleResponse(c, InternalError, err.Error())
		return
	}
	err = h.db.
		WithContext(c.Request.Context()).
		Delete(&domain.Role{}, "id = ?", id).Error
	if err != nil {
		h.log.Error(err.Error())
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}

// ListRoleWithPermissions godoc
// @Summary List all permissions with active roles
// @Description Returns full permission tree, each permission contains list of role IDs that have it active
// @Tags roles
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /role/list-with-permissions [get]
func (h *RoleHandler) ListRoleWithPermissions(c *gin.Context) {
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	sqlDB, err := h.db.DB()
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	var total int64
	if err = sqlDB.QueryRowContext(c.Request.Context(),
		`SELECT COUNT(*) FROM permissions WHERE deleted_at IS NULL AND parent_id IS NULL`,
	).Scan(&total); err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	const query = `
		WITH RECURSIVE top_level AS (
			SELECT id
			FROM permissions
			WHERE deleted_at IS NULL AND parent_id IS NULL
			ORDER BY name
			LIMIT $1 OFFSET $2
		),
		tree AS (
			SELECT id, parent_id FROM permissions
			WHERE id IN (SELECT id FROM top_level)
			UNION ALL
			SELECT p.id, p.parent_id
			FROM permissions p
			INNER JOIN tree t ON p.parent_id = t.id
			WHERE p.deleted_at IS NULL
		)
		SELECT
			p.id,
			COALESCE(p.parent_id::text, '') AS parent_id,
			p.name,
			p.key,
			p.route,
			p.type,
			p.method,
			COALESCE(p.description, '') AS description,
			COALESCE(
				json_agg(json_build_object('id', r.id, 'name', r.name)) FILTER (WHERE r.id IS NOT NULL),
				'[]'::json
			) AS roles
		FROM permissions p
		INNER JOIN tree ON tree.id = p.id
		LEFT JOIN role_permissions rp ON rp.permission_id = p.id AND rp.is_active = true
		LEFT JOIN roles r ON r.id = rp.role_id
		GROUP BY p.id, p.parent_id, p.name, p.key, p.route, p.type, p.method, p.description
		ORDER BY NULLIF(p.parent_id::text, '') NULLS FIRST, p.name
	`

	rows, err := sqlDB.QueryContext(c.Request.Context(), query, limit, offset)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	defer rows.Close()

	// permByID holds every scanned permission node, keyed by its ID.
	// childrenByParentID maps each parentID to its direct children IDs.
	type permNode struct {
		name, key, route, pType, parentID, description string
		method                            utils.StringArray
		roles                             []domain.RoleRef
	}

	permByID         := make(map[string]permNode)
	childrenByParent := make(map[string][]string)
	var rootIDs       []string
	
	for rows.Next() {
		var (
			id, parentID, name   string
			key, route, pType    string
			description          string
			method               utils.StringArray
			rolesJSON            []byte
		)
		if err = rows.Scan(&id, &parentID, &name, &key, &route, &pType, &method, &description, &rolesJSON); err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}

		var roles []domain.RoleRef
		if jsonErr := json.Unmarshal(rolesJSON, &roles); jsonErr != nil {
			h.log.Error("failed to unmarshal roles for permission", id, jsonErr)
			roles = []domain.RoleRef{}
		}

		permByID[id] = permNode{
			name:        name,
			key:         key,
			route:       route,
			pType:       pType,
			parentID:    parentID,
			description: description,
			method:      method,
			roles:       roles,
		}

		if parentID == "" {
			rootIDs = append(rootIDs, id)
		} else {
			childrenByParent[parentID] = append(childrenByParent[parentID], id)
		}
	}
	if err = rows.Err(); err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	// Orphaned nodes (parent was deleted) are surfaced at root level to avoid data loss.
	for id, node := range permByID {
		if node.parentID != "" {
			if _, exists := permByID[node.parentID]; !exists {
				rootIDs = append(rootIDs, id)
			}
		}
	}

	var buildChildren func(parentID string) []domain.PermissionWithRoles
	buildChildren = func(parentID string) []domain.PermissionWithRoles {
		ids := childrenByParent[parentID]
		out := make([]domain.PermissionWithRoles, 0, len(ids))
		for _, id := range ids {
			n := permByID[id]
			out = append(out, domain.PermissionWithRoles{
				Id:          id,
				Name:        n.name,
				Description: n.description,
				Key:         n.key,
				Route:       n.route,
				Type:        n.pType,
				ParentId:    n.parentID,
				Method:      n.method,
				Roles:       n.roles,
				Children:    buildChildren(id),
			})
		}
		return out
	}

	result := make([]domain.MainPermWithRoles, 0, len(rootIDs))
	for _, id := range rootIDs {
		n := permByID[id]
		result = append(result, domain.MainPermWithRoles{
			ID:          id,
			Key:         n.key,
			Name:	     n.name,
			Description: n.description,			
			Permissions: buildChildren(id),
		})
	}

	data := utils.ListResponse(result, total, limit, offset)

	handleResponse(c, OK, data, total)
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
	err = h.db.
		WithContext(c.Request.Context()).
		Delete(&domain.RolePermission{}, "role_id IN (?)", ids).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	err = h.db.
		Table("roles").Where("id IN (?)", ids).
		Updates(map[string]interface{}{"status": 2}).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")

}
