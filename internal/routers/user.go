package routers

import (
	mw "github.com/emiliogozo/panahon-api-go/internal/middlewares"
	"github.com/gin-gonic/gin"
)

func (r *DefaultRouter) userRouter(gr *gin.RouterGroup) {
	users := gr.Group("/users")
	{
		users.POST("/login", r.handler.LoginUser)
		users.POST("/logout", r.handler.LogoutUser)
		users.POST("/register", r.handler.RegisterUser)

		// API token routes: the user themselves or an admin.
		tokens := addMiddleware(users,
			mw.AuthMiddleware(r.tokenMaker, r.store, false),
			mw.RequireSelfOrAdmin(r.store, "id"))
		tokens.GET(":id/api-tokens", r.handler.ListAPITokens)
		tokens.POST(":id/api-tokens", r.handler.CreateAPIToken)
		tokens.DELETE(":id/api-tokens/:token_id", r.handler.DeleteAPIToken)

		// Admin-only user management.
		admin := addMiddleware(users,
			mw.AuthMiddleware(r.tokenMaker, r.store, false),
			mw.AdminMiddleware())
		admin.GET("", r.handler.ListUsers)
		admin.GET(":id", r.handler.GetUser)
		admin.POST("", r.handler.CreateUser)
		admin.PUT(":id", r.handler.UpdateUser)
		admin.DELETE(":id", r.handler.DeleteUser)

		// Auth user endpoint (requires authentication).
		auth := addMiddleware(users, mw.AuthMiddleware(r.tokenMaker, r.store, false))
		auth.GET("/auth", r.handler.GetAuthUser)
	}
}
