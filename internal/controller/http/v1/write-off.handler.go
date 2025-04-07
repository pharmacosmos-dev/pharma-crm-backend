package v1

import "github.com/gin-gonic/gin"

type WriteOffHandler struct {
	*Handler
}

func (h *Handler) NewWriteOffHandler(r *gin.RouterGroup) {
	writeOffHandler := &WriteOffHandler{h}
	writeOffHandler.WriteOffRoutes(r)
}

func (h *WriteOffHandler) WriteOffRoutes(r *gin.RouterGroup) {
	imports := r.Group("/write-off")
	{
		imports.POST("", h.Create)
		// imports.GET("/:id", h.Get)
		// imports.GET("/list", h.List)
		// imports.GET("/export-excel", h.ExportImportExcel)
		// imports.POST("/excel-upload", h.UploadExcelFile)
	}

}

func (h *WriteOffHandler) Create(c *gin.Context) {

}
