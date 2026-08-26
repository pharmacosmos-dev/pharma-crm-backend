package v1

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/helper"
	"github.com/pharma-crm-backend/pkg/utils"
)

type HolidayHandler struct {
	*Handler
}

func (h *Handler) NewHolidayHandler(r *gin.RouterGroup) {
	holiday := &HolidayHandler{h}
	holiday.HolidayRoutes(r)
}

func (h *HolidayHandler) HolidayRoutes(r *gin.RouterGroup) {
	holiday := r.Group("/holiday")
	{
		holiday.GET("/list", h.List)
		holiday.POST("", h.Create)
		holiday.PUT("/:id", h.Update)
		holiday.DELETE("/:id", h.Delete)
	}
}

// List godoc
// @Summary      List holidays
// @Description  Dam olish (bayram) kunlari ro'yxati. Payroll ish kunlarini shu jadvaldan hisoblaydi:
// @Description  ish kuni = kalendar kun − yakshanba − shu ro'yxatdagi sanalar.
// @Description  Yakshanba avtomatik hisoblanadi, bu yerda faqat bayramlar turadi.
// @Tags         holidays
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        year    query  int     false  "Yil bo'yicha filtr"
// @Param        month   query  int     false  "Oy bo'yicha filtr (year bilan birga)"
// @Param        search  query  string  false  "Nom bo'yicha qidiruv"
// @Param        limit   query  int     false  "Limit"
// @Param        offset  query  int     false  "Offset"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /holiday/list [get]
func (h *HolidayHandler) List(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.HolidayQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, nil, domain.InvalidQueryError)
		return
	}
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, totalCount, err := h.service.GetHolidays(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, utils.ListResponse(res, totalCount, params.Limit, params.Offset))
}

// Create godoc
// @Summary      Create holiday
// @Description  Yangi bayram kuni qo'shadi. Hayit sanalari har yili o'zgargani uchun
// @Description  ular shu endpoint orqali qo'lda kiritiladi.
// @Description  DIQQAT: qo'shilgan sana o'sha oyning ish kunlari sonini kamaytiradi va
// @Description  keyingi payroll hisobida KPI foiziga ta'sir qiladi. Ta'sir yo'nalishi
// @Description  sanaga bog'liq: o'tgan kunga qo'yilsa kutilgan reja kamayadi (foiz oshadi),
// @Description  kelajakdagi kunga qo'yilsa kutilgan reja ortadi (foiz tushadi).
// @Tags         holidays
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        input  body  domain.HolidayRequest  true  "Bayram sanasi (YYYY-MM-DD) va nomi"
// @Success      201  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      403  {object}  v1.Response
// @Failure      409  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /holiday [post]
func (h *HolidayHandler) Create(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}
	// Bayram kunlari barcha kompaniyalarga umumiy va oylik hisobga ta'sir qiladi.
	if !helper.IsAdmin(user) {
		handleServiceResponse(c, nil, domain.ForbiddinError)
		return
	}

	var body domain.HolidayRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.CreateHoliday(ctx, &body)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, CREATED, res)
}

// Update godoc
// @Summary      Update holiday
// @Description  Bayram sanasi yoki nomini o'zgartiradi.
// @Tags         holidays
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id     path  string                 true  "Holiday id"
// @Param        input  body  domain.HolidayRequest  true  "Yangi sana va nom"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      403  {object}  v1.Response
// @Failure      404  {object}  v1.Response
// @Failure      409  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /holiday/{id} [put]
func (h *HolidayHandler) Update(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}
	if !helper.IsAdmin(user) {
		handleServiceResponse(c, nil, domain.ForbiddinError)
		return
	}

	id := c.Param("id")
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid id")
		return
	}

	var body domain.HolidayRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.UpdateHoliday(ctx, id, &body)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, res)
}

// Delete godoc
// @Summary      Delete holiday
// @Description  Bayram kunini o'chiradi. O'chirilgan sana yana oddiy ish kuniga aylanadi.
// @Tags         holidays
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id  path  string  true  "Holiday id"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      403  {object}  v1.Response
// @Failure      404  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /holiday/{id} [delete]
func (h *HolidayHandler) Delete(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}
	if !helper.IsAdmin(user) {
		handleServiceResponse(c, nil, domain.ForbiddinError)
		return
	}

	id := c.Param("id")
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid id")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	if err := h.service.DeleteHoliday(ctx, id); err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, "DELETED")
}
