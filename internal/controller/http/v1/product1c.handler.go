package v1

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"gorm.io/gorm"
)

type Product1cHandler struct {
	*Handler
}

func (h *Handler) NewProduct1cHandler(r *gin.RouterGroup) {
	product1c := &Product1cHandler{h}
	product1c.Product1cRoutes(r)
}

func (h *Product1cHandler) Product1cRoutes(r *gin.RouterGroup) {
	r.POST("/product1c", h.Create)
	r.POST("/store1c", h.CreateStore)
}

// Create godoc
// @Summary Create a product
// @Description Create a product from the request body
// @Tags 1C Api
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param product body domain.CreateProduct1C true "product"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product1c [post]
func (h *Product1cHandler) Create(c *gin.Context) {
	var (
		body     domain.CreateProduct1C
		document domain.ProductReceipt
		err      error
	)
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	body.Dok.StoreCode = body.Apteka.StoreCode
	body.Dok.ID = uuid.New().String()
	err = h.db.
		WithContext(c.Request.Context()).
		Table("product_receipts").
		Create(&body.Dok).Scan(&document).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "unique constraint") {
			h.log.Warn(fmt.Sprintf("duplicate document_number: %v", err))
			handleResponse(c, BadRequest, "Document with this number already exists")
			return
		}
		// Log and handle other errors
		h.log.Error(fmt.Errorf("ERROR on creating dok: %v", err.Error()))
		handleResponse(c, InternalError, "Failed to create document")
		return
	}
	for i := range body.Товары {
		body.Товары[i].ReceiptID = document.ID
		body.Товары[i].StoreCode = body.Apteka.StoreCode
	}
	err = h.db.
		WithContext(c.Request.Context()).
		Table("products").
		Create(&body.Товары).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, "CREATED")
}

// Create godoc
// @Summary Create a store
// @Description Create a store from the request body
// @Tags 1C Api
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param store body []domain.StoreRequest1C true "store"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /store1c [post]
func (h *Product1cHandler) CreateStore(c *gin.Context) {
	var (
		body []domain.StoreRequest1C
		err  error
	)
	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}

	// Create stores from 1C
	err = h.db.
		WithContext(c.Request.Context()).
		Table("stores").
		Create(&body).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "CREATED")
}
