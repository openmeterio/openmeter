#!/usr/bin/env bash
set -euo pipefail

goos="${GOOS:-}"
goarch="${GOARCH:-}"

if [ -z "$goos" ] || [ -z "$goarch" ]; then
  echo "ERROR: GOOS and GOARCH are required" >&2
  exit 1
fi

name="benthos-collector_${goos}_${goarch}"
tar -C build/release -czf "build/release/${name}.tar.gz" "$name"
