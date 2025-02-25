package v1

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/etc"
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
		employee.PUT("/block", h.BlockEmployee)
		employee.PUT("/unblock", h.UnBlockEmployee)
		employee.GET("/bonus", h.SmenaBonus)
	}
}

// @Summary      Create employee
// @Description  Create a new employee in the system
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

	hashedPassword, err := etc.Encrypt(*body.Password, h.cfg.HeshKey)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	body.Password = &hashedPassword
	body.Id = uuid.New().String()
	body.Status = "active"
	body.FullName = body.FirstName + " " + body.LastName
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
// @Success      200  {array}   v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      403  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee/list [get]
func (h *EmployeeHandler) List(c *gin.Context) {
	var (
		res        []domain.Employee
		totalCount int64
	)
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// get employee list data
	res, totalCount, err = h.service.ListEmployee(c, limit, offset)
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}
	// add _meta for pagination response
	result := utils.ListResponse(res, totalCount, limit, offset)

	handleResponse(c, OK, result)
}

// @Summary      Download employee list as Excel
// @Description  Export filtered employee list to an Excel file
// @Tags         employees
// @Security     BearerAuth
// @Accept       json
// @Produce      application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param        role_id        query     string   false "Role ID"
// @Param        store_id       query     string   false "Store ID"
// @Param        search         query     string   false "Search"
// @Param        status         query     string   false "Status (deleted || blocked || active)"
// @Success      200  {file}   application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Failure      400  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee/export-excel [get]
func (h *EmployeeHandler) ExportEmployeeExcel(c *gin.Context) {
	var (
		employees []domain.Employee
		err       error
	)
	// get limit and offset
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}

	// get employee list data
	employees, _, err = h.service.ListEmployee(c, limit, offset)
	if err != nil {
		handleResponse(c, InternalError, err.Error())
		return
	}

	// Excel fayl yaratish
	f := excelize.NewFile()
	sheetName := "Employees"
	f.SetSheetName("Sheet1", sheetName)

	// Headerlar
	headers := []string{"ID", "ФИО", "Филиал", "Телефон", "Роль", "Статус"}

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
	for i, emp := range employees {
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

	// Faylni HTTP response orqali yuborish
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=employees.xlsx")

	if err := f.Write(c.Writer); err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, "Failed to generate Excel file")
	}
}

// @Summary      Update employee
// @Description  Update an employee
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

	body.FullName = body.FirstName + " " + body.LastName
	if body.Password != nil {
		*body.Password, err = etc.Encrypt(*body.Password, h.cfg.HeshKey)
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
	}
	if len(body.RoleIds) > 0 {
		err = h.db.WithContext(c.Request.Context()).
			Delete(&domain.EmployeeRole{}, "employee_id = ?", id).Error
		if err != nil {
			h.log.Error(err)
			handleResponse(c, InternalError, err.Error())
			return
		}
		var employeeRoles []domain.EmployeeRole
		for _, roleId := range body.RoleIds {
			employeeRoles = append(employeeRoles, domain.EmployeeRole{
				Id:         uuid.New().String(),
				EmployeeId: id,
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
	err = h.db.WithContext(c.Request.Context()).
		Table("employees").
		Where("id = ?", id).
		Updates(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
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

	err := h.db.
		WithContext(c.Request.Context()).
		Table("employees").
		Where("id IN (?)", ids).
		Updates(map[string]interface{}{
			"status":    "deleted",
			"is_active": false,
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
	var permissions []domain.Permission
	err := h.db.Debug().Raw(`
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
	`, userID, userID).Scan(&permissions).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	res.Permission = permissions
	var roles []domain.Role
	err = h.db.Raw(`
	SELECT 
		r.*
	FROM roles r 
	JOIN employee_roles er ON er.role_id = r.id
	WHERE er.employee_id = ?
	`, userID).Scan(&roles).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	res.Roles = roles

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

	hashedPassword, err := etc.Encrypt(body.ConfirmPassword, h.cfg.HeshKey)
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
	err := h.db.
		WithContext(c.Request.Context()).
		Table("employees").
		Where("id IN (?)", ids).
		Update("status", "blocked").Error
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
	err := h.db.
		WithContext(c.Request.Context()).
		Table("employees").
		Where("id IN (?)", ids).
		Update("is_active", true).
		Update("status", "active").Error
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
		handleResponse(c, BadRequest, "Operation ID is required")
		return
	}
	if employeeId == "" || employeeId == "undefined" {
		handleResponse(c, BadRequest, "Employee ID is required")
		return
	}
	err := h.db.Debug().
		Raw(`SELECT COALESCE(SUM(bonus_amount), 0) AS bonus FROM employee_bonus WHERE cashbox_operation_id = ? AND employee_id = ?`, operationId, employeeId).Scan(&bonus).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, gin.H{"bonus": bonus})
}
