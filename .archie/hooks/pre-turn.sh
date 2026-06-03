#!/usr/bin/env bash
# Archie pre-turn reset — clears the per-turn rule-injection marker so the
# next Write/Edit surfaces applicable rules again. Runs on UserPromptSubmit.
PROJECT_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || echo ".")"
TURN_HASH=$(printf '%s' "$PROJECT_ROOT" | cksum | awk '{print $1}')
rm -f "/tmp/.archie_turn_$TURN_HASH"
exit 0
