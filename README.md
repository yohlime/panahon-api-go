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
Both are gitignored, so copy the examples first:

```sh
cp app.env.example app.env             # fill in DB creds + TOKEN_SYMMETRIC_KEY
cp app_conf.yaml.example app_conf.yaml
```

Keep secrets out of the repo: `app.env` holds DB passwords, `TOKEN_SYMMETRIC_KEY`,
and the Glabs keys.

## Authorization

- The web app authenticates with an httpOnly cookie **session** (`/users/login`);
  external clients use a query `api_key` API token.
- Reads are scoped by the user's `user_access` **entitlement** (stations,
  variables, max days of history); admins are unrestricted.
- **API tokens default to, and are capped at, their owner's entitlement**:
  omitted `permissions` copy the entitlement, and provided permissions are
  intersected with it before being stored (so a key can never exceed its owner).
  The same intersection is applied at read time, so narrowing an entitlement
  narrows existing tokens — but widening it does not widen keys created earlier.

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
