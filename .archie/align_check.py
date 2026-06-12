"""Phase 3 alignment classifier — semantic comparison at plan + commit time.

Compares the agent's intent (plan text or staged diff) against each
architectural rule's semantic content (`description` + `why` + `example`)
via a single Claude CLI call. Returns structured per-rule diagnostics.

Usage (called from hooks, never directly):
    python3 align_check.py plan <project_root>     # stdin: PostToolUse JSON
    python3 align_check.py commit <project_root>   # reads git diff --cached

Exit codes:
    0  — no blocking violations OR claude CLI unavailable (advisory only)
    2  — at least one decision_violation or pitfall_triggered

Stdout: structured diagnostic block. Hook stdout is surfaced to Claude
Code as the tool result, so the agent sees the verdict and can revise.

Stderr: progress / fallback notes.

Architectural rules are pulled from `.archie/rule_index.json`'s
`for_classifier` list (Phase 2). Falls back to filtering all rules by
`severity_class != mechanical_violation` if the index is missing.

The classifier model is Haiku for cost + speed (per the richer-rules
plan). If the user's claude CLI is configured for a different default
the call still works — we just ask for JSON output and parse it.
"""

from __future__ import annotations

import json
import os
import subprocess
import sys
from pathlib import Path
from typing import Any

sys.path.insert(0, str(Path(__file__).resolve().parent))
from _common import safe_read_text  # noqa: E402

CLASSIFIER_MODEL = "claude-haiku-4-5-20251001"
CLASSIFIER_TIMEOUT_SECS = 90


# ---------------------------------------------------------------------------
# Rule loading
# ---------------------------------------------------------------------------


def _load_json(path: Path) -> Any:
    try:
        return json.loads(safe_read_text(path))
    except (OSError, json.JSONDecodeError, ValueError):
        return None


def _load_architectural_rules(archie_dir: Path) -> list[dict[str, Any]]:
    """Return rules eligible for plan/commit-time classification.

    Prefers the precomputed `for_classifier` list from rule_index.json.
    Falls back to filtering rules.json + platform_rules.json by
    severity_class (excluding mechanical_violation).
    """
    rules: dict[str, dict[str, Any]] = {}
    for fname in ("rules.json", "platform_rules.json"):
        data = _load_json(archie_dir / fname)
        if data is None:
            continue
        items = data if isinstance(data, list) else data.get("rules", [])
        if not isinstance(items, list):
            continue
        for r in items:
            if isinstance(r, dict) and r.get("id"):
                rules[r["id"]] = r

    index = _load_json(archie_dir / "rule_index.json")
    if isinstance(index, dict) and isinstance(index.get("for_classifier"), list):
        ids = index["for_classifier"]
        return [rules[i] for i in ids if i in rules]

    # Fallback: filter by severity_class. Old-shape rules with rationale
    # text are architectural enough to classify even if they also have a
    # mechanical check (the rationale is what the classifier reasons about).
    out: list[dict[str, Any]] = []
    for r in rules.values():
        sc = r.get("severity_class", "")
        if sc:
            if sc != "mechanical_violation":
                out.append(r)
        elif r.get("rationale") or r.get("why"):
            out.append(r)
    return out


# ---------------------------------------------------------------------------
# Input parsing — plan text from PostToolUse ExitPlanMode, or git diff
# ---------------------------------------------------------------------------


def _read_plan_from_stdin() -> str:
    """Pull the plan text out of a PostToolUse ExitPlanMode tool-call envelope.

    The hook receives the tool input/output JSON on stdin. ExitPlanMode's
    tool_input has a `plan` field (the markdown plan the agent submitted).
    """
    try:
        raw = sys.stdin.read()
    except Exception:
        return ""
    if not raw:
        return ""
    try:
        data = json.loads(raw)
    except json.JSONDecodeError:
        # Not JSON — assume it's raw plan text
        return raw

    # Try the various shapes a PostToolUse envelope might take
    if isinstance(data, dict):
        ti = data.get("tool_input", {})
        if isinstance(ti, dict) and ti.get("plan"):
            return str(ti["plan"])
        tr = data.get("tool_response", {})
        if isinstance(tr, dict):
            if tr.get("plan"):
                return str(tr["plan"])
            if tr.get("text"):
                return str(tr["text"])
        # PreToolUse-style (Bash on git commit)
        if isinstance(ti, dict) and ti.get("command"):
            return str(ti["command"])
    return raw


def _read_staged_diff(project_root: Path) -> str:
    """Read git diff --cached. Returns empty string if git not available."""
    try:
        result = subprocess.run(
            ["git", "-C", str(project_root), "diff", "--cached"],
            capture_output=True, text=True, timeout=15,
        )
        if result.returncode == 0:
            return result.stdout
    except (FileNotFoundError, subprocess.TimeoutExpired):
        pass
    return ""


# ---------------------------------------------------------------------------
# Classifier prompt + invocation
# ---------------------------------------------------------------------------


def _build_classifier_prompt(input_text: str, kind: str, rules: list[dict[str, Any]]) -> str:
    """Build the prompt for the Haiku classifier.

    `kind` is "plan" or "diff" — frames the artifact for the model.
    `rules` is the architectural rule set with full semantic content.
    """
    rule_blocks = []
    for r in rules:
        rid = r.get("id", "?")
        sc = r.get("severity_class", "") or ("error-legacy" if r.get("severity") == "error" else "warn-legacy")
        desc = r.get("description", "")
        why = r.get("why", "") or r.get("rationale", "")
        example = r.get("example", "")
        block = f"RULE {rid} [{sc}]\nDescription: {desc}\nWhy: {why}"
        if example:
            block += f"\nExample of the canonical shape: {example}"
        rule_blocks.append(block)

    if kind == "plan":
        artifact_label = "PLAN (the agent intends to execute these steps)"
    else:
        artifact_label = "STAGED DIFF (the agent is about to commit these changes)"

    return f"""You are an architectural reviewer. The codebase has the following architectural rules. Your job is to compare the {artifact_label} against EACH rule and decide whether the rule is violated, respected, or not relevant.

Reason about INTENT, not keywords. A rule may be violated even if the artifact uses different wording — what matters is whether the change crosses the rule's architectural boundary.

For each rule, return one entry in `diagnostics` with these fields:
  - `rule_id`: the rule's id
  - `severity_class`: copy from the rule
  - `verdict`: one of "violates" / "respects" / "not_relevant"
  - `evidence`: short quote / step reference that grounded your verdict (only for "violates")
  - `suggested_fix`: concrete action the agent should take (only for "violates")

Output ONLY valid JSON of the form:
{{
  "diagnostics": [...],
  "highest_severity": "decision_violation|pitfall_triggered|tradeoff_undermined|pattern_divergence|none"
}}

ARCHITECTURAL RULES:

{chr(10).join(rule_blocks)}

{artifact_label}:

{input_text}
"""


def _invoke_claude_classifier(prompt: str) -> dict[str, Any] | None:
    """Run claude CLI with the prompt. Returns parsed JSON verdict or None on any failure."""
    try:
        result = subprocess.run(
            [
                "claude",
                "-p", prompt,
                "--output-format", "json",
                "--model", CLASSIFIER_MODEL,
            ],
            capture_output=True,
            text=True,
            timeout=CLASSIFIER_TIMEOUT_SECS,
        )
    except (FileNotFoundError, subprocess.TimeoutExpired):
        return None
    if result.returncode != 0:
        print(f"[Archie] classifier exited {result.returncode}: {result.stderr[:300]}", file=sys.stderr)
        return None

    # claude --output-format json returns an envelope with `result` field
    # holding the assistant's stringified output. Parse the envelope first,
    # then parse the inner JSON.
    try:
        envelope = json.loads(result.stdout)
    except json.JSONDecodeError:
        # Sometimes the CLI just prints raw text — try parsing directly
        try:
            return json.loads(result.stdout)
        except json.JSONDecodeError:
            return None

    inner = envelope.get("result") if isinstance(envelope, dict) else None
    if isinstance(inner, str):
        # Strip any code fences the model may have wrapped around the JSON
        s = inner.strip()
        if s.startswith("```"):
            s = s.split("\n", 1)[1] if "\n" in s else s
            if s.endswith("```"):
                s = s.rsplit("```", 1)[0]
        try:
            return json.loads(s)
        except json.JSONDecodeError:
            return None
    return None


# ---------------------------------------------------------------------------
# Output rendering
# ---------------------------------------------------------------------------


SEVERITY_BLOCKING = {"decision_violation", "pitfall_triggered"}


def _render_diagnostics(verdict: dict[str, Any]) -> bool:
    """Print human-readable diagnostics. Return True if any blocking violation."""
    diags = verdict.get("diagnostics") or []
    if not isinstance(diags, list):
        return False
    blocking = False
    for d in diags:
        if not isinstance(d, dict):
            continue
        if d.get("verdict") != "violates":
            continue
        rid = d.get("rule_id", "?")
        sc = d.get("severity_class", "")
        evidence = d.get("evidence", "")
        fix = d.get("suggested_fix", "")
        if sc in SEVERITY_BLOCKING:
            label = "BLOCKED"
            blocking = True
        elif sc == "tradeoff_undermined":
            label = "WARN"
        elif sc == "pattern_divergence":
            label = "INFO"
        else:
            label = sc.upper() or "INFO"
        print(f"[Archie {label}] {rid} [{sc}]")
        if evidence:
            print(f"  Evidence: {evidence}")
        if fix:
            print(f"  Suggested fix: {fix}")
    return blocking


def _render_advisory(rules: list[dict[str, Any]]) -> None:
    """Fallback when claude CLI is unavailable: print the rule set as guidance."""
    print("[Archie] Classifier unavailable (claude CLI not on PATH or failed). "
          "Architectural rules below — review the plan/diff against each, "
          "block any decision_violation or pitfall_triggered before proceeding.")
    print("")
    for r in rules:
        rid = r.get("id", "?")
        sc = r.get("severity_class", "") or r.get("severity", "warn")
        desc = r.get("description", "")
        why = r.get("why", "") or r.get("rationale", "")
        print(f"  RULE {rid} [{sc}]: {desc}")
        if why:
            for ln in why.splitlines() or [why]:
                print(f"    WHY: {ln}")


# ---------------------------------------------------------------------------
# Subcommands
# ---------------------------------------------------------------------------


def cmd_plan(project_root: Path) -> int:
    archie_dir = project_root / ".archie"
    rules = _load_architectural_rules(archie_dir)
    if not rules:
        return 0  # nothing to compare against
    plan_text = _read_plan_from_stdin().strip()
    if not plan_text:
        return 0

    prompt = _build_classifier_prompt(plan_text, "plan", rules)
    verdict = _invoke_claude_classifier(prompt)
    if verdict is None:
        _render_advisory(rules)
        return 0
    blocking = _render_diagnostics(verdict)
    return 2 if blocking else 0


def cmd_commit(project_root: Path) -> int:
    archie_dir = project_root / ".archie"
    rules = _load_architectural_rules(archie_dir)
    if not rules:
        return 0
    diff = _read_staged_diff(project_root).strip()
    if not diff:
        return 0

    prompt = _build_classifier_prompt(diff, "diff", rules)
    verdict = _invoke_claude_classifier(prompt)
    if verdict is None:
        _render_advisory(rules)
        return 0
    blocking = _render_diagnostics(verdict)
    return 2 if blocking else 0


def main() -> int:
    if len(sys.argv) < 3:
        print("Usage:", file=sys.stderr)
        print("  python3 align_check.py plan <project_root>", file=sys.stderr)
        print("  python3 align_check.py commit <project_root>", file=sys.stderr)
        return 1
    subcmd = sys.argv[1]
    project_root = Path(sys.argv[2]).resolve()
    if not project_root.is_dir():
        print(f"Error: {project_root} is not a directory", file=sys.stderr)
        return 1
    # Disable via env var — useful for users who don't want the latency
    if os.environ.get("ARCHIE_DISABLE_ALIGN_CHECK") == "1":
        return 0
    if subcmd == "plan":
        return cmd_plan(project_root)
    if subcmd == "commit":
        return cmd_commit(project_root)
    print(f"Unknown subcommand: {subcmd}", file=sys.stderr)
    return 1


if __name__ == "__main__":
    sys.exit(main())
