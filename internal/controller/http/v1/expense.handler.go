package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ExpenseHandler struct {
	*Handler
}

func (h *Handler) NewExpenseHandler(r *gin.RouterGroup) {
	autoOrder := &ExpenseHandler{h}
	autoOrder.ExpenseRoutes(r)
}

func (h *ExpenseHandler) ExpenseRoutes(r *gin.RouterGroup) {
	expense := r.Group("/expense")
	{
		expense.POST("/send", h.Send)
	}
}

// CreateExpense godoc
// @Summary Create 1c expense
// @Description Create auto order
// @Security     BearerAuth
// @Tags 	Shift Expenses
// @Accept 	json
// @Produce json
// @Param 	send_date query string true "Send Date (2006-01-02)"
// @Success 201 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /expense/send 	[post]
func (h *ExpenseHandler) Send(c *gin.Context) {
	var (
		sendDate = c.Query("send_date")
		storeID  = c.Query("store_id")
		err      error
	)
	// validate storeID
	if err = uuid.Validate(storeID); err != nil {
		handleResponse(c, BadRequest, "StoreID is required")
		return
	}

	// get shift expense with store and date
	shiftExpense := h.service.CheckShiftExpense(sendDate, storeID)

	// check shift expense sent or not
	if shiftExpense {
		handleResponse(c, BadRequest, "Shift expense already sent for this date")
		return
	}

	// send expense with manual request
	err = h.service.SendExpenseTo1C(sendDate, storeID)
	if err != nil {
		h.log.Warn("ERROR on sending expense: %v", err)
		handleResponse(c, InternalError, "Can't send expense to 1C")
		return
	}

	handleResponse(c, OK, "Sent Successfully")
}
