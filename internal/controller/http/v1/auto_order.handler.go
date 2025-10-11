package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type AutoOrderHandler struct {
	*Handler
}

func (h *Handler) NewAutoOrderHandler(r *gin.RouterGroup) {
	autoOrder := &AutoOrderHandler{h}
	autoOrder.AutoOrderRoutes(r)
}

func (h *AutoOrderHandler) AutoOrderRoutes(r *gin.RouterGroup) {
	autoOrder := r.Group("/auto-order")
	{
		autoOrder.POST("", h.Create)
		autoOrder.DELETE("/:id", h.Delete)
		autoOrder.GET("/list", h.List)
		autoOrder.POST("/send/:id", h.SendAutoOrder)
		autoOrder.GET("/generated/list", h.ListGenerated)
	}
	autoOrderDetail := r.Group("auto-order-detail")
	{
		autoOrderDetail.GET("/list", h.AutoOrderDetailList)
		autoOrderDetail.GET("/export-excel", h.ExportAutoOrderDetail)
		autoOrderDetail.PUT("/change-quantity/:id", h.ChangeAdjustedOrder)
	}
}

// CreateAutoOrder godoc
// @Summary Create auto order
// @Description Create auto order
// @Security     BearerAuth
// @Tags 	auto_orders
// @Accept 	json
// @Produce json
// @Param 	input body 	domain.AutoOrderRequest true "Auto order information"
// @Success 201 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /auto-order 	[post]
func (h *AutoOrderHandler) Create(c *gin.Context) {
	var (
		body domain.AutoOrderRequest
		err  error
	)
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, domain.UnauthorizedError)
		return
	}

	err = c.ShouldBindJSON(&body) // bind request body
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}
	// start transaction
	tx := h.db.Begin()
	defer recoverTransaction(tx, h.log) // check revocer for panic error
	defer RollbackIfError(tx, &err)     // return transaction if error happened
	body.Id = uuid.New().String()
	body.CreatedBy = userId.(string)
	body.Status = constants.GeneralStatusNew
	body.AutoOrderDate = time.Now().Format("2006-01-02T15:04:05")
	// get auro order products based on store_id and interval day
	autoOrderDetails, err := h.service.GenerateAutoOrderDetail(body.Id, body.StoreId, body.IntervalDay)
	if err != nil {
		h.log.Error("could not generate new auto order details: %v", err)
		handleResponse(c, InternalError, domain.InternalServerError)
		return
	}

	// check if there are enough products for the auto order
	if len(autoOrderDetails) < 1 {
		handleResponse(c, CONFLICT, domain.NotEnoughProductError)
		return
	}

	// create auto order
	err = tx.
		Table("auto_orders").
		Create(&body).Error
	if err != nil {
		h.log.Error("cound not create auto order details: %v", err)
		handleResponse(c, InternalError, domain.InternalServerError)
		return
	}
	// create auto order details
	err = tx.Table("auto_order_details").Create(&autoOrderDetails).Error
	if err != nil {
		h.log.Warn("cound not create auto order details: %v", err)
		handleResponse(c, InternalError, domain.InternalServerError)
		return
	}
	// commit transaction
	err = tx.Commit().Error
	if err != nil {
		h.log.Error("cound not completed transaction: %v", err)
		handleResponse(c, InternalError, domain.InternalServerError)
		return
	}

	handleResponse(c, CREATED, "CREATED")
}

// ListAutoOrder godoc
// @Summary List auto orders
// @Description List auto orders
// @Tags 	auto_orders
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	limit query int false "Limit"
// @Param 	offset query int false "Offset"
// @Param 	store_id query string false "Store ID"
// @Param 	status query string false "Status"
// @Param 	start_date query string false "StartDate"
// @Param   end_date  query string false "EndDate"
// @Param 	search query string false "Search"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /auto-order/list [get]
func (h *AutoOrderHandler) List(c *gin.Context) {
	var param domain.AutoOrderParam
	// get user id from the header
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User not found from the context")
		return
	}
	err := c.ShouldBindQuery(&param)
	if err != nil {
		handleResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	// get defaul limit and offset
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)
	// get user id
	param.UserId = userId.(string)
	// get auto order list
	res, totalCount, err := h.service.ListAutoOrder(&param)
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}

	result := utils.ListResponse(res, totalCount, param.Limit, param.Offset)

	handleResponse(c, OK, result)
}

// AutoOrderDetailList godoc
// @Summary List auto order details
// @Description List auto order details
// @Tags auto_order_details
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param search query string false "Search"
// @Param auto_order_id query string false "Auto Order ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 401 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /auto-order-detail/list [get]
func (h *AutoOrderHandler) AutoOrderDetailList(c *gin.Context) {
	var (
		param      domain.AutoOrderParam
		totalCount int64
	)

	if err := c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, "invalid.request.queryparam")
		return
	}

	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	res, totalCount, err := h.service.AutoOrderDetailList(&param)
	if err != nil {
		h.log.Warn("ERROR on getting auto_order_details: %v", err)
		handleResponse(c, InternalError, "failed.get.auto_order_details")
		return
	}
	result := utils.ListResponse(res, totalCount, param.Limit, param.Offset)

	handleResponse(c, OK, result)
}

// AutoOrderDetailList godoc
// @Summary List auto order details
// @Description List auto order details
// @Tags auto_order_details
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param search query string false "Search"
// @Param auto_order_id query string false "Auto Order ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 401 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /auto-order-detail/export-excel [get]
func (h *AutoOrderHandler) ExportAutoOrderDetail(c *gin.Context) {
	var (
		param domain.AutoOrderParam
	)

	if err := c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, "invalid.request.queryparam")
		return
	}
	// get default
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)
	// get auto order detail list
	res, _, err := h.service.AutoOrderDetailList(&param)
	if err != nil {
		h.log.Warn("ERROR on getting auto_order_details: %v", err)
		handleResponse(c, InternalError, "failed.get.auto_order_details")
		return
	}

	// Excel fayl yaratish
	f := excelize.NewFile()
	sheetName := "List1"
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"Артикул", "Наименование", "Текущий Остаток", "Продажа кол-во", "Квант", "Мин зап", "Макс зап", "Срок Д/П", "Период продажа", "Сред продажа в ден", "Заказ кол-во", "Остаток на дату текущей поставки", "Остаток на дату следующей поставки"}

	err = setExcelHeaders(f, sheetName, headers)
	if err != nil {
		h.log.Error("Failed to create style:", err)
		handleResponse(c, InternalError, "Error on giving style to excel")
		return
	}

	// Ma'lumotlarni qo'shish
	for i, v := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, v.MaterialCode)
		f.SetCellValue(sheetName, "B"+row, v.ProductName)
		f.SetCellValue(sheetName, "C"+row, v.CurrentStock)
		f.SetCellValue(sheetName, "D"+row, v.SaleCount)
		f.SetCellValue(sheetName, "E"+row, v.Kvant)
		f.SetCellValue(sheetName, "F"+row, v.MinStock)
		f.SetCellValue(sheetName, "G"+row, v.MaxStock)
		f.SetCellValue(sheetName, "H"+row, v.ImportDay)
		f.SetCellValue(sheetName, "I"+row, 15)
		f.SetCellValue(sheetName, "J"+row, v.DailySaleCount)
		f.SetCellValue(sheetName, "K"+row, v.OrderCount)
		f.SetCellValue(sheetName, "L"+row, v.StockOnDeliveryDate)
		f.SetCellValue(sheetName, "M"+row, v.FutureStock)
	}
	saveExcelToUploads(c, f, *h.log, "auto_order_products")
}

// ChangeAdjustedOrder godoc
// @Summary Change adjusted order quantity
// @Description Change adjusted order quantity
// @Tags auto_order_details
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	id 	path string true "ID"
// @Param 	input body  domain.AdjustedOrderQuantity true "Adjusted Order Quantity"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /auto-order-detail/change-quantity/{id} [put]
func (h *AutoOrderHandler) ChangeAdjustedOrder(c *gin.Context) {
	var (
		body domain.AdjustedOrderQuantity
		err  error
		id   = c.Param("id")
	)
	if id == "" || id == "undefined" {
		handleResponse(c, BadRequest, "ID is required")
		return
	}
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// Initialize the map before using it
	data := make(map[string]interface{})
	if body.MinStock != 0 {
		data["min_stock"] = body.MinStock
	}
	if body.MaxStock != 0 {
		data["max_stock"] = body.MaxStock
	}
	if body.Kvant != 0 {
		data["kvant"] = body.Kvant
	}
	if body.AdjustedOrderQuantity != 0 {
		data["adjusted_order_quantity"] = body.AdjustedOrderQuantity
	}
	err = h.db.
		Model(&domain.AutoOrderDetail{}).
		Where("id = ?", id).
		Updates(&data).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	handleResponse(c, OK, "UPDATED")
}

// SendAutoOrder godoc
// @Summary Send auto order
// @Description Send auto order
// @Tags auto_orders
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Auto order ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /auto-order/send/{id} [post]
func (h *AutoOrderHandler) SendAutoOrder(c *gin.Context) {
	var (
		id        = c.Param("id")
		autoOrder *domain.AutoOrder
		data      domain.AutoOrderDetailSendRequest
		err       error
	)
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "user.not.authorized")
		return
	}
	// validate id
	if err = uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "invalid.auto_order_id")
		return
	}

	err = h.db.Preload("Store").First(&autoOrder, "id = ? AND status = 'new'", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleResponse(c, BadRequest, "already.completed")
			return
		}
		h.log.Error("could not get auto order(%s) info: %v", id, err)
		handleResponse(c, InternalError, domain.InternalServerError)
		return
	}
	err = h.db.Raw(`
	SELECT
		p.material_code, 
		p.name, 
		COALESCE(pr.name, '') AS manufacturer,  
		aod.order_count AS quantity
	FROM 
		auto_order_details aod
	JOIN 
		products p ON p.id = aod.product_id
	LEFT JOIN 
		producers pr ON p.producer_id = pr.id
	WHERE
		aod.order_count > 0 AND aod.auto_order_id = ?`, id).Scan(&data.Товары).Error
	if err != nil {
		h.log.Error("could not get auto order details: %v", err)
		handleResponse(c, InternalError, domain.InternalServerError)
		return
	}
	data.Dok.DataDok = autoOrder.CreatedAt.Format(constants.DateTimeFormatRFC3339)
	data.Dok.NomerDok = "AZ-" + strconv.Itoa(autoOrder.PublicID)
	data.Apteka.Name = autoOrder.Store.Name
	data.Apteka.StoreCode = autoOrder.Store.StoreCode

	res, err := h.DoRequest(context.Background(), data, "/zakaz")
	if err != nil {
		h.log.Error("could not do request auto order: %v", err)
		return
	}

	if len(res.Products) == 0 {
		return
	}

	for _, item := range res.Products {
		// update response_order_count after receive 1C response
		err = h.db.Exec(`UPDATE auto_order_details SET response_order_count = ? WHERE product_id = (SELECT id FROM products WHERE material_code = ?)`,
			item.QuantityFakt, item.MaterialCode).Error
		if err != nil {
			h.log.Error("could not update response order count: %v", err)
			return
		}
	}

	// update auto_order status to completed
	err = h.db.Exec(`UPDATE auto_orders SET status = ?, completed_date = NOW(), updated_by = ? WHERE id = ?`, constants.GeneralStatusCompleted, userId, id).Error
	if err != nil {
		h.log.Error("could not update auto_order(%s): %v", id, err)
		handleResponse(c, InternalError, domain.InternalServerError)
		return
	}

	handleResponse(c, OK, res.Data)
}

func (h *AutoOrderHandler) sentAutoOrderTo1C(id string, userId string, data *domain.AutoOrderDetailSendRequest) {

	res, err := h.DoRequest(context.Background(), data, "/zakaz")
	if err != nil {
		h.log.Error("could not do request auto order: %v", err)
		return
	}

	if len(res.Products) == 0 {
		return
	}

	for _, item := range res.Products {
		// update response_order_count after receive 1C response
		err = h.db.Exec(`UPDATE auto_order_details SET response_order_count = ? WHERE product_id = (SELECT id FROM products WHERE material_code = ?)`,
			item.QuantityFakt, item.MaterialCode).Error
		if err != nil {
			h.log.Error("could not update response order count: %v", err)
			return
		}
	}

	// update auto_order status to completed
	err = h.db.Exec(`UPDATE auto_orders SET status = ?, completed_date = NOW(), updated_by = ? WHERE id = ?`, constants.GeneralStatusCompleted, userId, id).Error
	if err != nil {
		h.log.Error("could not update auto_order(%s): %v", id, err)
		return
	}

}

// SendAutoOrder godoc
// @Summary Send auto order
// @Description Send auto order
// @Tags auto_orders
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	store_id query string true "Store ID"
// @Param 	day query string true "Interval Day"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /auto-order/generated/list [GET]
func (h *AutoOrderHandler) ListGenerated(c *gin.Context) {
	var (
		storeID = c.Query("store_id")
		day     = c.Query("day")
		err     error
	)
	intervalDay, err := strconv.ParseFloat(day, 64)
	if err != nil {
		handleResponse(c, BadRequest, "invalid.interval.day")
		return
	}

	res, err := h.service.GenerateAutoOrderDetail(uuid.NewString(), storeID, intervalDay)
	if err != nil {
		h.log.Warn("ERROR on generating auto order: %v", err)
		handleResponse(c, InternalError, "not.generated.auto_order")
		return
	}

	handleResponse(c, OK, res)
}

// DeleteAutoOrder godoc
// @Summary Delete auto order
// @Description Delete auto order
// @Tags 	auto_orders
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id path string true "Auto order ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /auto-order/{id} [delete]
func (h *AutoOrderHandler) Delete(c *gin.Context) {
	var id = c.Param("id")

	// delete auto order
	err := h.db.WithContext(c.Request.Context()).Delete(&domain.AutoOrder{}, "id = ? AND status = ?", id, constants.GeneralStatusNew).Error
	if err != nil {
		h.log.Error("could not delete auto_order(%s): %v", id, err)
		handleResponse(c, InternalError, domain.InternalServerError)
		return
	}

	handleResponse(c, OK, "DELETED")
}

// send request to 1C for creating auto order
func (h *AutoOrderHandler) DoRequest(ctx context.Context, data interface{}, url string) (*domain.AutoOrderResponse, error) {
	client := &http.Client{
		Timeout: 300 * time.Second,
	}
	buf := bytes.Buffer{}
	// Encode data to JSON
	err := json.NewEncoder(&buf).Encode(data)
	if err != nil {
		h.log.Error("failed to encode request data: %v", err)
		return nil, fmt.Errorf("failed to encode request data: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", h.cfg.OnecApiUrl+url, &buf)
	if err != nil {
		h.log.Error("failed to create HTTP request: %v", err)
		return nil, fmt.Errorf("failed to create HTTP request: %v", err)
	}
	req.SetBasicAuth(h.cfg.OnecApiUsername, h.cfg.OnecApiPassword)
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	response, err := client.Do(req)
	if err != nil {
		h.log.Error("failed to execute HTTP request: %v", err)
		return nil, fmt.Errorf("failed to execute HTTP request: %v", err)
	}
	body, _ := io.ReadAll(response.Body)
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return nil, fmt.Errorf("failed to send auto order: %v", string(body))
	}
	var info domain.AutoOrderResponse
	err = json.Unmarshal(body, &info)
	if err != nil {
		h.log.Error(err)
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}
	return &info, nil
}
