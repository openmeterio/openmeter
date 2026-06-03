"""ClaudeConnector — installs Archie for Claude Code.

Writes:
  .claude/commands/archie-*.md        — slash command shims
  .claude/hooks/*.sh                  — hook scripts (copies from archie/assets/hook_scripts/)
  .claude/settings.local.json         — hook bindings + permissions

See docs/plans/2026-05-18-multi-agent-connector-architecture.md §9.1.
"""
from __future__ import annotations

import json
import os
import shutil
import stat
from pathlib import Path

from ..manifest import CommandDef, ConfigPatch, HookDef
from .base import Connector


ASSETS_ROOT = Path(os.environ.get("ARCHIE_ASSETS_ROOT") or (Path(__file__).resolve().parent.parent / "assets"))
HOOK_SCRIPTS_DIR = ASSETS_ROOT / "hook_scripts"


_EVENT_NAME_CLAUDE = {
    "pre-tool-use": "PreToolUse",
    "post-tool-use": "PostToolUse",
    "user-prompt-submit": "UserPromptSubmit",
    "stop": "Stop",
}

# Permissions allowed by default so Archie commands run prompt-free in Claude
# Code. Mirrors archie/standalone/install_hooks.py::ARCHIE_PERMISSIONS — keep
# the two lists in sync if either changes (install_hooks.py is the legacy
# backwards-compat entry point; this list is what the connector loop writes).
ARCHIE_PERMISSIONS = [
    "Bash(python3 .archie/*.py *)", "Bash(python3 .archie/*.py)", "Bash(python3 -c *)",
    "Bash(git *)", "Bash(test *)", "Bash(cp *)", "Bash(ls *)", "Bash(wc *)",
    "Bash(cat *)", "Bash(echo *)", "Bash(for *)", "Bash(mkdir *)", "Bash(date *)",
    "Bash(sort *)", "Bash(head *)",
    "Bash(rm -f .archie/tmp/archie_*)", "Bash(rm -f .archie/health.json)",
    "Read(.archie/*)", "Read(.archie/**)",
    "Write(.archie/*)", "Write(.archie/**)",
    "Edit(.archie/*)", "Edit(.archie/**)",
    "Read(**)",
    "Write(**/CLAUDE.md)", "Edit(**/CLAUDE.md)",
    "Agent(*)",
]


# Where the rendered workflow tree lands for Claude. Matches {{WORKFLOW_ROOT}}.
CLAUDE_WORKFLOW_ROOT = ".archie/workflow/claude"


# Render map for the templated canonical workflow. Token values + native block
# partials. See HANDOFF_codex_command_parity.md §4 for the locked slot set.
_CLAUDE_RENDER_TOKENS = {
    "ANALYSIS_MODEL": "Sonnet",
    "REASONING_MODEL": "Opus",
    "VERIFY_MODEL": "Haiku",
    "WORKFLOW_ROOT": CLAUDE_WORKFLOW_ROOT,
    "COMMAND_PREFIX": "/",
}

# Block partials carry only the CLI-specific *mechanism*. The worker model
# and the task/question text stay inline in the canonical workflow (as
# {{ANALYSIS_MODEL}} / {{REASONING_MODEL}} tokens and verbatim prose) so the
# same partial is reusable at every dispatch site regardless of tier.
_CLAUDE_RENDER_PARTIALS = {
    # How to spawn N parallel analysis workers.
    "dispatch_parallel": (
        "Spawn each sub-agent with its own Agent tool call, ALL Agent tool "
        "calls emitted in a single message so they run in parallel. Pass each "
        "sub-agent's prompt text as the `prompt` parameter and set `model` to "
        "the lowercased model name given for that sub-agent."
    ),
    # How to spawn one worker.
    "dispatch_single": (
        "Spawn the sub-agent with a single Agent tool call. Pass its prompt "
        "text as the `prompt` parameter and set `model` to the lowercased "
        "model name given for it."
    ),
    # How to fan out one worker per selected workspace (monorepo SCOPE=per-package
    # / hybrid parallel mode).
    "dispatch_workspace_parallel": (
        "Spawn one Agent tool call per selected workspace, ALL Agent calls "
        "emitted in a single message so they run in parallel. Pass each "
        "workspace agent its workspace path; the agent sets `PROJECT_ROOT` to "
        "that path and runs the requested deep-scan steps for that workspace "
        "only, writing only that workspace's Archie artifacts. Wait for all "
        "workspace agents to finish before continuing."
    ),
    # How a spawned worker must write its output file.
    "output_contract": (
        "1. Use the Write tool to save your COMPLETE output to the file path named above.\n"
        "2. Write the raw output verbatim — the merge/finalize step handles JSON envelopes.\n"
        "3. After Writing, reply with exactly: \"Wrote <that file path>\".\n"
        "4. Do NOT print the output in your response body."
    ),
    # How to ask the user an interactive question. The question text, header,
    # and options stay inline in the canonical workflow — only the asking
    # mechanism is slotted. Carries the AskUserQuestion-specific mechanics
    # (multiSelect, 4-option cap) so the canonical workflow can stay neutral:
    # it just says "allow multiple selections" / lists N options, and this
    # partial maps that onto the tool's real constraints.
    "ask_user": (
        "Use the `AskUserQuestion` tool. Pass the question text, header, and "
        "options exactly as specified. When the step says to allow multiple "
        "selections, set `multiSelect: true`. `AskUserQuestion` accepts at "
        "most 4 options: if the step lists more than 4 choices, do not "
        "truncate them — instead ask in a plain follow-up message and accept "
        "a comma-separated reply (e.g. `1,3,5` or `all`)"
    ),
}


class ClaudeConnector(Connector):
    name = "claude"
    capabilities = frozenset({
        "commands",
        "hooks:pre-tool-use",
        "hooks:post-tool-use",
        "hooks:user-prompt-submit",
        "hooks:stop",
        "hooks:pre-commit",
        "parallel-agents",
    })

    render_tokens = _CLAUDE_RENDER_TOKENS
    render_partials = _CLAUDE_RENDER_PARTIALS

    def home_dir(self) -> Path:
        return Path.home() / ".claude"

    def install_command(self, project_root: Path, cmd: CommandDef) -> None:
        # Claude's slash commands at .claude/commands/<name>.md are thin shims
        # pointing at the rendered canonical workflow body under
        # .archie/workflow/claude/<command>/SKILL.md. cmd.body_path is the
        # command sub-path relative to the per-CLI workflow root.
        dest = project_root / ".claude" / "commands" / f"{cmd.name}.md"
        dest.parent.mkdir(parents=True, exist_ok=True)
        body_path = f"{CLAUDE_WORKFLOW_ROOT}/{cmd.body_path}"
        dest.write_text(
            f"---\ndescription: {cmd.description}\n---\n\n"
            f"Read `{body_path}` in full and execute the instructions as written.\n"
        )

    def install_hook(self, project_root: Path, hook: HookDef) -> None:
        script_name = Path(hook.script_path).name
        src = HOOK_SCRIPTS_DIR / script_name
        if not src.exists():
            raise FileNotFoundError(
                f"Canonical hook script missing: {src}. "
                f"Stage 1 should have extracted it from install_hooks.py."
            )
        # Copy script to .claude/hooks/ (Claude's expected location).
        hook_dir = project_root / ".claude" / "hooks"
        hook_dir.mkdir(parents=True, exist_ok=True)
        dest = hook_dir / script_name
        shutil.copyfile(src, dest)
        dest.chmod(dest.stat().st_mode | stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH)

        # Register in .claude/settings.local.json
        event_key = _EVENT_NAME_CLAUDE.get(hook.event)
        if event_key is None:
            return  # pre-commit handled separately by the install loop

        settings_path = project_root / ".claude" / "settings.local.json"
        settings: dict = {}
        if settings_path.exists():
            try:
                settings = json.loads(settings_path.read_text())
            except (json.JSONDecodeError, OSError):
                settings = {}

        hooks_root = settings.setdefault("hooks", {})
        bucket = hooks_root.setdefault(event_key, [])

        relative_cmd = f".claude/hooks/{script_name}"
        if not _hook_entry_present(bucket, hook.tool_match, relative_cmd):
            bucket.append({
                "matcher": hook.tool_match or "*",
                "hooks": [{"type": "command", "command": relative_cmd}],
            })

        settings_path.parent.mkdir(parents=True, exist_ok=True)
        settings_path.write_text(json.dumps(settings, indent=2) + "\n")

    def finalize(self, project_root: Path) -> None:
        # Merge Archie's default permissions into .claude/settings.local.json so
        # /archie-deep-scan and other Archie commands run without per-call user
        # prompts. Preserves any existing permissions the user has set (union,
        # not replace). Same set the legacy install_hooks.py heredoc wrote on `main`.
        settings_path = project_root / ".claude" / "settings.local.json"
        settings: dict = {}
        if settings_path.exists():
            try:
                settings = json.loads(settings_path.read_text())
            except (json.JSONDecodeError, OSError):
                settings = {}

        perms = settings.setdefault("permissions", {})
        allow = set(perms.get("allow", []))
        for p in ARCHIE_PERMISSIONS:
            allow.add(p)
        perms["allow"] = sorted(allow)

        settings_path.parent.mkdir(parents=True, exist_ok=True)
        settings_path.write_text(json.dumps(settings, indent=2) + "\n")


def _hook_entry_present(bucket: list, matcher: str | None, command: str) -> bool:
    needle = matcher or "*"
    for entry in bucket:
        if entry.get("matcher") != needle:
            continue
        for h in entry.get("hooks", []):
            if h.get("command") == command:
                return True
    return False
