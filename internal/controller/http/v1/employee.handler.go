package v1

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/etc"
	"github.com/pharma-crm-backend/pkg/utils"
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
		employee.PUT("/:id", h.Update)
		employee.DELETE("/:id", h.Delete)
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
	hashedPassword, err := etc.Encrypt(body.Password, h.cfg.HeshKey)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	// hashedPassword, err := etc.HashPassword(body.Password)
	// if err != nil {
	// 	h.log.Error(err)
	// 	handleResponse(c, InternalError, err.Error())
	// 	return
	// }
	body.Password = hashedPassword
	body.Id = uuid.New().String()
	err = h.db.WithContext(c.Request.Context()).
		Table("employees").
		Create(&body).Error
	if err != nil {
		h.log.Error(err.Error())
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, body)
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
	if err := h.db.Preload("Store").
		First(&res, "id = ?", c.Param("id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			handleResponse(c, NotFound, nil)
			return
		}
		h.log.Error("err: ", err)
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
// @Success      200  {array}   v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      403  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee/list [get]
func (h *EmployeeHandler) List(c *gin.Context) {
	var (
		res         []domain.Employee
		totalCount  int64
		searchField = fmt.Sprintf("%%%s%%", c.Query("search"))
		roleId      = c.Query("role_id")
	)
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, BadRequest, err.Error())
		return
	}
	query := h.db.Model(&domain.Employee{}).
		Count(&totalCount).
		Where("first_name ILIKE ? OR last_name ILIKE ?", searchField, searchField)

	if roleId != "" {
		query = query.Where("role_id = ?", roleId)
	}

	query = query.Preload("Store").
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&res)

	if query.Error != nil {
		h.log.Error(query.Error)
		handleResponse(c, InternalError, query.Error.Error())
		return
	}
	result := utils.ListResponse(res, totalCount, limit, offset)
	handleResponse(c, OK, result)
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
		body = new(domain.EmployeeRequest)
		err  error
	)
	err = c.ShouldBindJSON(&body)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	hashedPassword, err := etc.HashPassword(body.Password)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
	}
	body.Password = hashedPassword
	err = h.db.WithContext(c.Request.Context()).
		Model(&domain.Employee{}).
		Where("id = ?", c.Param("id")).
		Updates(body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, body)
}

// @Summary      Delete employee
// @Description  Delete an employee by id
// @Tags         employees
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id             path     string    true  "Employee id"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      403  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee/{id} [delete]
func (h *EmployeeHandler) Delete(c *gin.Context) {
	if err := h.db.WithContext(c.Request.Context()).Delete(&domain.Employee{}, "id = ?", c.Param("id")).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, OK, "DELETED")
}
