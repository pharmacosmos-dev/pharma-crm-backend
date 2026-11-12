package v1

import (
	// "context"
	// "errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	// "github.com/pharma-crm-backend/domain/constants"
)

type LikiHandler struct {
	*Handler
}

func (h *Handler) NewLikiHandler(r *gin.RouterGroup) {
	liki := &LikiHandler{h}
	liki.LikiRoutes(r)
}

func (h *LikiHandler) LikiRoutes(r *gin.RouterGroup) {
	liki := r.Group("/liki")
	liki.GET("/product/list", h.ProductList)
	liki.GET("/store-product/list", h.StoreProductList)
	liki.GET("/store-product/quantity", h.StoreProductQuantity)
	liki.GET("/store/list", h.StoreList)
	liki.GET("/category/list", h.CategoryList)
	// liki.POST("/order", h.CreateOrder)
	// liki.POST("/order/cancel/:order_id", h.CancelOrder)
	// liki.POST("/order/accept-courier/:order_id", h.AcceptCourierOrder)
}

// List Products
// @Summary List Products
// @Description List Products
// @Tags 		Liki API
// @Security    BasicAuth
// @Accept 		json
// @Produce 	json
// @Param   	limit 	query     int      false "Limit"
// @Param   	offset 	query     int      false "Offset"
// @Success 	200 {object} []domain.NoorProduct
// @Failure 	400 {object} v1.IntegrationErrorResponse
// @Failure 	500 {object} v1.IntegrationErrorResponse
// @Router 		/liki/product/list 	[GET]
func (h *LikiHandler) ProductList(c *gin.Context) {
	var params domain.NoorQueryParam

	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, http.StatusBadRequest, domain.InvalidQueryError)
		return
	}
	// get default product
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	res, err := h.service.GetLikiProducts(&params)
	if err != nil {
		handleServiceResponse(c, http.StatusInternalServerError, err)
		return
	}

	handleResponseNoor(c, http.StatusOK, res)
}

// Store Product List
// @Summary List Store Products
// @Description List Store Products
// @Tags 		Liki API
// @Security    BasicAuth
// @Accept 		json
// @Produce 	json
// @Param		shopId  query   string   false "Shop ID"
// @Param   	limit 	query     int      false "Limit"
// @Param   	offset 	query     int      false "Offset"
// @Success 	200 {object} []domain.NoorStoreProduct
// @Failure 	400 {object} v1.IntegrationErrorResponse
// @Failure 	500 {object} v1.IntegrationErrorResponse
// @Router 		/liki/store-product/list 	[GET]
func (h *LikiHandler) StoreProductList(c *gin.Context) {
	var params domain.NoorQueryParam
	// bind query param
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, http.StatusBadRequest, domain.InvalidQueryError)
		return
	}

	// get store products info
	res, err := h.service.GetLikiStoreProducts(&params)
	if err != nil {
		handleServiceResponse(c, http.StatusInternalServerError, err)
		return
	}

	handleResponseNoor(c, http.StatusOK, res)
}

// Store Product List
// @Summary List Store Products
// @Description List Store Products
// @Tags 		Liki API
// @Security    BasicAuth
// @Accept 		json
// @Produce     json
// @Success 	200 {object} []domain.NoorStore
// @Failure 	400 {object} v1.IntegrationErrorResponse
// @Failure 	500 {object} v1.IntegrationErrorResponse
// @Router 		/liki/store/list 	[GET]
func (h *LikiHandler) StoreList(c *gin.Context) {
	// get store products info
	res, err := h.service.GetLikiStores()
	if err != nil {
		handleResponseNoor(c, http.StatusInternalServerError, err.Error())
		return
	}

	handleResponseNoor(c, http.StatusOK, res)
}

// Bitta so'rovda so'ralishi mumkin bo'lgan product_id soni.
const likiMaxProductIds = 500

// Store Product Quantity
// @Summary 	Get product quantities in a store
// @Description Berilgan do'kondagi berilgan mahsulotlarning qoldig'i, qadoqda (uzum availability bilan bir xil).
// @Description Har bir mahsulotdan faqat eng katta partiya olinadi: buyurtma ham bitta partiyadan yoziladi.
// @Description shop_id va product_ids body'da yuboriladi (eski store_id kaliti ham qabul qilinadi).
// @Description Javob har doim massiv. Qoldiq tugagan bo'lsa ham qatori qaytadi, quantity: 0 bilan.
// @Description price online_products_price'dan olinadi. Narxi kiritilmagan mahsulot javobga tushmaydi.
// @Tags 		Liki API
// @Security    BasicAuth
// @Accept 		json
// @Produce 	json
// @Param 		body body domain.NoorStoreProductQuantityRequest true "shop_id va product_ids (ko'pi bilan 500 ta)"
// @Success 	200 {object} []domain.NoorStoreProductQuantity
// @Failure 	400 {object} v1.IntegrationErrorResponse
// @Failure 	500 {object} v1.IntegrationErrorResponse
// @Router 		/liki/store-product/quantity 	[GET]
func (h *LikiHandler) StoreProductQuantity(c *gin.Context) {
	var body domain.NoorStoreProductQuantityRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Errorf("could not bind liki store_product quantity request body: %v", err)
		handleResponseNoor(c, http.StatusBadRequest, "invalid request body")
		return
	}

	shopId := strings.TrimSpace(body.StoreId)
	if shopId == "" {
		shopId = strings.TrimSpace(body.StoreId)
	}
	if _, err := uuid.Parse(shopId); err != nil {
		handleResponseNoor(c, http.StatusBadRequest, "invalid store_id")
		return
	}

	var (
		productIds = make([]string, 0, len(body.ProductIds))
		seen       = make(map[string]struct{}, len(body.ProductIds))
	)

	for _, productId := range body.ProductIds {
		productId = strings.TrimSpace(productId)
		if productId == "" {
			continue
		}

		if _, err := uuid.Parse(productId); err != nil {
			handleResponseNoor(c, http.StatusBadRequest, "invalid product_id: "+productId)
			return
		}

		// Takrorlangan id javobda ikki marta chiqmasligi uchun.
		if _, ok := seen[productId]; ok {
			continue
		}
		seen[productId] = struct{}{}
		productIds = append(productIds, productId)
	}

	if len(productIds) == 0 {
		handleResponseNoor(c, http.StatusBadRequest, "product_ids is required")
		return
	}

	if len(productIds) > likiMaxProductIds {
		handleResponseNoor(c, http.StatusBadRequest, "too many product_ids, max "+strconv.Itoa(likiMaxProductIds))
		return
	}

	res, err := h.service.GetLikiStoreProductQuantities(shopId, productIds)
	if err != nil {
		handleResponseNoor(c, http.StatusInternalServerError, err)
		return
	}

	// Narxi yo'q mahsulotlar filtrlanadi, ya'ni javob bo'sh bo'lishi mumkin:
	// emptyIfNil bo'sh javobni null emas, [] qilib qaytaradi.
	handleResponseNoor(c, http.StatusOK, emptyIfNil(res))
}

// Category List godoc
// @Summary Get a category list for filter
// @Description Get a category list for filter
// @Tags 		Liki API
// @Security    BasicAuth
// @Accept 		json
// @Produce 	json
// @Param 		limit query int false "Limit"
// @Param 		offset query int false "Offset"
// @Success 	200 {object} []domain.NoorCategory
// @Failure 	400 {object} v1.IntegrationErrorResponse
// @Failure 	500 {object} v1.IntegrationErrorResponse
// @Router 		/liki/category/list [get]
func (h *LikiHandler) CategoryList(c *gin.Context) {
	var params domain.NoorQueryParam
	// bind query param
	err := c.ShouldBindQuery(&params)
	if err != nil {
		handleServiceResponse(c, http.StatusBadRequest, domain.InvalidQueryError)
		return
	}

	res, err := h.service.GetLikiCategories()
	if err != nil {
		handleResponseNoor(c, http.StatusInternalServerError, err)
		return
	}

	handleResponseNoor(c, http.StatusOK, res)
}


// func (h *LikiHandler) CreateOrder(c *gin.Context) {
// 	var body domain.OnlineOrderRequest
// 	if err := c.ShouldBindJSON(&body); err != nil {
// 		h.log.Errorf("could not bind liki order create request body: %v", err)
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"error": err.Error(),
// 		})
// 		return
// 	}

// 	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
// 	defer cancel()

// 	orderNumber, err := h.service.CreateLikiSale(ctx, &body)
// 	if err != nil {
// 		if notAddErr, ok := err.(*domain.NotAdditionError); ok {
// 			handleResponseNoor(c, http.StatusConflict, notAddErr.Data)
// 			return
// 		}
// 		if errors.Is(err, domain.ActiveInventoryError) {
// 			handleResponseNoor(c, http.StatusConflict, err)
// 			return
// 		}
// 		handleResponseNoor(c, http.StatusInternalServerError, err)
// 		return
// 	}

// 	handleResponseNoor(c, http.StatusOK, domain.OnlineOrderResponse{Message: "success", OrderId: orderNumber})
// }


// func (h *LikiHandler) CancelOrder(c *gin.Context) {
// 	orderId := c.Param("order_id")

// 	saleNumber, err := strconv.ParseInt(orderId, 10, 64)
// 	if err != nil {
// 		handleResponseNoor(c, http.StatusBadRequest, "invalid order_id")
// 		return
// 	}

// 	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
// 	defer cancel()

// 	err = h.service.CancelLikiOrder(ctx, saleNumber)
// 	if err != nil {
// 		handleServiceResponse(c, nil, err)
// 		return
// 	}

// 	handleResponse(c, NoContent, nil)
// }



// func (h *LikiHandler) SendWSTestMessage(c *gin.Context) {
// 	storeID := c.Query("store_id")
// 	if storeID == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "store_id is required"})
// 		return
// 	}

// 	go h.service.NotifyOnlineOrder(storeID, 12345)

// 	c.JSON(http.StatusOK, gin.H{"message": "Test message sent"})
// }
