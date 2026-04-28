#!/usr/bin/env bash
# Run every unit_test_suite found in every *.json collection in a given directory.
# Usage: [INSOMNIA_ENV="Local Dev"] run.sh [collections-dir]
set -euo pipefail

INSOMNIA_ENV="${INSOMNIA_ENV:-Local Dev}"
dir="${1:-$(dirname "$0")}"
tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT

shopt -s nullglob
collections=("$dir"/*.json)

if [[ ${#collections[@]} -eq 0 ]]; then
    echo "No *.json collections found in $dir" >&2
    exit 1
fi

for collection in "${collections[@]}"; do
    printf '\n==> %s\n' "$collection"

    python3 - "$collection" > "$tmp" <<'PYEOF'
import json, sys
data = json.load(open(sys.argv[1]))
for r in data.get("resources", []):
    if r.get("_type") == "unit_test_suite":
        print(r["name"])
PYEOF

    while IFS= read -r suite; do
        inso --ci -w "$collection" run test "$suite" --env "$INSOMNIA_ENV"
    done < "$tmp"
done
