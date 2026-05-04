#!/usr/bin/env bash
set -euo pipefail

go mod download github.com/oapi-codegen/oapi-codegen/v2

OAPI_MOD_DIR="$(go list -m -f '{{.Dir}}' github.com/oapi-codegen/oapi-codegen/v2)"
if [ -z "$OAPI_MOD_DIR" ]; then
  echo "error: could not locate oapi-codegen/v2 module dir" >&2
  exit 1
fi

cp "$OAPI_MOD_DIR/pkg/codegen/templates/chi/chi-middleware.tmpl" api/v3/templates/chi-middleware.tmpl
chmod u+w api/v3/templates/chi-middleware.tmpl
patch -p1 -d api/v3/templates < api/v3/templates/chi-middleware.tmpl.patch
