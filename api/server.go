package api

import (
	"fmt"

	db "github.com/emiliogozo/panahon-api-go/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/token"
	"github.com/emiliogozo/panahon-api-go/util"
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
func NewServer(config util.Config, store db.Store, logger *zerolog.Logger) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}
	server := &Server{
		config:     config,
		store:      store,
		tokenMaker: tokenMaker,
		logger:     logger,
	}

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("mobile_number", validMobileNumber)
		v.RegisterValidation("alphanumspace", validAlphaNumSpace)
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
			authMiddleware(s.tokenMaker))
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
			authMiddleware(s.tokenMaker),
			roleMiddleware(string(superAdminRole)))
		rolesAuth.GET("", s.ListRoles)
		rolesAuth.GET(":id", s.GetRole)
		rolesAuth.POST("", s.CreateRole)
		rolesAuth.PUT(":id", s.UpdateRole)
		rolesAuth.DELETE(":id", s.DeleteRole)
	}

	stations := api.Group("/stations")
	{
		stations.GET("", s.ListStations)
		stations.GET(":station_id", s.GetStation)

		stnObservations := stations.Group(":station_id/observations")
		{
			stnObservations.GET("", s.ListStationObservations)
			stnObservations.GET(":id", s.GetStationObservation)
		}

		stationsAuth := addMiddleware(stations,
			authMiddleware(s.tokenMaker),
			adminMiddleware())
		stationsAuth.POST("", s.CreateStation)
		stationsAuth.PUT(":station_id", s.UpdateStation)
		stationsAuth.DELETE(":station_id", s.DeleteStation)

		stnObservationsAuth := addMiddleware(stnObservations,
			authMiddleware(s.tokenMaker),
			adminMiddleware())
		{
			stnObservationsAuth.POST("", s.CreateStationObservation)
			stnObservationsAuth.PUT(":id", s.UpdateStationObservation)
			stnObservationsAuth.DELETE(":id", s.DeleteStationObservation)
		}

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
