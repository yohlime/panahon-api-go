# Backup and restore runbook

This runbook covers the PostgreSQL/PostGIS database used by Panahon API:
users, stations, observations, sessions, API tokens, and access grants. It does
not cover `app.env`, `app_conf.yaml`, container images, or any frontend data.

Backups contain password hashes, hashed session/API tokens, and user contact
details. Treat every dump as a secret.

## Prerequisites

- PostgreSQL client tools compatible with the server major version:
  `pg_dump`, `pg_restore`, and `psql`.
- A `DB_SOURCE` connection string for the source database, normally in
  `app.env`.
- Enough free disk space for a full dump plus the retention window.
- PostGIS installed on the target PostgreSQL server.
- Encrypted storage for dumps and a tested off-host copy.

## Back up

### Script

```sh
./scripts/backup-db.sh --output /var/backups/panahon --keep 14
```

The script:

- loads `DB_SOURCE` from `app.env` (override with `--env FILE`),
- writes a `pg_dump -Fc` custom-format archive as `panahon-<UTC>.dump`,
- uses `--no-owner --no-privileges` so the archive can be restored by a
  different database owner,
- verifies the archive with `pg_restore --list`,
- writes a SHA-256 sidecar by default (`--no-checksum` disables it),
- sets files to mode `0600` and removes backups older than `--keep`.

### Manual equivalent

```sh
set -a
. ./app.env
set +a

stamp="$(date -u +%Y%m%dT%H%M%SZ)"
pg_dump -Fc --no-owner --no-privileges \
  -f "panahon-$stamp.dump" "$DB_SOURCE"
pg_restore -l "panahon-$stamp.dump" >/dev/null
sha256sum "panahon-$stamp.dump" >"panahon-$stamp.dump.sha256"
chmod 600 "panahon-$stamp.dump" "panahon-$stamp.dump.sha256"
```

### Schedule and off-host copies

Example daily cron entry:

```cron
15 2 * * * cd /opt/panahon-api && ./scripts/backup-db.sh --output /var/backups/panahon --keep 14 >> /var/log/panahon-backup.log 2>&1
```

Alert on non-zero exit. Copy dumps off-host with an encrypted tool such as
`restic`, `borg`, or an object-storage client. A backup that only exists on the
database host is not a disaster-recovery backup.

Suggested baseline: daily full backups, 14 local copies, 30-90 days off-host,
and a restore drill at least quarterly. Adjust to the site's RPO/RTO.

## Restore to a new database

Always restore into a **new** database first. Never overwrite production while
testing a backup.

1. Take a fresh backup of the current database and record the current
   `DB_SOURCE`.

2. Create an empty target database and install PostGIS as a database
   administrator:

   ```sh
   createdb -h db.example -U postgres panahon_restore
   psql -h db.example -U postgres -d panahon_restore \
     -c 'CREATE EXTENSION IF NOT EXISTS postgis;'
   ```

3. Set a target connection string for the new database:

   ```sh
   export TARGET_DSN='postgresql://panahon:...@db.example:5432/panahon_restore?sslmode=require'
   ```

4. Verify the archive before restoring:

   ```sh
   ./scripts/restore-db.sh --source /var/backups/panahon/panahon-20261030T021500Z.dump --verify-only
   ```

5. Restore. The script verifies the checksum sidecar (when present), validates
   the archive, refuses a target that already has `public.users` unless
   `--clean` is given, and asks for confirmation:

   ```sh
   ./scripts/restore-db.sh \
     --source /var/backups/panahon/panahon-20261030T021500Z.dump \
     --target "$TARGET_DSN"
   ```

6. Verify the restored database:

   ```sh
   psql "$TARGET_DSN" -Atc "SELECT extname FROM pg_extension WHERE extname = 'postgis';"
   psql "$TARGET_DSN" -Atc "SELECT count(*) FROM users;"
   psql "$TARGET_DSN" -Atc "SELECT count(*) FROM observations_station;"
   psql "$TARGET_DSN" -Atc "SELECT count(*) FROM observations_observation;"
   psql "$TARGET_DSN" -Atc "SELECT max(timestamp) FROM observations_observation;"
   ```

   Expected counts and the newest observation should match the backup source.
   The restore script runs `ANALYZE` and warns if PostGIS is missing.

7. Point a test/staging API at `TARGET_DSN` first. Start it, check
   `/readyz`, log in, and request a station/observation through the API. When
   that passes, switch production:

   ```sh
   # edit DB_SOURCE in app.env, then restart the API
   docker compose restart api
   ```

   Migrations run on startup. If the binary is older than the dump's schema,
   deploy the matching or newer application version before starting.

8. Check `/readyz` and a real login/API request. Keep the old database and the
   pre-switch backup until the new database has been verified. To roll back,
   restore the previous `DB_SOURCE` and restart the API.

## Restore over an existing database

Not recommended. If it is unavoidable:

1. Take a fresh backup of the target.
2. Put the API in maintenance mode or stop it.
3. Re-run the restore script with `--clean` and confirm the prompt.
4. Verify and restart.

`--clean` drops objects present in the archive; it is not a substitute for a
fresh backup. If the restore fails halfway, drop and recreate the target
database before retrying.

## Disaster recovery on a new host

1. Provision PostgreSQL with the same or newer major version and install
   PostGIS.
2. Create the application role and database, then install PostGIS in the
   database.
3. Fetch the newest verified dump from off-host storage and run the restore
   steps above.
4. Configure `app.env` with the new `DB_SOURCE`, start the API, and verify
   `/readyz`, login, and an observation query.
5. Restore the reverse-proxy/DNS route and monitor errors for the first hour.

## Troubleshooting

- **`role "..." does not exist`**: the archive was taken with ownership
  metadata. Use `--no-owner` (the scripts do) or create the roles first.
- **`type "geometry" does not exist` / PostGIS errors**: enable PostGIS in the
  target database as a superuser before restoring.
- **`pg_restore: error: input file appears to be a text format dump`**: the
  source is not a custom-format (`-Fc`) archive.
- **Checksum mismatch**: discard the file and retrieve another copy. Do not
  restore a corrupt backup.
- **Target already contains `public.users`**: use a new database, or take a
  fresh backup and use `--clean` deliberately.
- **Client/server version mismatch**: use `pg_restore` from the same or newer
  PostgreSQL major version as the server.

## Security checklist

- Dumps are mode `0600` and the backup directory is not world-readable.
- `backups/` is gitignored; never commit dumps.
- Off-host copies are encrypted and access-controlled.
- Restore drills use an isolated target database, never production.
- Old backups are destroyed according to the retention policy.
