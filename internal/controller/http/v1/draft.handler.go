package v1

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
	"gorm.io/gorm"
)

type DraftHandler struct {
	*Handler
}

func (h *Handler) NewDraftHandler(r *gin.RouterGroup) {
	draft := &DraftHandler{h}
	draft.DraftRoutes(r)
}

func (h *DraftHandler) DraftRoutes(r *gin.RouterGroup) {
	draft := r.Group("/draft")
	{
		draft.POST("", h.Create)
		draft.GET("/:id", h.Get)
		draft.GET("/list", h.List)
		draft.PUT("/:id", h.Update)
		draft.DELETE("/:id", h.Delete)
	}
}

// Create godoc
// @Summary Create a draft
// @Description Create a draft from the request body
// @Tags drafts
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param   input body domain.DraftRequest true "Draft information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /draft [post]
func (h *DraftHandler) Create(c *gin.Context) {
	var (
		body domain.DraftRequest
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	body.ID = uuid.New().String()
	body.DraftNumber = utils.GenerateCode()
	if err = h.db.WithContext(c.Request.Context()).
		Table("drafts").Create(&body).Error; err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, body)
}

// Get godoc
// @Summary Get a draft
// @Description Get a draft from the request body
// @Tags drafts
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "draft ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /draft/{id} [get]
func (h *DraftHandler) Get(c *gin.Context) {
	var res domain.Draft
	if err := h.db.First(&res, "id = ?", c.Param("id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
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
// @Summary Get a draft
// @Description Get a draft from the request body
// @Tags drafts
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limmit query int false "Limit"
// @Param offset query int false "Offset"
// @Param store_id query string false "Store ID"
// @Param cash_box_id query string false "Cash Box ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /draft/list [get]
func (h *DraftHandler) List(c *gin.Context) {
	var (
		res        []*domain.Draft
		totalCount int64
	)
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}

	query := h.db.Model(&domain.Draft{}).
		Preload("Store").Preload("Product").
		Limit(limit).Offset(offset).Count(&totalCount).Order("created_at DESC")
	if storeID := c.Query("store_id"); storeID != "" {
		query = query.Where("store_id = ?", storeID)
	}
	if cashBoxID := c.Query("cash_box_id"); cashBoxID != "" {
		query = query.Where("cash_box_id = ?", cashBoxID)
	}
	if err := query.Find(&res).Error; err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	data := utils.ListResponse(res, totalCount, limit, offset)
	handleResponse(c, OK, data)
}

// Update godoc
// @Summary Update a draft
// @Description Update a draft from the request body
// @Tags drafts
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "draft ID"
// @Param   input body domain.DraftRequest true "Draft information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /draft/{id} [put]
func (h *DraftHandler) Update(c *gin.Context) {
	var (
		body domain.DraftRequest
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	if err = h.db.WithContext(c.Request.Context()).
		Table("drafts").Where("id = ?", c.Param("id")).
		Updates(&body).Error; err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}

// Delete godoc
// @Summary Delete a draft
// @Description Delete a draft from the request body
// @Tags drafts
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "draft ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /draft/{id} [delete]
func (h *DraftHandler) Delete(c *gin.Context) {
	if err := h.db.WithContext(c.Request.Context()).Delete(&domain.Draft{}, "id = ?", c.Param("id")).Error; err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}
