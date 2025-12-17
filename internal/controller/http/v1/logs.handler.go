package v1

import "github.com/gin-gonic/gin"

type LogHandler struct {
	*Handler
}

func (h *Handler) NewLogHandler(r *gin.RouterGroup) {
	helper := &LogHandler{h}
	helper.LogRoutes(r)
}

func (h *LogHandler) LogRoutes(r *gin.RouterGroup) {
	logs := r.Group("/logs")
	{
		logs.GET("", h.FetchLogs)
	}
}

func (h *LogHandler) FetchLogs(c *gin.Context) {

}
