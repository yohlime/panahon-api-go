#!/bin/bash
set -euo pipefail

LEVEL="patch"
DO_COMMIT=false
DO_TAG=false
VERSION_FILE="VERSION"

usage() {
  cat <<EOF
Usage: $0 [patch|minor|major] [--commit] [--tag]

Bumps $VERSION_FILE (default: patch). Without flags only the file is updated.
  --commit  commit the bump (unsigned)
  --tag     commit (implied) and create the lightweight tag vX.Y.Z
EOF
  exit "${1:-1}"
}

for arg in "$@"; do
  case "$arg" in
  patch | minor | major) LEVEL="$arg" ;;
  --commit) DO_COMMIT=true ;;
  --tag) DO_TAG=true ;;
  -h | --help) usage 0 ;;
  *) echo "Unknown argument: $arg" >&2; usage ;;
  esac
done

[[ -f "$VERSION_FILE" ]] || {
  echo "Missing $VERSION_FILE" >&2
  exit 1
}

current="$(tr -d '[:space:]' <"$VERSION_FILE")"
current="${current#v}"
base="${current%%[-+]*}"
IFS=. read -r major minor patch <<<"$base"
major="${major:-0}"
minor="${minor:-0}"
patch="${patch:-0}"

case "$LEVEL" in
major)
  major=$((major + 1))
  minor=0
  patch=0
  ;;
minor)
  minor=$((minor + 1))
  patch=0
  ;;
patch) patch=$((patch + 1)) ;;
esac
new="$major.$minor.$patch"

printf '%s\n' "$new" >"$VERSION_FILE"
echo "version: ${current:-?} -> $new"

if $DO_TAG && ! $DO_COMMIT; then
  DO_COMMIT=true
fi
if $DO_COMMIT; then
  git add "$VERSION_FILE"
  git commit --no-gpg-sign -m "chore: bump version to $new" -- "$VERSION_FILE"
fi
if $DO_TAG; then
  git tag "v$new"
  echo "tagged v$new"
fi
