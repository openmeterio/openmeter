#!/usr/bin/env bash
set -euo pipefail

PGPASSWORD="${PGPASSWORD:-postgres}" \
  psql \
    -h "${POSTGRES_HOST:-127.0.0.1}" \
    -U "${POSTGRES_USER:-postgres}" \
    "${POSTGRES_DB:-postgres}" \
    -c "SELECT version();" || {
      echo "!!! Postgres is not running. Please start it with 'docker compose up -d postgres' !!!"
      exit 1
    }
