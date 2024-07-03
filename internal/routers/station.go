package routers

import (
	mw "github.com/emiliogozo/panahon-api-go/internal/middlewares"
	"github.com/gin-gonic/gin"
)

func (r *DefaultRouter) stationRouter(gr *gin.RouterGroup) {
	stations := gr.Group("/stations")
	{
		stations.GET("", mw.AuthMiddleware(r.tokenMaker, true), r.handler.ListStations)
		stations.GET(":station_id", r.handler.GetStation)
		stations.GET("/nearest/observations/latest", r.handler.GetNearestLatestStationObservation)

		stnObs := stations.Group(":station_id/observations")
		{
			stnObs.GET("", r.handler.ListStationObservations)
			stnObs.GET("/latest", r.handler.GetLatestStationObservation)
			stnObs.GET(":id", r.handler.GetStationObservation)
		}

		stnAuth := addMiddleware(stations,
			mw.AuthMiddleware(r.tokenMaker, false),
			mw.AdminMiddleware())
		stnAuth.POST("", r.handler.CreateStation)
		stnAuth.PUT(":station_id", r.handler.UpdateStation)
		stnAuth.DELETE(":station_id", r.handler.DeleteStation)

		stnObsAuth := addMiddleware(stnObs,
			mw.AuthMiddleware(r.tokenMaker, false),
			mw.AdminMiddleware())
		{
			stnObsAuth.POST("", r.handler.CreateStationObservation)
			stnObsAuth.PUT(":id", r.handler.UpdateStationObservation)
			stnObsAuth.DELETE(":id", r.handler.DeleteStationObservation)
		}
	}
}
