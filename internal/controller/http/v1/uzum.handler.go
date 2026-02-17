package v1

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/domain/constants"
)

type UzumHandler struct {
	*Handler
}

func (h *Handler) NewUzumHandler(r *gin.RouterGroup) {
	uzum := &UzumHandler{h}
	uzum.UzumRoutes(r)
}

func (h *UzumHandler) UzumRoutes(r *gin.RouterGroup) {
	r.GET("/nomenclature/:storeId/composition", h.GetNomenclature)
	r.GET("/nomenclature/:storeId/availability", h.GetAvailability)
}

// @Summary      Get Nomenclature Composition
// @Description  Returns the current product catalog with categories for a specific store
// @Tags         uzum
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        storeId path string true "Store ID (UUID)"
// @Param        page query int false "Page number"
// @Param        limit query int false "Items per page"
// @Success      200 {object} domain.NomenclatureResponse
// @Failure      400 {array}  domain.UzumErrorList
// @Failure      401 {array}  domain.UzumErrorList
// @Failure      404 {array}  domain.UzumErrorList
// @Failure      500 {array}  domain.UzumErrorList
// @Router       /integrations/nomenclature/{storeId}/composition [get]
func (h *UzumHandler) GetNomenclature(c *gin.Context) {
	storeId := c.Param("storeId")

	if storeId == "" {
		c.JSON(http.StatusBadRequest, domain.UzumErrorList{
			{Code: 400, Description: "storeId is required"},
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	result, err := h.service.GetNomenclature(ctx, storeId, page, limit)
	if err != nil {
		h.log.Errorf("failed to get nomenclature: %v", err)
		c.JSON(http.StatusInternalServerError, domain.UzumErrorList{
			{Code: 500, Description: "Internal server error"},
		})
		return
	}

	if result == nil || len(result.Items) == 0 {
		c.JSON(http.StatusNotFound, domain.UzumErrorList{
			{Code: 404, Description: "No products found for this store"},
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// @Summary      Get Product Availability
// @Description  Returns product stock levels for a specific store
// @Tags         uzum
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        storeId path string true "Store ID (UUID)"
// @Param        page query int false "Page number"
// @Param        limit query int false "Items per page"
// @Success      200 {object} domain.AvailabilityResponse
// @Failure      400 {array}  domain.UzumErrorList
// @Failure      401 {array}  domain.UzumErrorList
// @Failure      404 {array}  domain.UzumErrorList
// @Failure      500 {array}  domain.UzumErrorList
// @Router       /integrations/nomenclature/{storeId}/availability [get]
func (h *UzumHandler) GetAvailability(c *gin.Context) {
	storeId := c.Param("storeId")

	if storeId == "" {
		c.JSON(http.StatusBadRequest, domain.UzumErrorList{
			{Code: 400, Description: "storeId is required"},
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultContextTimeout)
	defer cancel()

	result, err := h.service.GetAvailability(ctx, storeId, page, limit)
	if err != nil {
		h.log.Errorf("failed to get availability: %v", err)
		c.JSON(http.StatusInternalServerError, domain.UzumErrorList{
			{Code: 500, Description: "Internal server error"},
		})
		return
	}

	c.JSON(http.StatusOK, result)
}
