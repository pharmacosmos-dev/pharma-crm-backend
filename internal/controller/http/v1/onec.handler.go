package v1

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/etc"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

type ProductOnecHandler struct {
	*Handler
}

func (h *Handler) NewProductOnecHandler(r *gin.RouterGroup) {
	product1c := &ProductOnecHandler{h}
	product1c.ProductOnecRoutes(r)
}

func (h *ProductOnecHandler) ProductOnecRoutes(r *gin.RouterGroup) {
	onec := r.Group("/product1c")
	{
		onec.POST("", h.Create)
		onec.GET("/list", h.ListProductByStoreCode)
		onec.POST("/repricing", h.ProductRepricing)
		onec.POST("/multi-repricing", h.MultiProductRepricing)
		onec.POST("/quantity", h.UpdateQuantity)
		onec.POST("/token-asil-belgi", h.GetToken)
	}
	r.POST("/generate-token", h.GenerateOnecToken)
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
func (h *ProductOnecHandler) Create(c *gin.Context) {
	var body domain.CreateProduct1C
	// bind request body
	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	var company domain.Company
	if body.Apteka.Franshiza {
		err := h.db.First(&company, "name ilike ?", "%"+constants.PharmaCosmos+"%").Error // todo 1c given companyName
		if err != nil {
			handleResponse(c, InternalError, "Failed to get company info")
			return
		}
	} else {
		err := h.db.First(&company, "name ilike ?", "%"+constants.PharmaCosmos+"%").Error
		if err != nil {
			handleResponse(c, InternalError, "Failed to get company info")
			return
		}
	}
	// start transaction
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()
	// get store info
	var store domain.Store
	err := h.db.First(&store, "store_code = ?", body.Apteka.StoreCode).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		store, err = h.service.CreateStoreOnImport(&domain.StoreRequest{Name: body.Apteka.Name, StoreCode: body.Apteka.StoreCode, CompanyId: company.ID})
		if err != nil {
			_ = tx.Rollback()
			handleResponse(c, InternalError, "could.not.create.store")
			return
		}
	} else if err != nil {
		_ = tx.Rollback()
		handleResponse(c, InternalError, "could.not.check.store.info")
		return
	}
	// collect import data
	newImport := domain.ImportRequest{
		Id:             uuid.New().String(),
		StoreID:        store.Id,
		Status:         constants.GeneralStatusNew,
		ImportDate:     body.Dok.DocumentDate,
		DocumentNumber: body.Dok.DocumentNumber,
	}
	// create new import
	err = tx.Table("imports").Create(&newImport).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "unique constraint") {
			_ = tx.Rollback()
			h.log.Errorf("duplicate document_number: %v", err)
			handleServiceResponse(c, OK, domain.AlreadyExistsError)
			return
		}
		_ = tx.Rollback()
		h.log.Errorf("could not create dok: %v", err)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
		return
	}
	for i := range body.Товары {
		// get producer by code
		producer, err := h.service.GetProducerByCode(c.Request.Context(), body.Товары[i].Manufacturer)
		if err != nil {
			_ = tx.Rollback()
			h.log.Errorf("could not get producer by code: %v", err)
			handleServiceResponse(c, InternalError, domain.InternalServerError)
			return
		}
		// create product id
		productId := uuid.New().String()
		// create or update product
		err = tx.Raw(`
		INSERT INTO products (
			material_code, 
			name, 
			barcode, 
			producer_id, 
			mxik, 
			is_marking,
			company_id
			)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (material_code) DO UPDATE
		SET
			producer_id = EXCLUDED.producer_id,
			is_marking = EXCLUDED.is_marking
		RETURNING id`,
			body.Товары[i].MaterialCode,
			body.Товары[i].Name,
			body.Товары[i].Barcode,
			producer.Id,
			body.Товары[i].Ikpu,
			body.Товары[i].Mar,
			company.ID,
		).Scan(&productId).Error
		if err != nil {
			_ = tx.Rollback()
			h.log.Errorf("could not creating new product: %v", err)
			handleServiceResponse(c, BadRequest, domain.InternalServerError)
			return
		}
		err = tx.Exec(`
    		INSERT INTO product_barcodes (
				product_id, 
				barcode, 
				status
				)
    		SELECT 
				?, ?, ?
    		WHERE NOT EXISTS (
    		    SELECT 1 FROM product_barcodes 
    		    WHERE product_id = ? AND barcode = ? AND status = ?
    		)
		`, productId,
			body.Товары[i].Barcode,
			constants.GeneralStatusCompleted,
			productId,
			body.Товары[i].Barcode,
			constants.GeneralStatusCompleted,
		).Error
		if err != nil {
			_ = tx.Rollback()
			h.log.Errorf("could not create product barcode: %v", err)
			handleServiceResponse(c, BadRequest, domain.InternalServerError)
			return
		}
		// create import_detail
		var id string
		err = tx.Raw(`
		INSERT INTO import_details(
			product_id, 
			import_id,
			received_count, 
			scanned_count, 
			supply_price, 
			supply_price_vat,
			retail_price, 
			retail_price_vat,
			vat, 
			vat_sum, 
			expire_date, 
			series_number, 
			sum_vat, 
			marking
			) 
			VALUES(
				?, ?, ?, 
				?, ?, ?, 
				?, ?, ?, 
				?, ?, ?, 
				?, ?) 
			RETURNING id`,
			productId,
			newImport.Id,
			body.Товары[i].Quantity,
			body.Товары[i].Quantity,
			body.Товары[i].SupplyPrice,
			body.Товары[i].SupplyPriceVat,
			body.Товары[i].RetailPrice,
			body.Товары[i].RetailPriceVat,
			cast.ToInt(body.Товары[i].Vat),
			body.Товары[i].VatSum,
			body.Товары[i].ExpireDate,
			body.Товары[i].ProductSeriesNumber,
			body.Товары[i].SumVat,
			utils.StringArray(body.Товары[i].Markirovka),
		).Scan(&id).Error
		if err != nil {
			_ = tx.Rollback()
			h.log.Errorf("could not create import_details: %v", err)
			handleServiceResponse(c, InternalError, domain.InternalServerError)
			return
		}
		for _, marking := range body.Товары[i].Markirovka {
			err = tx.Exec(`
				INSERT INTO product_markings (
					import_detail_id, 
					product_id, 
					marking, 
					store_id
					)
				VALUES(?, ?, ?, ?)`,
				id,
				productId,
				marking,
				store.Id,
			).Error
			if err != nil {
				_ = tx.Rollback()
				h.log.Errorf("could not insert marking on importing: %v", err)
				handleServiceResponse(c, InternalError, domain.InternalServerError)
				return
			}
		}
	}

	// check transaction completed
	if err = tx.Commit().Error; err != nil {
		h.log.Errorf("could not commited transaction: %v", err)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
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
func (h *ProductOnecHandler) ListProductByStoreCode(c *gin.Context) {
	var (
		storeCode    = c.Query("store_code")
		materialCode = c.Query("material_code")
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
func (h *ProductOnecHandler) ProductRepricing(c *gin.Context) {
	var (
		body  domain.RepricingRequest1C
		store domain.Store
	)
	// bind request body
	if err := c.ShouldBindJSON(&body); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}

	// get store info
	err := h.db.First(&store, "store_code = ?", body.Apteka.StoreCode).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleServiceResponse(c, NotFound, domain.NotFoundError)
			return
		}
		h.log.Errorf("could not get store info: %v", err)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
		return
	}

	// validate products exists or no
	if len(body.Товары) < 1 {
		handleResponse(c, BadRequest, "received.empty.products")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// start transaction
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
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
				Status:    constants.GeneralStatusCompleted,
			})
			if err != nil {
				_ = tx.Rollback()
				h.log.Errorf("could not create new price_revalution: %v", err)
				handleServiceResponse(c, nil, domain.InternalServerError)
				return
			}
		}

	}

	// collect price revalution details
	var products []domain.PriceRevalutionDetailRequest
	for _, v := range body.Товары {
		// get product_id by material_code
		var productId string
		productId, err = h.service.GetProductIDByCode(int64(v.MaterialCode))
		if err != nil {
			_ = tx.Rollback()
			h.log.Errorf("could not get product_id by material_code: %v", err)
			handleServiceResponse(c, nil, domain.NotFoundError)
			return
		}
		// update retail price by store_product id
		err = h.service.UpdateRetailPrice(v.Id, v.NewRetailPrice)
		if err != nil {
			_ = tx.Rollback()
			h.log.Errorf("could not update retail_price: %v", err)
			handleServiceResponse(c, InternalError, domain.InternalServerError)
			return
		}
		// collect detail data
		products = append(products, domain.PriceRevalutionDetailRequest{
			PriceRevalutionId: res.Id,
			ProductID:         productId,
			StoreProductID:    v.Id,
			OldRetailPrice:    v.RetailPrice,
			NewRetailPrice:    v.NewRetailPrice,
			OldSupplyPrice:    v.SupplyPrice,
			OldExpireDate:     v.ExpireDate,
			SerialNumber:      v.SerialNumber,
		})
	}

	// create price revalution details
	err = h.service.CreatePriceRevalutionDetail(ctx, tx, products)
	if err != nil {
		_ = tx.Rollback()
		handleServiceResponse(c, InternalError, err)
		return
	}

	// commit transaction
	if err = tx.Commit().Error; err != nil {
		h.log.Errorf("could not commit product repricing transaction: %v", err)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
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
func (h *ProductOnecHandler) UpdateQuantity(c *gin.Context) {
	var body domain.UpdateQuantityRequest1C
	// bind request body
	if err := c.ShouldBindJSON(&body); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()
	// update quantity service
	err := h.service.UpdateProductQuantity(ctx, &body)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
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
func (h *ProductOnecHandler) MultiProductRepricing(c *gin.Context) {
	var body domain.MultiRepricingRequest1C
	// bind request body
	if err := c.ShouldBindJSON(&body); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}

	if len(body.Aptekas) < 1 {
		handleResponse(c, BadRequest, "received.empty.aptekas")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// start transaction (bitta hujjat bo‘yicha umumiy transaction)
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for _, aptekaReq := range body.Aptekas {
		// get store info
		var store domain.Store
		if err := tx.First(&store, "store_code = ?", aptekaReq.Apteka.StoreCode).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				_ = tx.Rollback()
				handleServiceResponse(c, NotFound, domain.NotFoundError)
				return
			}
			_ = tx.Rollback()
			h.log.Errorf("could not get store info: %v", err)
			handleResponse(c, InternalError, domain.InternalServerError)
			return
		}

		if len(aptekaReq.Товары) < 1 {
			continue // skip empty products
		}

		// check if price revalution already exists
		var res *domain.PriceRevalution
		err := tx.First(&res, "name = ? AND store_id = ?", body.Dok.DocumentNumber, store.Id).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// create new price revalution for this store
				res, err = h.service.CreateRepricingBy1C(tx, &domain.RepricingRequest{
					Name:      body.Dok.DocumentNumber,
					StoreId:   store.Id,
					CreatedBy: nil,
					Type:      "retail_price",
					Status:    constants.GeneralStatusCompleted,
				})
				if err != nil {
					_ = tx.Rollback()
					h.log.Errorf("could not create price_revalution: %v", err)
					handleServiceResponse(c, InternalError, domain.InternalServerError)
					return
				}
			}
		}

		// collect details
		var products []domain.PriceRevalutionDetailRequest
		for _, v := range aptekaReq.Товары {
			// get product_id by material_code
			var productId string
			productId, err = h.service.GetProductIDByCode(int64(v.MaterialCode))
			if err != nil {
				_ = tx.Rollback()
				h.log.Errorf("could not get product_id: %v", err)
				handleServiceResponse(c, nil, domain.NotFoundError)
				return
			}

			// update retail price
			if err = h.service.UpdateRetailPrice(v.Id, v.NewRetailPrice); err != nil {
				_ = tx.Rollback()
				h.log.Errorf("could not update retail_price: %v", err)
				handleServiceResponse(c, InternalError, domain.InternalServerError)
				return
			}

			products = append(products, domain.PriceRevalutionDetailRequest{
				PriceRevalutionId: res.Id,
				ProductID:         productId,
				StoreProductID:    v.Id,
				OldRetailPrice:    v.RetailPrice,
				NewRetailPrice:    v.NewRetailPrice,
				OldSupplyPrice:    v.SupplyPrice,
				OldExpireDate:     v.ExpireDate,
				SerialNumber:      v.SerialNumber,
			})
		}

		// create details
		if err = h.service.CreatePriceRevalutionDetail(ctx, tx, products); err != nil {
			_ = tx.Rollback()
			handleServiceResponse(c, InternalError, err)
			return
		}
	}

	// commit transaction
	if err := tx.Commit().Error; err != nil {
		h.log.Errorf("could not commit multi repricing transaction: %v", err)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
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
func (h *ProductOnecHandler) GetToken(c *gin.Context) {
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

// @Summary Generate 1C token
// @Description Generate 1C token
// @Tags 1C token
// @Accept json
// @Produce json
// @Param 	request body domain.GenerateOnecTokenRequest true "Generate 1C token"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /generate-token [post]
func (h *ProductOnecHandler) GenerateOnecToken(c *gin.Context) {
	var req domain.GenerateOnecTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}

	token, err := etc.Encrypt(req.Password, h.cfg.HashKey)
	if err != nil {
		h.log.Errorf("could not encrypt password: %v", err)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
		return
	}
	handleResponse(c, OK, token)
}
