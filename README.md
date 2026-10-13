# Panahon-API

Backend for panahon website. Go + Gin + pgx/v5 + sqlc.

## Development

The project uses [devenv](https://devenv.sh) (Go, air, sqlc, mockery, swag, migrate, postgresql).

```sh
devenv up
```

`devenv up` starts the PostGIS `postgres` service (pinned PostgreSQL 16, database
`weather`), seeds dev data, then runs the API with `air` (live reload) on
`0.0.0.0:3000`.

Configuration is read from `app.env` (viper), plus `app_conf.yaml` for cron jobs.

## Seeding dev data

Dev data lives in `scripts/seed/` (stations, weather observations). All commands
are idempotent and target the configured dev database (`weather`):

```sh
go run ./cmd/testcli seed all        # users + stations + records
go run ./cmd/testcli seed users      # user1 (USER), admin (ADMIN), admin2 (SUPERADMIN); password: password
go run ./cmd/testcli seed stations   # observations_station from scripts/seed/stations.dump
go run ./cmd/testcli seed records    # observations_* from scripts/seed/*.csv(.gz)
```

Flags: `--seed-dir` (default `scripts/seed`), `--force` (reload records even if
present), `--pg-restore` (path to `pg_restore`; resolved on `PATH` by default).
The old random generator is `go run ./cmd/testcli seed random-stations`.

## Tests

```sh
make short_test   # unit tests only (no Docker)
make test         # everything; needs Docker/Podman (testcontainers + PostGIS)
```

## Code generation

```sh
make sqlc   # internal/db/sqlc   (from internal/db/query, internal/db/migration)
make mock   # internal/mocks
make swag   # internal/docs/api
make migrateup / migratedown / migratenew name=<name>
```

Generated directories are never hand-edited.
