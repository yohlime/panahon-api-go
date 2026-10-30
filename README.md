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

## User registration

- Public self-registration (`POST /users/register`) creates a `pending` user and
  waits for admin approval. It still requires email.
- Admins create users without a password (`POST /users`). The response includes a
  one-time `registration_url`; copy it and send it to the user.
- The link opens `/complete-registration` on the frontend (production:
  `https://panahon.observatory.ph/sensors/complete-registration`). There the
  invited user confirms their username, optionally sets/corrects their full name
  and email, and creates a password.
- Links are hashed in the database, single-use, and expire after
  `REGISTRATION_TOKEN_DURATION` (default `72h`). Regenerating a link invalidates
  the previous one (`POST /users/{id}/registration-link`).
- On completion the user becomes `active` and is logged in automatically.
- Email is optional for admin-created users and can be changed immediately as
  long as the new address is unique. It is format-checked only, not
  inbox-verified, so password recovery uses admin-issued reset links instead.

## Password reset

- Admins generate a single-use reset link for an active user with
  `POST /users/{id}/password-reset-link`; copy the returned `reset_url` and send
  it to the user. No email is sent.
- The link opens `/reset-password` on the frontend (production:
  `https://panahon.observatory.ph/sensors/reset-password`). Links are hashed,
  single-use, and expire after `PASSWORD_RESET_TOKEN_DURATION` (default `1h`).
  Regenerating a link invalidates the previous one.
- Accepting the link sets the new password, revokes every existing session, and
  logs the user in. Password reset and account creation require at least 8
  characters; login still accepts older 6-character passwords.

## Health and readiness

- `GET /healthz` is a liveness probe: `200` while the HTTP process is running.
- `GET /readyz` is a readiness probe: it pings the database with a 2-second
  timeout and returns `200`, or `503` when the database is unreachable.
- Both live at the server root, outside `API_BASE_PATH`, require no auth, and
  send `Cache-Control: no-store`.

## Backup and restore

- `./scripts/backup-db.sh --output /var/backups/panahon --keep 14` writes a
  compressed `pg_dump -Fc` archive with a SHA-256 sidecar.
- `./scripts/restore-db.sh --source FILE --target DSN` restores into a new
  database, verifying the checksum and refusing a non-empty target unless
  `--clean` is given.
- Full procedure, retention, off-host copies, verification queries, rollback,
  and disaster recovery: [docs/backup-restore.md](docs/backup-restore.md).
- `backups/` is gitignored. Dumps contain password hashes and hashed tokens, so
  store them encrypted.

## API docs

- Full Swagger UI: `/api/v1/docs` (redirects to `index.html`).
- Public Swagger UI: `/api/v1/public-docs`, backed by
  `/api/v1/public-openapi.json`. It is filtered to read-only station and
  observation endpoints and advertises `api_key` authentication only.
- The full docs stay public for now. Neither docs route requires a session.
- Set `SWAG_API_BASE_PATH` to the externally visible API prefix (`/data/api/v1`
  in production). It is used both for Swagger's **Try it out** and for the
  `/docs` and `/public-docs` index redirects, so proxy prefixes are preserved.

## Pagination

- Station and observation lists default to 100 rows per page.
- Station lists are capped at 1,000 rows; observation lists at 5,000 rows.
- Other admin lists keep their existing defaults (users/roles/Lufft 5, access requests 10).
- `GET /stations/{id}/observations` and `GET /observations` support cursor mode:
  ```text
  ?limit=1000&cursor=<next_cursor>
  ```
  Cursor responses are `{ "items": [...], "next_cursor": "...", "has_more": true }`
  and skip the `COUNT(*)` query. Page and cursor modes use separate response shapes.
- Cursor pagination is ordered by `timestamp DESC, id DESC`; keep the same
  filters between pages.

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
| `FRONTEND_BASE_URL` | public frontend base, including Astro base (e.g. `https://panahon.observatory.ph/sensors`) |
| `REGISTRATION_TOKEN_DURATION` | admin registration-link lifetime (default `72h`) |
| `PASSWORD_RESET_TOKEN_DURATION` | admin password-reset-link lifetime (default `1h`) |
| `TRUSTED_PROXIES` | comma-separated proxy IPs/CIDRs allowed to supply `X-Forwarded-For`; empty ignores forwarded headers |
| `RATE_LIMIT_ENABLED` | in-memory per-IP rate limits (default `1`); `0` disables all |
| `RATE_LIMIT_LOGIN_PER_MINUTE` | login attempts per IP per minute (default `10`) |
| `RATE_LIMIT_REGISTRATION_PER_HOUR` | registration/registration-token requests per IP per hour (default `10`) |
| `RATE_LIMIT_RESET_PER_MINUTE` | password-reset requests per IP per minute (default `10`) |
| `RATE_LIMIT_API_KEY_PER_MINUTE` | API-key requests per IP per minute (default `120`) |

Rate limits are in-memory and per process, so they reset on restart and are
enforced separately by each replica. Set `TRUSTED_PROXIES` to the reverse
proxy's IP/CIDR in production; with an empty value forwarded headers are
ignored and every client behind the proxy shares one bucket.

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
