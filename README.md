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

## Deployment

### Build

```sh
./scripts/deploy-container.sh                       # local: panahon-api:latest + :<version>-<git-rev>
./scripts/deploy-container.sh -U yohlime -p         # build and push to ghcr.io/yohlime/panahon-api
```

Bump the version in `VERSION` with `./scripts/bump-version.sh patch|minor|major`
(add `--commit` to commit unsigned, `--tag` to also create `vX.Y.Z`). The
version is injected into the binary and logged at startup.

### Runtime configuration

The image does **not** contain `app.env` or `app_conf.yaml`; mount them read-only
into `/app` (see `docker-compose.yaml`). Create them from the `.example`
templates first. Required production values:

| Key | Notes |
|-----|-------|
| `ENVIRONMENT` | `production` (enables `Secure` cookies) |
| `TOKEN_SYMMETRIC_KEY` | exactly 32 random characters |
| `DB_SOURCE` | `postgresql://…` connection string |
| `ACCESS_TOKEN_DURATION` / `REFRESH_TOKEN_DURATION` | e.g. `5m` / `24h` |
| `CORS_ALLOW_ORIGINS` | comma-separated origins (empty = same-origin only) |
| `COOKIE_DOMAIN` | cookie domain (host-only when empty) |

Migrations run automatically on start; the target database must already exist
with PostGIS. For file logging, `LOG_DIRECTORY` must be writable in the
container, or set `ENABLE_FILE_LOGGING=0` (console only).

### Frontend

```sh
cd ../panahon-data-web
./scripts/deploy-container.sh -U yohlime -p
```

`PUBLIC_PANAHON_API` in `.env.production` must point at this API
(same-origin `/data/api/v1` today); `BACKEND_URL` (in `.env.development`) is
only used by the dev proxy.
