package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	docs "github.com/emiliogozo/panahon-api-go/internal/docs"
	"github.com/emiliogozo/panahon-api-go/internal/server"
	"github.com/emiliogozo/panahon-api-go/internal/service"
	"github.com/emiliogozo/panahon-api-go/internal/token"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

var interruptSignals = []os.Signal{
	os.Interrupt,
	syscall.SIGTERM,
	syscall.SIGINT,
}

// PanahonAPI
//
//	@title						Panahon API
//	@version					1.0
//	@description				Panahon API.
//	@contact.name				Emilio Gozo
//	@contact.email				emiliogozo@proton.me
//
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
func main() {
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal().Err(err).Msg("cannot load config")
	}

	logger := util.NewLogger(config)

	docs.SwaggerInfo.BasePath = config.SwagAPIBasePath

	ctx, stop := signal.NotifyContext(context.Background(), interruptSignals...)
	defer stop()

	connPool, err := pgxpool.New(ctx, config.DBSource)
	if err != nil {
		logger.Fatal().Err(err).Msg("cannot connect to db")
	}

	migration, err := migrate.New("file://"+config.MigrationPath, config.DBSource)
	if err != nil {
		logger.Fatal().Err(err).Msg("db migration problem")
	}
	err = migration.Up()
	if err != nil && err != migrate.ErrNoChange {
		logger.Fatal().Err(err).Msg("db migration problem")
	}

	store := db.NewStore(connPool)

	service.ScheduleJobs(ctx, store, config, logger)

	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		logger.Fatal().Err(err).Msg("cannot create token maker")
	}

	g, ctx := errgroup.WithContext(ctx)
	runGinServer(ctx, g, config, store, tokenMaker, logger)

	err = g.Wait()
	if err != nil {
		log.Fatal().Err(err).Msg("error from wait group")
	}
}

func runGinServer(ctx context.Context, g *errgroup.Group, config util.Config, store db.Store, tokenMaker token.Maker, logger *zerolog.Logger) {
	server, err := server.NewServer(config, store, tokenMaker, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("cannot create server")
	}

	server.Start(ctx, g)
}
