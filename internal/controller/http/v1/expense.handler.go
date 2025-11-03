package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
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
		expense.POST("/send-temporary", h.SendTemporary)
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
		h.log.Errorf("could not send expenses to onec: %v", err)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
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
// @Param 	send_date query string true "send_date (2006-01-02)"
// @Param 	store_id query string true "store_id(required)"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /expense/send-with-number 	[post]
func (h *ExpenseHandler) SendWithNumber(c *gin.Context) {
	var (
		sendDate = c.Query("send_date")
		storeId  = c.Query("store_id")
	)
	// Validate required parameters
	if sendDate == "" || storeId == "" {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	// send expense with manual request
	err := h.service.SendExpenseWithNumberToOnec(sendDate, storeId)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
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
// @Router /expense/send-temporary 	[post]
func (h *ExpenseHandler) SendTemporary(c *gin.Context) {
	var sendDate = c.Query("send_date")

	// send expense with manual request
	go h.service.SendChequesTemporary(sendDate)

	handleResponse(c, OK, "Sent Successfully")
}
