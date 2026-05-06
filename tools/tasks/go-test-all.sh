#!/usr/bin/env bash
set -euo pipefail

docker compose up -d postgres svix redis
./tools/wait-for-compose.sh postgres svix redis

GO_BUILD_FLAGS="${GO_BUILD_FLAGS--tags=dynamic}"
GO_TEST_PACKAGE_PARALLELISM="${GO_TEST_PACKAGE_PARALLELISM-128}"
GO_TEST_FLAGS="${GO_TEST_FLAGS--p ${GO_TEST_PACKAGE_PARALLELISM} -parallel 16 ${GO_BUILD_FLAGS}}"

# GO_TEST_FLAGS intentionally follows the old Makefile word-splitting behavior.
# shellcheck disable=SC2086
POSTGRES_HOST="${POSTGRES_HOST:-127.0.0.1}" \
  SVIX_HOST="${SVIX_HOST-localhost}" \
  SVIX_JWT_SECRET="${SVIX_JWT_SECRET-DUMMY_JWT_SECRET}" \
  go test ${GO_TEST_FLAGS} -count=1 ./...
