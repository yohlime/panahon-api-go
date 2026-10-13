package routers

import (
	mw "github.com/emiliogozo/panahon-api-go/internal/middlewares"
	"github.com/gin-gonic/gin"
)

func (r *DefaultRouter) observationRouter(gr *gin.RouterGroup) {
	observations := gr.Group("/observations")
	{
		// Public snapshot (registered before the gated middleware is added).
		observations.GET("/latest", r.handler.ListLatestObservations)

		// The bulk list exposes whole stations, so it requires auth (admin or a
		// validated, scoped API token).
		gated := addMiddleware(observations,
			mw.AuthMiddleware(r.tokenMaker, true),
			mw.APITokenMiddleware(r.store, true),
			mw.RequireAuthOrAPIToken(),
		)
		gated.GET("", r.handler.ListObservations)
	}
}
