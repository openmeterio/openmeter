#!/usr/bin/env bash
# Archie post-plan review — Phase 3 semantic comparison.
# Pipes the PostToolUse ExitPlanMode envelope to align_check.py, which
# pulls the plan text, runs the architectural-rule classifier (Haiku
# via claude CLI), and exits 2 if any decision_violation or
# pitfall_triggered fires. Falls back to advisory mode if classifier unavailable.
PROJECT_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || echo ".")"
[ ! -f "$PROJECT_ROOT/.archie/blueprint.json" ] && exit 0
TOOL_INPUT=$(cat || true)
[ -z "$TOOL_INPUT" ] && exit 0
ALIGN="$PROJECT_ROOT/.archie/align_check.py"
if [ -f "$ALIGN" ]; then
    printf '%s' "$TOOL_INPUT" | python3 "$ALIGN" plan "$PROJECT_ROOT"
    rc=$?
    if [ $rc -ne 0 ]; then
        exit $rc
    fi
fi
python3 "$PROJECT_ROOT/.archie/arch_review.py" plan "$PROJECT_ROOT"
