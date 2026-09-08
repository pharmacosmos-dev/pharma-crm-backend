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

type DeductionHandler struct {
	*Handler
}

func (h *Handler) NewDeductionHandler(r *gin.RouterGroup) {
	deduction := &DeductionHandler{h}
	deduction.DeductionRoutes(r)
}

func (h *DeductionHandler) DeductionRoutes(r *gin.RouterGroup) {
	types := r.Group("/deduction-type")
	{
		types.POST("", h.CreateType)
		types.GET("/list", h.ListTypes)
		types.GET("/:id", h.GetType)
		types.PUT("/:id", h.UpdateType)
		types.DELETE("/:id", h.DeleteType)
	}

	deduction := r.Group("/deduction")
	{
		deduction.POST("", h.Create)
		deduction.GET("/list", h.List)
		deduction.GET("/:id", h.Get)
		deduction.PUT("/:id", h.Update)
		deduction.DELETE("/:id", h.Delete)
	}

	details := r.Group("/deduction-detail")
	{
		details.POST("", h.CreateDetail)
		details.GET("/list", h.ListDetails)
		details.GET("/:id", h.GetDetail)
		details.PUT("/:id", h.UpdateDetail)
		details.DELETE("/:id", h.DeleteDetail)
	}
}

// signedUser — tizimga kirgan foydalanuvchini qaytaradi, kirmagan bo'lsa 401
// yozib false qaytaradi.
func (h *DeductionHandler) signedUser(c *gin.Context) (*domain.EmployeeClaims, bool) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return nil, false
	}
	return user, true
}

// validId — yo'ldagi id UUID ekanini tekshiradi.
func validId(c *gin.Context) (string, bool) {
	id := c.Param("id")
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid id")
		return "", false
	}
	return id, true
}

// region Types

// CreateType godoc
// @Summary      Create deduction type
// @Description  Yangi ushlab qolish turi qo'shadi (masalan "Kamomad").
// @Description  code — kodda ishlatiladigan barqaror kalit, keyin o'zgartirilmasligi kerak; name — ko'rinadigan nom.
// @Tags         deductions
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        input  body  domain.DeductionTypeRequest  true  "Tur"
// @Success      201  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      403  {object}  v1.Response
// @Failure      409  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /deduction-type [post]
func (h *DeductionHandler) CreateType(c *gin.Context) {
	user, ok := h.signedUser(c)
	if !ok {
		return
	}
	// Turlar barcha kompaniyalarga umumiy va hisobotlarga ta'sir qiladi
	if !helper.IsAdmin(user) {
		handleServiceResponse(c, nil, domain.ForbiddinError)
		return
	}

	var body domain.DeductionTypeRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.CreateDeductionType(ctx, &body)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}
	handleResponse(c, CREATED, res)
}

// ListTypes godoc
// @Summary      List deduction types
// @Description  Ushlab qolish turlari ro'yxati. Nofaollari ham qaytadi — eski qatorlarda ular hali ishlatilgan bo'lishi mumkin.
// @Tags         deductions
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /deduction-type/list [get]
func (h *DeductionHandler) ListTypes(c *gin.Context) {
	if _, ok := h.signedUser(c); !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.GetDeductionTypes(ctx)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}
	handleResponse(c, OK, res)
}

// GetType godoc
// @Summary      Get deduction type
// @Tags         deductions
// @Security     BearerAuth
// @Produce      json
// @Param        id  path  string  true  "Type ID"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      404  {object}  v1.Response
// @Router       /deduction-type/{id} [get]
func (h *DeductionHandler) GetType(c *gin.Context) {
	if _, ok := h.signedUser(c); !ok {
		return
	}
	id, ok := validId(c)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.GetDeductionTypeById(ctx, id)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}
	handleResponse(c, OK, res)
}

// UpdateType godoc
// @Summary      Update deduction type
// @Description  code va name'ni yangilaydi. is_active berilsa turni faol/nofaol qiladi.
// @Tags         deductions
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id     path  string                       true  "Type ID"
// @Param        input  body  domain.DeductionTypeRequest  true  "Tur"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      403  {object}  v1.Response
// @Failure      404  {object}  v1.Response
// @Failure      409  {object}  v1.Response
// @Router       /deduction-type/{id} [put]
func (h *DeductionHandler) UpdateType(c *gin.Context) {
	user, ok := h.signedUser(c)
	if !ok {
		return
	}
	if !helper.IsAdmin(user) {
		handleServiceResponse(c, nil, domain.ForbiddinError)
		return
	}
	id, ok := validId(c)
	if !ok {
		return
	}

	var body domain.DeductionTypeRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.UpdateDeductionType(ctx, id, &body)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}
	handleResponse(c, OK, res)
}

// DeleteType godoc
// @Summary      Delete deduction type
// @Description  Turni o'chiradi. Unga bog'langan qatorlar bo'lsa 409 qaytadi — bunday turni o'chirish o'rniga is_active = false qilib qo'yish kerak.
// @Tags         deductions
// @Security     BearerAuth
// @Produce      json
// @Param        id  path  string  true  "Type ID"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      403  {object}  v1.Response
// @Failure      404  {object}  v1.Response
// @Failure      409  {object}  v1.Response
// @Router       /deduction-type/{id} [delete]
func (h *DeductionHandler) DeleteType(c *gin.Context) {
	user, ok := h.signedUser(c)
	if !ok {
		return
	}
	if !helper.IsAdmin(user) {
		handleServiceResponse(c, nil, domain.ForbiddinError)
		return
	}
	id, ok := validId(c)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.DefaultContextTimeout)
	defer cancel()

	if err := h.service.DeleteDeductionType(ctx, id); err != nil {
		handleServiceResponse(c, nil, err)
		return
	}
	handleResponse(c, OK, "DELETED")
}

// region Deductions

// Create godoc
// @Summary      Create deduction
// @Description  Do'kon + oy sarlavhasini yaratadi. Bir do'konga bir oyda bitta sarlavha (takrorlansa 409).
// @Description  total_amount, paid_amount va status qo'lda kiritilmaydi — ular qatorlardan avtomatik hisoblanadi.
// @Tags         deductions
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        input  body  domain.DeductionRequest  true  "Sarlavha"
// @Success      201  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      404  {object}  v1.Response
// @Failure      409  {object}  v1.Response
// @Router       /deduction [post]
func (h *DeductionHandler) Create(c *gin.Context) {
	user, ok := h.signedUser(c)
	if !ok {
		return
	}

	var body domain.DeductionRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.CreateDeduction(ctx, user.UserId, &body)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}
	handleResponse(c, CREATED, res)
}

// List godoc
// @Summary      List deductions
// @Description  Do'kon + oy sarlavhalari. Admin bo'lmagan foydalanuvchi faqat o'z kompaniyasi va do'konini ko'radi.
// @Tags         deductions
// @Security     BearerAuth
// @Produce      json
// @Param        store_id  query  string  false  "Store ID"
// @Param        status    query  string  false  "open yoki paid"
// @Param        year      query  int     false  "Year"
// @Param        month     query  int     false  "Month 1-12"
// @Param        limit     query  int     false  "Limit"
// @Param        offset    query  int     false  "Offset"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Router       /deduction/list [get]
func (h *DeductionHandler) List(c *gin.Context) {
	user, ok := h.signedUser(c)
	if !ok {
		return
	}

	var params domain.DeductionQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, nil, domain.InvalidQueryError)
		return
	}
	if !helper.IsAdmin(user) {
		params.CompanyId = user.CompanyId
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
	}
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.DefaultContextTimeout)
	defer cancel()

	res, totalCount, err := h.service.GetDeductions(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}
	handleResponse(c, OK, utils.ListResponse(res, totalCount, params.Limit, params.Offset))
}

// Get godoc
// @Summary      Get deduction
// @Tags         deductions
// @Security     BearerAuth
// @Produce      json
// @Param        id  path  string  true  "Deduction ID"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      404  {object}  v1.Response
// @Router       /deduction/{id} [get]
func (h *DeductionHandler) Get(c *gin.Context) {
	if _, ok := h.signedUser(c); !ok {
		return
	}
	id, ok := validId(c)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.GetDeductionById(ctx, id)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}
	handleResponse(c, OK, res)
}

// Update godoc
// @Summary      Update deduction
// @Description  Izohni o'zgartiradi va tasdiqlaydi. approve=true bo'lsa approved_by joriy foydalanuvchiga qo'yiladi, false bo'lsa tasdiq bekor qilinadi.
// @Description  total_amount/paid_amount/status bu yerdan o'zgartirilmaydi — ular qatorlardan hisoblanadi.
// @Tags         deductions
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id     path  string                         true  "Deduction ID"
// @Param        input  body  domain.DeductionUpdateRequest  true  "O'zgarishlar"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      404  {object}  v1.Response
// @Router       /deduction/{id} [put]
func (h *DeductionHandler) Update(c *gin.Context) {
	user, ok := h.signedUser(c)
	if !ok {
		return
	}
	id, ok := validId(c)
	if !ok {
		return
	}

	var body domain.DeductionUpdateRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	if body.IsEmpty() {
		handleServiceResponse(c, nil, domain.InvalidRequestBodyError)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.UpdateDeduction(ctx, id, user.UserId, &body)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}
	handleResponse(c, OK, res)
}

// Delete godoc
// @Summary      Delete deduction
// @Description  Sarlavhani va uning BARCHA qatorlarini o'chiradi (CASCADE).
// @Tags         deductions
// @Security     BearerAuth
// @Produce      json
// @Param        id  path  string  true  "Deduction ID"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      403  {object}  v1.Response
// @Failure      404  {object}  v1.Response
// @Router       /deduction/{id} [delete]
func (h *DeductionHandler) Delete(c *gin.Context) {
	user, ok := h.signedUser(c)
	if !ok {
		return
	}
	// Qatorlari bilan birga ketadi — faqat admin
	if !helper.IsAdmin(user) {
		handleServiceResponse(c, nil, domain.ForbiddinError)
		return
	}
	id, ok := validId(c)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.DefaultContextTimeout)
	defer cancel()

	if err := h.service.DeleteDeduction(ctx, id); err != nil {
		handleServiceResponse(c, nil, err)
		return
	}
	handleResponse(c, OK, "DELETED")
}

// region Details

// CreateDetail godoc
// @Summary      Create deduction detail
// @Description  Xodimga ushlab qolish yozadi. store_id, year va month sarlavhadan olinadi — ularni yuborish shart emas.
// @Description  Bir xodimga bir oyda bir necha qator bo'lishi mumkin (masalan ikkita shtraf).
// @Description  Yozilgandan keyin sarlavhaning total_amount/paid_amount/status'i qayta hisoblanadi.
// @Tags         deductions
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        input  body  domain.DeductionDetailRequest  true  "Qator"
// @Success      201  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      404  {object}  v1.Response
// @Router       /deduction-detail [post]
func (h *DeductionHandler) CreateDetail(c *gin.Context) {
	user, ok := h.signedUser(c)
	if !ok {
		return
	}

	var body domain.DeductionDetailRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.CreateDeductionDetail(ctx, user.UserId, &body)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}
	handleResponse(c, CREATED, res)
}

// ListDetails godoc
// @Summary      List deduction details
// @Tags         deductions
// @Security     BearerAuth
// @Produce      json
// @Param        deduction_id       query  string  false  "Deduction ID"
// @Param        employee_id        query  string  false  "Employee ID"
// @Param        store_id           query  string  false  "Store ID"
// @Param        deduction_type_id  query  string  false  "Type ID"
// @Param        is_paid            query  bool    false  "To'langanmi"
// @Param        year               query  int     false  "Year"
// @Param        month              query  int     false  "Month 1-12"
// @Param        limit              query  int     false  "Limit"
// @Param        offset             query  int     false  "Offset"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Router       /deduction-detail/list [get]
func (h *DeductionHandler) ListDetails(c *gin.Context) {
	user, ok := h.signedUser(c)
	if !ok {
		return
	}

	var params domain.DeductionDetailQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, nil, domain.InvalidQueryError)
		return
	}
	if !helper.IsAdmin(user) && user.StoreId != "" {
		params.StoreId = user.StoreId
	}
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.DefaultContextTimeout)
	defer cancel()

	res, totalCount, err := h.service.GetDeductionDetails(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}
	handleResponse(c, OK, utils.ListResponse(res, totalCount, params.Limit, params.Offset))
}

// GetDetail godoc
// @Summary      Get deduction detail
// @Tags         deductions
// @Security     BearerAuth
// @Produce      json
// @Param        id  path  string  true  "Detail ID"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      404  {object}  v1.Response
// @Router       /deduction-detail/{id} [get]
func (h *DeductionHandler) GetDetail(c *gin.Context) {
	if _, ok := h.signedUser(c); !ok {
		return
	}
	id, ok := validId(c)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.GetDeductionDetailById(ctx, id)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}
	handleResponse(c, OK, res)
}

// UpdateDetail godoc
// @Summary      Update deduction detail
// @Description  Hammasi ixtiyoriy — berilgani yoziladi, berilmagani eski qiymatida qoladi.
// @Description  is_paid=true bo'lsa paid_at hozirgi vaqtga, false bo'lsa NULL'ga qo'yiladi.
// @Description  approve=true bo'lsa approved_by joriy foydalanuvchiga qo'yiladi.
// @Description  Har qanday o'zgarishdan keyin sarlavha yig'indilari qayta hisoblanadi.
// @Tags         deductions
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id     path  string                               true  "Detail ID"
// @Param        input  body  domain.DeductionDetailUpdateRequest  true  "O'zgarishlar"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      404  {object}  v1.Response
// @Router       /deduction-detail/{id} [put]
func (h *DeductionHandler) UpdateDetail(c *gin.Context) {
	user, ok := h.signedUser(c)
	if !ok {
		return
	}
	id, ok := validId(c)
	if !ok {
		return
	}

	var body domain.DeductionDetailUpdateRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	if body.IsEmpty() {
		handleServiceResponse(c, nil, domain.InvalidRequestBodyError)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.UpdateDeductionDetail(ctx, id, user.UserId, &body)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}
	handleResponse(c, OK, res)
}

// DeleteDetail godoc
// @Summary      Delete deduction detail
// @Description  Qatorni o'chiradi va sarlavha yig'indilarini qayta hisoblaydi.
// @Tags         deductions
// @Security     BearerAuth
// @Produce      json
// @Param        id  path  string  true  "Detail ID"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      404  {object}  v1.Response
// @Router       /deduction-detail/{id} [delete]
func (h *DeductionHandler) DeleteDetail(c *gin.Context) {
	if _, ok := h.signedUser(c); !ok {
		return
	}
	id, ok := validId(c)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.DefaultContextTimeout)
	defer cancel()

	if err := h.service.DeleteDeductionDetail(ctx, id); err != nil {
		handleServiceResponse(c, nil, err)
		return
	}
	handleResponse(c, OK, "DELETED")
}
