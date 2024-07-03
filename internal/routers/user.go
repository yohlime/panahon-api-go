package routers

import (
	mw "github.com/emiliogozo/panahon-api-go/internal/middlewares"
	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/gin-gonic/gin"
)

func (r *DefaultRouter) userRouter(gr *gin.RouterGroup) {
	users := gr.Group("/users")
	{
		users.POST("/login", r.handler.LoginUser)
		users.POST("/logout", r.handler.LogoutUser)
		users.POST("/register", r.handler.RegisterUser)

		usersAuth := addMiddleware(users,
			mw.AuthMiddleware(r.tokenMaker, false))
		usersAuth.GET("/auth", r.handler.GetAuthUser)
		usersAuth = addMiddleware(users,
			mw.RoleMiddleware(string(models.SuperAdminRole)))
		usersAuth.GET("", r.handler.ListUsers)
		usersAuth.GET(":id", r.handler.GetUser)
		usersAuth.POST("", r.handler.CreateUser)
		usersAuth.PUT(":id", r.handler.UpdateUser)
		usersAuth.DELETE(":id", r.handler.DeleteUser)
	}
}
