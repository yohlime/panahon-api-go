#!/bin/bash
set -euo pipefail

SOURCE=""
TARGET=""
CLEAN=false
ASSUME_YES=false
VERIFY_ONLY=false

usage() {
  cat <<EOF
Usage: $0 --source FILE --target DSN [options]

Restore a custom-format Panahon backup into an existing database.

Options:
  --source FILE     Backup file to restore
  --target DSN      PostgreSQL connection string for the target database
  --verify-only     Only verify the archive and print its table of contents
  --clean           Drop existing objects before restoring (dangerous)
  --yes             Do not prompt for confirmation
  -h, --help        Show this help

Create an empty target database first, for example:
  createdb -h db.example ph_restore
  psql -h db.example -d ph_restore -c 'CREATE EXTENSION IF NOT EXISTS postgis;'

The script refuses a non-empty target unless --clean is supplied. Always take
a fresh backup before using --clean.
EOF
}

die() {
  echo "error: $*" >&2
  exit 1
}

redact_dsn() {
  printf '%s' "$1" | sed -E 's#(://)[^/@:]+:[^/@]*@#\1***:***@#'
}

while [[ $# -gt 0 ]]; do
  case "$1" in
  --source)
    SOURCE="${2:?--source requires a file}"
    shift 2
    ;;
  --target)
    TARGET="${2:?--target requires a connection string}"
    shift 2
    ;;
  --verify-only) VERIFY_ONLY=true; shift ;;
  --clean) CLEAN=true; shift ;;
  --yes) ASSUME_YES=true; shift ;;
  -h | --help)
    usage
    exit 0
    ;;
  *)
    echo "Unknown argument: $1" >&2
    usage >&2
    exit 2
    ;;
  esac
done

[[ -n "$SOURCE" ]] || {
  usage >&2
  exit 2
}
[[ -f "$SOURCE" ]] || die "backup file not found: $SOURCE"
command -v pg_restore >/dev/null 2>&1 || die "pg_restore is not on PATH"
command -v psql >/dev/null 2>&1 || die "psql is not on PATH"

if [[ -f "$SOURCE.sha256" ]]; then
  checksum_dir="$(cd "$(dirname "$SOURCE")" && pwd)"
  checksum_file="$(basename "$SOURCE").sha256"
  (cd "$checksum_dir" && sha256sum -c "$checksum_file") || die "backup checksum verification failed"
else
  echo "warning: no checksum sidecar next to $SOURCE" >&2
fi

pg_restore --list "$SOURCE" >/dev/null || die "not a valid custom-format backup"

if $VERIFY_ONLY; then
  pg_restore --list "$SOURCE"
  exit 0
fi

[[ -n "$TARGET" ]] || {
  usage >&2
  exit 2
}

safe_target="$(redact_dsn "$TARGET")"
echo "target: $safe_target"

# A new database may already have PostGIS installed. Treat only an existing
# public.users table as a previous Panahon restore; anything else is checked
# after connection so a missing database fails with a clear message.
has_users="$(psql "$TARGET" -Atqc "SELECT to_regclass('public.users') IS NOT NULL;" 2>/dev/null || true)"
if [[ -z "$has_users" ]]; then
  die "cannot connect to target database; create it first and check the connection string"
fi
if [[ "$has_users" == "t" && "$CLEAN" != true ]]; then
  die "target already contains public.users; use a new database or pass --clean only after taking a backup"
fi

if ! $ASSUME_YES; then
  echo
  echo "This will restore $SOURCE into $safe_target."
  read -r -p "Continue? [y/N] " reply
  [[ "$reply" =~ ^[Yy]$ ]] || {
    echo "aborted"
    exit 1
  }
fi

restore_args=(--no-owner --no-privileges --exit-on-error --dbname="$TARGET" "$SOURCE")
if $CLEAN; then
  restore_args=(--clean --if-exists "${restore_args[@]}")
fi

pg_restore "${restore_args[@]}"

# pg_restore does not update planner statistics; ANALYZE keeps the first queries
# after a restore from being unnecessarily slow.
if ! psql "$TARGET" -v ON_ERROR_STOP=1 -c "ANALYZE;" >/dev/null; then
  echo "warning: ANALYZE failed; run it manually before switching traffic" >&2
fi

postgis="$(psql "$TARGET" -Atqc "SELECT extname FROM pg_extension WHERE extname = 'postgis';" 2>/dev/null || true)"
if [[ "$postgis" != "postgis" ]]; then
  echo "warning: PostGIS extension is missing in the target database" >&2
fi

echo "restore complete: $safe_target"
