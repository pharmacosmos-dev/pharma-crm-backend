package v1

import (
	"fmt"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
		upload.POST("/file", h.Upload)
	}
}

// Upload godoc
// @Summary Upload a file
// @Description Upload a file from the request body
// @Tags file upload
// @Security     BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Product file"
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /upload/file [post]
func (h *UploadHandler) Upload(c *gin.Context) {
	var (
		file domain.File
	)

	// Bind the file data
	err := c.ShouldBind(&file)
	if err != nil {
		h.log.Error("Failed to bind file: ", err)
		handleResponse(c, BadRequest, err.Error())
		return
	}

	// Check file type
	ext := filepath.Ext(file.File.Filename)
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		h.log.Error("Invalid file type")
		handleResponse(c, BadRequest, "Invalid file type")
		return
	}

	// Generate a new filename using UUID
	newFilename := uuid.New().String() + ext

	// Define the save path (adjust the directory as needed)
	savePath := filepath.Join("uploads", newFilename)

	// Save the file
	err = c.SaveUploadedFile(file.File, savePath)
	if err != nil {
		h.log.Error("Failed to save file: ", err)
		handleResponse(c, InternalError, "Failed to save file")
		return
	}

	scheme := "http" // Default to http
	if c.Request.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	// Construct the file URL
	fileURL := fmt.Sprintf("%s://%s/uploads/%s", scheme, c.Request.Host, newFilename)

	// Return the file URL in the response
	handleResponse(c, OK, fileURL)
}
