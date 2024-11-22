package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/etc"
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
		employee.GET("", h.Get)
		employee.GET("/get-list", h.List)
		employee.PUT("", h.Update)
		employee.DELETE("", h.Delete)
	}
}

// @Summary      Create employee
// @Description  Create a new employee in the system
// @Tags         employees
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        input body     domain.Employee  true  "Employee data"
// @Success      201  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      403  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee [post]
func (h *EmployeeHandler) Create(c *gin.Context) {
	var body = new(domain.Employee)

	if err := c.ShouldBindJSON(&body); err != nil {
		h.Log.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}

	hashedPassword, err := etc.HashPassword(body.Password)
	if err != nil {
		h.Log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
	}
	body.Password = hashedPassword
	body.Id = uuid.New().String()
	if err := h.Db.WithContext(c.Request.Context()).Create(body).Scan(body).Error; err != nil {
		h.Log.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusCreated, MsgSuccessCreate, body)
}

// @Summary      Get employee
// @Description  Get an employee by id
// @Tags         employees
// @Accept       json
// @Produce      json
// @Param        id   query     string  true  "Employee id"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      403  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee [get]
// @Security     BearerAuth
func (h *EmployeeHandler) Get(c *gin.Context) {
	var res domain.Employee
	if err := h.Db.First(&res, "id = ?", c.Query("id")).Error; err != nil {
		h.Log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessFetch, res)
}

// @Summary      List employees
// @Description  List all employees
// @Tags         employees
// @Accept       json
// @Produce      json
// @Param        limit          query     int             false "Limit"
// @Param        offset         query     int             false "Offset"
// @Success      200  {array}   v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      403  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee [get]
// @Security     BearerAuth
func (h *EmployeeHandler) List(c *gin.Context) {
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	var res []domain.Employee
	if err := h.Db.Limit(limit).Offset(offset).Find(&res).Error; err != nil {
		h.Log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessFetch, res)
}

// @Summary      Update employee
// @Description  Update an employee
// @Tags         employees
// @Accept       json
// @Produce      json
// @Param        input         body  domain.Employee true  "Employee data"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      403  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee [put]
// @Security     BearerAuth
func (h *EmployeeHandler) Update(c *gin.Context) {
	var body = new(domain.Employee)

	if err := c.ShouldBindJSON(&body); err != nil {
		h.Log.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	hashedPassword, err := etc.HashPassword(body.Password)
	if err != nil {
		h.Log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
	}
	body.Password = hashedPassword
	if err := h.Db.WithContext(c.Request.Context()).Model(body).Where("id = ?", body.Id).
		Updates(body).Error; err != nil {
		h.Log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessUpdate, body)
}

// @Summary      Delete employee
// @Description  Delete an employee by id
// @Tags         employees
// @Accept       json
// @Produce      json
// @Param        id             query     string    true  "Employee id"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      403  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee [delete]
// @Security     BearerAuth
func (h *EmployeeHandler) Delete(c *gin.Context) {
	if err := h.Db.WithContext(c.Request.Context()).Delete(&domain.Employee{}, "id = ?", c.Query("id")).Error; err != nil {
		h.Log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessDelete, MsgSuccessDelete)
}
