package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/domain"
)

type BrandController struct {
	*Handler
}

func (h *Handler) NewBrandController(r *gin.RouterGroup) {
	brand := &BrandController{h}
	brand.BrandRoutes(r)
}

func (b *BrandController) BrandRoutes(r *gin.RouterGroup) {
	apiGroup := r.Group("/brand")
	{
		apiGroup.POST("", b.Create)
		apiGroup.GET("", b.Get)
		apiGroup.GET("/get-list", b.List)
		apiGroup.PUT("", b.Update)
		apiGroup.DELETE("", b.Delete)
	}

}

func (b *BrandController) Create(c *gin.Context) {
	var (
		brand = new(domain.Brand)
		res   = new(domain.Brand)
	)

	if err := c.ShouldBind(brand); err != nil {
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	brand.Id = uuid.New().String()
	if err := b.Db.WithContext(c.Request.Context()).Create(brand).Scan(&res).Error; err != nil {
		b.Log.Error("Error on creating brand: ", err.Error())
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusCreated, MsgSuccessCreate, res)
}

func (b *BrandController) Get(c *gin.Context) {
	res := new(domain.Brand)
	if err := b.Db.First(res, "id = ?", c.Query("id")).Error; err != nil {
		b.Log.Error("Error on getting brand: ", err.Error())
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessFetch, res)
}

func (b *BrandController) List(c *gin.Context) {
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}
	var res []domain.Brand
	if err := b.Db.Limit(limit).Offset(offset).Find(&res).Error; err != nil {
		b.Log.Error("Error on list brand: ", err.Error())
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessFetch, res)
}

func (b *BrandController) Update(c *gin.Context) {
	var brand RequestBody[domain.Brand]
	var res = new(domain.Brand)
	if err := c.ShouldBindJSON(&brand); err != nil {
		handleResponse(c, http.StatusBadRequest, MsgErrInvalidRequest, err.Error())
		return
	}

	if err := b.Db.WithContext(c.Request.Context()).Model(res).Where("id = ?", brand.Data.Id).
		Updates(brand.Data).Error; err != nil {
		b.Log.Error("Error on update brand: ", err.Error())
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessUpdate, res)
}

func (b *BrandController) Delete(c *gin.Context) {

	if err := b.Db.WithContext(c.Request.Context()).Delete(&domain.Brand{}, "id = ?", c.Query("id")).Error; err != nil {
		b.Log.Error("Error on delete brand: ", err.Error())
		handleResponse(c, http.StatusInternalServerError, MsgErrInternal, err.Error())
		return
	}
	handleResponse(c, http.StatusOK, MsgSuccessDelete, MsgSuccessDelete)
}
