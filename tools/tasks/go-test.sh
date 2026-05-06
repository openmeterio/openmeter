#!/usr/bin/env bash
set -euo pipefail

count_arg=()

case "${1:-}" in
  "")
    ;;
  --nocache)
    count_arg=(-count=1)
    ;;
  *)
    echo "usage: $0 [--nocache]" >&2
    exit 2
    ;;
esac

GO_BUILD_FLAGS="${GO_BUILD_FLAGS--tags=dynamic}"
GO_TEST_PACKAGE_PARALLELISM="${GO_TEST_PACKAGE_PARALLELISM-128}"
GO_TEST_FLAGS="${GO_TEST_FLAGS--p ${GO_TEST_PACKAGE_PARALLELISM} -parallel 16 ${GO_BUILD_FLAGS}}"

# GO_TEST_FLAGS intentionally follows the old Makefile word-splitting behavior.
# shellcheck disable=SC2086
POSTGRES_HOST="${POSTGRES_HOST:-127.0.0.1}" go test ${GO_TEST_FLAGS} "${count_arg[@]}" ./...
