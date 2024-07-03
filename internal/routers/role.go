package routers

import (
	mw "github.com/emiliogozo/panahon-api-go/internal/middlewares"
	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/gin-gonic/gin"
)

func (r *DefaultRouter) roleRouter(gr *gin.RouterGroup) {
	roles := gr.Group("/roles")
	{
		rolesAuth := addMiddleware(roles,
			mw.AuthMiddleware(r.tokenMaker, false),
			mw.RoleMiddleware(string(models.SuperAdminRole)))
		rolesAuth.GET("", r.handler.ListRoles)
		rolesAuth.GET(":id", r.handler.GetRole)
		rolesAuth.POST("", r.handler.CreateRole)
		rolesAuth.PUT(":id", r.handler.UpdateRole)
		rolesAuth.DELETE(":id", r.handler.DeleteRole)
	}
}
