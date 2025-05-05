package v1

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
)

type TasnifHandler struct {
	*Handler
}

func (h *Handler) NewTasnifHandler(r *gin.RouterGroup) {
	t := TasnifHandler{h}
	t.TasnifRoutes(r)
}

func (h *TasnifHandler) TasnifRoutes(r *gin.RouterGroup) {
	tasnif := r.Group("/tasnif")
	{
		tasnif.POST("/update-package-code", h.UpdatePackageCode)
	}
}

// update package code with tasnif API
// @Summary update package code with tasnif API
// @Description update package code with tasnif API
// @Tags Tasnif
// @Accept 	json
// @Produce json
// @Success 200 {object} v1.Response
// @Failure 400 {object} v1.Response
// @Failure 500 {object} v1.Response
// @Router /tasnif/update-package-code [POST]
func (h *TasnifHandler) UpdatePackageCode(c *gin.Context) {

	client := &http.Client{}

	url := "https://tasnif.soliq.uz/api/cls-api/mxik/get/by-mxik?mxikCode=03004061008008002&lang=uz_latn"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		h.log.Warn("ERROR on creating new request: %v", err)
		handleResponse(c, InternalError, "Can't create new request")
		return
	}
	// add headers
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		h.log.Warn("ERROR on doing request: %v", err)
		handleResponse(c, InternalError, "Can't do request")
		return
	}
	defer resp.Body.Close()
	var res domain.TasnifResponse
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		h.log.Warn("ERROR on decoding epos response %v", err)
		handleResponse(c, InternalError, "Can't decode response data")
		return
	}

	c.JSON(http.StatusOK, res)
}
