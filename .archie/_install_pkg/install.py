"""Install loop — drives all registered connectors for a project install.

Called from npm-package/bin/archie.mjs as:
    python3 -m archie.install <project_root> [--target=auto|claude|codex|all]

Stage 2 work. See docs/plans/2026-05-18-multi-agent-connector-architecture.md §11.
"""
from __future__ import annotations

import argparse
import os
import re
import shutil
import sys
from pathlib import Path

from .connectors import ALL_CONNECTORS
from .connectors.base import Connector
from .manifest_data import COMMANDS, CONFIG_PATCHES, HOOKS


PACKAGE_ROOT = Path(__file__).resolve().parent
ASSETS_ROOT = Path(os.environ.get("ARCHIE_ASSETS_ROOT") or (PACKAGE_ROOT / "assets"))
STANDALONE_ROOT = Path(os.environ.get("ARCHIE_STANDALONE_ROOT") or (PACKAGE_ROOT / "standalone"))


def _resolve_targets(requested: list[str] | None, connectors: list[Connector]) -> list[Connector]:
    if not requested or requested == ["auto"]:
        return [c for c in connectors if c.detect()]
    if requested == ["all"]:
        return list(connectors)
    named = {c.name: c for c in connectors}
    selected = []
    for name in requested:
        if name not in named:
            print(f"Unknown target: {name}. Known: {sorted(named)}", file=sys.stderr)
            sys.exit(2)
        selected.append(named[name])
    return selected


_STANDALONE_SCRIPTS = [
    # Analysis pipeline (referenced by SKILL bodies via `python3 .archie/<name>`)
    "scanner.py", "renderer.py", "validate.py", "intent_layer.py",
    "finalize.py", "merge.py", "measure_health.py", "detect_cycles.py",
    "drift.py", "extract_output.py", "arch_review.py", "align_check.py",
    "check_rules.py", "code_shape.py", "rule_index.py", "lint_gate.py",
    "agent_cli.py", "verify_findings.py", "apply_verdicts.py", "migrate_blueprint_rules.py",
    "rule_kinds.py", "backfill_kinds.py",
    "telemetry.py", "telemetry_sync.py",
    "analytics.py", "config.py",
    "update_check.py", "upload.py", "share_setup.py", "refresh.py",
    "viewer.py", "install_hooks.py", "_common.py",
]


def _replace_tree(src: Path, dest: Path) -> None:
    if dest.exists():
        shutil.rmtree(dest)
    shutil.copytree(src, dest)


def _clean_legacy_layout(project_root: Path) -> None:
    """Remove install artifacts from superseded Archie layouts.

    Earlier versions copied the deep-scan step tree into
    .claude/skills/archie-deep-scan/ and command bodies into .archie/prompts/.
    The canonical workflow now renders per-CLI into .archie/workflow/<cli>/.
    On upgrade, drop the stale trees so a returning user is not left with a
    dead skill registration or a duplicated, out-of-date workflow body.

    Also sweeps per-CLI command shims that no longer correspond to a
    current entry in `COMMANDS` — e.g. an `archie-scan` shim from before
    that command was removed. The connectors below recreate shims for the
    current command set; anything left behind is from a previous version.
    """
    for stale_dir in (
        project_root / ".claude" / "skills" / "archie-deep-scan",
        project_root / ".claude" / "commands" / "archie-deep-scan",
        project_root / ".archie" / "prompts",
    ):
        if stale_dir.is_dir():
            shutil.rmtree(stale_dir, ignore_errors=True)
    stale_shared = project_root / ".claude" / "commands" / "_shared" / "scope_resolution.md"
    try:
        stale_shared.unlink()
    except OSError:
        pass

    current_command_names = {c.name for c in COMMANDS}

    # Sweep stale Claude command shims (.claude/commands/archie-X.md).
    claude_cmd_dir = project_root / ".claude" / "commands"
    if claude_cmd_dir.is_dir():
        for entry in claude_cmd_dir.iterdir():
            if (
                entry.is_file()
                and entry.name.startswith("archie-")
                and entry.suffix == ".md"
                and entry.stem not in current_command_names
            ):
                try:
                    entry.unlink()
                except OSError:
                    pass

    # Sweep stale Codex skill shims (.agents/skills/archie-X/).
    codex_skills_dir = project_root / ".agents" / "skills"
    if codex_skills_dir.is_dir():
        for entry in codex_skills_dir.iterdir():
            if (
                entry.is_dir()
                and entry.name.startswith("archie-")
                and entry.name not in current_command_names
            ):
                shutil.rmtree(entry, ignore_errors=True)


# ---------- workflow template renderer ----------

# Matches a block partial: {{>partial_name}}
_PARTIAL_RE = re.compile(r"\{\{>\s*([A-Za-z0-9_]+)\s*\}\}")
# Matches an inline token: {{TOKEN}}  (not a partial — partials start with '>')
_TOKEN_RE = re.compile(r"\{\{\s*([A-Za-z0-9_]+)\s*\}\}")
# Catches any leftover {{ ... }} after rendering — a missing slot.
_LEFTOVER_RE = re.compile(r"\{\{.*?\}\}")


def render_template(text: str, tokens: dict[str, str], partials: dict[str, str]) -> str:
    """Render a workflow template through a connector's render map.

    - `{{>partial}}` is replaced with the connector's native block body.
    - `{{TOKEN}}` is replaced with the connector's native token value.
    Partials are themselves rendered for nested tokens. Any unresolved `{{ }}`
    after rendering raises ValueError — a missing slot is a hard install error.
    """
    def sub_partial(m: re.Match) -> str:
        key = m.group(1)
        if key not in partials:
            raise ValueError(f"Unknown block partial: {{{{>{key}}}}}")
        # Partials may contain tokens; render those too.
        return _TOKEN_RE.sub(sub_token, partials[key])

    def sub_token(m: re.Match) -> str:
        key = m.group(1)
        if key not in tokens:
            raise ValueError(f"Unknown render token: {{{{{key}}}}}")
        return tokens[key]

    rendered = _PARTIAL_RE.sub(sub_partial, text)
    rendered = _TOKEN_RE.sub(sub_token, rendered)
    leftover = _LEFTOVER_RE.search(rendered)
    if leftover:
        raise ValueError(f"Unresolved template slot after render: {leftover.group(0)}")
    return rendered


def _render_workflow_tree(conn: Connector, project_root: Path) -> None:
    """Render archie/assets/workflow/* through `conn`'s render map.

    Output goes to <project>/.archie/workflow/<cli>/, preserving the source
    tree shape. Every text file is rendered; the rendered output is fully
    native (no `{{ }}` slots, no foreign-CLI paths).
    """
    src_root = ASSETS_ROOT / "workflow"
    if not src_root.is_dir():
        return
    dest_root = project_root / ".archie" / "workflow" / conn.name
    if dest_root.exists():
        shutil.rmtree(dest_root)
    dest_root.mkdir(parents=True, exist_ok=True)

    tokens = conn.render_tokens
    partials = conn.render_partials
    for src in sorted(src_root.rglob("*")):
        if src.name == ".DS_Store":
            continue
        rel = src.relative_to(src_root)
        dest = dest_root / rel
        if src.is_dir():
            dest.mkdir(parents=True, exist_ok=True)
            continue
        dest.parent.mkdir(parents=True, exist_ok=True)
        if src.suffix == ".md":
            rendered = render_template(src.read_text(), tokens, partials)
            dest.write_text(rendered)
        else:
            shutil.copyfile(src, dest)


def _copy_canonical_assets(project_root: Path) -> None:
    """Copies archie/assets/* and archie/standalone/*.py into <project>/.archie/."""
    dest_archie = project_root / ".archie"
    dest_archie.mkdir(parents=True, exist_ok=True)

    # Hook scripts: <project>/.archie/hooks/*.sh (canonical, all CLIs reference these)
    src_hooks = ASSETS_ROOT / "hook_scripts"
    if src_hooks.exists():
        dest_hooks = dest_archie / "hooks"
        dest_hooks.mkdir(parents=True, exist_ok=True)
        for sh in src_hooks.glob("*.sh"):
            target = dest_hooks / sh.name
            shutil.copyfile(sh, target)
            target.chmod(0o755)

    # Viewer source: shared by the local viewer sidecar and the share flow.
    src_viewer = ASSETS_ROOT / "viewer"
    if src_viewer.exists() and any(src_viewer.iterdir()):
        dest_viewer = dest_archie / "viewer"
        _replace_tree(src_viewer, dest_viewer)

    # Analysis pipeline scripts — needed by every SKILL body. Sourced from
    # archie/standalone/ (the canonical Python module location), copied to
    # <project>/.archie/<name>.py at install time.
    if STANDALONE_ROOT.is_dir():
        for name in _STANDALONE_SCRIPTS:
            src = STANDALONE_ROOT / name
            if not src.exists():
                continue
            target = dest_archie / name
            shutil.copyfile(src, target)
            target.chmod(0o755)

    # Data files consumed by scanner / hooks / viewer.
    for name in ("platform_rules.json",):
        src = ASSETS_ROOT / name
        if not src.exists():
            continue
        shutil.copyfile(src, dest_archie / name)

    # Transient run artifacts (Wave 1 outputs, intent-layer enrichments, rules
    # JSON, etc.) land under .archie/tmp/. Drop a self-ignoring .gitignore so
    # they never get committed even if the user's repo .gitignore doesn't cover
    # .archie/tmp/ explicitly.
    tmp_dir = dest_archie / "tmp"
    tmp_dir.mkdir(parents=True, exist_ok=True)
    (tmp_dir / ".gitignore").write_text("*\n")

    # Default ignore config is installed once and then left to the user.
    for src_name, dest_name in (
        ("archieignore.default", ".archieignore"),
        ("archiebulk.default", ".archiebulk"),
    ):
        src = ASSETS_ROOT / src_name
        dest = project_root / dest_name
        if src.exists() and not dest.exists():
            shutil.copyfile(src, dest)



def _install_git_pre_commit(project_root: Path) -> None:
    """Universal git pre-commit hook — runs validators before commit.

    Stage 1 stub: writes to .git/hooks/pre-commit.archie (non-clobbering;
    user wires it into their pre-commit chain). Full implementation
    deferred — see Q-A2 in the design doc.
    """
    git_hooks = project_root / ".git" / "hooks"
    if not git_hooks.exists():
        return  # not a git repo or hooks dir absent
    archie_hook = git_hooks / "pre-commit.archie"
    archie_hook.write_text(
        "#!/usr/bin/env bash\n"
        '# Archie git pre-commit gate. Add this to your pre-commit chain:\n'
        '#   echo \'bash .git/hooks/pre-commit.archie\' >> .git/hooks/pre-commit\n'
        'PROJECT_ROOT="$(git rev-parse --show-toplevel)"\n'
        'if [ -f "$PROJECT_ROOT/.archie/validate.py" ]; then\n'
        '    python3 "$PROJECT_ROOT/.archie/validate.py" all "$PROJECT_ROOT" --pre-commit\n'
        'fi\n'
    )
    archie_hook.chmod(0o755)


def install(project_root: Path, requested: list[str] | None = None) -> None:
    selected = _resolve_targets(requested, ALL_CONNECTORS)
    if not selected:
        print(
            "No supported CLI detected. Install one of:\n"
            "  Claude Code: https://docs.claude.com/claude-code\n"
            "  Codex CLI:   https://developers.openai.com/codex/cli",
            file=sys.stderr,
        )
        sys.exit(1)

    _clean_legacy_layout(project_root)
    _copy_canonical_assets(project_root)
    _install_git_pre_commit(project_root)

    for conn in selected:
        # Render the canonical workflow templates through this connector's
        # render map into .archie/workflow/<cli>/ — fully native, no slots.
        _render_workflow_tree(conn, project_root)
        for cmd in COMMANDS:
            conn.install_command(project_root, cmd)
        for hook in HOOKS:
            if conn.supports_event(hook.event):
                conn.install_hook(project_root, hook)
        if "config-patch" in conn.capabilities:
            conn.patch_config([p for p in CONFIG_PATCHES if p.cli == conn.name])
        conn.finalize(project_root)
        print(f"installed for {conn.name}", file=sys.stderr)


def _parse_targets(value: str | None) -> list[str] | None:
    if not value:
        return None
    return [t.strip() for t in value.split(",") if t.strip()]


def main() -> int:
    parser = argparse.ArgumentParser(prog="archie.install")
    parser.add_argument("project_root", type=Path)
    parser.add_argument(
        "--target",
        default="auto",
        help="auto | claude | codex | all | comma-separated subset",
    )
    args = parser.parse_args()
    install(args.project_root.resolve(), _parse_targets(args.target))
    return 0


if __name__ == "__main__":
    sys.exit(main())
