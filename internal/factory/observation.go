package factory

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// ObservationDataset maps a CSV export to a table.
type ObservationDataset struct {
	Table    string
	File     string
	Columns  string
	Select   string
	Conflict string
}

// ObservationDatasets are loaded from scripts/seed by default.
var ObservationDatasets = []ObservationDataset{
	{
		Table:    "observations_observation",
		File:     "observations.csv",
		Columns:  "created_at, pres, rr, rh, temp, td, wdir, wspd, wspdx, srad, mslp, hi, station_id, timestamp, wchill, qc_level, rain_tips, rain_cumulative_tips",
		Select:   "COALESCE(created_at, CURRENT_TIMESTAMP), pres, rr, rh, temp, td, wdir, wspd, wspdx, srad, mslp, hi, station_id, timestamp, wchill, COALESCE(qc_level, 0), rain_tips, rain_cumulative_tips",
		Conflict: "station_id, timestamp",
	},
	{
		Table:    "observations_mo_observation",
		File:     "mo_observations.csv.gz",
		Columns:  "created_at, pres, rr, rh, temp, td, wdir, wspd, wspdx, srad, hi, station_id, timestamp, wchill, rain, tx, tn, wrun, thwi, thswi, senergy, sradx, uvi, uvdose, uvx, hdd, cdd, et, qc_level, wdirx",
		Select:   "COALESCE(created_at, CURRENT_TIMESTAMP), pres, rr, rh, temp, td, wdir, wspd, wspdx, srad, hi, station_id, timestamp, wchill, rain, tx, tn, wrun, thwi, thswi, senergy, sradx, uvi, uvdose, uvx, hdd, cdd, et, COALESCE(qc_level, 0), wdirx",
		Conflict: "station_id, timestamp",
	},
}

// ObservationSeedOptions configures loading record CSVs.
type ObservationSeedOptions struct {
	Dir   string
	Force bool
}

// SeedObservations loads each dataset from opts.Dir. Datasets whose target table
// already holds rows are skipped unless Force is set.
func SeedObservations(ctx context.Context, pool *pgxpool.Pool, logger *zerolog.Logger, opts ObservationSeedOptions) error {
	for _, ds := range ObservationDatasets {
		path := filepath.Join(opts.Dir, ds.File)
		if err := seedObservationDataset(ctx, pool, logger, ds, path, opts.Force); err != nil {
			return err
		}
	}
	return nil
}

func seedObservationDataset(ctx context.Context, pool *pgxpool.Pool, logger *zerolog.Logger, ds ObservationDataset, path string, force bool) error {
	if !force {
		var exists bool
		if err := pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM "+ds.Table+" LIMIT 1)").Scan(&exists); err != nil {
			return fmt.Errorf("check %s: %w", ds.Table, err)
		}
		if exists {
			logger.Info().Str("table", ds.Table).Msg("records already present, skipping")
			return nil
		}
	}

	file, err := openSeedFile(path)
	if err != nil {
		return err
	}
	defer file.Close()

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	stg := "stg_" + ds.Table
	if _, err := tx.Exec(ctx, fmt.Sprintf("CREATE TEMP TABLE %s AS SELECT %s FROM %s WITH NO DATA", stg, ds.Columns, ds.Table)); err != nil {
		return fmt.Errorf("create %s staging table: %w", ds.Table, err)
	}

	copyStmt := fmt.Sprintf("COPY %s (%s) FROM STDIN WITH (FORMAT csv, HEADER)", stg, ds.Columns)
	if _, err := tx.Conn().PgConn().CopyFrom(ctx, file, copyStmt); err != nil {
		return fmt.Errorf("copy %s: %w", ds.Table, err)
	}

	insert := fmt.Sprintf("INSERT INTO %s (%s) SELECT %s FROM %s ON CONFLICT (%s) DO NOTHING", ds.Table, ds.Columns, ds.Select, stg, ds.Conflict)
	tag, err := tx.Exec(ctx, insert)
	if err != nil {
		return fmt.Errorf("insert %s: %w", ds.Table, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	logger.Info().Str("table", ds.Table).Int64("inserted", tag.RowsAffected()).Msg("seeded records")
	return nil
}

func openSeedFile(path string) (io.ReadCloser, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	if !strings.HasSuffix(path, ".gz") {
		return file, nil
	}

	reader, err := gzip.NewReader(file)
	if err != nil {
		file.Close()
		return nil, err
	}

	return &gzipFile{reader: reader, file: file}, nil
}

type gzipFile struct {
	reader *gzip.Reader
	file   *os.File
}

func (g *gzipFile) Read(p []byte) (int, error) { return g.reader.Read(p) }

func (g *gzipFile) Close() error {
	err := g.reader.Close()
	if cerr := g.file.Close(); err == nil {
		err = cerr
	}
	return err
}
