#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${APP_ENV_FILE:-$ROOT/app.env}"
OUTPUT_DIR="${BACKUP_DIR:-$ROOT/backups}"
KEEP=14
CHECKSUM=true

usage() {
  cat <<EOF
Usage: $0 [options]

Create a compressed PostgreSQL custom-format backup of DB_SOURCE.

Options:
  --env FILE        Config file to load (default: $ROOT/app.env)
  --output DIR      Backup directory (default: $ROOT/backups)
  --keep N          Keep the N newest backups; 0 keeps all (default: 14)
  --no-checksum     Do not write a SHA-256 sidecar
  -h, --help        Show this help

The backup is written as panahon-<UTC timestamp>.dump with mode 0600. The
directory should be on encrypted storage and copied off-host regularly.
EOF
}

die() {
  echo "error: $*" >&2
  exit 1
}

while [[ $# -gt 0 ]]; do
  case "$1" in
  --env)
    ENV_FILE="${2:?--env requires a file}"
    shift 2
    ;;
  --output)
    OUTPUT_DIR="${2:?--output requires a directory}"
    shift 2
    ;;
  --keep)
    KEEP="${2:?--keep requires a non-negative integer}"
    shift 2
    ;;
  --no-checksum) CHECKSUM=false; shift ;;
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

[[ "$KEEP" =~ ^[0-9]+$ ]] || die "--keep must be a non-negative integer"
[[ -f "$ENV_FILE" ]] || die "config file not found: $ENV_FILE"
command -v pg_dump >/dev/null 2>&1 || die "pg_dump is not on PATH"
command -v pg_restore >/dev/null 2>&1 || die "pg_restore is not on PATH"
if $CHECKSUM; then
  command -v sha256sum >/dev/null 2>&1 || die "sha256sum is not on PATH"
fi

# app.env is trusted local configuration. Export it so DB_SOURCE is visible to
# pg_dump and to the checks in this script.
set -a
# shellcheck disable=SC1090
. "$ENV_FILE"
set +a
: "${DB_SOURCE:?DB_SOURCE must be set in $ENV_FILE}"

umask 077
mkdir -p "$OUTPUT_DIR"

timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
final="$OUTPUT_DIR/panahon-$timestamp.dump"
if [[ -e "$final" ]]; then
  final="$OUTPUT_DIR/panahon-$timestamp-$$.dump"
fi
name="$(basename "$final")"
tmp="$final.partial.$$"

cleanup() {
  rm -f -- "$tmp"
}
trap cleanup EXIT

echo "backing up to $final"
pg_dump --format=custom --compress=9 --no-owner --no-privileges \
  --file "$tmp" "$DB_SOURCE" || die "pg_dump failed"

# A custom-format archive has a readable table of contents. This catches
# truncated writes before the file is promoted to a real backup.
pg_restore --list "$tmp" >/dev/null || die "backup verification failed"

chmod 600 "$tmp"
mv -- "$tmp" "$final"
trap - EXIT

if $CHECKSUM; then
  (cd "$OUTPUT_DIR" && sha256sum "$name" >"$name.sha256")
  chmod 600 "$final.sha256"
fi

echo "backup: $final"
echo "size: $(du -h "$final" | cut -f1)"
if $CHECKSUM; then
  echo "sha256: $(cut -d' ' -f1 "$final.sha256")"
fi

if [[ "$KEEP" -gt 0 ]]; then
  mapfile -t backups < <(find "$OUTPUT_DIR" -maxdepth 1 -type f -name 'panahon-*.dump' -printf '%T@\t%f\n' | sort -rn | cut -f2-)
  for ((i = KEEP; i < ${#backups[@]}; i++)); do
    old="$OUTPUT_DIR/${backups[$i]}"
    rm -f -- "$old" "$old.sha256"
    echo "removed old backup: $old"
  done
fi
