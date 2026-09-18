#!/usr/bin/env bash
#
# backup.sh — dump the defShows Postgres database to a timestamped, gzipped file.
#
# Runs pg_dump inside the running `postgres` compose service (no host Postgres
# client or published DB port required) and streams the dump to the host.
#
# Usage:
#   ./backup.sh                    # writes ./backups/defshows-YYYYmmdd-HHMMSS.sql.gz
#   BACKUP_DIR=/mnt/backups ./backup.sh
#   RETENTION_DAYS=14 ./backup.sh  # prune dumps older than N days (default 7; 0 = keep all)
#
# Restore:
#   gunzip -c backups/defshows-XXXX.sql.gz | \
#     docker compose -f docker-compose.prod.yml exec -T postgres \
#       psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"
#
# Cron example (daily at 03:30, from this directory):
#   30 3 * * * cd /opt/defshows/deploy && ./backup.sh >> backup.log 2>&1
#
set -euo pipefail

cd "$(dirname "$0")"

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.prod.yml}"
SERVICE="${POSTGRES_SERVICE:-postgres}"
BACKUP_DIR="${BACKUP_DIR:-./backups}"
RETENTION_DAYS="${RETENTION_DAYS:-7}"

# Load POSTGRES_* from .env if present (so we don't duplicate credentials here).
if [[ -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
fi

: "${POSTGRES_USER:?POSTGRES_USER is required (set it in .env or the environment)}"
: "${POSTGRES_DB:?POSTGRES_DB is required (set it in .env or the environment)}"

mkdir -p "$BACKUP_DIR"
STAMP="$(date +%Y%m%d-%H%M%S)"
OUT="$BACKUP_DIR/${POSTGRES_DB}-${STAMP}.sql.gz"

compose() { docker compose -f "$COMPOSE_FILE" "$@"; }

echo "[backup] dumping database '$POSTGRES_DB' via service '$SERVICE'…"
# -T: no TTY (required when piping). pg_dump inside the container connects locally.
compose exec -T "$SERVICE" pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
  | gzip -c > "$OUT"

SIZE="$(du -h "$OUT" | cut -f1)"
echo "[backup] wrote $OUT ($SIZE)"

if [[ "$RETENTION_DAYS" -gt 0 ]]; then
  echo "[backup] pruning dumps older than ${RETENTION_DAYS} day(s) in $BACKUP_DIR"
  find "$BACKUP_DIR" -type f -name "${POSTGRES_DB}-*.sql.gz" -mtime +"$RETENTION_DAYS" -print -delete
fi

echo "[backup] done."
