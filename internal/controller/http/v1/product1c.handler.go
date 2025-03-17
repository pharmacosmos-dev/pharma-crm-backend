package v1

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/spf13/cast"
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
}

// Create 	godoc
// @Summary Create a product
// @Description Create a product from the request body
// @Tags 	1C Api
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	product body domain.CreateProduct1C true "product"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product1c [post]
func (h *Product1cHandler) Create(c *gin.Context) {
	var (
		body domain.CreateProduct1C
		err  error
	)
	// bind request body
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// start transaction
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	// get store info
	var store domain.Store
	err = h.db.First(&store, "store_code = ?", body.Apteka.StoreCode).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleResponse(c, OK, "Store not found")
			return
		}
		tx.Rollback()
		handleResponse(c, InternalError, err.Error())
		return
	}
	// collect import data
	newImport := domain.ImportRequest{
		Id:             uuid.New().String(),
		StoreID:        store.Id,
		Status:         config.NEW_IMPORT,
		ImportDate:     body.Dok.DocumentDate,
		DocumentNumber: body.Dok.DocumentNumber,
	}
	// create new import
	err = tx.Table("imports").Create(&newImport).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "unique constraint") {
			h.log.Warn("duplicate document_number: %v", err)
			handleResponse(c, OK, "Document with this number already exists")
			tx.Rollback()
			return
		}
		h.log.Error(fmt.Errorf("ERROR on creating dok: %v", err.Error()))
		handleResponse(c, InternalError, "Failed to creating new import")
		tx.Rollback()
		return
	}
	for i := range body.Товары {
		// get producer by code
		producer, err := h.service.GetProducerByCode(c.Request.Context(), body.Товары[i].Manufacturer)
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, "Manufacturer not found or not created")
			tx.Rollback()
			return
		}
		// create product id
		productID := uuid.New().String()
		// create or update product
		err = tx.Raw(`
		INSERT INTO products (material_code, name, barcode, producer_id, mxik)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (material_code) DO UPDATE
		SET name = EXCLUDED.name,
			barcode = EXCLUDED.barcode,
			producer_id = EXCLUDED.producer_id,
			mxik = EXCLUDED.mxik
		RETURNING id`,
			body.Товары[i].MaterialCode,
			body.Товары[i].Name, body.Товары[i].Barcode, producer.Id, body.Товары[i].Ikpu).Scan(&productID).Error
		if err != nil {
			h.log.Warn("ERROR on creating new product: %v", err.Error())
			handleResponse(c, BadRequest, "Error on checking product data")
			tx.Rollback()
			return
		}

		// create import_detail
		var id string
		err = tx.Raw(`
		INSERT INTO import_details(
			product_id, import_id, 
			received_count, scanned_count, supply_price, supply_price_vat,
			retail_price, retail_price_vat,
			vat, vat_sum, expire_date, series_number, 
			sum_vat, marking) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`,
			productID, newImport.Id, body.Товары[i].Quantity, body.Товары[i].Quantity, body.Товары[i].SupplyPrice,
			body.Товары[i].SupplyPriceVat, body.Товары[i].RetailPrice, body.Товары[i].RetailPriceVat,
			cast.ToInt(body.Товары[i].Vat), body.Товары[i].VatSum,
			body.Товары[i].ExpireDate, body.Товары[i].ProductSeriesNumber,
			body.Товары[i].SumVat, utils.StringArray(body.Товары[i].Markirovka)).Scan(&id).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, "ERROR on creating import details")
			tx.Rollback()
			return
		}
		for _, marking := range body.Товары[i].Markirovka {
			err = tx.Exec(`
				INSERT INTO product_markings (import_detail_id, product_id, marking)
				VALUES(?, ?, ?)`,
				id, productID, marking).Error
			if err != nil {
				h.log.Error("Failed to insert marking on importing: ", err)
				handleResponse(c, InternalError, err.Error())
				tx.Rollback()
				return
			}
		}
	}

	// check transaction completed
	if err = tx.Commit().Error; err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, "Failed to commit transaction")
		tx.Rollback()
		return
	}

	handleResponse(c, OK, "CREATED")
}
