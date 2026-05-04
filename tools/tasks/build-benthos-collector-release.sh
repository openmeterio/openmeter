#!/usr/bin/env bash
set -euo pipefail

goos="${GOOS:-}"
goarch="${GOARCH:-}"
version="${VERSION:-unknown}"

if [ -z "$goos" ] || [ -z "$goarch" ]; then
  echo "ERROR: GOOS and GOARCH are required" >&2
  exit 1
fi

outdir="build/release/benthos-collector_${goos}_${goarch}"
rm -rf "$outdir"
mkdir -p "$outdir"

CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
  go build -trimpath \
    -ldflags "-s -w -X main.version=${version}" \
    -o "$outdir/benthos" ./cmd/benthos-collector

cp README.md LICENSE "$outdir/"
