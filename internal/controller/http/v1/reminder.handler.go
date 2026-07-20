package v1

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/utils"
)

type ReminderHandler struct {
	*Handler
}

func (h *Handler) NewReminderHandler(r *gin.RouterGroup) {
	handler := &ReminderHandler{h}
	handler.ReminderRoutes(r)
}

func (h *ReminderHandler) ReminderRoutes(r *gin.RouterGroup) {
	reminder := r.Group("/reminder")
	{
		reminder.POST("", h.Create)
		reminder.GET("/list", h.List)
		reminder.DELETE("/:id", h.Delete)
	}
}

// Create godoc
// @Summary      Create reminder for stores
// @Description  Admin tomonidan bir yoki bir nechta aptekaga (from_date - to_date oralig'ida ko'rsatiladigan) matnli eslatma yuboriladi. created_by tokendagi user_id orqali avtomatik saqlanadi. Yaratilgandan so'ng belgilangan har bir apteka uchun websocket orqali xabar yuboriladi.
// @Tags         reminder
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        input body domain.CreateReminderRequest true "Reminder data"
// @Success      201 {object} v1.Response
// @Failure      400 {object} v1.Response
// @Failure      401 {object} v1.Response
// @Failure      403 {object} v1.Response
// @Failure      500 {object} v1.Response
// @Router       /reminder [post]
func (h *ReminderHandler) Create(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	if !utils.In(user.Role, constants.AllAdminRoles...) {
		handleServiceResponse(c, nil, domain.ForbiddinError)
		return
	}

	var body domain.CreateReminderRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	result, err := h.service.CreateReminder(ctx, &body, user.UserId)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, CREATED, result)
}

// List godoc
// @Summary      Reminder list
// @Description  Eslatmalar ro'yxati. active=true bo'lsa faqat muddati (to_date) hozirgi vaqtdan hali o'tmagan eslatmalar qaytariladi. Admin bo'lmagan foydalanuvchilar faqat o'z do'koniga tegishli eslatmalarni ko'radi.
// @Tags         reminder
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        store_id  query  string  false  "Store ID (faqat admin uchun filter sifatida ishlaydi)"
// @Param        active    query  bool    false  "true bo'lsa faqat muddati o'tmaganlarni qaytaradi"
// @Param        limit     query  int     false  "Limit"
// @Param        offset    query  int     false  "Offset"
// @Success      200 {object} v1.Response
// @Failure      400 {object} v1.Response
// @Failure      401 {object} v1.Response
// @Failure      500 {object} v1.Response
// @Router       /reminder/list [get]
func (h *ReminderHandler) List(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.ReminderQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	if !utils.In(user.Role, constants.AllAdminRoles...) {
		if user.StoreId == "" {
			handleResponse(c, BadRequest, "store_id not found for user")
			return
		}
		params.StoreId = user.StoreId
	}

	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	results, count, err := h.service.GetReminderList(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, utils.ListResponse(results, count, params.Limit, params.Offset))
}

// Delete godoc
// @Summary      Delete reminder
// @Description  Eslatmani soft delete qiladi: is_active=false qilinadi va deleted_at joriy vaqtga o'rnatiladi (qator jismonan o'chirilmaydi).
// @Tags         reminder
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id  path  string  true  "Reminder ID"
// @Success      200 {object} v1.Response
// @Failure      401 {object} v1.Response
// @Failure      403 {object} v1.Response
// @Failure      404 {object} v1.Response
// @Failure      500 {object} v1.Response
// @Router       /reminder/{id} [delete]
func (h *ReminderHandler) Delete(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	if !utils.In(user.Role, constants.AllAdminRoles...) {
		handleServiceResponse(c, nil, domain.ForbiddinError)
		return
	}

	id := c.Param("id")
	if id == "" {
		handleResponse(c, BadRequest, "id is required")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	if err := h.service.DeleteReminder(ctx, id); err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, "DELETED")
}
