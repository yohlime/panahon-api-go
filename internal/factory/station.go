package factory

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// StationSeedOptions configures loading stations from a pg_dump archive.
type StationSeedOptions struct {
	DumpPath  string
	PGRestore string
}

type copyBlock struct {
	Columns string
	Data    string
}

// parseCopyBlock extracts the column list and tab-separated data from the
// `COPY ... FROM stdin;` block emitted by pg_restore.
func parseCopyBlock(out string) (copyBlock, error) {
	scanner := bufio.NewScanner(strings.NewReader(out))
	scanner.Buffer(make([]byte, 0, 1024*1024), 8*1024*1024)

	var (
		columns string
		found   bool
		data    strings.Builder
	)

	for scanner.Scan() {
		line := scanner.Text()

		if !found {
			if strings.HasPrefix(line, "COPY ") && strings.HasSuffix(strings.TrimSpace(line), "FROM stdin;") {
				open := strings.Index(line, "(")
				close := strings.LastIndex(line, ")")
				if open < 0 || close < 0 || close < open {
					return copyBlock{}, fmt.Errorf("malformed COPY statement: %s", line)
				}
				columns = line[open+1 : close]
				found = true
			}
			continue
		}

		if line == "\\." {
			return copyBlock{Columns: columns, Data: data.String()}, nil
		}

		data.WriteString(line)
		data.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		return copyBlock{}, err
	}

	return copyBlock{}, fmt.Errorf("no COPY data found in pg_restore output")
}

// readStationDump shells out to pg_restore to get the station COPY block.
func readStationDump(dumpPath, pgRestore string) (copyBlock, error) {
	if pgRestore == "" {
		resolved, err := exec.LookPath("pg_restore")
		if err != nil {
			return copyBlock{}, fmt.Errorf("pg_restore not found on PATH: %w", err)
		}
		pgRestore = resolved
	}

	cmd := exec.Command(pgRestore, "-a", "--no-owner", "-n", "public", "-t", "observations_station", "-f", "-", dumpPath)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return copyBlock{}, fmt.Errorf("pg_restore: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	return parseCopyBlock(stdout.String())
}

// SeedStations loads stations from a custom-format pg_dump archive. Idempotent
// via ON CONFLICT (id) DO NOTHING, and advances the id sequence afterwards.
func SeedStations(ctx context.Context, pool *pgxpool.Pool, logger *zerolog.Logger, opts StationSeedOptions) error {
	block, err := readStationDump(opts.DumpPath, opts.PGRestore)
	if err != nil {
		return err
	}

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

	if _, err := tx.Exec(ctx, "CREATE TEMP TABLE stg_stations AS SELECT * FROM observations_station WITH NO DATA"); err != nil {
		return fmt.Errorf("create station staging table: %w", err)
	}

	copyStmt := fmt.Sprintf("COPY stg_stations (%s) FROM STDIN", block.Columns)
	if _, err := tx.Conn().PgConn().CopyFrom(ctx, strings.NewReader(block.Data), copyStmt); err != nil {
		return fmt.Errorf("copy stations: %w", err)
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO observations_station (
			id, name, lat, lon, elevation, date_installed, mo_station_id, sms_system_type,
			mobile_number, station_type, station_type2, station_url, status, logger_version,
			priority_level, provider_id, province, region, address, created_at, updated_at, deleted_at, geom
		)
		SELECT
			id, name, lat, lon, elevation, date_installed, mo_station_id, sms_system_type,
			mobile_number, station_type, station_type2, station_url, status, logger_version,
			priority_level, provider_id, province, region, address,
			COALESCE(created_at, CURRENT_TIMESTAMP),
			COALESCE(updated_at, '0001-01-01 00:00:00Z'::timestamptz),
			COALESCE(deleted_at, '0001-01-01 00:00:00Z'::timestamptz),
			geom
		FROM stg_stations
		ON CONFLICT DO NOTHING`)
	if err != nil {
		return fmt.Errorf("insert stations: %w", err)
	}

	if _, err := tx.Exec(ctx, "SELECT setval(pg_get_serial_sequence('observations_station', 'id'), COALESCE((SELECT MAX(id) FROM observations_station), 1))"); err != nil {
		return fmt.Errorf("reset station id sequence: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	logger.Info().Int64("inserted", tag.RowsAffected()).Msg("seeded stations")
	return nil
}
