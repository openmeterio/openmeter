#!/usr/bin/env bash
# Archie pre-validate hook (canonical, source of truth).
# Reads rules.json + platform_rules.json, matches against the tool_input,
# emits agent-facing rule context, and exits 2 if any blocking rule fires.
#
# This script is invoked by:
#   - Claude Code's .claude/settings.json PreToolUse[Edit|Write|MultiEdit] hook
#   - Codex's .codex/hooks.json PreToolUse[apply_patch] hook (Stage 3)
#   - Pi's .pi/extensions/archie-hooks.ts (shells out to this; Stage 5)
# Each CLI normalizes its tool_call envelope to the same JSON shape that this
# script expects on stdin: { tool_name, tool_input, ... }.
PROJECT_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || echo ".")"
RULES_FILE="$PROJECT_ROOT/.archie/rules.json"
PLATFORM_RULES_FILE="$PROJECT_ROOT/.archie/platform_rules.json"
# Fail open if no rules at all.
if [ ! -f "$RULES_FILE" ] && [ ! -f "$PLATFORM_RULES_FILE" ]; then
    exit 0
fi
TOOL_INPUT=$(cat || true)
[ -z "$TOOL_INPUT" ] && exit 0

# Stable-named turn marker (cleared by pre-turn.sh on UserPromptSubmit).
# Holds rule IDs already injected this turn so we don't re-surface the same
# rule on every subsequent Write/Edit in the chain.
TURN_HASH=$(printf '%s' "$PROJECT_ROOT" | cksum | awk '{print $1}')
TURN_FILE="/tmp/.archie_turn_$TURN_HASH"

# Stash the whole tool-call JSON in a temp file so Python can parse it
# without shell quoting corrupting backslashes, newlines, or quotes in content.
TOOL_JSON=$(mktemp)
printf '%s' "$TOOL_INPUT" > "$TOOL_JSON"
export _ARCHIE_TOOL_JSON="$TOOL_JSON"
export _ARCHIE_ROOT="$PROJECT_ROOT"
export _ARCHIE_RULES="$RULES_FILE"
export _ARCHIE_PLATFORM_RULES="$PLATFORM_RULES_FILE"
export _ARCHIE_TURN_FILE="$TURN_FILE"
trap 'rm -f "$TOOL_JSON"' EXIT

python3 << 'PYEOF'
import json, sys, os, re, fnmatch

tool_json_file = os.environ.get("_ARCHIE_TOOL_JSON", "")
project_root = os.environ.get("_ARCHIE_ROOT", ".")
rules_file = os.environ.get("_ARCHIE_RULES", "")
platform_rules_file = os.environ.get("_ARCHIE_PLATFORM_RULES", "")
turn_file = os.environ.get("_ARCHIE_TURN_FILE", "")

try:
    with open(tool_json_file) as f:
        data = json.load(f)
except Exception:
    sys.exit(0)

tool_name = data.get("tool_name", "")
# Accept Claude's Write|Edit|MultiEdit and Codex's apply_patch.
if tool_name not in ("Write", "Edit", "MultiEdit", "apply_patch"):
    sys.exit(0)

ti = data.get("tool_input", {})
fp = ti.get("file_path", "") or ti.get("path", "")
if not fp:
    sys.exit(0)

def _real(p):
    try:
        return os.path.realpath(p)
    except Exception:
        return p
fp_real = _real(fp) if fp.startswith("/") else fp
project_root_real = _real(project_root)
if fp_real.startswith("/") and not fp_real.startswith(project_root_real):
    sys.exit(0)

content = ti.get("content", "") or ti.get("new_string", "")
if not content and ti.get("edits"):
    content = "\n".join(e.get("new_string", "") for e in ti["edits"])

rules = []
for rpath in (rules_file, platform_rules_file):
    if not rpath or not os.path.isfile(rpath):
        continue
    try:
        loaded = json.load(open(rpath))
        rules.extend(loaded.get("rules", []) if isinstance(loaded, dict) else loaded)
    except Exception:
        pass

rel_path = fp
if fp_real.startswith(project_root_real):
    rel_path = fp_real[len(project_root_real):].lstrip("/")
filename = os.path.basename(fp)

SEVERITY_BLOCKING = {"decision_violation", "pitfall_triggered", "mechanical_violation"}
SEVERITY_WARN = {"tradeoff_undermined"}
SEVERITY_INFO = {"pattern_divergence"}

def severity_label(rule):
    sc = rule.get("severity_class", "")
    if sc:
        return sc
    return "mechanical_violation" if rule.get("severity") == "error" else "tradeoff_undermined"

def is_blocking(rule):
    sc = rule.get("severity_class", "")
    if sc:
        return sc in SEVERITY_BLOCKING
    return rule.get("severity") == "error"

def _emit_labeled_block(label, value, prefix, sink):
    if not value:
        return
    pad = " " * len(label + ": ")
    first = True
    for ln in value.splitlines() or [value]:
        sink(f"{prefix}  {label + ': ' if first else pad}{ln}")
        first = False

already_injected = set()
if turn_file and os.path.isfile(turn_file):
    try:
        already_injected = {line.strip() for line in open(turn_file) if line.strip()}
    except Exception:
        pass

to_inject = []
newly_injected = []
for r in rules:
    rid = r.get("id", "")
    if not rid or rid in already_injected:
        continue
    applies_to = r.get("applies_to", "")
    always = bool(r.get("always_inject"))
    path_match = bool(applies_to) and rel_path.startswith(applies_to)
    if always or path_match:
        to_inject.append((r, "always" if (always and not path_match) else "path"))
        already_injected.add(rid)
        newly_injected.append(rid)

if to_inject:
    print(f"[Archie] Rules applying to {rel_path}:")
    for r, how in to_inject:
        tag = " (global)" if how == "always" else ""
        rid = r.get("id", "unknown")
        sev = severity_label(r)
        desc = r.get("description", "")
        print(f"  RULE {rid}{tag} [{sev}]: {desc}")
        _emit_labeled_block("WHY", r.get("why", "") or r.get("rationale", ""), "  ", print)
        _emit_labeled_block("FORCED BY", r.get("forced_by", ""), "  ", print)
        _emit_labeled_block("ENABLES", r.get("enables", ""), "  ", print)
        _emit_labeled_block("DO INSTEAD", r.get("alternative", ""), "  ", print)
        _emit_labeled_block("EXAMPLE", r.get("example", ""), "  ", print)

if newly_injected and turn_file:
    try:
        with open(turn_file, "a") as f:
            for rid in newly_injected:
                f.write(rid + "\n")
    except Exception:
        pass

def any_match(patterns, text):
    for p in patterns:
        try:
            if re.search(p, text):
                return True
        except re.error:
            continue
    return False

def _coerce_list(v):
    if v is None:
        return []
    if isinstance(v, str):
        return [v]
    if isinstance(v, list):
        return [str(x) for x in v if x]
    return []

def _path_glob_match(rel_path, pattern):
    if not pattern:
        return False
    if pattern.endswith("/") and "*" not in pattern:
        return rel_path.startswith(pattern) or rel_path == pattern.rstrip("/")
    out = ["^"]
    i = 0; n = len(pattern)
    while i < n:
        c = pattern[i]
        if c == "*":
            if i + 1 < n and pattern[i + 1] == "*":
                left_slash = i > 0 and pattern[i - 1] == "/"
                right_slash = i + 2 < n and pattern[i + 2] == "/"
                if left_slash and right_slash:
                    out[-1] = "(?:/.*)?/"
                    i += 3; continue
                if right_slash:
                    out.append("(?:.*/)?"); i += 3; continue
                if left_slash:
                    out[-1] = "(?:/.*)?"; i += 2; continue
                out.append(".*"); i += 2; continue
            out.append("[^/]*"); i += 1; continue
        if c in r".+?^$()[]{}|\\":
            out.append("\\" + c)
        else:
            out.append(c)
        i += 1
    out.append("$")
    try:
        return bool(re.match("".join(out), rel_path))
    except re.error:
        return False

def _code_shape_match(content, shape):
    if not isinstance(shape, dict) or shape.get("kind", "regex_in_content") != "regex_in_content":
        return False
    must_match = _coerce_list(shape.get("must_match"))
    must_not = _coerce_list(shape.get("must_not_match"))
    if not must_match:
        return False
    if not any_match(must_match, content):
        return False
    if must_not and any_match(must_not, content):
        return False
    return True

def _trigger_fires(rule, rel_path, content):
    triggers = rule.get("triggers")
    if not isinstance(triggers, dict):
        return False, True
    globs = _coerce_list(triggers.get("path_glob"))
    shapes = triggers.get("code_shape") or []
    if not isinstance(shapes, list):
        shapes = []
    if not globs and not shapes:
        return True, False
    if globs and not any(_path_glob_match(rel_path, g) for g in globs):
        return True, False
    if shapes and not any(_code_shape_match(content, s) for s in shapes):
        return True, False
    return True, True

errors = []
warns = []
infos = []

for r in rules:
    check = r.get("check", "")
    conf = r.get("confidence", 1.0)
    matched = False

    has_triggers, trigger_fires = _trigger_fires(r, rel_path, content)
    if has_triggers and not trigger_fires:
        continue

    if check == "file_placement":
        dirs = r.get("allowed_dirs", [])
        if dirs and not any(fp.startswith(d) or rel_path.startswith(d) for d in dirs):
            matched = True
    elif check == "naming":
        pat = r.get("pattern", "")
        if pat and not re.search(pat, filename):
            matched = True
    elif check == "forbidden_import":
        applies_to = r.get("applies_to", "")
        if applies_to and rel_path.startswith(applies_to) and content:
            matched = any_match(r.get("forbidden_patterns", []), content)
    elif check == "required_pattern":
        file_pat = r.get("file_pattern", "")
        if file_pat and fnmatch.fnmatch(filename, file_pat) and content:
            required = r.get("required_in_content", [])
            if required and not any(req in content for req in required):
                matched = True
    elif check == "forbidden_content":
        applies_to = r.get("applies_to", "")
        if (not applies_to or rel_path.startswith(applies_to)) and content:
            matched = any_match(r.get("forbidden_patterns", []), content)
    elif check == "architectural_constraint":
        file_pat = r.get("file_pattern", "")
        if file_pat and fnmatch.fnmatch(filename, file_pat) and content:
            matched = any_match(r.get("forbidden_patterns", []), content)
    elif check == "file_naming":
        applies_to = r.get("applies_to", "")
        file_pat = r.get("file_pattern", "")
        if applies_to and fnmatch.fnmatch(rel_path, applies_to) and file_pat:
            try:
                if not re.match(file_pat, filename):
                    matched = True
            except re.error:
                pass
    elif not check and has_triggers and trigger_fires:
        matched = True

    if not matched:
        continue

    sc = r.get("severity_class", "")
    if sc in SEVERITY_BLOCKING or (not sc and r.get("severity") == "error"):
        errors.append((r, conf))
    elif sc in SEVERITY_INFO:
        infos.append((r, conf))
    else:
        warns.append((r, conf))

def render_fired(rule, conf, prefix_label):
    rid = rule.get("id", "unknown")
    sev = severity_label(rule)
    desc = rule.get("description", "")
    confidence_tag = f" (confidence {int(conf * 100)}%)" if conf < 1.0 else ""
    print(f"[Archie] {prefix_label}{confidence_tag} {rid} [{sev}]: {desc}")
    _emit_labeled_block("WHY", rule.get("why", "") or rule.get("rationale", ""), "", print)
    _emit_labeled_block("FORCED BY", rule.get("forced_by", ""), "", print)
    _emit_labeled_block("ENABLES", rule.get("enables", ""), "", print)
    _emit_labeled_block("DO INSTEAD", rule.get("alternative", ""), "", print)
    _emit_labeled_block("EXAMPLE", rule.get("example", ""), "", print)

for r, c in infos[:5]:
    render_fired(r, c, "INFO")
for r, c in warns[:5]:
    render_fired(r, c, "WARN")
for r, c in errors:
    render_fired(r, c, "BLOCKED")
if errors:
    sys.exit(2)
PYEOF
