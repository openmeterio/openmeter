#!/usr/bin/env bash
# Archie blueprint nudge — reminds the AI about project architecture before searching.
# Fires on Glob|Grep so the agent reads architecture context before exploring the
# codebase. Inspired by Graphify's always-on hook pattern.
PROJECT_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || echo ".")"
BLUEPRINT="$PROJECT_ROOT/.archie/blueprint.json"
[ ! -f "$BLUEPRINT" ] && exit 0
MARKER="/tmp/.archie_nudge_$$"
[ -f "$MARKER" ] && exit 0
touch "$MARKER"
python3 -c "
import json, os
bp_path = os.path.join('$PROJECT_ROOT', '.archie', 'blueprint.json')
try:
    with open(bp_path) as f:
        bp = json.load(f)
    comps = bp.get('components', {}).get('components', [])
    names = [c.get('name', '') for c in comps[:5] if c.get('name')]
    style = bp.get('decisions', {}).get('architectural_style', {})
    ptype = style.get('style', '') if isinstance(style, dict) else str(style)
    parts = []
    if ptype:
        parts.append(ptype)
    if names:
        parts.append('Components: ' + ', '.join(names))
    if parts:
        print('[Archie] ' + ' | '.join(parts))
        print('[Archie] Read .archie/blueprint.json for architecture context before searching.')
except Exception:
    pass
" 2>/dev/null
exit 0
