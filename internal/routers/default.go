package routers

import (
	"strings"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/handlers"
	mw "github.com/emiliogozo/panahon-api-go/internal/middlewares"
	"github.com/emiliogozo/panahon-api-go/internal/token"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type DefaultRouter struct {
	handler    *handlers.DefaultHandler
	tokenMaker token.Maker
	store      db.Store
	*gin.Engine
}

func NewDefaultRouter(config util.Config, handler *handlers.DefaultHandler, tokenMaker token.Maker, store db.Store, logger *zerolog.Logger) *DefaultRouter {
	gin.SetMode(config.GinMode)
	g := gin.New()
	g.Use(mw.Zerologger(logger), gin.Recovery())

	corsConfig := cors.DefaultConfig()
	var origins []string
	for _, o := range strings.Split(config.CorsAllowOrigins, ",") {
		if o = strings.TrimSpace(o); o != "" {
			origins = append(origins, o)
		}
	}
	if len(origins) == 0 {
		// No configured origins: allow any origin but never send credentials,
		// which keeps cookie-auth requests effectively same-origin only.
		corsConfig.AllowAllOrigins = true
		corsConfig.AllowCredentials = false
	} else {
		corsConfig.AllowOrigins = origins
		corsConfig.AllowCredentials = true
	}
	corsConfig.AddAllowMethods("OPTIONS")
	corsConfig.AddAllowHeaders("Authorization")
	g.Use(cors.New(corsConfig))

	api := g.Group(config.APIBasePath)

	r := DefaultRouter{
		handler:    handler,
		tokenMaker: tokenMaker,
		store:      store,
		Engine:     g,
	}

	r.userRouter(api)
	r.accessRouter(api)
	r.roleRouter(api)
	r.stationRouter(api)
	r.observationRouter(api)
	r.glabsRouter(api)
	r.smsRouter(api)
	r.lufftRouter(api)
	r.csiRouter(api)

	api.POST("/tokens/renew", r.handler.RenewAccessToken)

	api.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return &r
}

// addMiddleware returns a new group with the given middleware so that
// registering one group's middleware never leaks into sibling groups
// (gin's RouterGroup.Use mutates the group, accumulating on the parent).
func addMiddleware(r *gin.RouterGroup, m ...gin.HandlerFunc) gin.IRoutes {
	if gin.Mode() != gin.TestMode {
		return r.Group("", m...)
	}

	return r
}
