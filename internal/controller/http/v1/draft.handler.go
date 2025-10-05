package v1

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"
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
		draft.DELETE("/:id", h.Delete)
		draft.PUT("/complete/:id", h.CompleteDraft)
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
	var body domain.DraftRequest
	// bind request body
	if err := c.ShouldBindJSON(&body); err != nil {
		handleResponse(c, BadRequest, domain.InvalidRequestBodyError.Message)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.CreateDraft(ctx, &body)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, CREATED, res)
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
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	draft, err := h.service.GetDraftById(ctx, id)
	if err != nil {
		handleServiceResponse(c, draft, err)
		return
	}

	handleResponse(c, OK, draft)
}

// CompleteDraft
// @Summary Complete a draft
// @Description Complete a draft from the request body
// @Tags   drafts
// @Security     BearerAuth
// @Accept  json
// @Produce json
// @Param 	id path string true "draft ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /draft/complete/{id} [put]
func (h *DraftHandler) CompleteDraft(c *gin.Context) {
	var id = c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	draft, err := h.service.CompleteDraft(ctx, id)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, draft.SaleId)
}

// List godoc
// @Summary Get a draft
// @Description Get a draft from the request body
// @Tags drafts
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param store_id query string false "Store ID"
// @Param customer_id query string false "Customer ID"
// @Param search query string false "Search"
// @Param draft_date query string false "Draft Date"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /draft/list [get]
func (h *DraftHandler) List(c *gin.Context) {
	var params domain.DraftQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		handleResponse(c, BadRequest, domain.InvalidRequestBodyError.Message)
		return
	}

	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	params.Limit = limit
	params.Offset = offset

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, totalCount, err := h.service.GetDrafts(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	// Prepare and send the response
	data := utils.ListResponse(res, totalCount, limit, offset)
	handleResponse(c, OK, data)
}

// Delete godoc
// @Summary Delete a draft
// @Description Delete a draft from the request body
// @Tags drafts
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	id path string true "draft ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /draft/{id} [delete]
func (h *DraftHandler) Delete(c *gin.Context) {
	var id = c.Param("id")

	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, domain.InvalidQueryError.Message)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	err := h.service.DeleteDraft(ctx, id)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, "DELETED")
}
