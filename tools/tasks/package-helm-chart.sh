#!/usr/bin/env bash
set -euo pipefail

chart="${CHART:-}"
version="${VERSION:-}"

if [ -z "$chart" ] || [ -z "$version" ]; then
  echo "ERROR: CHART and VERSION are required" >&2
  exit 1
fi

chart_dir="deploy/charts/${chart}"
version_no_v="${version#v}"

mkdir -p build/helm
helm-docs --log-level info -s file -c "$chart_dir" \
  -t "deploy/charts/template.md" \
  -t "$chart_dir/README.tmpl.md"
helm dependency update "$chart_dir"
helm package "$chart_dir" \
  --version "$version_no_v" \
  --app-version "$version" \
  --destination build/helm
