package routers

import (
	"github.com/gin-gonic/gin"
)

func (r *DefaultRouter) glabsRouter(gr *gin.RouterGroup) {
	glabs := gr.Group("/glabs")
	{
		glabs.GET("", r.handler.GLabsOptIn)
		glabs.POST("", r.handler.GLabsUnsubscribe)
		glabs.POST("/load", r.handler.CreateGLabsLoad)
	}
}
