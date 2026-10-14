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

		// Self-service API token routes (for authenticated users)
		me := addMiddleware(users, mw.AuthMiddleware(r.tokenMaker, r.store, false))
		me.GET("/me/api-tokens", r.handler.ListAPITokens)
		me.POST("/me/api-tokens", r.handler.CreateAPIToken)
		me.DELETE("/me/api-tokens/:token_id", r.handler.DeleteAPIToken)

		// Admin-only routes
		admin := addMiddleware(users,
			mw.AuthMiddleware(r.tokenMaker, r.store, false),
			mw.AdminMiddleware())
		admin.GET("", r.handler.ListUsers)
		admin.GET(":id", r.handler.GetUser)
		admin.POST("", r.handler.CreateUser)
		admin.PUT(":id", r.handler.UpdateUser)
		admin.DELETE(":id", r.handler.DeleteUser)

		// Admin API token routes
		admin.GET(":id/api-tokens", r.handler.ListUserAPITokens)
		admin.POST(":id/api-tokens", r.handler.CreateUserAPIToken)
		admin.DELETE(":id/api-tokens/:token_id", r.handler.DeleteUserAPIToken)

		// Auth user endpoint (requires authentication)
		auth := addMiddleware(users, mw.AuthMiddleware(r.tokenMaker, r.store, false))
		auth.GET("/auth", r.handler.GetAuthUser)
	}
}
