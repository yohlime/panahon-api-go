package routers

import (
	"github.com/gin-gonic/gin"
)

func (r *DefaultRouter) lufftRouter(gr *gin.RouterGroup) {
	lufft := gr.Group("/lufft")
	{
		lufft.GET(":station_id/logs", r.handler.LufftMsgLog)
	}
}
