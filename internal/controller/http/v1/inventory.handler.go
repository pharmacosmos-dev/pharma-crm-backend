package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
)

type InventoryHandler struct {
	*Handler
}

func (h *Handler) NewInventoryHandler(r *gin.RouterGroup) {
	inventoryHandler := &InventoryHandler{h}
	inventoryHandler.InventoryRoutes(r)
}

func (h *InventoryHandler) InventoryRoutes(r *gin.RouterGroup) {
	inventory := r.Group("/import")
	{
		inventory.POST("", h.Create)
		// imports.GET("/:id", h.Get)
		// imports.GET("/list", h.List)
		// imports.GET("/export-excel", h.ExportImportExcel)
		// imports.POST("/excel-upload", h.UploadExcelFile)
	}

}

// Create godoc
// @Summary Create inventory
// @Description Create inventory
// @Tags Inventory
// @Security     BearerAuth
// @Accept 	json
// @Produce json
// @Param 	inventory body domain.InventoryRequest true "Inventory"
// @Success 201 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /inventory [POST]
func (h *InventoryHandler) Create(c *gin.Context) {
	var inventoryRequest domain.InventoryRequest
	// Bind the request body to the InventoryRequest struct
	if err := c.ShouldBindJSON(&inventoryRequest); err != nil {
		h.log.Warn("Error on binding request: %v", err.Error())
		handleResponse(c, BadRequest, "Invalid request body")
		return
	}
	// create inventory
	err := h.service.CreateInventory(&inventoryRequest)
	if err != nil {
		h.log.Warn("Error on creating inventory: %v", err.Error())
		handleResponse(c, InternalError, "Failed to create inventory")
		return
	}

	handleResponse(c, CREATED, "CREATED")
}

func (h *InventoryHandler) Get(c *gin.Context) {

}
func (h *InventoryHandler) List(c *gin.Context) {

}

func (h *InventoryHandler) ExportImportExcel(c *gin.Context) {

}
