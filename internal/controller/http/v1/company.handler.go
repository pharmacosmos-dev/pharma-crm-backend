package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/utils"
)

type CompanyHandler struct {
	*Handler
}

func (h *Handler) NewCompanyHandler(r *gin.RouterGroup) {
	company := &CompanyHandler{h}
	company.CompanyRoutes(r)
}

func (h *CompanyHandler) CompanyRoutes(r *gin.RouterGroup) {
	company := r.Group("/company")
	{
		company.POST("", h.Create)
		company.GET("/:id", h.Get)
		company.GET("/info", h.GetInfo)
		company.PUT("/:id", h.Update)
	}

}

// Create company
// @Summary Create a company
// @Description Create a company from the request body
// @Tags companies
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	input body domain.CompanyRequest true "company"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /company [post]
func (h *CompanyHandler) Create(c *gin.Context) {
	var (
		body domain.CompanyRequest
		err  error
	)
	// bind request body
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// validate phone
	if !utils.IsValidPhone(body.Phone) {
		handleResponse(c, BadRequest, "Invalid phone number, Format: 998901234567")
		return
	}
	// create company info
	err = h.db.Table("companies").Create(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, CREATED, "CREATED")
}

// Get company
// @Summary Get a company
// @Description Get a company by id
// @Tags companies
// @Security     BearerAuth
// @Produce json
// @Param 	id path string true "company id"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /company/{id} [get]
func (h *CompanyHandler) Get(c *gin.Context) {
	var (
		id      = c.Param("id")
		err     error
		company domain.Company
	)
	// validate company id
	if err = uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid id")
		return
	}
	// get company info
	err = h.db.First(&company, "id = ?", id).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, company)
}

// Get company
// @Summary Get a company
// @Description Get a company by id
// @Tags companies
// @Security     BearerAuth
// @Produce json
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /company/info [get]
func (h *CompanyHandler) GetInfo(c *gin.Context) {
	var (
		err     error
		company domain.Company
	)
	// get company info
	err = h.db.First(&company).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, company)
}

// Update company
// @Summary Update a company
// @Description Update a company by id
// @Tags companies
// @Security     BearerAuth
// @Produce json
// @Param 	id path string true "company id"
// @Param 	body body domain.CompanyRequest true "company request body"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /company [put]
func (h *CompanyHandler) Update(c *gin.Context) {
	var (
		id   = c.Param("id")
		err  error
		body domain.CompanyRequest
	)
	// validate company id
	if err = uuid.Validate(id); err != nil {
		handleResponse(c, BadRequest, "Invalid id")
		return
	}
	// bind request body
	if err = c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	// validate phone
	if !utils.IsValidPhone(body.Phone) {
		handleResponse(c, BadRequest, "Invalid phone number, Format: 998901234567")
		return
	}
	// update company info
	err = h.db.Model(&domain.Company{}).Where("id = ?", id).Updates(&body).Error
	if err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}

	handleResponse(c, OK, "UPDATED")
}
