package v1

import (
	"github.com/gin-gonic/gin"
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
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /expense/send 	[post]
func (h *ExpenseHandler) Send(c *gin.Context) {
	var (
		sendDate = c.Query("send_date")
		err      error
	)

	// send expense with manual request
	err = h.service.SendExpenseTo1C(sendDate)
	if err != nil {
		h.log.Warn("ERROR on sending expense: %v", err)
		handleResponse(c, InternalError, "Can't send expense to 1C")
		return
	}

	handleResponse(c, OK, "Sent Successfully")
}
