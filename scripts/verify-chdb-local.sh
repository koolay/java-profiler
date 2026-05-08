#!/usr/bin/env sh
set -eu

root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
required="${CHDB_REQUIRED:-0}"

has_libchdb() {
  if command -v chdb >/dev/null 2>&1; then
    return 0
  fi
  search_path="$(printf '%s:%s:/usr/local/lib:/opt/homebrew/lib:/opt/lib' "${LD_LIBRARY_PATH:-}" "${DYLD_LIBRARY_PATH:-}")"
  old_ifs="$IFS"
  IFS=":"
  for dir in $search_path; do
    [ -n "$dir" ] || continue
    if ls "$dir"/libchdb.* >/dev/null 2>&1; then
      IFS="$old_ifs"
      return 0
    fi
  done
  IFS="$old_ifs"
  return 1
}

if ! has_libchdb; then
  cat <<'MSG'
libchdb was not found; skipping chDB local smoke verification.

Install it with the official chDB installer before running this check:
  curl -sL https://lib.chdb.io | bash

To make missing libchdb fail automation, run:
  CHDB_REQUIRED=1 scripts/verify-chdb-local.sh
MSG
  if [ "$required" = "1" ]; then
    exit 1
  fi
  exit 0
fi

cd "$root/tools/chdb-smoke"
go mod download
go run . "$root/db/clickhouse/001_initial_profile_schema.sql"
