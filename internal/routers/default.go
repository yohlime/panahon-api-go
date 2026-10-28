package routers

import (
	"strings"
	"time"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/handlers"
	mw "github.com/emiliogozo/panahon-api-go/internal/middlewares"
	"github.com/emiliogozo/panahon-api-go/internal/ratelimit"
	"github.com/emiliogozo/panahon-api-go/internal/token"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

type DefaultRouter struct {
	handler    *handlers.DefaultHandler
	tokenMaker token.Maker
	store      db.Store
	config     util.Config
	limiter    *ratelimit.Limiter
	*gin.Engine
}

func NewDefaultRouter(config util.Config, handler *handlers.DefaultHandler, tokenMaker token.Maker, store db.Store, logger *zerolog.Logger) *DefaultRouter {
	gin.SetMode(config.GinMode)
	g := gin.New()

	// Ignore forwarded headers unless the immediate peer is a configured
	// proxy. Otherwise clients could spoof X-Forwarded-For and bypass per-IP
	// limits.
	trustedProxies, err := util.ParseTrustedProxies(config.TrustedProxies)
	if err != nil {
		logger.Error().Err(err).Msg("invalid TRUSTED_PROXIES; ignoring forwarded headers")
		trustedProxies = nil
	}
	if err := g.SetTrustedProxies(trustedProxies); err != nil {
		logger.Error().Err(err).Msg("cannot set trusted proxies; ignoring forwarded headers")
		_ = g.SetTrustedProxies(nil)
	}

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
		config:     config,
		limiter:    ratelimit.New(),
		Engine:     g,
	}

	// Probes live outside the API prefix so orchestrators can reach them
	// regardless of how the public API is mounted.
	g.GET("/healthz", r.handler.Healthz)
	g.GET("/readyz", r.handler.Readyz)

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

	registerDocsRoutes(api, config.SwagAPIBasePath)

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

func (r *DefaultRouter) rateLimit(name string, limit int, window time.Duration) gin.HandlerFunc {
	if !r.config.RateLimit.Enabled {
		limit = 0
	}
	return mw.RateLimit(r.limiter, name, limit, window)
}

func (r *DefaultRouter) apiKeyRateLimit() gin.HandlerFunc {
	limit := r.config.RateLimit.APIKeyPerMinute
	if !r.config.RateLimit.Enabled {
		limit = 0
	}
	return mw.APIKeyRateLimit(r.limiter, limit, time.Minute)
}
