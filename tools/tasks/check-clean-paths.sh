#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 2 ]; then
  echo "usage: $0 <message> <path>..." >&2
  exit 2
fi

message="$1"
shift

if ! git diff --quiet -- "$@" || [ -n "$(git ls-files --others --exclude-standard -- "$@")" ]; then
  git --no-pager diff -- "$@"
  git ls-files --others --exclude-standard -- "$@"
  echo "$message"
  exit 1
fi
