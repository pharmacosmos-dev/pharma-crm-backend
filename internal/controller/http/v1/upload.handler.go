package v1

import (
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
)

type UploadHandler struct {
	*Handler
}

func (h *Handler) NewUploadHandler(r *gin.RouterGroup) {
	upload := &UploadHandler{h}
	upload.UploadRoutes(r)
}

func (h *UploadHandler) UploadRoutes(r *gin.RouterGroup) {
	upload := r.Group("/upload")
	{
		upload.POST("", h.Upload)
	}
}

// Upload godoc
// @Summary Upload a product
// @Description Upload a product from the request body
// @Tags file upload
// @Security     BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Product file"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /product/import [post]
func (h *UploadHandler) Upload(c *gin.Context) {
	var (
		file domain.File
	)

	err := c.ShouldBind(&file)
	if err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	ext := filepath.Ext(file.File.Filename)
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		h.log.Error("Invalid file type")
		handleResponse(c, BadRequest, "Invalid file type")
		return
	}
}
