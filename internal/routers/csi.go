package routers

import (
	"github.com/gin-gonic/gin"
)

func (r *DefaultRouter) csiRouter(gr *gin.RouterGroup) {
	ptexter := gr.Group("/csi")
	{
		ptexter.POST("", r.handler.CSIStoreMisol)
	}
}
