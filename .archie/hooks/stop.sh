#!/usr/bin/env bash
# Archie Stop hook (canonical, source of truth).
# Runs when the agent finishes a turn/session depending on CLI semantics.
# Currently used as a parity extension point and light cleanup hook.
PROJECT_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || echo ".")"
TURN_HASH=$(printf '%s' "$PROJECT_ROOT" | cksum | awk '{print $1}')
# Avoid leaking per-turn rule injection state if a session ends without the
# next UserPromptSubmit event clearing it.
rm -f "/tmp/.archie_turn_$TURN_HASH"
exit 0
