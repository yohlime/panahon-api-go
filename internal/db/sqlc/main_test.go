package db

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
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
	testConfig    util.Config
	testStore     Store
	testLogger    *zerolog.Logger
	testMigration *migrate.Migrate
)

func TestMain(m *testing.M) {
	testConfig = util.Config{
		MigrationPath:        "../../db/migration",
		EnableConsoleLogging: true,
		EnableFileLogging:    false,
	}

	testLogger = util.NewLogger(testConfig)

	dbPool, dbClose := createDBConn()

	var err error
	testMigration, err = migrate.New("file://"+testConfig.MigrationPath, testConfig.DBSource)
	if err != nil {
		testLogger.Fatal().Err(err).Msg("could not construct migration")
	}

	testStore = NewStore(dbPool)

	code := m.Run()
	dbClose()
	os.Exit(code)
}

func initDocker() *dockertest.Pool {
	pool, err := dockertest.NewPool("")
	if err != nil {
		testLogger.Fatal().Err(err).Msg("could not construct docker pool")
	}

	err = pool.Client.Ping()
	if err != nil {
		testLogger.Fatal().Err(err).Msg("could not connect to Docker")
	}

	pool.MaxWait = 120 * time.Second
	return pool
}

func createPGInstance(dockerPool *dockertest.Pool) (*dockertest.Resource, error) {
	container, err := dockerPool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgis/postgis",
		Tag:        "12-3.4",
		Env: []string{
			"POSTGRES_PASSWORD=" + dbPasswd,
			"POSTGRES_USER=postgres",
			"POSTGRES_DB=" + dbName,
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		testLogger.Fatal().Err(err).Msg("could not start postgres")
		return nil, nil
	}

	container.Expire(120)
	return container, nil
}

func createDBConn() (*pgxpool.Pool, func()) {
	dockerPool := initDocker()

	container, err := createPGInstance(dockerPool)
	if err != nil {
		return nil, nil
	}

	dbHostAndPort := container.GetHostPort("5432/tcp")
	testConfig.DBSource = fmt.Sprintf("postgres://postgres:%s@%s/%s?sslmode=disable", dbPasswd, dbHostAndPort, dbName)

	var dbPool *pgxpool.Pool

	if err := dockerPool.Retry(func() error {
		ctx := context.Background()
		var err error
		dbPool, err = pgxpool.New(ctx, testConfig.DBSource)
		if err != nil {
			return err
		}

		return dbPool.Ping(ctx)
	}); err != nil {
		testLogger.Fatal().Err(err).Msg("postgres container not intialized")
		return nil, nil
	}

	close := func() {
		if err := dockerPool.Purge(container); err != nil {
			testLogger.Fatal().Err(err).Msg("could not purge postgres")
		}
	}

	return dbPool, close
}
