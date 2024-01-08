package api

import (
	db "github.com/emiliogozo/panahon-api-go/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/token"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Server struct {
	config     util.Config
	router     *gin.Engine
	store      db.Store
	tokenMaker token.Maker
	logger     *zerolog.Logger
}

// NewServer creates a new HTTP server and setup routing
func NewServer(config util.Config, store db.Store, tokenMaker token.Maker, logger *zerolog.Logger) (*Server, error) {
	server := &Server{
		config:     config,
		store:      store,
		tokenMaker: tokenMaker,
		logger:     logger,
	}

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("mobile_number", validMobileNumber)
		v.RegisterValidation("alphanumspace", validAlphaNumSpace)
		v.RegisterValidation("date_time", validDateTimeStr)
	}

	server.setupRouter()

	return server, nil
}

func (s *Server) setupRouter() {
	gin.SetMode(s.config.GinMode)
	r := gin.Default()

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowCredentials = true
	corsConfig.AddAllowMethods("OPTIONS")
	corsConfig.AddAllowHeaders("Authorization")
	r.Use(cors.New(corsConfig))

	api := r.Group(s.config.APIBasePath)

	users := api.Group("/users")
	{
		users.POST("/login", s.LoginUser)
		users.POST("/register", s.RegisterUser)

		usersAuth := addMiddleware(users,
			authMiddleware(s.tokenMaker, false))
		usersAuth.GET("/auth", s.GetAuthUser)
		usersAuth = addMiddleware(users,
			roleMiddleware(string(superAdminRole)))
		usersAuth.GET("", s.ListUsers)
		usersAuth.GET(":id", s.GetUser)
		usersAuth.POST("", s.CreateUser)
		usersAuth.PUT(":id", s.UpdateUser)
		usersAuth.DELETE(":id", s.DeleteUser)
	}

	api.POST("/tokens/renew", s.RenewAccessToken)

	roles := api.Group("/roles")
	{
		rolesAuth := addMiddleware(roles,
			authMiddleware(s.tokenMaker, false),
			roleMiddleware(string(superAdminRole)))
		rolesAuth.GET("", s.ListRoles)
		rolesAuth.GET(":id", s.GetRole)
		rolesAuth.POST("", s.CreateRole)
		rolesAuth.PUT(":id", s.UpdateRole)
		rolesAuth.DELETE(":id", s.DeleteRole)
	}

	stations := api.Group("/stations")
	{
		stations.GET("", authMiddleware(s.tokenMaker, true), s.ListStations)
		stations.GET(":station_id", s.GetStation)
		stations.GET("/nearest/observations/latest", s.GetNearestLatestStationObservation)

		stnObs := stations.Group(":station_id/observations")
		{
			stnObs.GET("", s.ListStationObservations)
			stnObs.GET("/latest", s.GetLatestStationObservation)
			stnObs.GET(":id", s.GetStationObservation)
		}

		stnAuth := addMiddleware(stations,
			authMiddleware(s.tokenMaker, false),
			adminMiddleware())
		stnAuth.POST("", s.CreateStation)
		stnAuth.PUT(":station_id", s.UpdateStation)
		stnAuth.DELETE(":station_id", s.DeleteStation)

		stnObsAuth := addMiddleware(stnObs,
			authMiddleware(s.tokenMaker, false),
			adminMiddleware())
		{
			stnObsAuth.POST("", s.CreateStationObservation)
			stnObsAuth.PUT(":id", s.UpdateStationObservation)
			stnObsAuth.DELETE(":id", s.DeleteStationObservation)
		}
	}

	observations := api.Group("/observations")
	{
		observations.GET("", s.ListObservations)
		observations.GET("/latest", s.ListLatestObservations)
	}

	glabs := api.Group("/glabs")
	{
		glabs.GET("", s.GLabsOptIn)
		glabs.POST("", s.GLabsUnsubscribe)
		glabs.POST("/load", s.CreateGLabsLoad)
	}

	ptexter := api.Group("/ptexter")
	{
		ptexter.POST("", s.PromoTexterStoreLufft)
	}

	lufft := api.Group("/lufft")
	{
		lufft.GET(":station_id/logs", s.LufftMsgLog)
	}

	api.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	s.router = r
}

func (s *Server) Start(address string) error {
	return s.router.Run(address)
}

func addMiddleware(r *gin.RouterGroup, m ...gin.HandlerFunc) gin.IRoutes {
	if gin.Mode() != gin.TestMode {
		return r.Use(m...)
	}

	return r
}

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}
