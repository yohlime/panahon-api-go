package server

import (
	"context"
	"net/http"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/handlers"
	"github.com/emiliogozo/panahon-api-go/internal/routers"
	"github.com/emiliogozo/panahon-api-go/internal/token"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

type Server struct {
	config util.Config
	router *routers.DefaultRouter
	logger *zerolog.Logger
}

// NewServer creates a new HTTP server and setup routing
func NewServer(config util.Config, store db.Store, tokenMaker token.Maker, logger *zerolog.Logger) (*Server, error) {
	server := &Server{
		config: config,
		logger: logger,
	}

	handler := handlers.NewDefaultHandler(config, store, tokenMaker, logger)

	server.router = routers.NewDefaultRouter(config, handler, tokenMaker, logger)

	return server, nil
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
