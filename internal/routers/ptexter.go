package routers

import (
	"github.com/gin-gonic/gin"
)

func (r *DefaultRouter) ptexterRouter(gr *gin.RouterGroup) {
	ptexter := gr.Group("/ptexter")
	{
		ptexter.POST("", r.handler.PromoTexterStoreLufft)
	}
}
