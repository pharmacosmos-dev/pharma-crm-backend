package v1

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
)

type ProducerHandler struct {
	*Handler
}

func (h *Handler) NewProducerHandler(r *gin.RouterGroup) {
	producer := &ProducerHandler{h}
	producer.ProducerRoutes(r)
}

func (h *ProducerHandler) ProducerRoutes(r *gin.RouterGroup) {
	producer := r.Group("/producer")
	{
		producer.POST("", h.Create)
		producer.GET("/list", h.List)
		producer.PUT("/:id", h.Update)
		producer.DELETE("/:id", h.Delete)
	}
	shelf := r.Group("/shelf")
	{
		shelf.POST("", h.CreateShelf)
		shelf.GET("/list", h.ListShelf)
		shelf.PUT("/:id", h.UpdateShelf)
		shelf.DELETE("/:id", h.DeleteShelf)
	}
}

// Create a new producer
// @Summary Create a new producer
// @Description Create a new producer
// @Tags producers
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.Producer true "Producer information"
// @Success 201 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /producer 	[post]
func (h *ProducerHandler) Create(c *gin.Context) {
	var (
		body domain.Producer
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.Create(&body).Error
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, "CREATED")
}

// List all producers
// @Summary List all producers
// @Description List all producers
// @Tags producers
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param search query string false "Search"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /producer/list [get]
func (h *ProducerHandler) List(c *gin.Context) {
	var (
		res        []*domain.Producer
		err        error
		totalCount int64
		search     = c.Query("search")
	)
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	query := h.db.Model(&domain.Producer{})

	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		query = query.Where("name ILIKE ?", search)
	}
	err = query.Count(&totalCount).Limit(limit).Offset(offset).Find(&res).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	result := utils.ListResponse(res, totalCount, limit, offset)

	handleResponse(c, OK, result)
}

// Update a producer
// @Summary Update a producer
// @Description Update a producer from the request body
// @Tags producers
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "producer ID"
// @Param input body domain.Producer true "Producer information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /producer/{id} [put]
func (h *ProducerHandler) Update(c *gin.Context) {
	var (
		id   = c.Param("id")
		body domain.Producer
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.Model(&domain.Producer{}).
		Where("id = ?", id).
		Updates(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, "UPDATED")
}

// Delete a producer
// @Summary Delete a producer
// @Description Delete a producer from the request body
// @Tags producers
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id path string true "producer ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /producer/{id} [delete]
func (h *ProducerHandler) Delete(c *gin.Context) {
	var id = c.Param("id")
	err := h.db.Delete(&domain.Producer{}, "id = ?", id).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}

// Create a new shelf
// @Summary Create a new shelf
// @Description Create a new shelf from the request body
// @Tags shelves
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.Shelf true "Shelf information"
// @Success 201 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /shelf 	[post]
func (h *ProducerHandler) CreateShelf(c *gin.Context) {
	var (
		body domain.Shelf
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.Create(&body).Error
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, "CREATED")
}

// List all shelves
// @Summary List all shelves
// @Description List all shelves
// @Tags shelves
// @Security     BearerAuth
// @Accept 	json
// @Produce 	json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param search query string false "Search"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /shelf/list [get]
func (h *ProducerHandler) ListShelf(c *gin.Context) {
	var (
		res        []*domain.Shelf
		err        error
		search     = c.Query("search")
		totalCount int64
	)
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	query := h.db.Model(&domain.Shelf{})
	if search != "" {
		search = fmt.Sprintf("%%%s%%", search)
		query = query.Where("name ILIKE ?", search)
	}
	err = query.Count(&totalCount).Limit(limit).Offset(offset).Find(&res).Error
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	result := utils.ListResponse(res, totalCount, limit, offset)

	handleResponse(c, OK, result)
}

// Update a shelf
// @Summary Update a shelf
// @Description Update a shelf from the request body
// @Tags shelves
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "shelf ID"
// @Param input body domain.Shelf true "Shelf information"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /shelf/{id} [put]
func (h *ProducerHandler) UpdateShelf(c *gin.Context) {
	var (
		id   = c.Param("id")
		body domain.Shelf
		err  error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.Model(&domain.Shelf{}).
		Where("id = ?", id).
		Updates(&body).Error
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "UPDATED")
}

// Delete a shelf
// @Summary Delete a shelf
// @Description Delete a shelf from the request body
// @Tags shelves
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	id path string true "shelf ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /shelf/{id} [delete]
func (h *ProducerHandler) DeleteShelf(c *gin.Context) {
	var id = c.Param("id")
	err := h.db.Delete(&domain.Shelf{}, "id = ?", id).Error
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}
