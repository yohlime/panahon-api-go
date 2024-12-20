package db

import (
	"os"
	"testing"

	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/rs/zerolog/log"
)

var (
	testStore     Store
	testMigration *migrate.Migrate
)

func TestMain(m *testing.M) {
	config, err := util.LoadConfig("../../..")
	if err != nil {
		log.Fatal().Err(err).Msg("cannot load config")
	}

	db := util.NewDockerPostgres(config)

	migrationPath := "../../db/migration"
	testMigration, err = migrate.New("file://"+migrationPath, db.Source)
	if err != nil {
		log.Fatal().Err(err).Msg("could not construct migration")
	}

	testStore = NewStore(db.Conn)

	code := m.Run()
	db.Close()
	os.Exit(code)
}
