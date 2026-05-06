#!/usr/bin/env bash
set -euo pipefail

cd e2e

compose=(
  docker compose
  -f docker-compose.infra.yaml
  -f docker-compose.debug-ports.yaml
  -f docker-compose.openmeter.yaml
  -f docker-compose.openmeter-local.yaml
)

"${compose[@]}" down
cleanup() {
  "${compose[@]}" down
}
trap cleanup EXIT

"${compose[@]}" up -d --build --force-recreate

# wait for sink-worker to be ready
curl --retry 10 --retry-max-time 120 --retry-all-errors http://localhost:30000/healthz

TZ=UTC OPENMETER_ADDRESS=http://localhost:38888 go test -count=1 -v ./...
