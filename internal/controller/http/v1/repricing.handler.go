package v1

import (
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/xuri/excelize/v2"
)

type RepricingHandler struct {
	*Handler
}

func (h *Handler) NewRepricingHandler(r *gin.RouterGroup) {
	returnHandler := &RepricingHandler{h}
	returnHandler.RepricingRoutes(r)
}

func (h *RepricingHandler) RepricingRoutes(r *gin.RouterGroup) {
	repricing := r.Group("/repricing")
	{
		repricing.POST("", h.Create)
		repricing.GET("/:id", h.Get)
		repricing.GET("/list", h.List)
		repricing.GET("/export-excel", h.ExportRepricingExcel)
		repricing.POST("/confirm/:id", h.Confirm)
		repricing.POST("/cancel/:id", h.Cancel)
	}
	// detail := r.Group("repricing-detail")
	// {
	// 	// detail.GET("/list", h.ReturnDetailList)
	// 	// detail.GET("/export-excel", h.ExportReturnDetailList)
	// }
}

// Create godoc
// @Summary Create Repricing
// @Description Create Repricing
// @Tags Repricing
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	repricing body domain.RepricingRequest true "Repricing"
// @Success 201 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /repricing [POST]
func (h *RepricingHandler) Create(c *gin.Context) {
	var request domain.RepricingRequest
	// Bind the request body to the ReturnRequest struct
	if err := c.ShouldBindJSON(&request); err != nil {
		h.log.Warn("Error on binding request: %v", err.Error())
		handleResponse(c, BadRequest, "Invalid request body")
		return
	}
	// get user_id from the set header
	userId, ok := c.Get("user_id")
	if !ok {
		h.log.Warn("Error on getting user id from context")
		handleResponse(c, BadRequest, "User not authorized")
		return
	}
	request.CreatedBy = userId.(string) // get creator id from set header

	// create repricing
	_, err := h.service.CreateRepricing(&request)
	if err != nil {
		h.log.Warn("Error on creating repricing: %v", err)
		handleResponse(c, InternalError, "Failed to create repricing")
		return
	}

	handleResponse(c, CREATED, "CREATED")
}

// Get godoc
// @Summary Get Repricing
// @Description Get Repricing
// @Tags Repricing
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id path string true "Repricing ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /repricing/{id} [GET]
func (h *RepricingHandler) Get(c *gin.Context) {
	// get return by id
	id := c.Param("id")
	// validate id
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid repricing id")
		return
	}
	// get repricing by id
	res, err := h.service.GetRepricingByID(id)
	if err != nil {
		h.log.Warn("Error on getting repricing: %v", err.Error())
		handleResponse(c, InternalError, "Failed to get repricing")
		return
	}

	handleResponse(c, OK, res)
}

// List godoc
// @Summary List Repricing
// @Description List Repricing
// @Tags Repricing
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	limit query int false "Limit"
// @Param 	offset query int false "Offset"
// @Param   search query string false "Search"
// @Param   store_id query string false "Store ID"
// @Param   start_date 	query string false "Start Date"
// @Param   end_date 	query string false "End Date"
// @Param   status 	query string false "Status"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /repricing/list [get]
func (h *RepricingHandler) List(c *gin.Context) {
	var param domain.QueryParam

	if err := c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	res, totalCount, err := h.service.RepricingList(&param)
	if err != nil {
		h.log.Warn("ERROR on getting repricing list: %v", err)
		handleResponse(c, InternalError, "Failed to get repricing list")
		return
	}

	data := utils.ListResponse(res, totalCount, param.Limit, param.Offset)

	handleResponse(c, OK, data)
}

// List godoc
// @Summary List Repricing Export excel
// @Description List Repricing
// @Tags Repricing
// @Security     BearerAuth
// @Accept json
// @Produce json
// @Param 	limit query int false "Limit"
// @Param 	offset query int false "Offset"
// @Param   search query string false "Search"
// @Param   store_id query string false "Store ID"
// @Param   start_date 	query string false "Start Date"
// @Param   end_date 	query string false "End Date"
// @Param   status 	query string false "Status"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /repricing/export-excel [get]
func (h *RepricingHandler) ExportRepricingExcel(c *gin.Context) {
	var param domain.QueryParam

	if err := c.ShouldBindQuery(&param); err != nil {
		handleResponse(c, BadRequest, "Invalid query param")
		return
	}
	param.Limit, param.Offset = defaultLimitOffset(param.Limit, param.Offset)

	res, _, err := h.service.RepricingList(&param)
	if err != nil {
		h.log.Warn("ERROR on getting repricing list: %v", err)
		handleResponse(c, InternalError, "Failed to get repricing list")
		return
	}

	// Create excel file
	f := excelize.NewFile()
	sheetName := "List1"
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"ID", "Название", "Филиал", "Тип", "Кол-во", "Статус", "Создал", "Завершил", "Дата переоценки"}

	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Color: "000000",
		},
	})
	if err != nil {
		h.log.Error("Failed to create style:", err)
		handleResponse(c, InternalError, "Error on giving style to excel")
		return
	}

	for i, h := range headers {
		col := string(rune('A'+i)) + "1"
		f.SetCellValue(sheetName, col, h)
		f.SetCellStyle(sheetName, col, col, headerStyle)
	}

	// Ma'lumotlarni qo'shish
	for i, imp := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, imp.Id)
		f.SetCellValue(sheetName, "B"+row, imp.Name)
		if imp.Store != nil {
			f.SetCellValue(sheetName, "C"+row, imp.Store.Name)
		} else {
			f.SetCellValue(sheetName, "C"+row, "N/A")
		}
		f.SetCellValue(sheetName, "D"+row, imp.Type)
		f.SetCellValue(sheetName, "E"+row, imp.Count)
		f.SetCellValue(sheetName, "F"+row, imp.Status)
		if imp.CreatedBy != nil {
			f.SetCellValue(sheetName, "G"+row, imp.CreatedBy.FirstName)
		} else {
			f.SetCellValue(sheetName, "G"+row, "N/A")
		}
		if imp.CreatedBy != nil {
			f.SetCellValue(sheetName, "H"+row, imp.UpdatedBy.FirstName)
		} else {
			f.SetCellValue(sheetName, "H"+row, "N/A")
		}
		f.SetCellValue(sheetName, "I"+row, imp.CreatedAt.Format(time.DateTime))

	}

	// Faylni uploads/ papkasiga UUID bilan saqlash
	fileName := "Qayta_baholash_" + time.Now().Add(time.Hour*5).Format("2006-01-02_15-04-05") + ".xlsx"
	filePath := filepath.Join("uploads", fileName)

	// uploads/ papkasi mavjud bo‘lmasa, yaratish
	if _, err := os.Stat("uploads"); os.IsNotExist(err) {
		err := os.Mkdir("uploads", os.ModePerm)
		if err != nil {
			h.log.Error("Failed to create uploads directory:", err)
			handleResponse(c, InternalError, "Failed to create uploads folder")
			return
		}
	}

	// Faylni diskka yozish
	if err := f.SaveAs(filePath); err != nil {
		h.log.Error("Failed to save Excel file:", err)
		handleResponse(c, InternalError, "Failed to save Excel file")
		return
	}

	// Foydalanuvchiga file path yoki URLni qaytarish
	handleResponse(c, OK, gin.H{
		"file_name": fileName,
	})
}

// confirm repricing
// @Summary Confirm Repricing
// @Description Confirm Repricing
// @Tags Repricing
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id 	path string true "Repricing ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /repricing/confirm/{id} [POST]
func (h *RepricingHandler) Confirm(c *gin.Context) {
	var id = c.Param("id")
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid repricing id")
		return
	}
	// get user id from the context
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "user id not found from the context")
		return
	}
	// confirm repricing service
	err := h.service.ConfirmRepricing(id, userId.(string))
	if err != nil {
		handleResponse(c, InternalError, "Failed to confirm repricing")
		return
	}

	handleResponse(c, OK, "CONFIRMED")
}

// cancel Repricing
// @Summary Cancel Repricing
// @Description Cancel Repricing
// @Tags Repricing
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	id 	path string true "Repricing ID"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /repricing/cancel/{id} [POST]
func (h *RepricingHandler) Cancel(c *gin.Context) {
	var id = c.Param("id")
	// validate request uuid
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid repricing id")
		return
	}
	// get user id from the context
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "user id not found from the context")
		return
	}
	// confirm repricing service
	err := h.service.CancelRepricing(id, userId.(string))
	if err != nil {
		handleResponse(c, InternalError, "Failed to cancel repricing")
		return
	}

	handleResponse(c, OK, "CANCELED")
}
