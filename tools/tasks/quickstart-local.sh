#!/usr/bin/env bash
set -euo pipefail

cd quickstart

compose=(
  docker compose
  -f docker-compose.yaml
  -f docker-compose.debug-ports.yaml
)

"${compose[@]}" down
cleanup() {
  "${compose[@]}" down
}
trap cleanup EXIT

"${compose[@]}" up -d --force-recreate

curl --retry 10 --retry-max-time 120 --retry-all-errors http://localhost:40000/healthz

OPENMETER_ADDRESS=http://localhost:48888 go test -count=1 -v ./...
