package v1

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
	"gorm.io/gorm"
)

type ReturnHandler struct {
	*Handler
}

func (h *Handler) NewReturnHandler(r *gin.RouterGroup) {
	returnHandler := &ReturnHandler{h}
	returnHandler.ReturnRoutes(r)
}

func (h *ReturnHandler) ReturnRoutes(r *gin.RouterGroup) {
	returned := r.Group("/return")
	{
		returned.POST("", h.Create)
		returned.GET("/:id", h.Get)
		returned.GET("/list", h.List)
		returned.PATCH("/:id/add-product-by-barcode", h.AddProductByBarcode)
		returned.POST("/send/:id", h.Send)
		returned.POST("/confirm/:id", h.Confirm)
		returned.POST("/cancel/:id", h.Cancel)
	}
	detail := r.Group("return-detail")
	{
		detail.GET("/list", h.InventoryDetailList)
	}

}

// Create godoc
// @Summary Create Return
// @Description Create Return
// @Tags Return
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	return body domain.ReturnRequest true "Return"
// @Success 201 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /return [POST]
func (h *ReturnHandler) Create(c *gin.Context) {
	var returnRequest domain.ReturnRequest
	// Bind the request body to the InventoryRequest struct
	if err := c.ShouldBindJSON(&returnRequest); err != nil {
		h.log.Warn("Error on binding request: %v", err.Error())
		handleResponse(c, BadRequest, "Invalid request body")
		return
	}
	userId, ok := c.Get("user_id")
	if !ok {
		h.log.Warn("Error on getting user id from context")
		handleResponse(c, BadRequest, "User not authorized")
		return
	}
	returnRequest.CreatedBy = userId.(string)     // get creator id from set header
	returnRequest.PublicId = utils.GenerateCode() // generate public id

	// create inventory
	err := h.service.CreateReturn(&returnRequest)
	if err != nil {
		h.log.Warn("Error on creating inventory: %v", err.Error())
		handleResponse(c, InternalError, "Failed to create inventory")
		return
	}

	handleResponse(c, CREATED, "CREATED")
}

// Get godoc
// @Summary Get Return
// @Description Get Return
// @Tags Return
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id path string true "Return ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /return/{id} [GET]
func (h *ReturnHandler) Get(c *gin.Context) {
	// get return by id
	id := c.Param("id")
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid return id")
		return
	}
	var res domain.Return
	err := h.db.Model(&domain.Transfer{}).Preload("Store").Raw(`SELECT * FROM transfers WHERE id = ?`, id).Scan(&res).Error
	if err != nil {
		h.log.Warn("Error on getting return: %v", err.Error())
		handleResponse(c, InternalError, "Failed to get return")
		return
	}

	handleResponse(c, OK, res)
}

// Get List
// @Summary Get Return list
// @Description Get Return list
// @Tags Return
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	limit query int false "LIMIT"
// @Param 	offset query int false "OFFSET"
// @Param   store_id query string false "STORE ID"
// @Param   search 	query string false "SEARCH KEY"
// @Param   type 	query string false "TYPE"
// @Param   status 	query string false "STATUS (0->new|1->sent|2->completed)"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /return/list [GET]
func (h *ReturnHandler) List(c *gin.Context) {
	var param domain.ReturnParam

	if err := c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	res, totalCount, err := h.service.ReturnList(&param)
	if err != nil {
		handleResponse(c, InternalError, "Failed to get return list")
		return
	}
	data := utils.ListResponse(res, totalCount, param.Limit, param.Offset)

	handleResponse(c, OK, data)
}

// Add product by barcode
// @Summary Add product by barcode
// @Description Add product by barcode
// @Tags Return
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id 	path string true "Return ID"
// @Param 	body body domain.ReturnAddProduct true "Add product by barcode"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /return/{id}/add-product-by-barcode [PATCH]
func (h *ReturnHandler) AddProductByBarcode(c *gin.Context) {
	var request domain.ReturnAddProduct
	id := c.Param("id")
	// validate inventory id
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Return id is invalid")
		return
	}
	// bind request body
	err := c.ShouldBindJSON(&request)
	if err != nil {
		handleResponse(c, BadRequest, "Invalid request body")
		return
	}
	var productId string
	if request.ProductId != "" {
		productId = request.ProductId
	} else if request.Barcode != "" {
		err = h.db.Raw(`
		SELECT transfer_details.product_id FROM transfer_details JOIN products p ON p.id = transfer_details.product_id
		WHERE p.barcode = ? AND transfer_details.transfer_id = ?
		`, request.Barcode, id).Scan(&productId).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				handleResponse(c, BadRequest, "Product barcode not found")
				return
			}
			h.log.Warn("ERROR on getting product by barcode: %v", err)
			handleResponse(c, InternalError, "Failed to find product by barcode")
			return
		}
	} else {
		handleResponse(c, BadRequest, "product_id or barcode not sent")
		return
	}

	if request.Type == "SCAN" {
		if request.Count == 0 {
			request.Count = 1
		}
		// add scanned count by product barcode
		err = h.db.Exec(`
		UPDATE transfer_details
		SET scanned_count = scanned_count + ?, updated_at = NOW()
		WHERE product_id = ? AND transfer_id = ?`,
			request.Count, productId, id).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, "Failed to add count")
			return
		}
	} else if request.Type == "MANUAL" {
		// add scanned count by product barcode
		err = h.db.Debug().Exec(`
		UPDATE transfer_details
		SET scanned_count = ?, updated_at = NOW()
		WHERE product_id = ? AND transfer_id = ?`,
			request.Count, productId, id).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, "Failed to add count")
			return
		}
	}
	handleResponse(c, OK, "ADDED")
}

// send Return
// @Summary Send Return
// @Description Send Return
// @Tags Return
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id 	path string true "Return ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /return/send/{id} [POST]
func (h *ReturnHandler) Send(c *gin.Context) {
	id := c.Param("id")
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid return id")
		return
	}
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "user id not found from the context")
		return
	}
	// confirm inventory service
	err := h.service.SendReturn(id, userId.(string))
	if err != nil {
		handleResponse(c, InternalError, "Failed to send return")
		return
	}

	handleResponse(c, OK, "SENT")
}

// confirm Return
// @Summary Confirm Return
// @Description Confirm Return
// @Tags Return
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id 	path string true "Return ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /return/confirm/{id} [POST]
func (h *ReturnHandler) Confirm(c *gin.Context) {
	id := c.Param("id")
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid return id")
		return
	}
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "user id not found from the context")
		return
	}
	// confirm inventory service
	err := h.service.ConfirmReturn(id, userId.(string))
	if err != nil {
		handleResponse(c, InternalError, "Failed to confirm return")
		return
	}

	handleResponse(c, OK, "COMFIRMED")
}

// cancel return
// @Summary Cancel Return
// @Description Cancel Return
// @Tags Return
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id 	path string true "Return ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /return/cancel/{id} [POST]
func (h *ReturnHandler) Cancel(c *gin.Context) {
	var id = c.Param("id")
	// validate inventory id
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid inventory id")
		return
	}
	// get user_id from the header
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "user id not found from the context")
		return
	}
	// confirm inventory service
	err := h.service.CancelReturn(id, userId.(string))
	if err != nil {
		handleResponse(c, InternalError, "Failed to confirm return")
		return
	}

	handleResponse(c, OK, "CANCELED")
}

// Get List
// @Summary Get return list
// @Description Get return list
// @Tags Return
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	limit query int false "LIMIT"
// @Param 	offset query int false "OFFSET"
// @Param   return_id query string true "Return ID"
// @Param   search 	query string false "SEARCH KEY"
// @Param   type 	query string false "TYPE"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /return-detail/list [GET]
func (h *ReturnHandler) InventoryDetailList(c *gin.Context) {
	var param domain.ReturnDetailParam
	// bind query param
	if err := c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	res, totalCount, err := h.service.ReturnDetailList(&param)
	if err != nil {
		handleResponse(c, InternalError, "Failed to get inventory detail list")
		return
	}
	// get inventory details status count
	statsCount, err := h.service.ReturnDetailStatsCount(&param)
	if err != nil {
		handleResponse(c, InternalError, "Failed to get inventory detail stats count")
		return
	}
	data := map[string]any{
		"_meta": utils.Meta{
			TotalCount:  totalCount,
			PerPage:     param.Limit,
			CurrentPage: (param.Offset / param.Limit) + 1,
			PageCount:   int((totalCount + int64(param.Limit) - 1) / int64(param.Limit)),
		},
		"stats_count": statsCount,
		"data":        res,
	}

	handleResponse(c, OK, data)
}
