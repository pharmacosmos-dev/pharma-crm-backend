package v1

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pharma-crm-backend/domain"
	"github.com/spf13/cast"
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
	var products []domain.ProductData
	// get product list
	err := h.db.Table("products").Where("mxik is not null AND mxik <> '' and unit_code is null").Find(&products).Error
	if err != nil {
		h.log.Warn("ERROR on getting product list: %v", err)
		handleResponse(c, InternalError, "Can't get product get list")
		return
	}
	var count = 0
	// declare client
	client := &http.Client{}

	for _, p := range products {
		// Rebuild URL correctly for each product
		url := "https://tasnif.soliq.uz/api/cls-api/mxik/get/by-mxik?mxikCode=" + p.MXIK + "&lang=uz_latn"
		// create new request
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			h.log.Warn("ERROR on creating new request: %v", err)
			handleResponse(c, InternalError, "Can't create new request")
			return
		}
		// add headers
		req.Header.Add("Content-Type", "application/json")

		// doing the request
		resp, err := client.Do(req)
		if err != nil {
			h.log.Warn("ERROR on doing request: %v", err)
			handleResponse(c, InternalError, "Can't do request")
			return
		}
		defer resp.Body.Close()
		var res domain.TasnifResponse
		result, _ := io.ReadAll(resp.Body)

		// decode the tasnif response
		err = json.Unmarshal(result, &res)
		// err = json.NewDecoder(resp.Body).Decode(&res)
		if err != nil {
			h.log.Warn("ERROR on decoding tasnif %v", err)
			fmt.Println("--->>> ", string(result))
			fmt.Println("====>> ", p.MXIK)
			handleResponse(c, InternalError, "Can't decode response data")
			return
		}

		err = h.db.Debug().Model(&domain.MxikPackage{}).Create(&res.Packages).Error
		if err != nil {
			h.log.Warn("ERROR on inserting mxik_packages: %v", err)
		}
		count++
	}
	fmt.Println("COUNT: ", count)
	c.JSON(http.StatusOK, "SUCCESS: "+cast.ToString(count))
}
