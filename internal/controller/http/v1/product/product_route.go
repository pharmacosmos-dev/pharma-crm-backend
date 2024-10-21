package product

import "github.com/gin-gonic/gin"

func NewProductRouter(r *gin.Engine) {
	product := r.Group("/product")

	product.GET("/new")

}
