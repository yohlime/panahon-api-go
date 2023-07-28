package main

import (
	"context"
	"os"

	"github.com/emiliogozo/panahon-api-go/api"
	db "github.com/emiliogozo/panahon-api-go/db/sqlc"
	docs "github.com/emiliogozo/panahon-api-go/docs"
	"github.com/emiliogozo/panahon-api-go/util"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

//	@title			Panahon API
//	@version		1.0
//	@description	Panahon API.

//	@contact.name	Emilio Gozo
//	@contact.email	emiliogozo@proton.me

//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
func main() {
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal().Err(err).Msg("cannot load config")
	}

	if config.Environment == "development" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	docs.SwaggerInfo.BasePath = config.SwagAPIBasePath

	connPool, err := pgxpool.New(context.Background(), config.DBSource)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot connect to db")
	}

	util.RunDBMigration(config.MigrationPath, config.DBSource)

	store := db.NewStore(connPool)

	runGinServer(config, store)
}

func runGinServer(config util.Config, store db.Store) {
	server, err := api.NewServer(config, store)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create server")
	}

	err = server.Start(config.HTTPServerAddress)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot start server")
	}
}
