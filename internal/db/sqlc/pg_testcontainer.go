package db

import (
	"context"
	"fmt"

	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rs/zerolog/log"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

const (
	dbName   = "test"
	dbUser   = "pguser"
	dbPasswd = "pGpaSsw0rd"
)

type PostgresContainer struct {
	Source    string
	Conn      *pgxpool.Pool
	container *postgres.PostgresContainer
	config    util.Config
	ctx       context.Context
}

func NewDockerPostgres(config util.Config) *PostgresContainer {
	pg := &PostgresContainer{config: config}
	pg.createDBConn()
	return pg
}

func (pg *PostgresContainer) createDBConn() {
	pg.ctx = context.Background()

	var err error
	pg.container, err = postgres.Run(pg.ctx,
		fmt.Sprintf("%s:%s", pg.config.DockerTestPGRepo, pg.config.DockerTestPGTag),
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPasswd),
		postgres.BasicWaitStrategies(),
		postgres.WithSQLDriver("pgx"),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("could not start postgres")
		return
	}

	pg.Source, err = pg.container.ConnectionString(pg.ctx, "sslmode=disable")
	if err != nil {
		log.Fatal().Err(err).Msg("could not retrieve postgres connection string")
		return
	}

	pg.Conn, err = pgxpool.New(pg.ctx, pg.Source)
	if err != nil {
		log.Fatal().Err(err).Msg("could not retrieve postgres connection string")
		return
	}
}

func (pg *PostgresContainer) Close() {
	if err := tc.TerminateContainer(pg.container); err != nil {
		log.Fatal().Err(err).Msg("failed to terminate container")
	}
	pg.Conn.Close()
}
