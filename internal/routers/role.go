package routers

import (
	mw "github.com/emiliogozo/panahon-api-go/internal/middlewares"
	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/gin-gonic/gin"
)

func (r *DefaultRouter) roleRouter(gr *gin.RouterGroup) {
	roles := gr.Group("/roles")
	{
		// Anyone at admin level can read roles (e.g. to populate the user role
		// picker), but only a superadmin may create/update/delete them.
		rolesRead := addMiddleware(roles,
			mw.AuthMiddleware(r.tokenMaker, r.store, false),
			mw.AdminMiddleware())
		rolesRead.GET("", r.handler.ListRoles)
		rolesRead.GET(":id", r.handler.GetRole)

		rolesWrite := addMiddleware(roles,
			mw.AuthMiddleware(r.tokenMaker, r.store, false),
			mw.RoleMiddleware(string(models.SuperAdminRole)))
		rolesWrite.POST("", r.handler.CreateRole)
		rolesWrite.PUT(":id", r.handler.UpdateRole)
		rolesWrite.DELETE(":id", r.handler.DeleteRole)
	}
}
