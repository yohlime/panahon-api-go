package routers

import (
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
	*gin.Engine
}

func NewDefaultRouter(config util.Config, handler *handlers.DefaultHandler, tokenMaker token.Maker, logger *zerolog.Logger) *DefaultRouter {
	gin.SetMode(config.GinMode)
	g := gin.New()
	g.Use(mw.Zerologger(logger), gin.Recovery())

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowCredentials = true
	corsConfig.AddAllowMethods("OPTIONS")
	corsConfig.AddAllowHeaders("Authorization")
	g.Use(cors.New(corsConfig))

	api := g.Group(config.APIBasePath)

	r := DefaultRouter{
		handler:    handler,
		tokenMaker: tokenMaker,
		Engine:     g,
	}

	r.userRouter(api)
	r.roleRouter(api)
	r.stationRouter(api)
	r.observationRouter(api)
	r.glabsRouter(api)
	r.ptexterRouter(api)
	r.lufftRouter(api)
	r.csiRouter(api)

	api.POST("/tokens/renew", r.handler.RenewAccessToken)

	api.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return &r
}

func addMiddleware(r *gin.RouterGroup, m ...gin.HandlerFunc) gin.IRoutes {
	if gin.Mode() != gin.TestMode {
		return r.Use(m...)
	}

	return r
}
