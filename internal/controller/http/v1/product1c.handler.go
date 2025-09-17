package v1

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
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
	group1C := r.Group("/product1c")
	{
		group1C.POST("", h.Create)
		group1C.GET("/list", h.ListProductByStoreCode)
		group1C.POST("/repricing", h.ProductRepricing)
		group1C.POST("/multi-repricing", h.MultiProductRepricing)
		group1C.POST("/quantity", h.UpdateQuantity)
		group1C.POST("/token-asil-belgi", h.GetToken)
	}
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
// @Router /product1c [POST]
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
	var company domain.Company
	if body.Apteka.Franshiza {
		err = h.db.First(&company, "name ilike ?", "%"+constants.PHARMA_COSMOS+"%").Error // todo 1c given companyName
		if err != nil {
			tx.Rollback()
			handleResponse(c, InternalError, "Failed to get company info")
			return
		}
	} else {
		err = h.db.First(&company, "name ilike ?", "%"+constants.PHARMA_COSMOS+"%").Error
		if err != nil {
			tx.Rollback()
			handleResponse(c, InternalError, "Failed to get company info")
			return
		}
	}
	// get store info
	var store domain.Store
	err = h.db.First(&store, "store_code = ?", body.Apteka.StoreCode).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		store, err = h.service.CreateStoreOnImport(&domain.StoreRequest{Name: body.Apteka.Name, StoreCode: body.Apteka.StoreCode, CompanyId: company.ID})
		if err != nil {
			tx.Rollback()
			handleResponse(c, InternalError, "Failed to create new store")
			return
		}
	} else if err != nil {
		tx.Rollback()
		handleResponse(c, InternalError, "Failed to check store info")
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
		INSERT INTO products (material_code, name, barcode, producer_id, mxik, is_marking,company_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (material_code) DO UPDATE
		SET
			producer_id = EXCLUDED.producer_id, 
			mxik = EXCLUDED.mxik, 
			is_marking = EXCLUDED.is_marking
		RETURNING id`,
			body.Товары[i].MaterialCode,
			body.Товары[i].Name, body.Товары[i].Barcode, producer.Id, body.Товары[i].Ikpu, body.Товары[i].Mar, company.ID).Scan(&productID).Error
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
				INSERT INTO product_markings (import_detail_id, product_id, marking, store_id)
				VALUES(?, ?, ?, ?)`,
				id, productID, marking, store.Id).Error
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

// Get product list godoc
// @Summary Get product list by store_code
// @Description Get product list by store_code
// @Tags 	1C Api
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param   limit 			query int false "Limit"
// @Param   offset 			query int false "Offset"
// @Param 	store_code 		query string false "Store CODE"
// @Param 	material_code 	query string false "Material Code"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product1c/list [GET]
func (h *Product1cHandler) ListProductByStoreCode(c *gin.Context) {
	var (
		storeCode    = c.Query("store_code")
		materialCode = c.Query("material_code")
		// limitStr     = c.Query("limit")
		// offsetStr    = c.Query("offset")
	)

	query := h.db.
		Table("store_products sp").
		Select(
			"sp.id",
			"p.material_code",
			"p.name",
			"s.name as store_name",
			"s.store_code",
			"p.barcode",
			"COALESCE(pr.code, '') as manufacturer",
			"ROUND(sp.pack_quantity::numeric + (sp.unit_quantity::numeric % p.unit_per_pack)/p.unit_per_pack, 4) as quantity",
			"sp.serial_number",
			"COALESCE(sp.expire_date, '3000-01-01') AS expire_date",
			"sp.retail_price",
			"sp.supply_price",
			"ROUND((sp.pack_quantity::numeric + (sp.unit_quantity::numeric % p.unit_per_pack)/p.unit_per_pack) * retail_price, 2) AS sum",
		).
		Joins("JOIN products p ON sp.product_id = p.id").
		Joins("JOIN stores s ON sp.store_id = s.id").
		Joins("LEFT JOIN producers pr ON p.producer_id = pr.id").
		Where("(sp.pack_quantity > 0 or sp.unit_quantity > 0)")

	if storeCode != "" {
		store, err := h.service.GetStoreByField("store_code", storeCode)
		if err != nil {
			handleResponse(c, InternalError, err.Error())
			return
		}
		query = query.Where("sp.store_id = ?", store.Id)
	}

	if materialCode != "" {
		code, err := strconv.ParseInt(materialCode, 10, 64)
		if err != nil {
			handleResponse(c, InternalError, err.Error())
			return
		}
		productID, err := h.service.GetProductIDByCode(code)
		if err != nil {
			handleResponse(c, InternalError, err.Error())
			return
		}
		query = query.Where("sp.product_id = ?", productID)
	}

	var res []domain.ProductRes1C
	err := query.Find(&res).Error
	if err != nil {
		h.log.Warn("ERROR on getting product list: %v", err)
		handleResponse(c, InternalError, "failed.to.get.product_list")
		return
	}

	handleResponse(c, OK, res)
}

// Update retail price by 1C
// @Summary Update retail price by 1C
// @Description Update retail price by 1C
// @Tags 	1C Api
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	product body domain.RepricingRequest1C true "product"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product1c/repricing [POST]
func (h *Product1cHandler) ProductRepricing(c *gin.Context) {
	var (
		body  domain.RepricingRequest1C
		err   error
		store domain.Store
	)
	// bind request body
	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Warn("ERROR on binding request body: %v", err)
		handleResponse(c, BadRequest, "invalid.request.body")
		return
	}

	// get store info
	err = h.db.First(&store, "store_code = ?", body.Apteka.StoreCode).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleResponse(c, NotFound, "store.not.found")
			return
		}
		h.log.Warn("ERROR on getting store info: %v", err)
		handleResponse(c, InternalError, "failed.to.get.store")
		return
	}

	// validate products exists or no
	if len(body.Товары) < 1 {
		handleResponse(c, BadRequest, "received.empty.products")
		return
	}

	// start transaction
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// rollback function
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()
	// check if price revalution already exists
	var res *domain.PriceRevalution
	err = tx.First(&res, "name = ? ", body.Dok.DocumentNumber).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// create new price revalution
			res, err = h.service.CreateRepricingBy1C(tx, &domain.RepricingRequest{
				Name:      body.Dok.DocumentNumber,
				StoreId:   store.Id,
				CreatedBy: nil,
				Type:      "retail_price",
				Status:    config.COMPLETED,
			})
			if err != nil {
				h.log.Warn("ERROR on creating new price_revalution: %v", err)
				handleResponse(c, InternalError, "failed.to.create.repricing")
				return
			}
		}

	}

	// collect price revalution details
	var products []domain.PriceRevalutionDetailRequest
	for _, v := range body.Товары {
		// get product_id by material_code
		var productID string
		productID, err = h.service.GetProductIDByCode(int64(v.MaterialCode))
		if err != nil {
			h.log.Warn("ERROR on getting product_id by material_code: %v", err)
		}
		// update retail price by store_product id
		err = h.service.UpdateRetailPrice(v.Id, v.NewRetailPrice)
		if err != nil {
			h.log.Warn("ERROR on updating retail_price: %v", err)
			handleResponse(c, InternalError, "failed.update.retail_price")
			return
		}
		// collect detail data
		products = append(products, domain.PriceRevalutionDetailRequest{
			PriceRevalutionId: res.Id,
			ProductID:         productID,
			StoreProductID:    v.Id,
			OldRetailPrice:    v.RetailPrice,
			NewRetailPrice:    v.NewRetailPrice,
			OldSupplyPrice:    v.SupplyPrice,
			OldExpireDate:     v.ExpireDate,
			SerialNumber:      v.SerialNumber,
		})
	}

	// create price revalution details
	err = h.service.CreatePriceRevalutionDetail(tx, products)
	if err != nil {
		handleResponse(c, InternalError, "failed to create price revalution")
		return
	}

	// commit transaction
	err = tx.Commit().Error
	if err != nil {
		handleResponse(c, InternalError, "not.committed.transcation")
		return
	}

	handleResponse(c, OK, "UPDATED")
}

// Update retail price by 1C
// @Summary Update retail price by 1C
// @Description Update retail price by 1C
// @Tags 	1C Api
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	product body domain.UpdateQuantityRequest1C true "product"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product1c/quantity [POST]
func (h *Product1cHandler) UpdateQuantity(c *gin.Context) {
	var (
		body domain.UpdateQuantityRequest1C
		err  error
	)
	// bind request body
	if err = c.ShouldBindJSON(&body); err != nil {
		handleResponse(c, BadRequest, "Invalid received info, Please try again")
		return
	}
	// update quantity service
	err = h.service.UpdateProductQuantity(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, "Can't update quantity")
		return
	}
	handleResponse(c, OK, "UPDATED")
}

// MultiProductRepricing godoc
// Update retail price by 1C for multiple aptekas
// @Summary Update retail price by 1C (multi-apteka)
// @Description Update retail price by 1C for multiple aptekas
// @Tags 	1C Api
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	request body domain.MultiRepricingRequest1C true "repricing request"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product1c/multi-repricing [POST]
func (h *Product1cHandler) MultiProductRepricing(c *gin.Context) {
	var (
		body domain.MultiRepricingRequest1C
		err  error
	)

	// bind request body
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Warn("ERROR on binding request body: %v", err)
		handleResponse(c, BadRequest, "invalid.request.body")
		return
	}

	if len(body.Aptekas) < 1 {
		handleResponse(c, BadRequest, "received.empty.aptekas")
		return
	}

	// start transaction (bitta hujjat bo‘yicha umumiy transaction)
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	for _, aptekaReq := range body.Aptekas {
		// get store info
		var store domain.Store
		if err = tx.First(&store, "store_code = ?", aptekaReq.Apteka.StoreCode).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				handleResponse(c, NotFound, "store.not.found")
				return
			}
			h.log.Warn("ERROR on getting store info: %v", err)
			handleResponse(c, InternalError, "failed.to.get.store")
			return
		}

		if len(aptekaReq.Товары) < 1 {
			continue // skip empty products
		}

		// check if price revalution already exists
		var res *domain.PriceRevalution
		err = tx.First(&res, "name = ? AND store_id = ?", body.Dok.DocumentNumber, store.Id).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// create new price revalution for this store
				res, err = h.service.CreateRepricingBy1C(tx, &domain.RepricingRequest{
					Name:      body.Dok.DocumentNumber,
					StoreId:   store.Id,
					CreatedBy: nil,
					Type:      "retail_price",
					Status:    config.COMPLETED,
				})
				if err != nil {
					h.log.Warn("ERROR on creating price_revalution: %v", err)
					handleResponse(c, InternalError, "failed.to.create.repricing")
					return
				}
			}
		}

		// collect details
		var products []domain.PriceRevalutionDetailRequest
		for _, v := range aptekaReq.Товары {
			// get product_id by material_code
			var productID string
			productID, err = h.service.GetProductIDByCode(int64(v.MaterialCode))
			if err != nil {
				h.log.Warn("ERROR on getting product_id: %v", err)
			}

			// update retail price
			if err = h.service.UpdateRetailPrice(v.Id, v.NewRetailPrice); err != nil {
				h.log.Warn("ERROR on updating retail_price: %v", err)
				handleResponse(c, InternalError, "failed.update.retail_price")
				return
			}

			products = append(products, domain.PriceRevalutionDetailRequest{
				PriceRevalutionId: res.Id,
				ProductID:         productID,
				StoreProductID:    v.Id,
				OldRetailPrice:    v.RetailPrice,
				NewRetailPrice:    v.NewRetailPrice,
				OldSupplyPrice:    v.SupplyPrice,
				OldExpireDate:     v.ExpireDate,
				SerialNumber:      v.SerialNumber,
			})
		}

		// create details
		if err = h.service.CreatePriceRevalutionDetail(tx, products); err != nil {
			handleResponse(c, InternalError, "failed.to.create.repricing.details")
			return
		}
	}

	// commit transaction
	if err = tx.Commit().Error; err != nil {
		handleResponse(c, InternalError, "not.committed.transaction")
		return
	}

	handleResponse(c, OK, "UPDATED")
}

// GetToken godoc
// @Summary Save Asil Belgi Token
// @Description Save new Asil Belgi token (provided by 1C), deactivate old tokens
// @Tags 	1C Api
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	token body domain.AsilBelgiTokenRequest true "token"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product1c/token-asil-belgi [POST]
func (h *Product1cHandler) GetToken(c *gin.Context) {
	var (
		body domain.AsilBelgiTokenRequest
		err  error
	)

	// bind request body
	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Warn("ERROR on binding request body: %v", err)
		handleResponse(c, BadRequest, "invalid.request.body")
		return
	}

	// start transaction
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// save new token (service call)
	err = h.service.SaveAsilBelgiToken(tx, &body)
	if err != nil {
		h.log.Warn("ERROR on saving Asil Belgi token: %v", err)
		handleResponse(c, InternalError, "failed.to.save.token")
		return
	}

	// commit
	err = tx.Commit().Error
	if err != nil {
		handleResponse(c, InternalError, "not.committed.transaction")
		return
	}

	handleResponse(c, OK, "UPDATED")
}
