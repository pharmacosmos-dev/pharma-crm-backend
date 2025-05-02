package v1

import (
	"github.com/gin-gonic/gin"
)

type DiscountCardHandler struct {
	*Handler
}

func (h *Handler) NewDiscountCardHandler(r *gin.RouterGroup) {
	discountCard := &DiscountCardHandler{h}
	discountCard.DiscountCardRoutes(r)
}

func (h *DiscountCardHandler) DiscountCardRoutes(r *gin.RouterGroup) {
	discountCard := r.Group("/discount-card")
	{
		discountCard.POST("")
	}
}
