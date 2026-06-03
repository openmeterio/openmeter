#!/usr/bin/env bash
# Archie pre-commit review — Phase 3 semantic comparison on the staged diff.
# Same flow as post-plan-review: align_check.py classifier first (blocks
# on decision_violation / pitfall_triggered), then arch_review prose context.
PROJECT_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || echo ".")"
[ ! -f "$PROJECT_ROOT/.archie/blueprint.json" ] && exit 0
TOOL_INPUT=$(cat || true)
COMMAND=$(echo "$TOOL_INPUT" | python3 -c "
import sys, json
try: print(json.load(sys.stdin).get('tool_input',{}).get('command',''))
except: print('')
" 2>/dev/null || echo "")
case "$COMMAND" in *git\ commit*|*git\ -C*commit*) ;; *) exit 0 ;; esac
ALIGN="$PROJECT_ROOT/.archie/align_check.py"
if [ -f "$ALIGN" ]; then
    python3 "$ALIGN" commit "$PROJECT_ROOT"
    rc=$?
    if [ $rc -ne 0 ]; then
        exit $rc
    fi
fi
python3 "$PROJECT_ROOT/.archie/arch_review.py" diff "$PROJECT_ROOT"
