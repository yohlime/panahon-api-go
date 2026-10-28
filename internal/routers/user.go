package routers

import (
	"time"

	mw "github.com/emiliogozo/panahon-api-go/internal/middlewares"
	"github.com/gin-gonic/gin"
)

func (r *DefaultRouter) userRouter(gr *gin.RouterGroup) {
	users := gr.Group("/users")
	{
		users.POST("/login", r.rateLimit("login", r.config.RateLimit.LoginPerMinute, time.Minute), r.handler.LoginUser)
		users.POST("/logout", r.handler.LogoutUser)
		users.POST("/register", r.rateLimit("registration", r.config.RateLimit.RegistrationPerHour, time.Hour), r.handler.RegisterUser)
		users.POST("/complete-registration/validate", r.rateLimit("registration", r.config.RateLimit.RegistrationPerHour, time.Hour), r.handler.ValidateCompleteRegistration)
		users.POST("/complete-registration/accept", r.rateLimit("registration", r.config.RateLimit.RegistrationPerHour, time.Hour), r.handler.CompleteRegistration)
		users.POST("/password-reset/validate", r.rateLimit("password_reset", r.config.RateLimit.ResetPerMinute, time.Minute), r.handler.ValidatePasswordReset)
		users.POST("/password-reset/accept", r.rateLimit("password_reset", r.config.RateLimit.ResetPerMinute, time.Minute), r.handler.AcceptPasswordReset)

		// API token routes: the user themselves or an admin.
		tokens := addMiddleware(users,
			mw.AuthMiddleware(r.tokenMaker, r.store, false),
			mw.RequireSelfOrAdmin(r.store, "id"))
		tokens.GET(":id/api-tokens", r.handler.ListAPITokens)
		tokens.POST(":id/api-tokens", r.handler.CreateAPIToken)
		tokens.DELETE(":id/api-tokens/:token_id", r.handler.DeleteAPIToken)

		// Self-service profile update; the handler requires the current password.
		profile := addMiddleware(users, mw.AuthMiddleware(r.tokenMaker, r.store, false))
		profile.PUT(":id/profile", r.handler.UpdateProfile)

		// Admin-only user management.
		admin := addMiddleware(users,
			mw.AuthMiddleware(r.tokenMaker, r.store, false),
			mw.AdminMiddleware())
		admin.GET("", r.handler.ListUsers)
		admin.GET(":id", r.handler.GetUser)
		admin.POST("", r.handler.CreateUser)
		admin.PUT(":id", r.handler.UpdateUser)
		admin.DELETE(":id", r.handler.DeleteUser)
		admin.POST(":id/approve", r.handler.ApproveUser)
		admin.POST(":id/reject", r.handler.RejectUser)
		admin.POST(":id/registration-link", r.handler.CreateRegistrationLink)
		admin.POST(":id/password-reset-link", r.handler.CreatePasswordResetLink)

		// Auth user endpoint (requires authentication).
		auth := addMiddleware(users, mw.AuthMiddleware(r.tokenMaker, r.store, false))
		auth.GET("/auth", r.handler.GetAuthUser)
	}
}
