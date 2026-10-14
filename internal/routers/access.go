package routers

import (
	mw "github.com/emiliogozo/panahon-api-go/internal/middlewares"
	"github.com/gin-gonic/gin"
)

func (r *DefaultRouter) accessRouter(gr *gin.RouterGroup) {
	users := gr.Group("/users")
	{
		me := addMiddleware(users, mw.AuthMiddleware(r.tokenMaker, r.store, false))
		me.GET("/me/access", r.handler.GetMyAccess)
		me.POST("/me/access-requests", r.handler.CreateMyAccessRequest)
		me.GET("/me/access-requests", r.handler.ListMyAccessRequests)
		me.DELETE("/me/access-requests/:id", r.handler.CancelMyAccessRequest)

		admin := addMiddleware(users, mw.AuthMiddleware(r.tokenMaker, r.store, false), mw.AdminMiddleware())
		admin.GET("/:id/access", r.handler.GetUserAccess)
		admin.PUT("/:id/access", r.handler.UpdateUserAccess)
	}

	requests := gr.Group("/access-requests")
	adminRequests := addMiddleware(requests, mw.AuthMiddleware(r.tokenMaker, r.store, false), mw.AdminMiddleware())
	adminRequests.GET("", r.handler.ListAccessRequests)
	adminRequests.POST("/:id/approve", r.handler.ApproveAccessRequest)
	adminRequests.POST("/:id/deny", r.handler.DenyAccessRequest)
}
