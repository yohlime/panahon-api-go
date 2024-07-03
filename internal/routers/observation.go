package routers

import (
	"github.com/gin-gonic/gin"
)

func (r *DefaultRouter) observationRouter(gr *gin.RouterGroup) {
	observations := gr.Group("/observations")
	{
		observations.GET("", r.handler.ListObservations)
		observations.GET("/latest", r.handler.ListLatestObservations)
	}
}
