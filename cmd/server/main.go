package main

import (
	"context"

	db "github.com/emiliogozo/panahon-api-go/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/api"
	docs "github.com/emiliogozo/panahon-api-go/internal/docs"
	"github.com/emiliogozo/panahon-api-go/internal/service"
	"github.com/emiliogozo/panahon-api-go/internal/util"
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

	service.ScheduleJobs(ctx, store, config, logger)

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
