package server

import (
	"context"
	"net/http"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/handlers"
	mw "github.com/emiliogozo/panahon-api-go/internal/middlewares"
	"github.com/emiliogozo/panahon-api-go/internal/models"
	"github.com/emiliogozo/panahon-api-go/internal/token"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"golang.org/x/sync/errgroup"
)

type Server struct {
	config     util.Config
	router     *gin.Engine
	tokenMaker token.Maker
	handler    *handlers.DefaultHandler
	logger     *zerolog.Logger
}

// NewServer creates a new HTTP server and setup routing
func NewServer(config util.Config, store db.Store, tokenMaker token.Maker, logger *zerolog.Logger) (*Server, error) {
	server := &Server{
		config:     config,
		tokenMaker: tokenMaker,
		logger:     logger,
	}

	server.handler = handlers.NewDefaultHandler(config, store, tokenMaker, logger)

	server.setupRouter()

	return server, nil
}

func (s *Server) setupRouter() {
	gin.SetMode(s.config.GinMode)
	r := gin.New()
	r.Use(mw.Zerologger(s.logger), gin.Recovery())

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowCredentials = true
	corsConfig.AddAllowMethods("OPTIONS")
	corsConfig.AddAllowHeaders("Authorization")
	r.Use(cors.New(corsConfig))

	api := r.Group(s.config.APIBasePath)

	users := api.Group("/users")
	{
		users.POST("/login", s.handler.LoginUser)
		users.POST("/logout", s.handler.LogoutUser)
		users.POST("/register", s.handler.RegisterUser)

		usersAuth := addMiddleware(users,
			mw.AuthMiddleware(s.tokenMaker, false))
		usersAuth.GET("/auth", s.handler.GetAuthUser)
		usersAuth = addMiddleware(users,
			mw.RoleMiddleware(string(models.SuperAdminRole)))
		usersAuth.GET("", s.handler.ListUsers)
		usersAuth.GET(":id", s.handler.GetUser)
		usersAuth.POST("", s.handler.CreateUser)
		usersAuth.PUT(":id", s.handler.UpdateUser)
		usersAuth.DELETE(":id", s.handler.DeleteUser)
	}

	api.POST("/tokens/renew", s.handler.RenewAccessToken)

	roles := api.Group("/roles")
	{
		rolesAuth := addMiddleware(roles,
			mw.AuthMiddleware(s.tokenMaker, false),
			mw.RoleMiddleware(string(models.SuperAdminRole)))
		rolesAuth.GET("", s.handler.ListRoles)
		rolesAuth.GET(":id", s.handler.GetRole)
		rolesAuth.POST("", s.handler.CreateRole)
		rolesAuth.PUT(":id", s.handler.UpdateRole)
		rolesAuth.DELETE(":id", s.handler.DeleteRole)
	}

	stations := api.Group("/stations")
	{
		stations.GET("", mw.AuthMiddleware(s.tokenMaker, true), s.handler.ListStations)
		stations.GET(":station_id", s.handler.GetStation)
		stations.GET("/nearest/observations/latest", s.handler.GetNearestLatestStationObservation)

		stnObs := stations.Group(":station_id/observations")
		{
			stnObs.GET("", s.handler.ListStationObservations)
			stnObs.GET("/latest", s.handler.GetLatestStationObservation)
			stnObs.GET(":id", s.handler.GetStationObservation)
		}

		stnAuth := addMiddleware(stations,
			mw.AuthMiddleware(s.tokenMaker, false),
			mw.AdminMiddleware())
		stnAuth.POST("", s.handler.CreateStation)
		stnAuth.PUT(":station_id", s.handler.UpdateStation)
		stnAuth.DELETE(":station_id", s.handler.DeleteStation)

		stnObsAuth := addMiddleware(stnObs,
			mw.AuthMiddleware(s.tokenMaker, false),
			mw.AdminMiddleware())
		{
			stnObsAuth.POST("", s.handler.CreateStationObservation)
			stnObsAuth.PUT(":id", s.handler.UpdateStationObservation)
			stnObsAuth.DELETE(":id", s.handler.DeleteStationObservation)
		}
	}

	observations := api.Group("/observations")
	{
		observations.GET("", s.handler.ListObservations)
		observations.GET("/latest", s.handler.ListLatestObservations)
	}

	glabs := api.Group("/glabs")
	{
		glabs.GET("", s.handler.GLabsOptIn)
		glabs.POST("", s.handler.GLabsUnsubscribe)
		glabs.POST("/load", s.handler.CreateGLabsLoad)
	}

	ptexter := api.Group("/ptexter")
	{
		ptexter.POST("", s.handler.PromoTexterStoreLufft)
	}

	lufft := api.Group("/lufft")
	{
		lufft.GET(":station_id/logs", s.handler.LufftMsgLog)
	}

	api.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	s.router = r
}

func (s *Server) Start(ctx context.Context, g *errgroup.Group) {
	srv := &http.Server{
		Addr:    s.config.HTTPServerAddress,
		Handler: s.router,
	}

	g.Go(func() error {
		s.logger.Info().Msgf("starting gin server: %s", srv.Addr)
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			s.logger.Error().Err(err).Msg("failed to run gin server")
			return err
		}
		return nil
	})

	g.Go(func() error {
		<-ctx.Done()
		s.logger.Info().Msg("shutting down gin server")
		return srv.Shutdown(context.Background())
	})
}

func addMiddleware(r *gin.RouterGroup, m ...gin.HandlerFunc) gin.IRoutes {
	if gin.Mode() != gin.TestMode {
		return r.Use(m...)
	}

	return r
}
