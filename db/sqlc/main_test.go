package db

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"github.com/rs/zerolog"
)

const (
	dbName   = "test"
	dbPasswd = "secret"
)

var (
	testConfig util.Config
	testStore  Store
	testLogger *zerolog.Logger
)

func TestMain(m *testing.M) {
	testConfig = util.Config{
		MigrationPath:        "../../db/migration",
		EnableConsoleLogging: true,
		EnableFileLogging:    false,
	}

	testLogger = util.NewLogger(testConfig)

	connPool, dbCleanUp, err := newDBTest(&testConfig)
	if err != nil {
		testLogger.Fatal().Err(err).Msg("cannot connect to db")
	}

	testStore = NewStore(connPool)

	code := m.Run()
	dbCleanUp()
	os.Exit(code)
}

func newDBTest(config *util.Config) (connPool *pgxpool.Pool, fnCleanUp func(), err error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		testLogger.Fatal().Err(err).Msg("could not construct pool")
	}

	err = pool.Client.Ping()
	if err != nil {
		testLogger.Fatal().Err(err).Msg("could not connect to docker")
	}

	// pulls an image, creates a container based on it and runs it
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgis/postgis",
		Tag:        "12-3.4",
		Env: []string{
			"POSTGRES_PASSWORD=" + dbPasswd,
			"POSTGRES_USER=postgres",
			"POSTGRES_DB=" + dbName,
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.NeverRestart()
	})
	if err != nil {
		testLogger.Fatal().Err(err).Msg("could not start resource")
	}

	dbPort := resource.GetPort("5432/tcp")
	config.DBSource = fmt.Sprintf("postgresql://postgres:%s@localhost:%s/%s?sslmode=disable", dbPasswd, dbPort, dbName)

	testLogger.Info().Msgf("connecting to database on url: " + config.DBSource)

	resource.Expire(120) // Tell docker to hard kill the container in 120 seconds

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	pool.MaxWait = 120 * time.Second
	if err = pool.Retry(func() error {
		ctx := context.Background()
		connPool, err = pgxpool.New(ctx, config.DBSource)
		if err != nil {
			return err
		}
		return connPool.Ping(ctx)
	}); err != nil {
		testLogger.Fatal().Err(err).Msg("could not connect to docker")
	}

	fnCleanUp = func() {
		if err := pool.Purge(resource); err != nil {
			testLogger.Fatal().Err(err).Msg("could not purge resource")
		}
	}

	return
}
