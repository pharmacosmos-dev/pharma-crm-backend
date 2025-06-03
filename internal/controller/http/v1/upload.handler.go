package v1

import (
	"fmt"
	"io"
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
		upload.GET("/excel/:xlsx", h.ServeExcelFile)
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
	// get accept language
	lang := c.GetHeader("Accept-Language")
	if lang == "" {
		lang = "en"
	}

	// Bind the file data
	err := c.ShouldBind(&file)
	if err != nil {
		h.log.Error("Failed to bind file: ", err)
		handleResponse(c, BadRequest, "Failed to bind file")
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
		handleResponse(c, UnprocessableEntity, "Invalid file type. Only .jpg, .jpeg, and .png files are allowed.")
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

	// Return the file URL in the response
	c.JSON(http.StatusOK, gin.H{
		"file_url":  newFilename,
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

// ServeFile godoc
// @Summary Serve a file
// @Description Serve a file by its filename
// @Tags file upload
// @Produce octet/stream
// @Param xlsx path string true "File name"
// @Success 200 {file} file "File content"
// @Router /upload/excel/{xlsx} [get]
func (h *UploadHandler) ServeExcelFile(c *gin.Context) {
	xlsx := c.Param("xlsx")
	if xlsx == "" {
		h.log.Warn("Filename not provided: %v", xlsx)
		handleResponse(c, BadRequest, "Filename not provided")
		return
	}

	filePath := filepath.Join("./uploads", xlsx)

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			h.log.Warn("File not found: %v", err.Error())
			handleResponse(c, NotFound, "File not found")
			return
		}
		h.log.Error("Error opening file: %v", err)
		handleResponse(c, InternalError, "Could not open file")
		return
	}
	defer file.Close()

	// Set the headers
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", xlsx))
	c.Header("Content-Type", "application/octet-stream")

	// Stream the file to the client
	if _, err := io.Copy(c.Writer, file); err != nil {
		h.log.Error("Error writing file to response: %v", err)
		handleResponse(c, InternalError, "Could not send file")
		return
	}

	// Remove the file after sending
	if err := os.Remove(filePath); err != nil {
		h.log.Error("Error deleting file after send: %v", err)
	}
}
