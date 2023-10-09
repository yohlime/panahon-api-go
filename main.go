package main

import (
	"context"
	"time"

	"github.com/emiliogozo/panahon-api-go/api"
	db "github.com/emiliogozo/panahon-api-go/db/sqlc"
	docs "github.com/emiliogozo/panahon-api-go/docs"
	"github.com/emiliogozo/panahon-api-go/util"
	"github.com/go-co-op/gocron"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

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

	ctx := context.Background()

	connPool, err := pgxpool.New(ctx, config.DBSource)
	if err != nil {
		logger.Fatal().Err(err).Msg("cannot connect to db")
	}

	err = util.RunDBMigration(config.MigrationPath, config.DBSource)
	if err != nil {
		logger.Fatal().Err(err).Msg("db migration problem")
	}

	store := db.NewStore(connPool)

	s := gocron.NewScheduler(time.Local)
	_, err = s.Cron("2-59/10 * * * *").Tag("refreshMaterialiazedView").Do(refreshMaterialiazedView, ctx, store)
	if err != nil {
		log.Fatal().Err(err).Msg("error scheduling job")
	}
	s.StartAsync()

	runGinServer(config, store, logger)
}

func runGinServer(config util.Config, store db.Store, logger *zerolog.Logger) {
	server, err := api.NewServer(config, store, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("cannot create server")
	}

	err = server.Start(config.HTTPServerAddress)
	if err != nil {
		logger.Fatal().Err(err).Msg("cannot start server")
	}
}

func refreshMaterialiazedView(ctx context.Context, store db.Store) error {
	err := store.RefreshMVCurrentObservations(ctx)
	if err != nil {
		return err
	}
	return nil
}
