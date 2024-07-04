package cmd

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/emiliogozo/panahon-api-go/internal/server"
	"github.com/emiliogozo/panahon-api-go/internal/util"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var (
	testDBName string
	resetDB    bool
	config     util.Config
	logger     *zerolog.Logger
)

var rootCmd = &cobra.Command{
	Use:   "cli",
	Short: "A test cli for the server",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		verbosity, _ := cmd.Flags().GetCount("verbose")
		switch verbosity {
		case 1:
			config.LogLevel = "info"
		case 2:
			config.LogLevel = "debug"
		case 3:
			config.LogLevel = "trace"
		default:
			config.LogLevel = "warn"
		}
		logger = util.NewLogger(config)
	},
}

func init() {
	cobra.OnInitialize(initCmd)
	rootCmd.AddCommand(seedCmd, lufftCmd)
	rootCmd.PersistentFlags().CountP("verbose", "v", "increase verbosity level (up to -vvv)")
	rootCmd.PersistentFlags().StringVar(&testDBName, "db", "testweather", "db name")
	rootCmd.PersistentFlags().BoolVarP(&resetDB, "reset", "r", false, "reset db")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initCmd() {
	var err error
	config, err = util.LoadConfig(".")
	if err != nil {
		log.Fatal().Err(err).Msg("cannot load config")
	}
}

func dbConnect(ctx context.Context) (*pgxpool.Pool, db.Store) {
	config.DBSource, _ = replaceDBName(config.DBSource, testDBName)
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
		logger.Fatal().Err(err).Msg("error applying db migrations")
	}

	if resetDB {
		err = migration.Down()
		if err != nil {
			logger.Fatal().Err(err).Msg("error resetting db")
		}
		resetDB = false

		err = migration.Up()
		if err != nil && err != migrate.ErrNoChange {
			logger.Fatal().Err(err).Msg("error applying db migrations")
		}
	}

	store := db.NewStore(connPool)

	return connPool, store
}

func runGinServer(ctx context.Context, g *errgroup.Group, store db.Store) {
	server, err := server.NewServer(config, store, nil, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("cannot create server")
	}

	server.Start(ctx, g)
}

func isServerRunning(ready chan<- bool) {
	defer close(ready)
	logger.Info().Msg("waiting for server to be online")

	maxRetries := 5
	retryDelay := time.Second

	start := time.Now()
	for attempt := 0; attempt < maxRetries; attempt++ {
		conn, err := net.DialTimeout("tcp", config.HTTPServerAddress, 5*time.Second)
		if err == nil {
			conn.Close()
			ready <- true
			logger.Info().Dur("duration", time.Since(start)).Msg("server is online")
			return
		}

		logger.Info().Err(err).Msgf("Attempt %d failed", attempt)

		time.Sleep(retryDelay)
		retryDelay *= 2
	}

	ready <- false
}

func replaceDBName(connStr, newDBName string) (string, error) {
	u, err := url.Parse(connStr)
	if err != nil {
		return "", err
	}

	pathParts := strings.Split(u.Path, "/")
	if len(pathParts) > 1 {
		pathParts[1] = newDBName
	}
	u.Path = strings.Join(pathParts, "/")

	return u.String(), nil
}
