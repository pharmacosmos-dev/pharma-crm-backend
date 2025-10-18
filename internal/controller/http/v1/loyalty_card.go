package v1

import "github.com/gin-gonic/gin"

type LoyaltyCardHandler struct {
	*Handler
}

func (h *Handler) NewLoyaltyCardHandler(r *gin.RouterGroup) {
	loyaltyCardHandler := &LoyaltyCardHandler{h}
	loyaltyCardHandler.LoyaltyCardRoutes(r)
}

func (h *LoyaltyCardHandler) LoyaltyCardRoutes(r *gin.RouterGroup) {
	// loyaltyCard := r.Group("/loyalty_card")
	{
		// loyaltyCard.POST("", h.Create)
		// loyaltyCard.GET("/:id", h.Get)
		// loyaltyCard.GET("/list", h.List)
		// loyaltyCard.PUT("/:id", h.Update)
		// loyaltyCard.DELETE("/:id", h.Delete)
	}
}
