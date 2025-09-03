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
		expense.POST("/send-with-number", h.SendWithNumber)
		expense.POST("/expense-given-excel", h.SendFromExcel)
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

// CreateExpense godoc
// @Summary Create 1c expense
// @Description Create auto order
// @Security     BearerAuth
// @Tags 	Shift Expenses
// @Accept 	json
// @Produce json
// @Param 	send_date query string true "Send Date (2006-01-02)"
// @Param 	store_id query string true "Store ID (required)"
// @Param 	send_number query string true "Send Number (required)"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /expense/send-with-number 	[post]
func (h *ExpenseHandler) SendWithNumber(c *gin.Context) {
	var (
		sendDate   = c.Query("send_date")
		storeID    = c.Query("store_id")
		sendNumber = c.Query("send_number")
		err        error
	)
	// Validate required parameters
	if sendDate == "" || storeID == "" || sendNumber == "" {
		handleResponse(c, BadRequest, "send_date, store_id and send_number are required")
		return
	}

	// send expense with manual request
	err = h.service.SendExpenseWithNumberTo1C(sendDate, storeID, sendNumber)
	if err != nil {
		h.log.Warn("ERROR on sending expense: %v", err)
		handleResponse(c, InternalError, "Can't send expense to 1C")
		return
	}

	handleResponse(c, OK, "Sent Successfully")
}

// SendFromExcel godoc
// @Summary Send expenses to 1C from Excel
// @Description Read Excel (ID, Дата) and send each to 1C
// @Security BearerAuth
// @Tags Shift Expenses
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Excel File"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /expense/expense-given-excel [post]
func (h *ExpenseHandler) SendFromExcel(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		handleResponse(c, BadRequest, "Excel file required")
		return
	}

	filePath := "/tmp/" + file.Filename
	if err = c.SaveUploadedFile(file, filePath); err != nil {
		handleResponse(c, InternalError, "Cannot save uploaded file")
		return
	}

	err = h.service.SendExpenseTo1CFromExcel(filePath)
	if err != nil {
		h.log.Warn("ERROR on sending expense from excel: %v", err)
		handleResponse(c, InternalError, "Can't send expense to 1C from excel")
		return
	}

	handleResponse(c, OK, "Sent Successfully from Excel")
}
