#!/usr/bin/env bash
# Generate SQLC testdata for a specific migration version.
#
# Given a golang-migrate version timestamp (e.g. 20240826120919) this script:
#   1. Spins up a clean postgres via docker compose (reusing the repo service)
#   2. Runs all migrations up to the requested version against a scratch database
#   3. Dumps the resulting schema with pg_dump
#   4. Writes sqlc.yaml + placeholder queries alongside the schema
#   5. Runs `sqlc generate` to produce the Go structs
#
# Requires: docker compose, migrate (golang-migrate), pg_dump, sqlc (all provided
# by the repo's nix dev shell).
#
# Usage: VERSION=20240826120919 ./tools/migrate/generate-sqlc-testdata.sh
#        (or: make generate-sqlc-testdata VERSION=20240826120919)

set -euo pipefail

if [[ -z "${VERSION:-}" ]]; then
	echo "ERROR: VERSION is required (e.g. VERSION=20240826120919)" >&2
	exit 1
fi

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUT_DIR="${OUT_DIR:-${REPO_ROOT}/tools/migrate/testdata/sqlcgen/${VERSION}}"
MIGRATIONS_DIR="${REPO_ROOT}/tools/migrate/migrations"

PG_HOST="${POSTGRES_HOST:-127.0.0.1}"
PG_PORT="${POSTGRES_PORT:-5432}"
PG_USER="${POSTGRES_USER:-postgres}"
PG_PASSWORD="${POSTGRES_PASSWORD:-postgres}"
SCRATCH_DB="sqlc_gen_${VERSION}"

export PGPASSWORD="${PG_PASSWORD}"

for bin in docker migrate pg_dump psql sqlc; do
	if ! command -v "${bin}" >/dev/null 2>&1; then
		echo "ERROR: required binary '${bin}' not found in PATH" >&2
		echo "Hint: run inside the nix dev shell (e.g. 'nix develop --impure')" >&2
		exit 1
	fi
done

echo ">>> Ensuring postgres is running (docker compose up -d postgres)"
(cd "${REPO_ROOT}" && docker compose up -d postgres)
"${REPO_ROOT}/tools/wait-for-compose.sh" postgres

echo ">>> Creating scratch database ${SCRATCH_DB}"
psql -h "${PG_HOST}" -p "${PG_PORT}" -U "${PG_USER}" -d postgres \
	-c "DROP DATABASE IF EXISTS ${SCRATCH_DB};" \
	-c "CREATE DATABASE ${SCRATCH_DB};"

DB_URL="postgres://${PG_USER}:${PG_PASSWORD}@${PG_HOST}:${PG_PORT}/${SCRATCH_DB}?sslmode=disable&x-migrations-table=schema_om"

echo ">>> Migrating ${SCRATCH_DB} to version ${VERSION}"
migrate -path "${MIGRATIONS_DIR}" -database "${DB_URL}" goto "${VERSION}"

mkdir -p "${OUT_DIR}/sqlc"

echo ">>> Dumping schema to ${OUT_DIR}/sqlc/db-schema.sql"
pg_dump -h "${PG_HOST}" -p "${PG_PORT}" -U "${PG_USER}" -d "${SCRATCH_DB}" \
	-s --no-owner --no-acl > "${OUT_DIR}/sqlc/db-schema.sql"

echo ">>> Writing sqlc.yaml and placeholder queries"
cat > "${OUT_DIR}/sqlc.yaml" <<'YAML'
version: "2"
sql:
  - engine: "postgresql"
    queries: "sqlc/queries.sql"
    schema: "sqlc/db-schema.sql"
    gen:
      go:
        package: "db"
        out: "db"
YAML

cat > "${OUT_DIR}/sqlc/queries.sql" <<'SQL'
-- Add your SQL queries here
-- Example:
-- name: GetExampleByID :one
-- SELECT * FROM example_table WHERE id = $1;

-- Placeholder query for SQLC validation
-- name: GetSchemaVersion :one
SELECT version FROM schema_om ORDER BY version DESC LIMIT 1;
SQL

echo ">>> Running sqlc generate in ${OUT_DIR}"
(cd "${OUT_DIR}" && sqlc generate)

echo ">>> Dropping scratch database ${SCRATCH_DB}"
psql -h "${PG_HOST}" -p "${PG_PORT}" -U "${PG_USER}" -d postgres \
	-c "DROP DATABASE IF EXISTS ${SCRATCH_DB};" >/dev/null

echo ">>> Done — SQLC testdata written to ${OUT_DIR}"
