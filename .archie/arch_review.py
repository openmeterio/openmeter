#!/usr/bin/env python3
"""Archie architectural review — gathers context for post-plan and pre-commit hooks.

Called by hook scripts to assemble blueprint context + plan/diff for Claude
to spawn a reviewer subagent.

Subcommands:
  plan  <project_root>              — gather context for plan review
  diff  <project_root>              — gather context for pre-commit review

Outputs structured review prompt to stdout that the hook passes to Claude.

Zero dependencies beyond Python 3.9+ stdlib.
"""
from __future__ import annotations

import json
import subprocess
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from _common import _load_json, safe_read_text  # noqa: E402

# Comprehensive depth lifts content-comprehensiveness render slices. Set from
# __main__ when "--comprehensive" is present in argv. Render functions read this
# module global via _cap().
_COMPREHENSIVE = False


def _cap(seq, n):
    """Cap seq to n items unless comprehensive depth is active."""
    return seq if _COMPREHENSIVE else seq[:n]


def _get_blueprint_context(root: Path) -> str:
    """Extract key architectural constraints from blueprint."""
    bp = _load_json(root / ".archie" / "blueprint.json")
    if not bp:
        return ""

    parts = []

    # Key decisions
    decisions = bp.get("decisions", {})
    key_decs = decisions.get("key_decisions", [])
    if key_decs:
        parts.append("## Key Architectural Decisions")
        for d in _cap(key_decs, 8):
            if isinstance(d, dict):
                title = d.get("title", "")
                chosen = d.get("chosen", "")
                parts.append(f"- **{title}**: {chosen}")
            elif isinstance(d, str):
                parts.append(f"- {d}")
            else:
                print(f"Warning: unexpected {type(d).__name__} in key_decisions, coercing", file=sys.stderr)
                parts.append(f"- {str(d)}")

    # Trade-offs with violation signals
    trade_offs = decisions.get("trade_offs", [])
    if trade_offs:
        parts.append("\n## Trade-offs (violation signals)")
        for t in _cap(trade_offs, 5):
            if isinstance(t, dict):
                accept = t.get("accept", "")
                signals = t.get("violation_signals", [])
                if signals:
                    parts.append(f"- {accept} — signals: {', '.join(_cap(signals, 5))}")
            elif isinstance(t, str):
                parts.append(f"- {t}")
            else:
                print(f"Warning: unexpected {type(t).__name__} in trade_offs, coercing", file=sys.stderr)
                parts.append(f"- {str(t)}")

    # Decision chain
    chain = decisions.get("decision_chain", {})
    if chain:
        root_constraint = chain.get("root", "")
        if root_constraint:
            parts.append(f"\n## Root Constraint: {root_constraint}")

    # Development rules
    dev_rules = bp.get("development_rules", [])
    if dev_rules:
        parts.append("\n## Development Rules")
        for r in _cap(dev_rules, 10):
            if isinstance(r, dict):
                parts.append(f"- {r.get('rule', '')}")
            elif isinstance(r, str):
                parts.append(f"- {r}")
            else:
                print(f"Warning: unexpected {type(r).__name__} in development_rules, coercing", file=sys.stderr)
                parts.append(f"- {str(r)}")

    # Component boundaries
    comps = bp.get("components", {})
    comp_list = comps.get("components", []) if isinstance(comps, dict) else []
    if comp_list:
        parts.append("\n## Component Boundaries")
        for c in _cap(comp_list, 10):
            if isinstance(c, dict):
                name = c.get("name", "")
                loc = c.get("location", "")
                deps = c.get("depends_on", [])
                parts.append(f"- **{name}** ({loc}) depends on: {', '.join(deps) if deps else 'none'}")
            elif isinstance(c, str):
                parts.append(f"- {c}")
            else:
                print(f"Warning: unexpected {type(c).__name__} in components, coercing", file=sys.stderr)
                parts.append(f"- {str(c)}")

    return "\n".join(parts)


def _get_rules_summary(root: Path) -> str:
    """Extract architectural rules with semantic content for the AI reviewer.

    Reads new-shape fields (`severity_class`, `why`, `example`) when present,
    falls back to legacy `severity` + `rationale` so old rules.json still
    produces a useful summary.
    """
    rules_data = _load_json(root / ".archie" / "rules.json")
    rules = rules_data.get("rules", []) if isinstance(rules_data, dict) else []
    if not rules:
        return ""

    parts = ["## Architectural Rules"]
    parts.append(
        "Evaluate the change against each rule's *intent* (the WHY block). "
        "Block any decision_violation or pitfall_triggered. Warn on tradeoff_undermined. "
        "Pattern_divergence is informational unless the divergence is structural.\n"
    )
    for r in rules:
        if not isinstance(r, dict):
            continue
        rid = r.get("id", "")
        severity_class = r.get("severity_class", "")
        sev = severity_class or r.get("severity", "warn")
        desc = r.get("description", "")
        why = r.get("why", "") or r.get("rationale", "")
        example = r.get("example", "")
        applies = r.get("applies_to", "") or r.get("file_pattern", "")
        scope = f" (scope: `{applies}`)" if applies else ""
        header = f"- **[{sev}]** {rid}: {desc}{scope}" if rid else f"- **[{sev}]** {desc}{scope}"
        parts.append(header)
        if why:
            parts.append(f"  *Why:* {why}")
        if example:
            parts.append(f"  *Example:* {example}")

    return "\n".join(parts)


def _get_folder_context(root: Path, changed_files: list[str]) -> str:
    """Get relevant folder CLAUDE.md content for changed files."""
    seen_folders = set()
    parts = []

    for fp in changed_files[:20]:
        folder = str(Path(fp).parent)
        if folder in seen_folders or folder == ".":
            continue
        seen_folders.add(folder)

        claude_md = root / folder / "CLAUDE.md"
        if claude_md.exists():
            content = safe_read_text(claude_md, root)
            # Truncate to keep context manageable
            if len(content) > 1500:
                content = content[:1500] + "\n... (truncated)"
            parts.append(f"### {folder}/CLAUDE.md\n{content}")

    return "\n\n".join(parts[:5])


def cmd_plan(root: Path):
    """Gather context for plan review. Output review instructions to stdout."""
    bp_context = _get_blueprint_context(root)
    rules = _get_rules_summary(root)

    if not bp_context:
        # No blueprint = no architectural context to review against
        sys.exit(0)

    print("[Archie] Architectural plan review required.")
    print("")
    print("Before executing this plan, spawn a reviewer subagent (model: sonnet) to check it against the architecture.")
    print("Feed the reviewer the plan content (read it from the plan file) plus the following architectural context:")
    print("")
    print("---BEGIN ARCHITECTURAL CONTEXT---")
    print(bp_context)
    if rules:
        print("")
        print(rules)
    print("---END ARCHITECTURAL CONTEXT---")
    print("")
    print("The reviewer subagent should check:")
    print("1. Does the plan violate any key architectural decisions?")
    print("2. Does it respect component boundaries and dependency directions?")
    print("3. Does it trigger any trade-off violation signals?")
    print("4. Does it follow the development rules?")
    print("5. Does the plan violate the INTENT of any architectural rule? Read each rule's rationale — evaluate whether the plan undermines the reasoning, not just whether it matches a regex pattern.")
    print("")
    print("Return JSON: {\"approved\": true/false, \"violations\": [{\"decision\": \"...\", \"concern\": \"...\", \"suggestion\": \"...\"}]}")
    print("")
    print("If violations are found, revise the plan before proceeding. If approved, continue with execution.")


def cmd_diff(root: Path):
    """Gather context for pre-commit review. Output review instructions to stdout."""
    # Get staged diff
    try:
        diff_result = subprocess.run(
            ["git", "-C", str(root), "diff", "--cached", "--stat"],
            capture_output=True, text=True, timeout=10,
        )
        diff_stat = diff_result.stdout.strip()
    except (subprocess.TimeoutExpired, FileNotFoundError):
        diff_stat = ""

    if not diff_stat:
        # Nothing staged = nothing to review
        sys.exit(0)

    # Get changed file list
    try:
        files_result = subprocess.run(
            ["git", "-C", str(root), "diff", "--cached", "--name-only"],
            capture_output=True, text=True, timeout=10,
        )
        changed_files = [f for f in files_result.stdout.strip().split("\n") if f]
    except (subprocess.TimeoutExpired, FileNotFoundError):
        changed_files = []

    # Get the actual diff (truncated for context window)
    try:
        full_diff = subprocess.run(
            ["git", "-C", str(root), "diff", "--cached"],
            capture_output=True, text=True, timeout=10,
        )
        diff_content = full_diff.stdout
        if len(diff_content) > 15000:
            diff_content = diff_content[:15000] + "\n... (truncated, full diff too large)"
    except (subprocess.TimeoutExpired, FileNotFoundError):
        diff_content = ""

    bp_context = _get_blueprint_context(root)
    rules = _get_rules_summary(root)
    folder_context = _get_folder_context(root, changed_files)

    if not bp_context:
        sys.exit(0)

    print("[Archie] Pre-commit architectural review required.")
    print("")
    print(f"Staged changes: {diff_stat}")
    print("")
    print("Before committing, spawn a reviewer subagent (model: sonnet) to check the staged changes against the architecture.")
    print("Feed the reviewer the diff below plus the architectural context:")
    print("")
    print("---BEGIN STAGED DIFF---")
    print(diff_content)
    print("---END STAGED DIFF---")
    print("")
    if folder_context:
        print("---BEGIN FOLDER PATTERNS---")
        print(folder_context)
        print("---END FOLDER PATTERNS---")
        print("")
    print("---BEGIN ARCHITECTURAL CONTEXT---")
    print(bp_context)
    if rules:
        print("")
        print(rules)
    print("---END ARCHITECTURAL CONTEXT---")
    print("")
    print("The reviewer subagent should check:")
    print("1. Do the changes violate any key architectural decisions?")
    print("2. Do files follow the patterns described in their folder's CLAUDE.md?")
    print("3. Are component boundaries and dependency directions respected?")
    print("4. Are there any trade-off violation signals in the new code?")
    print("5. Do the changes follow the development rules?")
    print("6. Do the changes violate the INTENT of any architectural rule? Read each rule's rationale — evaluate whether the change undermines the reasoning, not just whether it matches a regex pattern.")
    print("")
    print("Return JSON: {\"approved\": true/false, \"violations\": [{\"file\": \"...\", \"rule\": \"...\", \"concern\": \"...\", \"suggestion\": \"...\"}]}")
    print("")
    print("If violations are found, fix them before committing. If approved, proceed with the commit.")


if __name__ == "__main__":
    if len(sys.argv) < 3:
        print("Usage:", file=sys.stderr)
        print("  python3 arch_review.py plan <project_root>", file=sys.stderr)
        print("  python3 arch_review.py diff <project_root>", file=sys.stderr)
        sys.exit(1)

    # Module-level: assign directly (no `global` needed at module scope).
    # Render functions read this module global via _cap().
    _COMPREHENSIVE = "--comprehensive" in sys.argv

    subcmd = sys.argv[1]
    root = Path(sys.argv[2]).resolve()

    if subcmd == "plan":
        cmd_plan(root)
    elif subcmd == "diff":
        cmd_diff(root)
    else:
        print(f"Unknown subcommand: {subcmd}", file=sys.stderr)
        sys.exit(1)
