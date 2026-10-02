#!/usr/bin/env bash

set -euo pipefail

variant="${1:?E2E variant is required}"
image="${2:?OpenMeter image is required}"
pull_policy="${3:?image pull policy is required}"
dependency_source="${4:?dependency image source is required}"
repository="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"

case "${dependency_source}" in
  depot|upstream) ;;
  *)
    echo "Unsupported dependency image source: ${dependency_source}"
    exit 1
    ;;
esac

cd "${repository}/e2e"

export OPENMETER_ADDRESS=http://localhost:38888
export TZ=UTC

compose=(
  docker compose
  -f docker-compose.infra.yaml
  -f docker-compose.openmeter.yaml
  -f docker-compose.override.yaml
)

case "${variant}" in
  base)
    if [ "${dependency_source}" = depot ]; then
      compose+=(-f ../.github/docker-compose.depot-registry.yaml)
    fi
    test_target=test-base
    ;;
  credits-disabled)
    compose+=(-f docker-compose.credits-disabled.yaml)
    test_target=test-credits-disabled
    ;;
  *)
    echo "Unsupported E2E variant: ${variant}"
    exit 1
    ;;
esac

log_dir="artifacts/logs/docker-compose/${variant}"
log_pid=""

cleanup() {
  status=$?
  trap - EXIT
  set +e
  if [ -n "${log_pid}" ]; then
    kill "${log_pid}" 2>/dev/null
    wait "${log_pid}" 2>/dev/null
  fi
  mkdir -p "${log_dir}"
  "${compose[@]}" ps --all > "${log_dir}/compose-ps.txt" 2>&1
  mapfile -t services < <("${compose[@]}" config --services)
  for service in "${services[@]}"; do
    "${compose[@]}" logs --no-color --timestamps "${service}" > "${log_dir}/${service}.log" 2>&1
  done
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
  billing-worker:
    image: ${image}
    pull_policy: ${pull_policy}
EOF

mkdir -p "${log_dir}"
"${compose[@]}" up -d
"${compose[@]}" logs --no-color --timestamps --follow > "${log_dir}/compose-follow.log" 2>&1 &
log_pid=$!
curl --fail --retry 10 --retry-max-time 120 --retry-all-errors http://localhost:30000/healthz
curl --fail --retry 10 --retry-max-time 120 --retry-all-errors http://localhost:30001/healthz
make "${test_target}"
