package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/emiliogozo/panahon-api-go/util"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
)

const (
	dbName   = "test"
	dbPasswd = "secret"
)

var testStore Store

func TestMain(m *testing.M) {
	config := util.Config{
		MigrationPath: "../../db/migration",
	}

	connPool, dbCleanUp, err := newDBTest(&config)
	if err != nil {
		log.Fatalf("cannot connect to db: %s", err)
	}

	util.RunDBMigration(config.MigrationPath, config.DBSource)

	testStore = NewStore(connPool)

	code := m.Run()
	dbCleanUp()
	os.Exit(code)
}

func newDBTest(config *util.Config) (connPool *pgxpool.Pool, fnCleanUp func(), err error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not construct pool: %s", err)
	}

	err = pool.Client.Ping()
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	// pulls an image, creates a container based on it and runs it
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "12",
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
		log.Fatalf("could not start resource: %s", err)
	}

	dbPort := resource.GetPort("5432/tcp")
	config.DBSource = fmt.Sprintf("postgresql://postgres:%s@localhost:%s/%s?sslmode=disable", dbPasswd, dbPort, dbName)

	log.Println("Connecting to database on url: " + config.DBSource)

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
		log.Fatalf("could not connect to docker: %s", err)
	}

	fnCleanUp = func() {
		if err := pool.Purge(resource); err != nil {
			log.Fatalf("could not purge resource: %s", err)
		}
	}

	return
}
