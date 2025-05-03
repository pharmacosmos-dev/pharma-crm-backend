package v1

import "github.com/gin-gonic/gin"

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
		tasnif.POST("")
	}
}


func (h *TasnifHandler) UpdateTasnifInfos(c *gin.Context) {

}