package routers

import (
	mw "github.com/emiliogozo/panahon-api-go/internal/middlewares"
	"github.com/gin-gonic/gin"
)

func (r *DefaultRouter) accessRouter(gr *gin.RouterGroup) {
	users := gr.Group("/users")
	{
		// Entitlement reads and access requests: the user themselves or an admin.
		selfOrAdmin := addMiddleware(users,
			mw.AuthMiddleware(r.tokenMaker, r.store, false),
			mw.RequireSelfOrAdmin(r.store, "id"))
		selfOrAdmin.GET(":id/access", r.handler.GetUserAccess)
		selfOrAdmin.GET(":id/access-requests", r.handler.ListUserAccessRequests)
		selfOrAdmin.POST(":id/access-requests", r.handler.CreateUserAccessRequest)
		selfOrAdmin.DELETE(":id/access-requests/:request_id", r.handler.CancelUserAccessRequest)

		// Setting an entitlement stays admin-only (users request it instead).
		admin := addMiddleware(users,
			mw.AuthMiddleware(r.tokenMaker, r.store, false),
			mw.AdminMiddleware())
		admin.PUT(":id/access", r.handler.UpdateUserAccess)
	}

	requests := gr.Group("/access-requests")
	adminRequests := addMiddleware(requests,
		mw.AuthMiddleware(r.tokenMaker, r.store, false),
		mw.AdminMiddleware())
	adminRequests.GET("", r.handler.ListAccessRequests)
	adminRequests.POST("/:id/approve", r.handler.ApproveAccessRequest)
	adminRequests.POST("/:id/deny", r.handler.DenyAccessRequest)
}
