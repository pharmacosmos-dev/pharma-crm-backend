package v1

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/etc"
	"github.com/pharma-crm-backend/pkg/logger"
	"github.com/pharma-crm-backend/pkg/token"
	"gorm.io/gorm"
)

type EmployeeHandler struct {
	cfg        *config.Config
	db         *gorm.DB
	log        *logger.Logger
	JwtHandler token.JWTHandler
}

func NewEmployeeHandler(cfg *config.Config, db *gorm.DB, log *logger.Logger, jwtHandler token.JWTHandler) *EmployeeHandler {
	return &EmployeeHandler{cfg: cfg, db: db, log: log, JwtHandler: jwtHandler}
}

// @Summary      Create employee
// @Description  Create a new employee in the system
// @Tags         employees
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true  "Bearer token"
// @Param        input body      RequestBody[domain.Employee]  true  "Employee data"
// @Success      201  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      403  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee [post]
func (h *EmployeeHandler) Create(c *gin.Context) {
	var (
		body RequestBody[domain.Employee]
		res  domain.Employee
	)
	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	hashedPassword, err := etc.HashPassword(body.Data.Password)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
	}
	body.Data.Password = hashedPassword
	body.Data.Id = uuid.New().String()
	if err := h.db.WithContext(ctx).Create(&body.Data).Scan(&res).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusCreated, MsgSuccessCreate, res)
}

// @Summary      Get employee
// @Description  Get an employee by id
// @Tags         employees
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string          true  "Bearer token"
// @Param        id             query     string          true  "Employee id"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      403  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee [get]
// @Security     BearerAuth
func (h *EmployeeHandler) Get(c *gin.Context) {
	var res domain.Employee
	if err := h.db.First(&res, "id = ?", c.Query("id")).Error; err != nil {
		h.log.Error(err)
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
// @Param        Authorization  header    string          true  "Bearer token"
// @Param        limit          query     int             false "Limit"
// @Param        offset         query     int             false "Offset"
// @Success      200  {array}   v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      403  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employees [get]
// @Security     BearerAuth
func (h *EmployeeHandler) List(c *gin.Context) {
	limit, err := getLimitParam(c)
	if err != nil {
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	offset, err := getOffsetParam(c)
	if err != nil {
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	var res []*domain.Employee
	if err := h.db.Limit(limit).Offset(offset).Find(&res).Error; err != nil {
		h.log.Error(err)
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
// @Param        Authorization  header    string          true  "Bearer token"
// @Param        input         body      RequestBody[domain.Employee]  true  "Employee data"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      403  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee [put]
// @Security     BearerAuth
func (h *EmployeeHandler) Update(c *gin.Context) {
	var body RequestBody[domain.Employee]
	var res domain.Employee
	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	hashedPassword, err := etc.HashPassword(body.Data.Password)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
	}
	body.Data.Password = hashedPassword

	if err := h.db.WithContext(ctx).Model(&res).Where("id = ?", body.Data.Id).
		Updates(&body.Data).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessUpdate, res)
}

// @Summary      Delete employee
// @Description  Delete an employee by id
// @Tags         employees
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string          true  "Bearer token"
// @Param        id             query     string          true  "Employee id"
// @Success      200  {object}  v1.Response
// @Failure      400  {object}  v1.Response
// @Failure      401  {object}  v1.Response
// @Failure      403  {object}  v1.Response
// @Failure      500  {object}  v1.Response
// @Router       /employee [delete]
// @Security     BearerAuth
func (h *EmployeeHandler) Delete(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	if err := h.db.WithContext(ctx).Delete(&domain.Employee{}, "id = ?", c.Query("id")).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessDelete, MsgSuccessDelete)
}
