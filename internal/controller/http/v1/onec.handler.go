package v1

import (
	"context"
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/etc"
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
		onec.POST("/max-price-changing", h.CreateMaxPriceChanging)
		onec.POST("/barcode/create-or-update", h.CreateOrUpdateBarcodes)
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
// @Param 	product body domain.CreateOnecImportDto true "product"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product1c [POST]
func (h *ProductOnecHandler) Create(c *gin.Context) {
	var body domain.CreateOnecImportDto
	// bind request body
	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Errorf("could not bind onec import request: %v", err)
		handleServiceResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.ContextTimeoutForReports)
	defer cancel()

	err := h.service.CreateImportFromOnec(ctx, &body)
	if err != nil {
		handleServiceResponse(c, nil, err)
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
	// parse limit and offset
	limitStr := c.DefaultQuery("limit", "10000000")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	query := h.db.WithContext(ctx).
		Select(
			"sp.id",
			"p.material_code",
			"p.name",
			"s.name as store_name",
			"s.store_code",
			"p.barcode",
			"COALESCE(pr.code, '') as manufacturer",
			"ROUND(sp.unit_quantity::numeric/p.unit_per_pack, 4) as quantity",
			"sp.serial_number",
			"COALESCE(sp.expire_date, '3000-01-01') AS expire_date",
			"sp.retail_price",
			"sp.supply_price",
			"ROUND((sp.unit_quantity::numeric/p.unit_per_pack) * retail_price, 2) AS sum",
		).
		Table("store_products sp").
		Joins("JOIN products p ON sp.product_id = p.id").
		Joins("JOIN stores s ON sp.store_id = s.id").
		Joins("LEFT JOIN producers pr ON p.producer_id = pr.id").
		Where("(sp.pack_quantity > 0 or sp.unit_quantity > 0)")

	if storeCode != "" {
		store, err := h.service.GetStoreByField("store_code", storeCode)
		if err != nil {
			handleServiceResponse(c, InternalError, domain.InternalServerError)
			return
		}
		query = query.Where("sp.store_id = ?", store.Id)
	}

	if materialCode != "" {
		code, err := strconv.ParseInt(materialCode, 10, 64)
		if err != nil {
			handleServiceResponse(c, InternalError, domain.InvalidQueryError)
			return
		}
		productId, err := h.service.GetProductIdByCode(ctx, code)
		if err != nil {
			handleServiceResponse(c, InternalError, err)
			return
		}
		query = query.Where("sp.product_id = ?", productId)
	}

	var res []domain.OnecProductRes
	err = query.WithContext(ctx).Limit(limit).Offset(offset).Find(&res).Error
	if err != nil {
		h.log.Errorf("could not get product list: %v", err)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
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
// @Param 	product body domain.OnecRepricingRequest true "product"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product1c/repricing [POST]
func (h *ProductOnecHandler) ProductRepricing(c *gin.Context) {
	var (
		body  domain.OnecRepricingRequest
		store domain.Store
	)
	// bind request body
	if err := c.ShouldBindJSON(&body); err != nil {
		handleServiceResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	// get store info
	err := h.db.WithContext(ctx).First(&store, "store_code = ?", body.Apteka.StoreCode).Error
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

	// start transaction
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// check if price revalution already exists
	var res *domain.PriceRevalution
	err = tx.WithContext(ctx).First(&res, "name = ? ", body.Dok.DocumentNumber).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// create new price revalution
			res, err = h.service.CreateRepricingByOnec(ctx, tx, &domain.RepricingRequest{
				Name:      body.Dok.DocumentNumber,
				StoreId:   store.Id,
				CreatedBy: nil,
				Type:      "retail_price",
				Status:    constants.GeneralStatusCompleted,
			})
			if err != nil {
				_ = tx.Rollback()
				h.log.Errorf("could not create new price_revalution: %v", err)
				handleServiceResponse(c, InternalError, domain.NotFoundError)
				return
			}
		} else {
			// Handle any other database error (not RecordNotFound)
			_ = tx.Rollback()
			handleServiceResponse(c, InternalError, domain.InternalServerError)
			return
		}
	}

	// collect price revalution details
	var products []domain.PriceRevalutionDetailRequest
	for _, v := range body.Товары {
		// get product_id by material_code
		var productId string
		productId, err = h.service.GetProductIdByCode(ctx, int64(v.MaterialCode))
		if err != nil {
			_ = tx.Rollback()
			handleServiceResponse(c, nil, domain.NotFoundError)
			return
		}
		// update retail price by store_product id
		err = h.service.UpdateRetailPrice(ctx, tx, v.Id, v.NewRetailPrice)
		if err != nil {
			_ = tx.Rollback()
			h.log.Errorf("could not update retail_price: %v", err)
			handleServiceResponse(c, InternalError, domain.InternalServerError)
			return
		}
		// collect detail data
		products = append(products, domain.PriceRevalutionDetailRequest{
			PriceRevalutionId: res.Id,
			ProductId:         productId,
			StoreProductId:    v.Id,
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
		handleServiceResponse(c, InternalError, domain.InternalServerError)
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
// @Param 	product body domain.OnecUpdateQuantityRequest true "product"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product1c/quantity [POST]
func (h *ProductOnecHandler) UpdateQuantity(c *gin.Context) {
	var body domain.OnecUpdateQuantityRequest
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
// @Param 	request body domain.OnecMultiRepricingRequest true "repricing request"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product1c/multi-repricing [POST]
func (h *ProductOnecHandler) MultiProductRepricing(c *gin.Context) {
	var body domain.OnecMultiRepricingRequest
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
		if err := tx.WithContext(ctx).First(&store, "store_code = ?", aptekaReq.Apteka.StoreCode).Error; err != nil {
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
		err := tx.WithContext(ctx).First(&res, "name = ? AND store_id = ?", body.Dok.DocumentNumber, store.Id).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// create new price revalution for this store
				res, err = h.service.CreateRepricingByOnec(ctx, tx, &domain.RepricingRequest{
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
			productId, err = h.service.GetProductIdByCode(ctx, int64(v.MaterialCode))
			if err != nil {
				_ = tx.Rollback()
				h.log.Errorf("could not get product_id: %v", err)
				handleServiceResponse(c, nil, domain.NotFoundError)
				return
			}

			// update retail price
			if err = h.service.UpdateRetailPrice(ctx, tx, v.Id, v.NewRetailPrice); err != nil {
				_ = tx.Rollback()
				h.log.Errorf("could not update retail_price: %v", err)
				handleServiceResponse(c, InternalError, domain.InternalServerError)
				return
			}

			products = append(products, domain.PriceRevalutionDetailRequest{
				PriceRevalutionId: res.Id,
				ProductId:         productId,
				StoreProductId:    v.Id,
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

	handleResponse(c, OK, "CREATED")
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
			_ = tx.Rollback()
		}
	}()

	// save new token (service call)
	err = h.service.SaveAsilBelgiToken(tx, &body)
	if err != nil {
		_ = tx.Rollback()
		h.log.Warn("ERROR on saving Asil Belgi token: %v", err)
		handleResponse(c, InternalError, "failed.to.save.token")
		return
	}

	// commit
	if err = tx.Commit().Error; err != nil {
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


// CreatePriceChanged godoc
// @Summary Create product price changing
// @Description Save product price changing data from 1C
// @Tags        1C Api
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       request body domain.ProductChangePriceRequest true "product price changed"
// @Success     200 {object} v1.Response
// @Failure     400 {object} v1.Response
// @Failure     500 {object} v1.Response
// @Router      /product1c/max-price-changing [POST]
func (h *ProductOnecHandler) CreateMaxPriceChanging(c *gin.Context) {

	var body domain.ProductChangePriceRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Errorf("could not bind price changed request: %v", err)
		handleServiceResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		constants.DefaultContextTimeout,
	)
	defer cancel()

	result, err := h.service.CreateProductMaxPriceChanged(ctx, &body)
	if err != nil {
		if notAddErr, ok := err.(*domain.NotAdditionError); ok {
			handleResponse(c, NotFound, notAddErr.Data)
			return
		}
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, result)
}

// CreateOrUpdateBarcodes godoc
// @Summary Upsert product barcodes and material codes
// @Description Save product barcode and material code data from 1C
// @Tags        1C Api
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       request body domain.CreateOrUpdateBarcodesRequest true "upsert barcode"
// @Success     200 {object} v1.Response
// @Failure     400 {object} v1.Response
// @Failure     500 {object} v1.Response
// @Router      /product1c/barcode/upsert [POST]
func (h *ProductOnecHandler) CreateOrUpdateBarcodes(c *gin.Context) {
	var body domain.CreateOrUpdateBarcodesRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Errorf("could not bind barcode upsert request: %v", err)
		handleServiceResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}

	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}
	body.CreatedBy = user.UserId

	ctx, cancel := context.WithTimeout(
		context.Background(),
		constants.DefaultContextTimeout,
	)
	defer cancel()

	err := h.service.CreateOrUpdateBarcodes(ctx, &body)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, "SUCCESS")
}

