#!/usr/bin/env bash
# Archie lint gate — opt-in via .archie/enforcement.json.
# Runs the project's native linter on a single changed file after Write/Edit/MultiEdit.
PROJECT_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || echo ".")"
CFG="$PROJECT_ROOT/.archie/enforcement.json"
GATE="$PROJECT_ROOT/.archie/lint_gate.py"
[ ! -f "$CFG" ] && exit 0
[ ! -f "$GATE" ] && exit 0
export _ARCHIE_ROOT="$PROJECT_ROOT"
python3 "$GATE"
