package v1

import (
	"fmt"
	"net/http"
	"os"
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
		upload.GET("/:filename", h.ServeFile)
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

	// Check the file size (maximum 5 MB)
	maxFileSize := int64(5 * 1024 * 1024) // 5 MB
	if file.File.Size > maxFileSize {
		h.log.Error("File size exceeds the maximum limit of 5 MB")
		handleResponse(c, BadRequest, "File size exceeds the maximum limit of 5 MB")
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
	savePath := filepath.Join("./app/uploads", newFilename)

	// Save the file
	err = c.SaveUploadedFile(file.File, savePath)
	if err != nil {
		h.log.Warn("Failed to save file: %v", err.Error())
		handleResponse(c, InternalError, "Failed to save file")
		return
	}

	scheme := "http" // Default to http
	if c.Request.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	// Construct the file URL
	fileURL := fmt.Sprintf("%s://%s/v1/upload/%s", scheme, c.Request.Host, newFilename)

	// Return the file URL in the response
	c.JSON(http.StatusOK, gin.H{
		"file_url":  fileURL,
		"file_name": newFilename,
	})
}

// ServeFile godoc
// @Summary Serve a file
// @Description Serve a file by its filename
// @Tags file upload
// @Produce octet/stream
// @Param filename path string true "File name"
// @Success 200 {file} file "File content"
// @Router /upload/{filename} [get]
func (h *UploadHandler) ServeFile(c *gin.Context) {
	// Get the filename from the query or route parameter
	filename := c.Param("filename")
	if filename == "" {
		h.log.Warn("Filename not provided: %v", filename)
		handleResponse(c, BadRequest, "Filename not provided")
		return
	}

	// Construct the full file path
	filePath := filepath.Join("./app/uploads", filename)

	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		h.log.Warn("File not found: %v", err.Error())
		handleResponse(c, NotFound, "File not found")
		return
	}

	// Serve the file
	c.File(filePath)
}
