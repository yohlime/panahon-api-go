package util

import (
	"context"
	"fmt"
	"time"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"github.com/rs/zerolog/log"
)

const (
	dbName   = "test"
	dbPasswd = "secret"
)

type DockerPostgres struct {
	Source    string
	Conn      *pgxpool.Pool
	container *dockertest.Resource
	config    Config
}

func NewDockerPostgres(config Config) *DockerPostgres {
	pg := &DockerPostgres{config: config}
	pg.createDBConn()
	return pg
}

func (pg *DockerPostgres) createDBConn() {
	dockerPool := initDocker()

	container, err := createPGInstance(pg.config, dockerPool)
	if err != nil {
		log.Fatal().Err(err).Msg("could not create postgres container")
		return
	}
	pg.container = container

	dbHostAndPort := container.GetHostPort("5432/tcp")
	pg.Source = fmt.Sprintf("postgres://postgres:%s@%s/%s?sslmode=disable", dbPasswd, dbHostAndPort, dbName)

	if err := dockerPool.Retry(func() error {
		ctx := context.Background()
		var err error
		pg.Conn, err = pgxpool.New(ctx, pg.Source)
		if err != nil {
			return err
		}

		return pg.Conn.Ping(ctx)
	}); err != nil {
		log.Fatal().Err(err).Msg("could not initialize postgres container")
		return
	}
}

func (pg *DockerPostgres) Close() {
	if err := pg.container.Close(); err != nil {
		log.Fatal().Err(err).Msg("could not purge postgres")
	}
}

func initDocker() *dockertest.Pool {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatal().Err(err).Msg("could not construct docker pool")
	}

	err = pool.Client.Ping()
	if err != nil {
		log.Fatal().Err(err).Msg("could not connect to Docker")
	}

	pool.MaxWait = 120 * time.Second
	return pool
}

func createPGInstance(config Config, dockerPool *dockertest.Pool) (*dockertest.Resource, error) {
	container, err := dockerPool.RunWithOptions(&dockertest.RunOptions{
		Repository: config.DockerTestPGRepo,
		Tag:        config.DockerTestPGTag,
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
		log.Fatal().Err(err).Msg("could not start postgres")
		return nil, nil
	}

	container.Expire(120)
	return container, nil
}
