package v1

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
	"github.com/pharma-crm-backend/pkg/etc"
	"github.com/pharma-crm-backend/pkg/helper"
	"github.com/pharma-crm-backend/pkg/utils"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type EmployeeHandler struct {
	*Handler
}

func (h *Handler) NewEmployeeHandler(r *gin.RouterGroup) {
	employee := &EmployeeHandler{h}
	employee.EmployeeRoutes(r)
}

func (h *EmployeeHandler) EmployeeRoutes(r *gin.RouterGroup) {
	employee := r.Group("/employee")
	{
		employee.POST("", h.Create)
		employee.GET("/:id", h.Get)
		employee.GET("/list", h.List)
		employee.GET("/export-excel", h.ExportEmployeeExcel)
		employee.PUT("/:id", h.Update)
		employee.DELETE("/delete", h.Delete)
		employee.GET("/info", h.GetInfo)
		employee.PUT("/reset-password", h.ResetPassword)
		employee.PUT("/info", h.UpdateEmployeeinfo)
		employee.PUT("/:id/status", h.UpdateEmployeeStatus)
		employee.PUT("/block", h.BlockEmployee)
		employee.PUT("/unblock", h.UnBlockEmployee)
		employee.GET("/bonus", h.SmenaBonus)
		employee.POST("/attendance-face-id", h.CheckInOut)
		employee.DELETE("/attendance-face-id/:id", h.DeleteAttendanceFaceId)
		employee.DELETE("/attendance-face-id/cleanup-old", h.CleanupOldAttendanceFaceIds)
		employee.GET("/attendance/list", h.AttendanceList)
		employee.GET("/attendance/stats", h.AttendanceStats)
		employee.GET("/attendance-days/list", h.EmployeeAttendanceDayList)
		employee.POST("/attendance-manual", h.CreateAttendanceLogManual)
		employee.PUT("/attendance-logs/:id", h.UpdateAttendanceLog)
		employee.DELETE("/attendance-logs/:id", h.DeleteAttendanceLog)
		employee.PATCH("/:id/face-descriptor", h.CreateOrUpdateFaceDescriptor)
		employee.GET("/:id/face-descriptor", h.GetEmployeeFaceDescriptor)
		employee.DELETE("/:id/face-descriptor", h.DeleteEmployeeFaceDescriptor)
		employee.GET("/list/face-descriptor", h.ListEmployeeFaceDescriptors)
		employee.GET("/payroll/stores", h.StorePayrollList)
		employee.GET("/payroll/employees", h.EmployeePayrollList)
		employee.GET("/payroll/my", h.MyPayroll)
		employee.POST("/payroll/recalculate", h.RecalculatePayroll)
		employee.GET("/payroll/management", h.EmployeePayrollManagementList)
		employee.GET("/payroll/management/statistics", h.PayrollManagementStatistics)
		employee.GET("/payroll/employees/statistics", h.PayrollStatistics)
		employee.GET("/payroll/stores/statistics", h.StorePayrollStatistics)
		employee.GET("/payroll/stores/export-excel", h.ExportStorePayrollExcel)
		employee.GET("/payroll/employees/export-excel", h.ExportEmployeePayrollExcel)
		employee.PUT("/payroll/:id/management", h.UpdateEmployeePayrollManagement)
	}
}

// floatOrZero — nil bo'lsa 0 ko'rsatuvchi pointer qaytaradi (NOT NULL ustunlar uchun).
func floatOrZero(v *float64) *float64 {
	if v != nil {
		return v
	}
	zero := 0.0
	return &zero
}

// nullIfBlank — bo'sh (yoki faqat probel) string uchun nil qaytaradi.
// DATE ustunlariga "" yozilsa Postgres 22007 xatosini beradi, shuning uchun
// bunday qiymatlar NULL sifatida saqlanadi.
func nullIfBlank(v *string) *string {
	if v == nil || strings.TrimSpace(*v) == "" {
		return nil
	}
	return v
}

// isPhoneTaken — telefon raqami o'chirilmagan boshqa xodimga biriktirilganmi.
// Login telefon raqami bo'yicha ishlagani uchun raqam butun tizim bo'ylab yagona
// bo'lishi kerak. excludeId berilsa (update), o'sha xodimning o'zi hisobga olinmaydi.
func (h *EmployeeHandler) isPhoneTaken(c *gin.Context, phone, excludeId string) (bool, error) {
	query := h.db.WithContext(c.Request.Context()).
		Table("employees").
		Where("phone = ? AND deleted_at IS NULL", phone)
	if excludeId != "" {
		query = query.Where("id <> ?", excludeId)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// @Summary      Create employee
// @Description  Create a new employee in the system.
// @Tags         employees
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        input body     domain.EmployeeRequest  true  "Employee data"
// @Success      201  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      403  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee [post]
func (h *EmployeeHandler) Create(c *gin.Context) {
	var (
		body = domain.EmployeeRequest{}
		err  error
	)
	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	if !utils.IsValidPhone(body.Phone) {
		handleResponse(c, BadRequest, "Invalid phone number, Format: 998901234567")
		return
	}
	// telefon raqami band emasligini tekshirish
	taken, err := h.isPhoneTaken(c, body.Phone, "")
	if err != nil {
		h.log.Errorf("ERROR on checking employee phone uniqueness: %v", err)
		handleResponse(c, InternalError, "Can't check employee phone number")
		return
	}
	if taken {
		handleServiceResponse(c, nil, domain.DuplicatePhoneError)
		return
	}

	hashedPassword, err := etc.Encrypt(*body.Password, h.cfg.HashKey)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	body.Password = &hashedPassword
	body.Id = uuid.New().String()
	body.Status = "active"
	body.FullName = body.FirstName + " " + body.LastName
	// bo'sh sana qiymatlari NULL bo'lishi kerak (DATE ustunlari "" ni qabul qilmaydi)
	body.Birthdate = nullIfBlank(body.Birthdate)
	body.StartDate = nullIfBlank(body.StartDate)
	body.EndDate = nullIfBlank(body.EndDate)
	// create employee
	err = h.db.
		WithContext(c.Request.Context()).
		Table("employees").
		Create(&body).Error

	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	// create employee_roles
	if len(body.RoleIds) > 0 {
		var employeeRoles []domain.EmployeeRole
		for _, roleId := range body.RoleIds {
			employeeRoles = append(employeeRoles, domain.EmployeeRole{
				Id:         uuid.New().String(),
				EmployeeId: body.Id,
				RoleId:     roleId,
			})
		}
		err = h.db.WithContext(c.Request.Context()).Create(&employeeRoles).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
	}

	handleResponse(c, CREATED, "CREATED")
}

// @Summary      Get employee
// @Description  Get an employee by id
// @Tags         employees
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id   path     string  true  "Employee id"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      403  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee/{id} [get]
func (h *EmployeeHandler) Get(c *gin.Context) {
	var res domain.Employee
	var id = c.Param("id")
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid id")
		return
	}

	err := h.db.
		Preload("Store").
		Preload("Roles").
		Preload("Company").
		First(&res, "id = ?", id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, "Employee not found")
			return
		}
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, res)
}

// @Summary      List employees
// @Description  List all employees
// @Tags         employees
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        limit          query     int             false "Limit"
// @Param        offset         query     int             false "Offset"
// @Param        search         query     string          false "Search"
// @Param        role_id        query     string          false "Role ID"
// @Param        store_id       query     string          false "Store ID"
// @Param        status 		query     string          false "Status (deleted || blocked || active)"
// @Param        is_dismissed   query     bool            false "false — ishdan bo'shatilganlarni (Уволен) chiqarib tashlaydi; true — faqat o'shalarni; berilmasa hammasi"
// @Success      200  {array}   v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      403  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee/list [get]
func (h *EmployeeHandler) List(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}
	var params domain.EmployeeQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, nil, domain.InvalidQueryError)
		return
	}
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	if !helper.IsAdmin(user) {
		params.CompanyId = user.CompanyId
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()
	// get employee list data
	res, totalCount, err := h.service.GetEmployees(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}
	// add _meta for pagination response
	result := utils.ListResponse(res, totalCount, params.Limit, params.Offset)

	handleResponse(c, OK, result)
}

// @Summary      Download employee list as Excel
// @Description  Export filtered employee list to an Excel file
// @Tags         employees
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        role_id        query     string   false "Role ID"
// @Param        store_id       query     string   false "Store ID"
// @Param        search         query     string   false "Search"
// @Param        status         query     string   false "Status (deleted || blocked || active)"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee/export-excel [get]
func (h *EmployeeHandler) ExportEmployeeExcel(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}
	var params domain.EmployeeQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		handleServiceResponse(c, nil, domain.InvalidQueryError)
		return
	}
	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	if !helper.IsAdmin(user) {
		params.CompanyId = user.CompanyId
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()
	// get employee list data
	res, _, err := h.service.GetEmployees(ctx, &params)
	if err != nil {
		handleServiceResponse(c, InternalError, err)
		return
	}

	// Excel fayl yaratish
	f := excelize.NewFile()
	sheetName := constants.DefaultSheetName
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"ID", "ФИО", "Филиал", "Телефон", "Роль", "Статус"}

	err = setExcelHeaders(f, sheetName, headers)
	if err != nil {
		h.log.Error("Failed to create style:", err)
		handleResponse(c, InternalError, "Error on giving style to excel")
		return
	}

	// Ma'lumotlarni qo'shish
	for i, emp := range res {
		row := strconv.Itoa(i + 2)
		f.SetCellValue(sheetName, "A"+row, emp.PublicId)
		f.SetCellValue(sheetName, "B"+row, emp.FullName)
		if emp.Store != nil {
			f.SetCellValue(sheetName, "C"+row, emp.Store.Name)
		} else {
			f.SetCellValue(sheetName, "C"+row, "N/A")
		}

		f.SetCellValue(sheetName, "D"+row, emp.Phone)

		// Agar employee bir nechta rolga ega bo‘lsa, ularni vergul bilan ajratib yozamiz
		var roles []string
		for _, role := range emp.Roles {
			roles = append(roles, role.Name)
		}
		f.SetCellValue(sheetName, "E"+row, strings.Join(roles, ", "))
		f.SetCellValue(sheetName, "F"+row, emp.Status)
	}
	saveExcelToUploads(c, f, *h.log, "Xodimlar")
}

// @Summary      Update employee
// @Description  Update an employee.
// @Tags         employees
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id            path     string    true  "Employee id"
// @Param        input         body  domain.EmployeeRequest true  "Employee data"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      403  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee/{id} [put]
func (h *EmployeeHandler) Update(c *gin.Context) {
	var (
		body = domain.EmployeeRequest{}
		id   = c.Param("id")
		err  error
	)
	// bind request body
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// validate phone number
	if !utils.IsValidPhone(body.Phone) {
		handleResponse(c, BadRequest, "Invalid phone number, Format: 998901234567")
		return
	}
	// telefon raqami boshqa xodimda band emasligini tekshirish
	taken, err := h.isPhoneTaken(c, body.Phone, id)
	if err != nil {
		h.log.Errorf("ERROR on checking employee phone uniqueness: %v", err)
		handleResponse(c, InternalError, "Can't check employee phone number")
		return
	}
	if taken {
		handleServiceResponse(c, nil, domain.DuplicatePhoneError)
		return
	}
	// collect full_name by adding first and last name
	body.FullName = body.FirstName + " " + body.LastName
	// bo'sh sana qiymatlari NULL bo'lishi kerak (DATE ustunlari "" ni qabul qilmaydi)
	body.Birthdate = nullIfBlank(body.Birthdate)
	body.StartDate = nullIfBlank(body.StartDate)
	body.EndDate = nullIfBlank(body.EndDate)
	// check password password not nill
	if body.Password != nil {
		// get encrypted new password
		*body.Password, err = etc.Encrypt(*body.Password, h.cfg.HashKey)
		if err != nil {
			h.log.Warn("ERROR on encrypting new password: %v", err)
			handleResponse(c, InternalError, "Can't encrypt new password")
			return
		}
	}
	if len(body.RoleIds) > 0 { // checking employee roles received or no with length
		// clean employee roles data which is depends on the employee
		err = h.db.WithContext(c.Request.Context()).
			Delete(&domain.EmployeeRole{}, "employee_id = ?", id).Error
		if err != nil {
			h.log.Warn("ERROR on deleting employee roles: %v", err)
			handleResponse(c, InternalError, "Can't delete employee roles")
			return
		}
		var employeeRoles []domain.EmployeeRole
		for _, roleId := range body.RoleIds {
			// collect new employee roles info
			employeeRoles = append(employeeRoles, domain.EmployeeRole{
				Id:         uuid.New().String(),
				EmployeeId: id,
				RoleId:     roleId,
			})
		}
		// create new employee roles
		err = h.db.WithContext(c.Request.Context()).Create(&employeeRoles).Error
		if err != nil {
			h.log.Warn("ERROR on creating employee roles on update: %v", err)
			handleResponse(c, InternalError, "Can't update employee roles")
			return
		}
	}
	// update employee information
	updateData := map[string]any{
		"full_name":  body.FullName,
		"first_name": body.FirstName,
		"last_name":  body.LastName,
		"company_id": body.CompanyId,
		"phone":      body.Phone,
		"gender":     body.Gender,
		"language":   body.Language,
		"birthdate":  body.Birthdate,
		"store_ids":  body.StoreIds,
		"start_date": body.StartDate,
		"end_date":   body.EndDate,
		"role_type":  body.RoleType,
	}

	if body.Passport != nil {
		updateData["passport"] = *body.Passport
	}

	if body.Password != nil {
		updateData["password"] = *body.Password
	}

	// Explicitly set store_id to NULL if it is not provided in the request
	var (
		currentStoreId *string
		storeIdChanged bool
	)
	var currentEmployee struct {
		StoreId *string `gorm:"column:store_id"`
	}
	if err = h.db.WithContext(c.Request.Context()).
		Table("employees").
		Select("store_id").
		Where("id = ?", id).
		Scan(&currentEmployee).Error; err != nil {
		h.log.Warn("ERROR on getting current employee store_id: %v", err)
		handleResponse(c, InternalError, "Can't get employee data")
		return
	}
	currentStoreId = currentEmployee.StoreId

	switch {
	case currentStoreId == nil && body.StoreId == nil:
		storeIdChanged = false
	case currentStoreId == nil && body.StoreId != nil:
		storeIdChanged = true
	case currentStoreId != nil && body.StoreId == nil:
		storeIdChanged = true
	default:
		storeIdChanged = *currentStoreId != *body.StoreId
	}

	if storeIdChanged {
		var openCashbox struct {
			ID string `gorm:"column:id"`
		}
		if err = h.db.WithContext(c.Request.Context()).
			Raw(`SELECT id FROM cashbox_operations WHERE current_employee_id = ? AND is_open = TRUE LIMIT 1`, id).
			Scan(&openCashbox).Error; err != nil {
			h.log.Warn("ERROR on checking open cashbox: %v", err)
			handleResponse(c, InternalError, "Can't check employee cashbox status")
			return
		}
		if openCashbox.ID != "" {
			handleResponse(c, BadRequest, "Employee has an open cashbox. Please close it before changing the store")
			return
		}
	}

	updateData["store_id"] = body.StoreId

	err = h.db.WithContext(c.Request.Context()).
		Table("employees").
		Where("id = ?", id).
		Updates(updateData).Error
	if err != nil {
		h.log.Warn("ERROR on updating employee info: %v", err)
		handleResponse(c, InternalError, "Can't update employee data")
		return
	}

	if storeIdChanged {
		oldStoreId := ""
		if currentStoreId != nil {
			oldStoreId = *currentStoreId
		}
		newStoreId := ""
		if body.StoreId != nil {
			newStoreId = *body.StoreId
		}
		go h.service.HandleEmployeeStoreChange(oldStoreId, newStoreId)
	}

	handleResponse(c, OK, body)
}

// @Summary      Delete employees
// @Description  Delete employees by ids
// @Tags         employees
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body  body     []string  true  "Employee ids"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      403  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee/delete [DELETE]
func (h *EmployeeHandler) Delete(c *gin.Context) {
	var ids []string
	if err := c.ShouldBindJSON(&ids); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, "Invalid input")
		return
	}

	if len(ids) == 0 {
		handleResponse(c, BadRequest, "No employee IDs provided")
		return
	}

	var openCashbox struct {
		EmployeeId string `gorm:"column:current_employee_id"`
	}
	if err := h.db.WithContext(c.Request.Context()).
		Raw(`SELECT current_employee_id FROM cashbox_operations WHERE current_employee_id IN (?) AND is_open = TRUE LIMIT 1`, ids).
		Scan(&openCashbox).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, "Can't check employee cashbox status")
		return
	}
	if openCashbox.EmployeeId != "" {
		handleResponse(c, BadRequest, "Your cash box is open, close your cash box operations")
		return
	}

	err := h.db.
		WithContext(c.Request.Context()).
		Table("employees").
		Where("id IN (?)", ids).
		Updates(map[string]any{
			"status":     "deleted",
			"is_active":  false,
			"deleted_at": time.Now(),
		}).Error

	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, "DELETED")
}

// @Summary      Get employee info
// @Description  Get employee info
// @Tags         employees
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      403  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee/info [get]
func (h *EmployeeHandler) GetInfo(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	var res domain.Employee
	if err := h.db.
		Preload("Store").
		First(&res, "id = ?", userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, OK, "Employee not found")
			return
		}
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	err := h.db.Raw(`
	SELECT
		p.*,
		COALESCE(NULLIF(p.route, ''), p.key) AS route
	FROM permissions p
	JOIN role_permissions rp ON rp.permission_id = p.id
	JOIN employee_roles er ON er.role_id = rp.role_id
	WHERE er.employee_id = ?

	UNION

	SELECT
		pp.*,
		COALESCE(NULLIF(pp.route, ''), pp.key) AS route
	FROM permissions pp
	WHERE pp.id IN (
		SELECT DISTINCT p.parent_id
		FROM permissions p
		JOIN role_permissions rp ON rp.permission_id = p.id
		JOIN employee_roles er ON er.role_id = rp.role_id
		WHERE er.employee_id = ?
		AND p.parent_id IS NOT NULL
	);
	`, userID, userID).Scan(&res.Permission).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	err = h.db.Raw(`
	SELECT roles.* FROM roles
	JOIN employee_roles er ON er.role_id = roles.id
	WHERE er.employee_id = ?
	`, userID).Scan(&res.Roles).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	err = h.db.Raw(`
	SELECT
		cb.id, 
		co.id AS cashbox_operation_id,
		co.operation_id, 
		cb.name, 
		cb.created_at, 
		cb.updated_at
	FROM 
		cashbox_operations co 
	JOIN 
		cash_boxes cb ON co.cash_box_id = cb.id
	WHERE 
		co.end_time IS NULL AND 
		co.current_employee_id = ?
	`, userID).Scan(&res.Cashbox).Error

	var lastAttendance domain.AttendanceLog
	err = h.db.Raw(`
	SELECT id, store_id, employee_id, event_type, event_at, created_at, updated_at
	FROM attendance_logs
	WHERE employee_id = ?
	ORDER BY created_at DESC
	LIMIT 1
	`, userID).Scan(&lastAttendance).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		h.log.Error(err)
		handleResponse(c, InternalError, "Failed to get attendance info")
		return
	}
	if err == nil {
		res.LastAttendance = &lastAttendance
	}

	err = h.db.Raw(`
	SELECT
		cp.id,
		cp.name,
		cp.email,
		cp.legal_name,
		cp.legal_address,
		cp.postal_code,
		cp.company_inn,
		cp.company_mfo,
		cp.phone,
		cp.country,
		cp.city,
		cp.is_franchise,
		cp.created_at,
		cp.updated_at
	FROM companies cp
	JOIN employees e ON e.company_id = cp.id
	WHERE e.id = ?
	`, userID).Scan(&res.Company).Error

	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		h.log.Error(err)
		handleResponse(c, InternalError, "Failed to get cashbox info")
		return
	}
	handleResponse(c, OK, res)
}

// @Summary      Reset employee password
// @Description  Reset employee password
// @Tags         employees
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        input         body  domain.ResetPasswordRequest true  "Password data"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      403  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee/reset-password [put]
func (h *EmployeeHandler) ResetPassword(c *gin.Context) {
	var (
		body = domain.ResetPasswordRequest{}
		err  error
	)
	userID, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}

	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	if body.NewPassword != body.ConfirmPassword {
		handleResponse(c, BadRequest, "Password and confirm password do not match")
		return
	}

	hashedPassword, err := etc.Encrypt(body.ConfirmPassword, h.cfg.HashKey)
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	err = h.db.WithContext(c.Request.Context()).
		Table("employees").
		Where("id = ?", userID).
		Update("password", hashedPassword).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, "UPDATED")
}

// @Summary      Update employee info
// @Description  Update employee info
// @Tags         employees
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        input         body  domain.EmployeeUpdateInfoRequest true  "Employee data"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      403  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee/info [put]
func (h *EmployeeHandler) UpdateEmployeeinfo(c *gin.Context) {
	var body domain.EmployeeUpdateInfoRequest
	userId, ok := c.Get("user_id")
	if !ok {
		handleResponse(c, UNAUTHORIZED, "User ID not found")
		return
	}
	err := c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, BadRequest, err.Error())
		return
	}
	err = h.db.WithContext(c.Request.Context()).
		Table("employees").
		Where("id = ?", userId).
		Updates(&body).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}

// @Summary      Update employee status
// @Description  Switch an employee between "active" (Активный) and "dismissed" (Уволен). A dismissed employee can no longer log in.
// @Tags         employees
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id    path     string                             true  "Employee ID"
// @Param        body  body     domain.EmployeeStatusUpdateRequest true  "New status"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      404  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee/{id}/status [put]
func (h *EmployeeHandler) UpdateEmployeeStatus(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var (
		body domain.EmployeeStatusUpdateRequest
		id   = c.Param("id")
	)
	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Errorf("could not bind employee status request body: %v", err)
		handleServiceResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.DefaultContextTimeout)
	defer cancel()

	updates := map[string]any{
		"status":     body.Status,
		"updated_by": user.UserId,
	}
	// Reactivating undoes a block as well, mirroring UnBlockEmployee.
	if body.Status == constants.GeneralStatusActive {
		updates["is_active"] = true
	}

	result := h.db.WithContext(ctx).
		Model(&domain.Employee{}).
		Where("id = ? AND status != ?", id, constants.GeneralStatusDeleted).
		Updates(updates)
	if result.Error != nil {
		h.log.Errorf("could not update employee status: %v", result.Error)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
		return
	}
	if result.RowsAffected == 0 {
		handleServiceResponse(c, NotFound, domain.NotFoundError)
		return
	}

	handleResponse(c, OK, body)
}

// @Summary      Block employee
// @Description  Block employee by id
// @Tags         employees
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body  body     []string  true  "Employee ids"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      403  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee/block [put]
func (h *EmployeeHandler) BlockEmployee(c *gin.Context) {
	var ids []string
	if err := c.ShouldBindJSON(&ids); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, "Invalid input")
		return
	}
	if len(ids) == 0 {
		handleResponse(c, BadRequest, "No employee IDs provided")
		return
	}
	// O'chirilgan xodimning statusi "deleted" bo'lib qolishi kerak, aks holda
	// keyingi unblock uni tiriltirib yuboradi.
	err := h.db.
		WithContext(c.Request.Context()).
		Table("employees").
		Where("id IN (?)", ids).
		Where("status <> ?", constants.GeneralStatusDeleted).
		Update("status", constants.GeneralStatusBlocked).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "BLOCKED")
}

// @Summary      Unblock employee
// @Description  Unblock employee by id
// @Tags         employees
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body  body     []string  true  "Employee ids"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      403  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee/unblock [put]
func (h *EmployeeHandler) UnBlockEmployee(c *gin.Context) {
	var ids []string
	if err := c.ShouldBindJSON(&ids); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, "Invalid input")
		return
	}
	if len(ids) == 0 {
		handleResponse(c, BadRequest, "No employee IDs provided")
		return
	}
	// DIQQAT: ikkita ketma-ket .Update() ishlatmang. Update — finisher metod;
	// birinchisi bajarilgach Statement.SQL to'lib qoladi va ikkinchi chaqiruv
	// yangi SET qurmasdan AYNAN o'sha SQL'ni qayta bajaradi. Natijada status
	// hech qachon yozilmasdan faqat is_active = true bo'lardi — o'chirilgan
	// xodim (status = "deleted") tirilib, login qilib ketardi.
	//
	// O'chirilgan xodim unblock orqali qaytarilmaydi: uni tiklash alohida amal.
	err := h.db.
		WithContext(c.Request.Context()).
		Table("employees").
		Where("id IN (?)", ids).
		Where("status <> ?", constants.GeneralStatusDeleted).
		Where("deleted_at IS NULL").
		Updates(map[string]any{
			"is_active": true,
			"status":    constants.GeneralStatusActive,
		}).Error
	if err != nil {
		h.log.Error(fmt.Errorf("err: %v", err))
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "UNBLOCKED")
}

// @Summary      Smena bonus
// @Description  Get smena bonus by employee id
// @Tags         employees
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        operation_id  query 	string     true  "Operation ID"
// @Param 		 employee_id  query string true "Employee ID"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee/bonus [get]
func (h *EmployeeHandler) SmenaBonus(c *gin.Context) {
	var (
		bonus       float64
		operationId = c.Query("operation_id")
		employeeId  = c.Query("employee_id")
	)
	if operationId == "" || operationId == "undefined" {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}
	if employeeId == "" || employeeId == "undefined" {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	err := h.db.WithContext(ctx).
		Raw(`SELECT COALESCE(SUM(bonus_amount), 0) AS bonus FROM employee_bonus WHERE cashbox_operation_id = ? AND employee_id = ?`, operationId, employeeId).Scan(&bonus).Error
	if err != nil {
		h.log.Errorf("could not get employee shift bonus: %v", err)
		handleServiceResponse(c, InternalError, domain.InternalServerError)
		return
	}
	handleResponse(c, OK, gin.H{"bonus": bonus})
}

// CheckInOut godoc
// @Summary      Employee attendance-face-id check-in / check-out
// @Description  JWT tokendagi employee_id orqali xodimning check-in yoki check-out voqeasini attendance_logs jadvaliga yozadi. event_type qat'iy "check-in" yoki "check-out" bo'lishi kerak, aks holda xatolik qaytariladi. Faqat bugungi kun (Toshkent vaqti) bo'yicha oxirgi voqeaga qarab tekshiriladi: hech qanday voqea yo'q yoki oxirgisi check-out bo'lsa faqat check-in, oxirgisi check-in bo'lsa faqat check-out yuborish mumkin.
// @Tags         employees
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        input body domain.CreateAttendanceLogRequest true "Attendance data"
// @Success      201  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      409  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee/attendance-face-id [post]
func (h *EmployeeHandler) CheckInOut(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var body domain.CreateAttendanceLogRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	result, err := h.service.CreateAttendanceLog(ctx, user.UserId, user.StoreId, body.EventType, body.FaceIdUrl)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, CREATED, result)
}

// CreateAttendanceLogManual godoc
// @Summary      Manually create attendance check-in/check-out (admin)
// @Description  Admin tomonidan berilgan employee_id, event_type va event_at bo'yicha attendance_logs yozuvini qo'lda yaratadi. Face id orqali check-in/check-out ishlamay qolgan hollarda ishlatiladi uchun. Faqat admin huquqiga ega foydalanuvchilar chaqira oladi.
// @Tags         employees
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        input body domain.ManualCreateAttendanceLogRequest true "Manual attendance data"
// @Success      201  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      403  {object}  v1.Response
// @Failure      404  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee/attendance-manual [post]
func (h *EmployeeHandler) CreateAttendanceLogManual(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	// if !helper.IsAdmin(user) {
	// 	handleResponse(c, FORBIDDEN, "Only admin can add attendance manually")
	// 	return
	// }

	var body domain.ManualCreateAttendanceLogRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	result, err := h.service.CreateManualAttendanceLog(ctx, body.EmployeeId, body.EventType, body.EventAt)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, CREATED, result)
}

// UpdateAttendanceLog godoc
// @Summary      Update attendance event time
// @Description  attendance_logs yozuvining event_at vaqtini qo'lda tuzatadi — face-id noto'g'ri vaqt yozgan
// @Description  yoki avtomatik yopish xato ishlagan hollar uchun.
// @Description  Faqat event_at o'zgaradi; xodim yoki voqea turini almashtirish uchun eskisini o'chirib,
// @Description  /employee/attendance-manual orqali yangisini yaratish kerak.
// @Description  DIQQAT: employee_attendance_days darhol qayta hisoblanmaydi — u kunlik cron bilan to'ladi
// @Description  va cron faqat kechagi kunni qamraydi. Eskiroq kunni tuzatgandan keyin o'sha kunning
// @Description  yig'indisi (ishlagan soat, kechikish) eski holicha qoladi.
// @Tags         employees
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id    path  string                              true  "Attendance log ID"
// @Param        body  body  domain.UpdateAttendanceLogRequest  true  "Yangi event_at"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      404  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee/attendance-logs/{id} [put]
func (h *EmployeeHandler) UpdateAttendanceLog(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	id := c.Param("id")
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid id")
		return
	}

	var body domain.UpdateAttendanceLogRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.UpdateAttendanceLogEventAt(ctx, id, body.EventAt)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, res)
}

// DeleteAttendanceFaceId godoc
// @Summary      Delete attendance photo
// @Description  attendance_logs yozuvining face_id_url maydonini NULL qiladi va tegishli faylni upload papkadan o'chiradi.
// @Tags         employees
// @Security     BearerAuth
// @Produce      json
// @Param        id  path  string  true  "Attendance log ID"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      404  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee/attendance-face-id/{id} [delete]
func (h *EmployeeHandler) DeleteAttendanceFaceId(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	id := c.Param("id")
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid id")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	fileName, err := h.service.ClearAttendanceLogFaceIdUrl(ctx, id)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	if fileName != "" {
		filePath := filepath.Join("./app/uploads", fileName)
		if removeErr := os.Remove(filePath); removeErr != nil && !os.IsNotExist(removeErr) {
			h.log.Errorf("could not delete attendance photo file: %v", removeErr)
		}
	}

	handleResponse(c, OK, "DELETED")
}

// CleanupOldAttendanceFaceIds godoc
// @Summary      Cleanup old attendance photos (admin)
// @Description  keep_days'dan eski (Toshkent vaqti bo'yicha kun sanog'i) barcha check-in/check-out rasmlarining face_id_url maydonini NULL qiladi va tegishli fayllarni upload papkadan o'chiradi. Masalan keep_days=2 (standart) bo'lsa, faqat bugungi va kechagi kun rasmlari saqlanadi, undan oldingi barcha kunlar tozalanadi. Faqat admin huquqiga ega foydalanuvchilar chaqira oladi.
// @Tags         employees
// @Security     BearerAuth
// @Produce      json
// @Param        keep_days  query  int  false  "Nechta oxirgi kun saqlanishi kerak (standart 2)"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      403  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee/attendance-face-id/cleanup-old [delete]
func (h *EmployeeHandler) CleanupOldAttendanceFaceIds(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	if !helper.IsAdmin(user) {
		handleResponse(c, FORBIDDEN, "Only admin can cleanup attendance photos")
		return
	}

	keepDays := 2
	if raw := c.Query("keep_days"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			handleResponse(c, BadRequest, "Invalid keep_days")
			return
		}
		keepDays = parsed
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	fileNames, err := h.service.CleanupOldAttendanceFaceIds(ctx, keepDays)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	for _, fileName := range fileNames {
		if fileName == "" {
			continue
		}
		filePath := filepath.Join("./app/uploads", fileName)
		if removeErr := os.Remove(filePath); removeErr != nil && !os.IsNotExist(removeErr) {
			h.log.Errorf("could not delete attendance photo file: %v", removeErr)
		}
	}

	handleResponse(c, OK, gin.H{"cleaned_count": len(fileNames)})
}

// AttendanceList godoc
// @Summary      Attendance check-in / check-out list
// @Description  Xodimlarning check-in/check-out yozuvlari ro'yxati. start_date va end_date SaleStatistic bilan bir xil ishlaydi: end_date berilmasa start_date kuni yakunigacha (23:59) qamrab olinadi, ya'ni faqat start_date sifatida bugungi kun berilsa faqat bugungi kun qaytadi. Aniq do'konga bog'langan foydalanuvchilar (user.store_id mavjud) uchun ro'yxat bo'sh qaytariladi; faqat store_id'siz foydalanuvchilar uchun ma'lumot qaytadi.
// @Tags         employees
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        store_id     query  string  false  "Store ID (faqat admin uchun filter sifatida ishlaydi)"
// @Param        employee_id  query  string  false  "Employee ID"
// @Param        event_type   query  string  false  "Event type (check-in yoki check-out)"
// @Param        is_auto_closed query bool   false  "true — faqat cron avtomatik yopgan check-out'lar, false — faqat xodim o'zi bosganlar, berilmasa ikkalasi ham"
// @Param        search       query  string  false  "Xodim ismi yoki telefoni bo'yicha qidiruv"
// @Param        start_date   query  string  true   "Start Date (RFC3339, masalan 2026-08-03T00:00:00+05:00)"
// @Param        end_date     query  string  false  "End Date (RFC3339)"
// @Param        limit        query  int     false  "Limit"
// @Param        offset       query  int     false  "Offset"
// @Success      200 {object} v1.Response
// @Failure      400 {object} v1.Response
// @Failure      401 {object} v1.Response
// @Failure      500 {object} v1.Response
// @Router       /employee/attendance/list [get]
func (h *EmployeeHandler) AttendanceList(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.AttendanceLogQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	if params.StartDate == nil {
		handleServiceResponse(c, BadRequest, domain.InvalidQueryError)
		return
	}

	if params.EventType != "" && params.EventType != domain.AttendanceEventCheckIn && params.EventType != domain.AttendanceEventCheckOut {
		handleServiceResponse(c, BadRequest, domain.InvalidEventTypeError)
		return
	}

	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	results, count, err := h.service.GetAttendanceLogList(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, utils.ListResponse(results, count, params.Limit, params.Offset))
}

// AttendanceStats godoc
// @Summary      Attendance statistics
// @Description  Bir kunlik davomat statistikasi. Do'konlar: jami / ochiq (kamida bitta xodimining oxirgi voqeasi check-in) / yopiq. Xodimlar: jami / working (hozir ishda) / left (kelib ketgan) / absent (umuman kelmagan) / came (working+left) / not_working (left+absent). Do'kon "ochiq" qoidasi xarita API'sidagi is_open bilan bir xil. date berilmasa bugungi kun (Toshkent) olinadi. Admin bo'lmagan foydalanuvchi faqat o'z kompaniyasi (va do'koni bo'lsa — o'z do'koni) bo'yicha ko'radi.
// @Tags         employees
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        date          query  string  false  "Sana (2006-01-02, default: bugun)"
// @Param        store_id      query  string  false  "Store ID (bitta do'kon bo'yicha)"
// @Param        is_pharma     query  bool    false  "true"
// @Param        is_franchise  query  bool    false  "true"
// @Success      200 {object} v1.Response
// @Failure      400 {object} v1.Response
// @Failure      401 {object} v1.Response
// @Failure      500 {object} v1.Response
// @Router       /employee/attendance/stats [get]
func (h *EmployeeHandler) AttendanceStats(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.AttendanceStatsQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	// Admin bo'lmasa faqat o'z kompaniyasi va o'z do'konini ko'radi
	if !helper.IsAdmin(user) {
		params.CompanyId = user.CompanyId
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.GetAttendanceStats(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, res)
}

// DeleteAttendanceDay godoc
// @Summary      Delete attendance log
// @Description  attendance_logs jadvalidagi bitta check-in/check-out yozuvini id bo'yicha butunlay o'chiradi (soft delete emas). Faqat ADMIN va SUPERADMIN rollari uchun ochiq.
// @Tags         employees
// @Security     BearerAuth
// @Produce      json
// @Param        id  path  string  true  "Attendance log ID"
// @Success      200 {object} v1.Response
// @Failure      400 {object} v1.Response
// @Failure      401 {object} v1.Response
// @Failure      403 {object} v1.Response
// @Failure      404 {object} v1.Response
// @Failure      500 {object} v1.Response
// @Router       /employee/attendance-logs/{id} [delete]
func (h *EmployeeHandler) DeleteAttendanceLog(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	// helper.IsAdmin bu yerda ataylab ishlatilmadi: u AllAdminRoles'ni tekshiradi
	// va buxgalter, menejer, direktor kabi rollarni ham "admin" deb hisoblaydi.
	// O'chirish faqat ADMIN va SUPERADMIN uchun ochiq.
	if !utils.In(user.Role, constants.RoleAdmin, constants.RoleSuperAdmin) {
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

	if err := h.service.DeleteAttendanceLog(ctx, id); err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, "DELETED")
}

// EmployeeAttendanceDayList godoc
// @Summary      Employee attendance day list
// @Description  Xodimlarning kunlik davomat yig'indisi ro'yxati (employee_attendance_days). Admin bo'lmagan foydalanuvchilar faqat o'z do'koniga tegishli yozuvlarni ko'radi, store_id filtri ular uchun e'tiborga olinmaydi.
// @Tags         employees
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        store_id     query  string  false  "Store ID (faqat admin uchun filter sifatida ishlaydi)"
// @Param        employee_id  query  string  false  "Employee ID"
// @Param        search       query  string  false  "Xodim ismi yoki telefoni bo'yicha qidiruv"
// @Param        start_date   query  string  false  "Oraliq boshlanishi (2006-01-02 yoki RFC3339, work_date bo'yicha)"
// @Param        end_date     query  string  false  "Oraliq tugashi (2006-01-02 yoki RFC3339, work_date bo'yicha)"
// @Param        limit        query  int     false  "Limit"
// @Param        offset       query  int     false  "Offset"
// @Success      200 {object} v1.Response
// @Failure      400 {object} v1.Response
// @Failure      401 {object} v1.Response
// @Failure      500 {object} v1.Response
// @Router       /employee/attendance-days/list [get]
func (h *EmployeeHandler) EmployeeAttendanceDayList(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.EmployeeAttendanceDayQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	if !helper.IsAdmin(user) {
		if user.StoreId == "" {
			handleResponse(c, BadRequest, "store_id not found for user")
			return
		}
		params.StoreId = user.StoreId
	}

	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	results, count, err := h.service.GetEmployeeAttendanceDayList(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, utils.ListResponse(results, count, params.Limit, params.Offset))
}

// StorePayrollList godoc
// @Summary      Payroll by stores
// @Description  Do'konlar kesimidagi oylik yig'indilar (ishlagan soati, oylik, KPI, bonus, ushlab qolishlar) — javobda faqat do'kon qatorlari keladi, xodimlar ro'yxati yo'q. Xodimlarni olish uchun /employee/payroll/employees?store_id=... ishlatiladi. Pagination do'konlarga qo'yiladi. Joriy oy so'ralsa ma'lumot jonli hisoblanadi (oy boshidan bugungi kungacha), o'tgan oy so'ralsa employee_payrolls jadvalidan olinadi. year/month berilmasa joriy oy.
// @Tags         employees
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        year      query  int     false  "Yil (default: joriy)"
// @Param        month     query  int     false  "Oy 1-12 (default: joriy)"
// @Param        store_id  query  string  false  "Bitta do'kon bo'yicha filtr"
// @Param        limit     query  int     false  "Limit (do'konlar soni)"
// @Param        offset    query  int     false  "Offset"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee/payroll/stores [get]
func (h *EmployeeHandler) StorePayrollList(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.EmployeePayrollQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	if err := params.ApplyDate(); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	// Admin bo'lmasa faqat o'z kompaniyasi va o'z do'konini ko'radi
	if !helper.IsAdmin(user) {
		params.CompanyId = user.CompanyId
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
	}

	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, totalCount, period, err := h.service.GetStorePayrolls(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	result := utils.ListResponse(res, totalCount, params.Limit, params.Offset)
	result["period"] = period

	handleResponse(c, OK, result)
}

// RecalculatePayroll godoc
// @Summary      Recalculate payroll manually
// @Description  Har kecha 02:00 da ishlaydigan payroll hisobini QO'LDA ishga tushiradi.
// @Description  Cron tunda ishlamay qolganda (server o'chgan, xato bo'lgan) ertalab shu chaqiriladi.
// @Description  Eski oyni qayta hisoblash uchun ham ishlatiladi — masalan davomat qo'lda tuzatilgandan keyin.
// @Description  Xavfsiz: hisob har safar oy boshidan qayta quriladi, necha marta chaqirilsa ham natija bir xil.
// @Description  Avans, ushlab qolish, status va approved_by o'zgarmaydi.
// @Description  year/month berilmasa joriy oy; date berilmasa kechagi kun.
// @Tags         employees
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        year   query  int     false  "Yil (default: joriy)"
// @Param        month  query  int     false  "Oy 1-12 (default: joriy)"
// @Param        date   query  string  false  "Hisob shu kungacha olib boriladi, YYYY-MM-DD (default: kecha)"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      403  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee/payroll/recalculate [post]
func (h *EmployeeHandler) RecalculatePayroll(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}
	// Haqiqiy ma'lumotga yozadi va butun kompaniyani qamraydi — faqat admin.
	if !helper.IsAdmin(user) {
		handleServiceResponse(c, nil, domain.ForbiddinError)
		return
	}

	var params struct {
		Year  int    `form:"year"`
		Month int    `form:"month"`
		Date  string `form:"date"`
	}
	if err := c.ShouldBindQuery(&params); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.RecalculatePayroll(ctx, params.Year, params.Month, params.Date)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, res)
}

// EmployeePayrollList godoc
// @Summary      Payroll by employees
// @Description  Xodimlarning oylik ko'rsatkichlari (ishlagan soati, oylik, KPI, bonus, avans, ushlab qolishlar).
// @Description  Ro'yxatga faqat faol xodimlar kiradi: is_active, status = "active" va roli "Кассир" yoki "Заведующий".
// @Description  KPI pog'onasi DO'KON bo'yicha: expected_plan = store_plan × (o'tgan ish kuni / oydagi ish kuni), achievement = store_sales / expected_plan × 100 — bu qiymatlar bitta do'kondagi hamma xodimda bir xil.
// @Description  KPI summasi esa shaxsiy: kpi_amount = individual_sales × kpi_percent. Do'kon rejani bajarmasa kpi_percent = 0 va hech kim KPI olmaydi.
// @Description  year/month berilmasa joriy oy olinadi.
// @Tags         employees
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        store_id  query  string  false  "Store ID (berilmasa barcha xodimlar)"
// @Param        search    query  string  false  "Employee full name search"
// @Param        year      query  int     false  "Year"
// @Param        month     query  int     false  "Month"
// @Param        limit     query  int     false  "Limit (employees)"
// @Param        offset    query  int     false  "Offset"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee/payroll/employees [get]
func (h *EmployeeHandler) EmployeePayrollList(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.EmployeePayrollQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	if err := params.ApplyDate(); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	// Admin bo'lmasa faqat o'z kompaniyasi va o'z do'konini ko'radi
	if !helper.IsAdmin(user) {
		params.CompanyId = user.CompanyId
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
	}

	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, totalCount, period, err := h.service.GetEmployeePayrolls(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	result := utils.ListResponse(res, totalCount, params.Limit, params.Offset)
	result["period"] = period

	handleResponse(c, OK, result)
}

// ExportStorePayrollExcel godoc
// @Summary      Export store payroll to Excel
// @Description  /employee/payroll/stores ro'yxatini Excel fayl qilib saqlaydi va fayl nomini qaytaradi.
// @Description  Ro'yxat bilan bir xil filtrlardan o'tadi, lekin SAHIFALANMAYDI: filtrga mos barcha do'kon fayldagi.
// @Description  Tartib ham bir xil — franshiza do'konlari eng oxirida.
// @Description  date berilsa yil/oy shundan olinadi; berilmasa year/month, ular ham bo'lmasa joriy oy.
// @Tags         employees
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        store_id  query  string  false  "Store ID"
// @Param        date      query  string  false  "Sana YYYY-MM-DD (year/month o'rniga)"
// @Param        year      query  int     false  "Year (default: joriy)"
// @Param        month     query  int     false  "Month 1-12 (default: joriy)"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee/payroll/stores/export-excel [get]
func (h *EmployeeHandler) ExportStorePayrollExcel(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.EmployeePayrollQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	if err := params.ApplyDate(); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	// Admin bo'lmasa faqat o'z kompaniyasi va o'z do'konini ko'radi
	if !helper.IsAdmin(user) {
		params.CompanyId = user.CompanyId
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
	}

	// Eksportda sahifalash yo'q: limit 0 qoldiriladi, xizmat uni "cheklovsiz"
	// deb tushunadi. defaultLimitOffset ataylab chaqirilmaydi — u 10 qo'yardi.
	params.Limit, params.Offset = 0, 0

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	rows, _, period, err := h.service.GetStorePayrolls(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	f := excelize.NewFile()
	sheet := constants.DefaultSheetName
	f.SetSheetName("Sheet1", sheet)

	headers := []string{
		"Филиал", "Франшиза", "Сотрудников", "Активных", "В расчёте",
		"Отработано часов", "Норма часов",
		"Оклад", "Начислено по окладу", "Личные продажи",
		"План магазина", "Продажи магазина",
		"KPI", "Бонус", "Начислено всего", "Доля ФОТ, %",
		"Аванс", "Удержания", "К выплате",
	}
	if err := setExcelHeaders(f, sheet, headers); err != nil {
		h.log.Errorf("could not style payroll excel header: %v", err)
		handleServiceResponse(c, nil, domain.InternalServerError)
		return
	}

	for i, r := range rows {
		// CoordinatesToCellName ustun harfini to'g'ri hisoblaydi (Z dan keyin AA),
		// qo'lda 'A'+i qilish 26 ustundan keyin buzilardi.
		set := func(col int, value any) {
			cell, err := excelize.CoordinatesToCellName(col, i+2)
			if err != nil {
				return
			}
			f.SetCellValue(sheet, cell, value)
		}
		franchise := "Нет"
		if r.IsFranchise {
			franchise = "Да"
		}
		values := []any{
			r.StoreName, franchise, r.EmployeeCount, r.ActiveStoreEmployeeCount, r.PayrollCount,
			r.WorkedHours, r.AvgMonthlyHours,
			r.SalaryRateAmount, r.ActualSalaryAmount, r.IndividualSalesAmount,
			r.StorePlanAmount, r.StoreSalesAmount,
			r.KpiAmount, r.BonusAmount, r.GrossSalaryAmount, r.SalaryPercent,
			r.AdvanceAmount, r.TotalDeduction, r.NetPayAmount,
		}
		for col, v := range values {
			set(col+1, v)
		}
	}

	saveExcelToUploads(c, f, *h.log, fmt.Sprintf("Oylik_dokonlar_%d-%02d", period.Year, period.Month))
}

// StorePayrollStatistics godoc
// @Summary      Store payroll statistics
// @Description  /employee/payroll/stores ro'yxatining yig'ma ko'rsatkichlari.
// @Description  Ro'yxat bilan bir xil filtrlardan o'tadi, lekin sahifalanmaydi: limit/offset ta'sir qilmaydi.
// @Description  Uchta xodim sanog'i uchta boshqa narsa: total_employee_count (do'kon kartochkasidagi son),
// @Description  total_active_store_employee_count (haqiqatan faol xodimlar), total_payroll_count (oylik hisobga kirganlar).
// @Description  DIQQAT: total_store_plan_amount va total_store_sales_amount do'kon darajasidagi qiymatlar —
// @Description  har bir do'kon bo'yicha BIR MARTA sanaladi, xodimlar soniga ko'paymaydi.
// @Description  date berilsa yil/oy shundan olinadi; berilmasa year/month, ular ham bo'lmasa joriy oy.
// @Tags         employees
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        store_id  query  string  false  "Store ID"
// @Param        date      query  string  false  "Sana YYYY-MM-DD (year/month o'rniga)"
// @Param        year      query  int     false  "Year (default: joriy)"
// @Param        month     query  int     false  "Month 1-12 (default: joriy)"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee/payroll/stores/statistics [get]
func (h *EmployeeHandler) StorePayrollStatistics(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.EmployeePayrollQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	if err := params.ApplyDate(); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	// Admin bo'lmasa faqat o'z kompaniyasi va o'z do'koni bo'yicha
	if !helper.IsAdmin(user) {
		params.CompanyId = user.CompanyId
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	stats, period, err := h.service.GetStorePayrollStatistics(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, gin.H{"statistics": stats, "period": period})
}

// ExportEmployeePayrollExcel godoc
// @Summary      Export employee payroll to Excel
// @Description  /employee/payroll/employees hisobotini Excel qilib saqlaydi va fayl nomini qaytaradi.
// @Description  Varaq do'konlar bo'yicha bloklarga bo'linadi: har blok do'kon yig'indisi qatori bilan boshlanadi,
// @Description  ostida o'sha do'kon xodimlari, eng oxirida esa barcha do'konlar bo'yicha umumiy jami.
// @Description  Sarlavhalar ikki qatorli va guruhlar bo'yicha ranglangan (Оклад, KPI, Аванс, Удержания, ...).
// @Description  Ro'yxat bilan bir xil filtrlardan o'tadi, lekin SAHIFALANMAYDI: filtrga mos barcha xodim fayldagi.
// @Description  date berilsa yil/oy shundan olinadi; berilmasa year/month, ular ham bo'lmasa joriy oy.
// @Tags         employees
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        store_id  query  string  false  "Store ID"
// @Param        search    query  string  false  "Employee full name search"
// @Param        date      query  string  false  "Sana YYYY-MM-DD (year/month o'rniga)"
// @Param        year      query  int     false  "Year (default: joriy)"
// @Param        month     query  int     false  "Month 1-12 (default: joriy)"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee/payroll/employees/export-excel [get]
func (h *EmployeeHandler) ExportEmployeePayrollExcel(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.EmployeePayrollQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	if err := params.ApplyDate(); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	if !helper.IsAdmin(user) {
		params.CompanyId = user.CompanyId
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
	}

	// Eksportda sahifalash yo'q. So'rovda LIMIT NULLIF(@limit, 0) turibdi,
	// shuning uchun 0 "cheklovsiz" degani.
	params.Limit, params.Offset = 0, 0

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	rows, _, period, err := h.service.GetEmployeePayrolls(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	f, err := buildPayrollExcel(constants.DefaultSheetName, rows)
	if err != nil {
		h.log.Errorf("could not build payroll excel: %v", err)
		handleServiceResponse(c, nil, domain.InternalServerError)
		return
	}

	saveExcelToUploads(c, f, *h.log, payrollExcelFileName(period.Year, period.Month))
}

// PayrollStatistics godoc
// @Summary      Payroll report statistics
// @Description  /employee/payroll/employees ro'yxatining yig'ma ko'rsatkichlari.
// @Description  Ro'yxat bilan AYNAN bir xil filtrlardan o'tadi (davr, do'kon, kompaniya, rol va davomat doirasi),
// @Description  shuning uchun raqamlar ekrandagi ro'yxatga mos keladi. Sahifalanmaydi: limit/offset ta'sir qilmaydi.
// @Description  DIQQAT: total_store_plan_amount, total_store_sales_amount va total_expected_plan_amount do'kon
// @Description  darajasidagi qiymatlar — har bir do'kon bo'yicha BIR MARTA sanaladi, xodimlar soniga ko'paymaydi.
// @Description  date berilsa yil/oy shundan olinadi; berilmasa year/month, ular ham bo'lmasa joriy oy.
// @Tags         employees
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        store_id  query  string  false  "Store ID"
// @Param        search    query  string  false  "Employee full name search"
// @Param        date      query  string  false  "Sana YYYY-MM-DD (year/month o'rniga)"
// @Param        year      query  int     false  "Year (default: joriy)"
// @Param        month     query  int     false  "Month 1-12 (default: joriy)"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee/payroll/employees/statistics [get]
func (h *EmployeeHandler) PayrollStatistics(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.EmployeePayrollQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	if err := params.ApplyDate(); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	// Admin bo'lmasa faqat o'z kompaniyasi va o'z do'koni bo'yicha
	if !helper.IsAdmin(user) {
		params.CompanyId = user.CompanyId
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	stats, period, err := h.service.GetPayrollStatistics(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, gin.H{"statistics": stats, "period": period})
}

// PayrollManagementStatistics godoc
// @Summary      Payroll management statistics
// @Description  /employee/payroll/management ro'yxatining yig'ma ko'rsatkichlari: nechta do'kon,
// @Description  nechta xodim, oylik fond stavkasi (employees.salary yig'indisi) va avanslar jami
// @Description  (karta + naqd birga).
// @Description  role_type_counts — employees.role_type bo'yicha xodimlar soni. Kalitlar bazadagi haqiqiy
// @Description  qiymatlar ("CASHIER", "HEADOFCASHIER", "ROP_APTEKA", "INTERN", ...), yangi rol qo'shilsa
// @Description  o'zi paydo bo'ladi. role_type to'ldirilmaganlar bo'sh kalit ("") ostida — shu sababli
// @Description  qiymatlar yig'indisi doim total_employees'ga teng.
// @Description  Filtrlar ro'yxat bilan AYNAN bir xil, shuning uchun raqamlar ekrandagi ro'yxatga mos keladi.
// @Description  Sahifalanmaydi: limit/offset ta'sir qilmaydi, filtrga tushgan hamma xodim hisobga olinadi.
// @Description  date berilsa yil/oy shundan olinadi; berilmasa year/month, ular ham bo'lmasa joriy oy.
// @Tags         employees
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        store_id     query  string  false  "Store ID"
// @Param        employee_id  query  string  false  "Employee ID"
// @Param        role_type    query  string  false  "employees.role_type (CASHIER, HEADOFCASHIER, ...)"
// @Param        shift_type   query  string  false  "employees.shift_type (day yoki night)"
// @Param        search       query  string  false  "Ism yoki telefon bo'yicha qidiruv"
// @Param        date         query  string  false  "Sana YYYY-MM-DD (year/month o'rniga)"
// @Param        year         query  int     false  "Year (default: joriy)"
// @Param        month        query  int     false  "Month 1-12 (default: joriy)"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee/payroll/management/statistics [get]
func (h *EmployeeHandler) PayrollManagementStatistics(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.EmployeePayrollAdvanceQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	if err := params.ApplyDate(); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	// Admin bo'lmasa faqat o'z kompaniyasi va o'z do'koni bo'yicha
	if !helper.IsAdmin(user) {
		params.CompanyId = user.CompanyId
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.DefaultContextTimeout)
	defer cancel()

	stats, period, err := h.service.GetPayrollManagementStatistics(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, gin.H{"statistics": stats, "period": period})
}

// EmployeePayrollmanagementList godoc
// @Summary      Payroll edit list (salary, KPI, advances)
// @Description  Xodim kartochkasidagi qiymatlar (role_type, ism, telefon, salary, daily_work_hours, shift_type, experience_years) va so'ralgan oyning payroll qatoridan kpi_percent bilan avanslar.
// @Description  Har bir qatordagi id — employee_payrolls qatorining id'si, uni to'g'ridan-to'g'ri PUT /employee/payroll/{id}/management ga berish mumkin.
// @Description  Ro'yxatga roli "Кассир"/"Заведующий" bo'lgan barcha xodimlar kiradi, shu jumladan oy davomida hali ishlamaganlar ham (ularga ham avans yozish mumkin bo'lishi kerak).
// @Description  Shu sababli bu ro'yxat /employee/payroll/employees hisobotidan ko'proq qator qaytarishi mumkin — hisobotda davomati borlar (worked_hours > 0) ko'rsatiladi.
// @Description  year/month berilmasa joriy oy olinadi. Kelajakdagi oy qabul qilinmaydi.
// @Tags         employees
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        store_id     query  string  false  "Store ID"
// @Param        employee_id  query  string  false  "Employee ID"
// @Param        role_type    query  string  false  "employees.role_type (CASHIER, HEADOFCASHIER, ...)"
// @Param        shift_type   query  string  false  "employees.shift_type (day yoki night)"
// @Param        search       query  string  false  "Ism yoki telefon bo'yicha qidiruv"
// @Param        date         query  string  false  "Sana YYYY-MM-DD (year/month o'rniga)"
// @Param        year         query  int     false  "Year (default: joriy)"
// @Param        month        query  int     false  "Month 1-12 (default: joriy)"
// @Param        limit        query  int     false  "Limit"
// @Param        offset       query  int     false  "Offset"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee/payroll/management [get]
func (h *EmployeeHandler) EmployeePayrollManagementList(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.EmployeePayrollAdvanceQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	if err := params.ApplyDate(); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	// Admin bo'lmasa faqat o'z kompaniyasi va o'z do'konini ko'radi
	if !helper.IsAdmin(user) {
		params.CompanyId = user.CompanyId
		if user.StoreId != "" {
			params.StoreId = user.StoreId
		}
	}

	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.DefaultContextTimeout)
	defer cancel()

	res, totalCount, period, err := h.service.GetEmployeePayrollManagement(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	result := utils.ListResponse(res, totalCount, params.Limit, params.Offset)
	result["period"] = period

	handleResponse(c, OK, result)
}

// UpdateEmployeePayrollManagement godoc
// @Summary      Update payroll salary, KPI and advance amounts
// @Description  kpi_percent, salary, daily_work_hours, shift_type, role_type, advance_card_amount va advance_cash_amount maydonlarini payroll id bo'yicha yangilaydi.
// @Description  Hammasi ixtiyoriy — berilgani yoziladi, berilmagani eski qiymatida qoladi.
// @Description  kpi_percent/salary ikkala jadvalga; daily_work_hours (4, 7, 8), shift_type va role_type faqat employees'ga; avanslar esa faqat shu oyning payroll qatoriga yoziladi.
// @Description  role_type xodim kartochkasida saqlanadi va keyingi oylarga ham amal qiladi — payroll hisob-kitobiga ta'sir qilmaydi.
// @Description  actual_salary_amount, kpi_amount, gross_salary_amount va net_pay_amount shu yerda qayta hisoblanadi — cron kutilmaydi.
// @Tags         employees
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id    path     string                                true  "Payroll ID"
// @Param        body  body     domain.EmployeePayrollAdvanceRequest  true  "Avans summalari"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      404  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee/payroll/{id}/management [put]
func (h *EmployeeHandler) UpdateEmployeePayrollManagement(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var (
		body domain.EmployeePayrollAdvanceRequest
		id   = c.Param("id")
	)
	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Errorf("could not bind payroll advance request body: %v", err)
		handleServiceResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}
	if body.IsEmpty() {
		handleServiceResponse(c, BadRequest, domain.InvalidRequestBodyError)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.UpdateEmployeePayrollAdvance(ctx, id, user.UserId, &body)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, res)
}

// MyPayroll godoc
// @Summary      My payroll
// @Description  Token egasining o'z oylik ko'rsatkichlari: ism-familiyasi, do'koni va shu oy uchun umumiy ma'lumotlar (qancha ishladi, oylik, KPI, bonus, avans, ushlab qolishlar, qo'lga tegadigan summa).
// @Description  So'ralgan oy uchun employee_payrolls'da snapshot bo'lsa o'sha qaytadi (avans va ushlab qolishlar bilan), bo'lmasa jonli hisoblanadi (oy boshidan bugungi kungacha).
// @Description  Xodim id token'dan olinadi — birovning oyligini so'rab bo'lmaydi. Xodim nofaol bo'lsa 404.
// @Tags         employees
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        year   query  int  false  "Yil (default: joriy)"
// @Param        month  query  int  false  "Oy 1-12 (default: joriy)"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      404  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee/payroll/my [get]
func (h *EmployeeHandler) MyPayroll(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.EmployeePayrollQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.GetMyPayroll(ctx, user.UserId, params.Year, params.Month)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, res)
}

// CreateOrUpdateFaceDescriptor godoc
// @Summary      Create or update employee face descriptor
// @Description  Xodim uchun yuz descriptorlarini (face-api.js orqali olingan array of arrays, "face_id" nomi bilan) yaratadi yoki mavjud bo'lsa yangilaydi. Har bir xodim uchun bitta yozuv saqlanadi.
// @Tags         employees
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id     path     string                         true  "Employee id"
// @Param        input  body     domain.FaceDescriptorRequest  true  "Face descriptor data"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      404  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee/{id}/face-descriptor [patch]
func (h *EmployeeHandler) CreateOrUpdateFaceDescriptor(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	id := c.Param("id")
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid id")
		return
	}

	var body domain.FaceDescriptorRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.UpsertEmployeeFaceDescriptor(ctx, id, body.Descriptor)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, res)
}

// GetEmployeeFaceDescriptor godoc
// @Summary      Get employee face descriptor
// @Description  Xodimning saqlangan yuz descriptor yozuvini qaytaradi.
// @Tags         employees
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id   path     string  true  "Employee id"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      404  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee/{id}/face-descriptor [get]
func (h *EmployeeHandler) GetEmployeeFaceDescriptor(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	id := c.Param("id")
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid id")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	res, err := h.service.GetEmployeeFaceDescriptor(ctx, id)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, res)
}

// DeleteEmployeeFaceDescriptor godoc
// @Summary      Delete employee face descriptor
// @Description  Xodimning saqlangan yuz descriptor yozuvini o'chiradi.
// @Tags         employees
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id   path     string  true  "Employee id"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      404  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee/{id}/face-descriptor [delete]
func (h *EmployeeHandler) DeleteEmployeeFaceDescriptor(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	id := c.Param("id")
	if err := uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid id")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	if err := h.service.DeleteEmployeeFaceDescriptor(ctx, id); err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, "DELETED")
}

// ListEmployeeFaceDescriptors godoc
// @Summary      List employee face descriptors
// @Description  Filtrlar (store_id, company_id, employee_id) bo'yicha xodimlarning yuz descriptor yozuvlari ro'yxati. Admin bo'lmagan foydalanuvchilar faqat o'z do'koniga tegishli yozuvlarni ko'radi, store_id filtri ular uchun e'tiborga olinmaydi.
// @Tags         employees
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        store_id     query  string  false  "Store ID (faqat admin uchun filter sifatida ishlaydi)"
// @Param        company_id   query  string  false  "Company ID (faqat admin uchun filter sifatida ishlaydi)"
// @Param        employee_id  query  string  false  "Employee ID"
// @Param        limit        query  int     false  "Limit"
// @Param        offset       query  int     false  "Offset"
// @Success      200 {object} v1.Response
// @Failure      400 {object} v1.Response
// @Failure      401 {object} v1.Response
// @Failure      500 {object} v1.Response
// @Router       /employee/list/face-descriptor [get]
func (h *EmployeeHandler) ListEmployeeFaceDescriptors(c *gin.Context) {
	user := h.service.GetSignedUser(c)
	if user.UserId == "" {
		handleServiceResponse(c, nil, domain.UnauthorizedError)
		return
	}

	var params domain.EmployeeFaceDescriptorQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	if !helper.IsAdmin(user) {
		if user.StoreId == "" {
			handleResponse(c, BadRequest, "store_id not found for user")
			return
		}
		params.StoreId = user.StoreId
	}

	params.Limit, params.Offset = defaultLimitOffset(params.Limit, params.Offset)

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	results, count, err := h.service.GetEmployeeFaceDescriptorList(ctx, &params)
	if err != nil {
		handleServiceResponse(c, nil, err)
		return
	}

	handleResponse(c, OK, utils.ListResponse(results, count, params.Limit, params.Offset))
}
