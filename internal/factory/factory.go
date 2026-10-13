// Package factory seeds the dev database with users, stations and observation
// records. All operations are idempotent and safe to re-run.
package factory

import (
	"context"
	"path/filepath"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// SeedOptions configures a full seed run.
type SeedOptions struct {
	SeedDir   string
	PGRestore string
	Force     bool
	Users     []UserSpec
}

// SeedAll seeds users/roles, stations and observation records in order.
func SeedAll(ctx context.Context, pool *pgxpool.Pool, store db.Store, logger *zerolog.Logger, opts SeedOptions) error {
	if len(opts.Users) == 0 {
		opts.Users = DefaultDevUsers()
	}

	logger.Info().Msg("seeding users")
	if err := SeedUsers(ctx, store, logger, opts.Users); err != nil {
		return err
	}

	logger.Info().Msg("seeding stations")
	if err := SeedStations(ctx, pool, logger, StationSeedOptions{
		DumpPath:  filepath.Join(opts.SeedDir, "stations.dump"),
		PGRestore: opts.PGRestore,
	}); err != nil {
		return err
	}

	logger.Info().Msg("seeding observation records")
	if err := SeedObservations(ctx, pool, logger, ObservationSeedOptions{
		Dir:   opts.SeedDir,
		Force: opts.Force,
	}); err != nil {
		return err
	}

	logger.Info().Msg("seeding complete")
	return nil
}
