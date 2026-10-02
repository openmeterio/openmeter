#!/usr/bin/env bash

set -euo pipefail

image="${1:?OpenMeter image is required}"
pull_policy="${2:?image pull policy is required}"
repository="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"

cd "${repository}/quickstart"

export COMPOSE_PROFILES=webhook
export OPENMETER_ADDRESS=http://localhost:48888

compose=(docker compose -f docker-compose.yaml -f docker-compose.override.yaml)
log_dir="artifacts/logs/docker-compose/quickstart"

cleanup() {
  status=$?
  trap - EXIT
  set +e
  mkdir -p "${log_dir}"
  "${compose[@]}" ps --all > "${log_dir}/compose-ps.txt" 2>&1
  "${compose[@]}" logs --no-color --timestamps > "${log_dir}/compose.log" 2>&1
  "${compose[@]}" down -v
  exit "${status}"
}
trap cleanup EXIT

cat > docker-compose.override.yaml <<EOF
services:
  openmeter:
    image: ${image}
    pull_policy: ${pull_policy}
  sink-worker:
    image: ${image}
    pull_policy: ${pull_policy}
  balance-worker:
    image: ${image}
    pull_policy: ${pull_policy}
  notification-service:
    image: ${image}
    pull_policy: ${pull_policy}
  billing-worker:
    image: ${image}
    pull_policy: ${pull_policy}
  openmeter-jobs:
    image: ${image}
    pull_policy: ${pull_policy}
EOF

"${compose[@]}" up -d
curl --fail --retry 10 --retry-max-time 120 --retry-all-errors http://localhost:40000/healthz
go test -v -count=1 .
