package v1

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/helper"
	"github.com/pharma-crm-backend/pkg/utils"
)

type ReserveHandler struct {
	*Handler
}

func (h *Handler) NewReserveHandler(r *gin.RouterGroup) {
	reserve := &ReserveHandler{h}
	reserve.ReserveRoutes(r)
}

func (h *ReserveHandler) ReserveRoutes(r *gin.RouterGroup) {
	reserve := r.Group("/reserve")
	{
		reserve.POST("", h.Create)
		reserve.GET("/list", h.List)
		reserve.GET("/:id", h.Get)
		reserve.PATCH("/:id/status", h.UpdateStatus)
		reserve.DELETE("/:id", h.Delete)
	}
	reserveDetail := r.Group("/reserve-detail")
	{
		reserveDetail.POST("", h.CreateDetail)
		reserveDetail.POST("/quick", h.QuickAddDetail)
		reserveDetail.GET("/list", h.ListDetails)
		reserveDetail.PUT("/:id", h.UpdateDetail)
		reserveDetail.DELETE("/:id", h.DeleteDetail)
	}
}

// Create godoc
// @Summary      Create reserve document
// @Description  Rezerv hujjati yaratadi. store_id bo'sh bo'lsa token'dagi do'kon olinadi,
// @Description  dok_number bo'sh bo'lsa avtomatik beriladi (RZ-1000, RZ-1001, ...).
// @Description  items berilsa qatorlar ham shu tranzaksiyada yoziladi; mahsulot topilmasa
// @Description  butun hujjat yaratilmaydi (ROLLBACK).
// @Tags         reserves
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        input  body  domain.ReserveRequest  true  "Hujjat va qatorlari"
// @Success      201  {object}  v1.Response{data=domain.Reserve}
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      409  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /reserve [post]
func (h *ReserveHandler) Create(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var body domain.ReserveRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		handleServiceResponse(c, nil, domain.InvalidRequestBodyError)
		return
	}
	// O'z do'koni bo'lgan xodim faqat o'sha do'kon uchun hujjat ochadi.
	if body.StoreId == "" || !helper.IsAdmin(user) {
		if user.StoreId != "" {
			body.StoreId = user.StoreId
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.CreateReserve(ctx, &body, user.UserId)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, CREATED, res)
}

// List godoc
// @Summary      List reserve documents
// @Description  Rezerv hujjatlari ro'yxati, created_at bo'yicha kamayish tartibida.
// @Tags         reserves
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        store_id    query  string  false  "Do'kon (admin uchun; boshqalarda token'dagi do'kon)"
// @Param        status      query  string  false  "new | checking | done"
// @Param        search      query  string  false  "dok_number yoki izoh bo'yicha"
// @Param        start_date  query  string  false  "YYYY-MM-DD (Toshkent kuni)"
// @Param        end_date    query  string  false  "YYYY-MM-DD (Toshkent kuni)"
// @Param        limit       query  int     false  "Limit"
// @Param        offset      query  int     false  "Offset"
// @Success      200  {object}  v1.Response{data=[]domain.Reserve}
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /reserve/list [get]
func (h *ReserveHandler) List(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.ReserveQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, nil, domain.InvalidQueryError)
		return
	}
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	// Admin bo'lmagan xodim faqat o'z do'koni hujjatlarini ko'radi.
	if !helper.IsAdmin(user) && user.StoreId != "" {
		params.StoreId = user.StoreId
	}
	// Bir nechta do'konga biriktirilgan xodim (ROP) ruxsatidan tashqari do'konni so'rasa — bo'sh javob.
	if len(user.StoreIds) > 0 && params.StoreId != "" && !utils.In(params.StoreId, user.StoreIds...) {
		handleResponse(c, OK, utils.ListResponse([]domain.Reserve{}, 0, params.Limit, params.Offset))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, totalCount, err := h.service.GetReserves(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, utils.ListResponse(res, totalCount, params.Limit, params.Offset))
}

// Get godoc
// @Summary      Get reserve document
// @Description  Hujjat va uning barcha qatorlari (details).
// @Tags         reserves
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id  path  string  true  "Reserve ID"
// @Success      200  {object}  v1.Response{data=domain.Reserve}
// @Failure      401  {object}  v1.Response
// @Failure      404  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /reserve/{id} [get]
func (h *ReserveHandler) Get(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.GetReserveById(ctx, c.Param("id"))
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, res)
}

// UpdateStatus godoc
// @Summary      Update reserve status
// @Description  new -> checking -> done. done bo'lgan hujjat qayta o'zgarmaydi (409).
// @Tags         reserves
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id     path  string                        true  "Reserve ID"
// @Param        input  body  domain.ReserveStatusRequest   true  "Yangi status"
// @Success      200  {object}  v1.Response{data=domain.Reserve}
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      404  {object}  v1.Response
// @Failure      409  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /reserve/{id}/status [patch]
func (h *ReserveHandler) UpdateStatus(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var body domain.ReserveStatusRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		handleServiceResponse(c, nil, domain.InvalidRequestBodyError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.UpdateReserveStatus(ctx, c.Param("id"), body.Status, user.UserId)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, res)
}

// Delete godoc
// @Summary      Delete reserve document
// @Description  Faqat 'new' holatidagi hujjat o'chiriladi, qatorlari ham birga ketadi.
// @Tags         reserves
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id  path  string  true  "Reserve ID"
// @Success      200  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      404  {object}  v1.Response
// @Failure      409  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /reserve/{id} [delete]
func (h *ReserveHandler) Delete(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	if err := h.service.DeleteReserve(ctx, c.Param("id")); err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, "DELETED")
}

// CreateDetail godoc
// @Summary      Add product to reserve document
// @Description  Hujjatga mahsulot qo'shadi; o'sha mahsulot allaqachon bo'lsa miqdori yangilanadi.
// @Description  Hujjat 'done' bo'lsa o'zgartirishga yo'l qo'yilmaydi (409).
// @Tags         reserves
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        input  body  domain.ReserveDetailRequest  true  "reserve_id, product_id, quantity"
// @Success      201  {object}  v1.Response{data=domain.ReserveDetail}
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      404  {object}  v1.Response
// @Failure      409  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /reserve-detail [post]
func (h *ReserveHandler) CreateDetail(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var body domain.ReserveDetailRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		handleServiceResponse(c, nil, domain.InvalidRequestBodyError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.UpsertReserveDetail(ctx, &body, user.UserId)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, CREATED, res)
}

// QuickAddDetail godoc
// @Summary      Quick add product to the store's open reserve document
// @Description  Bitta so'rov bilan qo'shish: do'konning ochiq hujjati (status != done) o'zi
// @Description  topiladi, bo'lmasa yangisi ochiladi. Mahsulot hujjatda allaqachon bo'lsa
// @Description  miqdor USTIGA QO'SHILADI (almashtirilmaydi — buning uchun PUT /reserve-detail/{id}).
// @Description  store_id bo'sh bo'lsa token'dagi do'kon ishlatiladi. Javobdagi reserve_id —
// @Description  qaysi hujjatga tushgani.
// @Tags         reserves
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        input  body  domain.ReserveQuickDetailRequest  true  "store_id (ixtiyoriy), product_id, quantity"
// @Success      201  {object}  v1.Response{data=domain.ReserveDetail}
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /reserve-detail/quick [post]
func (h *ReserveHandler) QuickAddDetail(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var body domain.ReserveQuickDetailRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		handleServiceResponse(c, nil, domain.InvalidRequestBodyError)
		return
	}
	// O'z do'koni bo'lgan xodim faqat o'sha do'kon hujjatiga qo'shadi.
	if body.StoreId == "" || !helper.IsAdmin(user) {
		if user.StoreId != "" {
			body.StoreId = user.StoreId
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.QuickAddReserveDetail(ctx, &body, user.UserId)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, CREATED, res)
}

// ListDetails godoc
// @Summary      List reserve document details
// @Description  Hujjat qatorlari. available_quantity — hujjat do'konidagi hozirgi qoldiq.
// @Tags         reserves
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        reserve_id  query  string  true   "Reserve ID"
// @Param        search      query  string  false  "Mahsulot nomi yoki material_code"
// @Param        limit       query  int     false  "Limit"
// @Param        offset      query  int     false  "Offset"
// @Success      200  {object}  v1.Response{data=[]domain.ReserveDetail}
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /reserve-detail/list [get]
func (h *ReserveHandler) ListDetails(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.ReserveDetailQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, nil, domain.InvalidQueryError)
		return
	}
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, totalCount, err := h.service.GetReserveDetails(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, utils.ListResponse(res, totalCount, params.Limit, params.Offset))
}

// UpdateDetail godoc
// @Summary      Update reserve document detail
// @Description  quantity va/yoki checked_quantity ni yangilaydi (faqat berilgan maydon).
// @Tags         reserves
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id     path  string                              true  "Reserve detail ID"
// @Param        input  body  domain.ReserveDetailUpdateRequest   true  "quantity / checked_quantity"
// @Success      200  {object}  v1.Response{data=domain.ReserveDetail}
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      404  {object}  v1.Response
// @Failure      409  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /reserve-detail/{id} [put]
func (h *ReserveHandler) UpdateDetail(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var body domain.ReserveDetailUpdateRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		handleServiceResponse(c, nil, domain.InvalidRequestBodyError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.UpdateReserveDetail(ctx, c.Param("id"), &body, user.UserId)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, res)
}

// DeleteDetail godoc
// @Summary      Delete reserve document detail
// @Description  Qatorni o'chiradi va hujjat jami miqdorlarini qayta hisoblaydi.
// @Tags         reserves
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id  path  string  true  "Reserve detail ID"
// @Success      200  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      404  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /reserve-detail/{id} [delete]
func (h *ReserveHandler) DeleteDetail(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	if err := h.service.DeleteReserveDetail(ctx, c.Param("id"), user.UserId); err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, "DELETED")
}
