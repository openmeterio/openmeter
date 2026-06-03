"""CodexConnector — installs Archie for OpenAI Codex CLI.

Writes:
  .agents/skills/archie-*/SKILL.md    — slash-command shims (parent-walk discovered)
  .codex/hooks.json                    — hook registrations referencing .archie/hooks/*.sh
  .codex/agents/archie-analysis.toml   — project-scoped custom subagent definition
  .codex/rules/archie.rules            — execpolicy Rules pre-approving the workflow's
                                          shell command surface (per
                                          developers.openai.com/codex/rules)
  ~/.codex/config.toml *(patched)*     — idempotent merge: project_doc_max_bytes +
                                          fallback_filenames (top-level), [agents]
                                          max_threads + max_depth (set-if-absent),
                                          [projects."<abs>"] trust_level = "trusted"
                                          (set-if-absent) so the project-scoped
                                          .codex/ layer above actually loads

See docs/plans/2026-05-18-multi-agent-connector-architecture.md §9.2 and
docs/plans/HANDOFF_CODEX.md for the full implementation contract. Codex
hooks schema documented at https://developers.openai.com/codex/hooks.
"""
from __future__ import annotations

import json
import re
from pathlib import Path

from ..manifest import CommandDef, ConfigPatch, HookDef
from .base import Connector
from .claude import HOOK_SCRIPTS_DIR

# Imported lazily inside finalize so connectors stay independent of
# manifest_data at module-import time.
def _load_command_catalogue():
    from ..manifest_data import COMMAND_RULES
    from ..install import _STANDALONE_SCRIPTS
    return COMMAND_RULES, _STANDALONE_SCRIPTS


_EVENT_NAME_CODEX = {
    "pre-tool-use": "PreToolUse",
    "post-tool-use": "PostToolUse",
    "user-prompt-submit": "UserPromptSubmit",
    "stop": "Stop",
}

_MATCHER_NAME_CODEX = {
    "Edit|Write|MultiEdit": "^apply_patch$",
    "Bash": "^Bash$",
    "Glob|Grep": "^(Glob|Grep)$",
    "ExitPlanMode": "^ExitPlanMode$",
}


# Where the rendered workflow tree lands for Codex. Matches {{WORKFLOW_ROOT}}.
CODEX_WORKFLOW_ROOT = ".archie/workflow/codex"


# Render map for the templated canonical workflow. Codex token values + native
# block partials. See HANDOFF_codex_command_parity.md §4 for the locked slot
# set. Codex's strongest reasoning model is gpt-5; the analysis/verify tiers
# also run gpt-5 (Codex exposes a single frontier model).
_CODEX_RENDER_TOKENS = {
    "ANALYSIS_MODEL": "gpt-5",
    "REASONING_MODEL": "gpt-5",
    "VERIFY_MODEL": "gpt-5",
    "WORKFLOW_ROOT": CODEX_WORKFLOW_ROOT,
    "COMMAND_PREFIX": "$",
}

# Block partials carry only the CLI-specific *mechanism*. The worker model and
# the task/question text stay inline in the canonical workflow so the same
# partial is reusable at every dispatch site.
_CODEX_RENDER_PARTIALS = {
    # How to spawn N parallel analysis workers.
    "dispatch_parallel": (
        "Spawn the whole wave with Codex's native subagent workflow. Start "
        "one Codex subagent per listed sub-agent/prompt, using the `archie_analysis` "
        "custom agent when it is available and the built-in `worker` agent "
        "otherwise. Send all subagent launches from the same orchestration "
        "step so Codex runs them in parallel, then wait for every subagent "
        "to finish before continuing. Give each subagent: `Project root: "
        "$PWD. Read $PWD/AGENTS.md first if it exists, then carry out this "
        "prompt in full: <that sub-agent's full prompt text>.` Each subagent "
        "writes its own output file per the output contract below. After "
        "all subagents finish, verify every expected output file exists; if "
        "any output is missing, stop and report the failed subagent id."
    ),
    # How to spawn one worker.
    "dispatch_single": (
        "Spawn one Codex subagent for this task, using the `archie_analysis` "
        "custom agent when it is available and the built-in `worker` agent "
        "otherwise. Wait for it to finish before continuing. Give it the "
        "project root, the exact prompt text or prompt-file path named in "
        "this workflow, and the one output file path it owns. After it "
        "finishes, verify the expected output file exists before continuing."
    ),
    # How to fan out one worker per selected workspace (monorepo SCOPE=per-package
    # / hybrid parallel mode). Codex docs: ask Codex to spawn subagents; Codex
    # waits for all to finish before returning a consolidated response.
    "dispatch_workspace_parallel": (
        "Ask Codex to spawn one native Codex subagent per selected workspace, "
        "using the `archie_analysis` custom agent (defined in "
        "`.codex/agents/archie-analysis.toml`) when available and the built-in "
        "`worker` agent otherwise. Each workspace subagent sets `PROJECT_ROOT` "
        "to its assigned `$PWD/<workspace>` path and runs the requested "
        "deep-scan steps for that workspace only. Send all workspace subagent "
        "launches from the same orchestration step, wait for every workspace "
        "subagent to finish, then continue with the parent workflow."
    ),
    # How a spawned worker must write its output file.
    # Targets all live under the workspace (`.archie/tmp/...`), so `apply_patch`
    # handles them natively under Codex's default `workspace-write` sandbox —
    # no shell-write fallback needed.
    "output_contract": (
        "1. Use `apply_patch` to write your COMPLETE output to the file path "
        "named above.\n"
        "2. Write the raw output only — no markdown fences, no prose, unless "
        "the target format explicitly expects them.\n"
        "3. Reply with exactly: \"Wrote <that file path>\".\n"
        "4. Do NOT paste the full output into the conversation."
    ),
    # How to ask the user an interactive question. The question text, header,
    # and options stay inline in the canonical workflow — only the asking
    # mechanism is slotted.
    "ask_user": (
        "Ask the user directly in the Codex conversation. Present the question "
        "text, then a numbered list of the options, explicitly say when "
        "multiple selections are allowed, accept comma-separated numbers or "
        "`all` when the workflow allows it, and wait for the user's reply "
        "before continuing"
    ),
}


class CodexConnector(Connector):
    name = "codex"
    capabilities = frozenset({
        "commands",
        "hooks:pre-tool-use",
        "hooks:post-tool-use",
        "hooks:user-prompt-submit",
        "hooks:stop",
        "hooks:pre-commit",
        "parallel-agents",
        "config-patch",
    })

    render_tokens = _CODEX_RENDER_TOKENS
    render_partials = _CODEX_RENDER_PARTIALS

    def home_dir(self) -> Path:
        return Path.home() / ".codex"

    def install_command(self, project_root: Path, cmd: CommandDef) -> None:
        # Codex parent-walks .agents/skills/<name>/SKILL.md
        # — verified by Q1 probe 2026-05-15. SKILL.md is a thin shim that points
        # at the rendered canonical body under
        # .archie/workflow/codex/<command>/SKILL.md.
        dest = project_root / ".agents" / "skills" / cmd.name / "SKILL.md"
        dest.parent.mkdir(parents=True, exist_ok=True)
        body_path = f"{CODEX_WORKFLOW_ROOT}/{cmd.body_path}"
        dest.write_text(
            f"---\nname: {cmd.name}\ndescription: {cmd.description}\n---\n\n"
            f"Read `{body_path}` in full and execute the instructions as written.\n"
        )

    def install_hook(self, project_root: Path, hook: HookDef) -> None:
        script_name = Path(hook.script_path).name
        src = HOOK_SCRIPTS_DIR / script_name
        if not src.exists():
            raise FileNotFoundError(f"Canonical hook script missing: {src}")

        event_key = _EVENT_NAME_CODEX.get(hook.event)
        if event_key is None:
            return  # pre-commit handled by the universal git hook in install.py

        hooks_path = project_root / ".codex" / "hooks.json"
        config: dict = {"hooks": {}}
        if hooks_path.exists():
            try:
                config = json.loads(hooks_path.read_text())
                config.setdefault("hooks", {})
            except (json.JSONDecodeError, OSError):
                config = {"hooks": {}}

        # Use an absolute command path so the hook fires regardless of whether
        # the user started Codex from a subdirectory of the project.
        command_path = str((project_root / hook.script_path).resolve())
        bucket = config["hooks"].setdefault(event_key, [])
        matcher = _codex_matcher(hook.tool_match)
        if not _codex_entry_present(bucket, matcher, command_path):
            entry = {
                "hooks": [{"type": "command", "command": command_path, "timeout": 30}],
            }
            if matcher is not None:
                entry["matcher"] = matcher
            bucket.append(entry)

        hooks_path.parent.mkdir(parents=True, exist_ok=True)
        hooks_path.write_text(json.dumps(config, indent=2) + "\n")

    def patch_config(self, patches: list[ConfigPatch]) -> None:
        cfg_path = self.home_dir() / "config.toml"
        cfg_path.parent.mkdir(parents=True, exist_ok=True)
        existing = cfg_path.read_text() if cfg_path.exists() else ""
        updated = existing
        for patch in patches:
            if patch.section is None:
                # Top-level patches reflect Archie requirements (the project-doc
                # context size + fallback filename list) — overwrite to enforce
                # the floor we need.
                updated = _toml_set_top_level(updated, patch.key, patch.value)
            else:
                # Section patches are defensive defaults ([agents] knobs Codex
                # itself has documented defaults for). Respect a user's existing
                # value if any — only write the key when it's absent.
                updated = _toml_set_section_key(
                    updated, patch.section, patch.key, patch.value, overwrite=False,
                )
        if updated != existing:
            cfg_path.write_text(updated)

    def finalize(self, project_root: Path) -> None:
        # 1. Project-scoped custom-agent definition. Codex docs document
        # `.codex/agents/*.toml` as the project-scoped location for custom
        # agents; [agents] globals live in `~/.codex/config.toml` and are
        # handled by patch_config() via CONFIG_PATCHES.
        agents_dir = project_root / ".codex" / "agents"
        agents_dir.mkdir(parents=True, exist_ok=True)
        agent_path = agents_dir / "archie-analysis.toml"
        if not agent_path.exists():
            agent_path.write_text(
                'name = "archie_analysis"\n'
                'description = "Archie analysis worker for scan, deep-scan, and intent-layer subagent tasks."\n'
                'model_reasoning_effort = "medium"\n'
                'sandbox_mode = "workspace-write"\n'
                'developer_instructions = """\n'
                'You are an Archie workflow worker. Follow the parent prompt exactly, stay within your assigned task, and write the requested artifact to the requested path. Do not modify unrelated source files. Do not paste large artifacts into the conversation after writing them.\n'
                '"""\n'
            )

        # 2. Execpolicy Rules file — pre-approves every shell command shape
        # the workflow runs so the user is not prompted mid-scan. The Rules
        # schema is documented at developers.openai.com/codex/rules.
        rules_dir = project_root / ".codex" / "rules"
        rules_dir.mkdir(parents=True, exist_ok=True)
        (rules_dir / "archie.rules").write_text(_build_archie_rules_content())

        # 3. Mark the project trusted in ~/.codex/config.toml so the
        # project-scoped .codex/ layer (rules, agents, hooks) actually
        # loads. Codex docs: "Untrusted projects skip project-scoped
        # `.codex/` layers, including project-local config, hooks, and
        # rules." Installing Archie is itself an explicit trust act; the
        # write is set-if-absent so a user who manually marked the project
        # "untrusted" is respected.
        abs_path = str(project_root.resolve())
        cfg_path = self.home_dir() / "config.toml"
        cfg_path.parent.mkdir(parents=True, exist_ok=True)
        existing = cfg_path.read_text() if cfg_path.exists() else ""
        # TOML quoted-key section header: [projects."<abs-path>"]
        section = f'projects."{_toml_string(abs_path)}"'
        updated = _toml_set_section_key(
            existing, section, "trust_level", "trusted", overwrite=False,
        )
        if updated != existing:
            cfg_path.write_text(updated)


# ---------- helpers ----------


def _starlark_str(value: str) -> str:
    """Serialize a Python string as a Starlark double-quoted literal."""
    return '"' + value.replace("\\", "\\\\").replace('"', '\\"') + '"'


def _prefix_rule_block(pattern: tuple[str, ...], justification: str, match: list[str] | None = None) -> str:
    """Render one Starlark prefix_rule(...) declaration for archie.rules."""
    pattern_repr = ", ".join(_starlark_str(p) for p in pattern)
    lines = [
        "prefix_rule(",
        f"    pattern = [{pattern_repr}],",
        "    decision = \"allow\",",
        f"    justification = {_starlark_str(justification)},",
    ]
    if match:
        match_repr = ",\n        ".join(_starlark_str(m) for m in match)
        lines.append("    match = [")
        lines.append(f"        {match_repr},")
        lines.append("    ],")
    lines.append(")")
    return "\n".join(lines)


def _build_archie_rules_content() -> str:
    """Produce the full Starlark `archie.rules` content from the catalogue +
    `_STANDALONE_SCRIPTS`. Adding a new Archie Python script automatically
    gets a prefix_rule here for free."""
    COMMAND_RULES, _STANDALONE_SCRIPTS = _load_command_catalogue()
    header = (
        "# Archie execpolicy Rules — auto-approve every shell command the\n"
        "# deep-scan / intent-layer / share / viewer workflows invoke,\n"
        "# so the user is not prompted mid-run. Generated by\n"
        "# CodexConnector.finalize() at install time from\n"
        "# archie/manifest_data.py COMMAND_RULES + install._STANDALONE_SCRIPTS.\n"
        "# Format: developers.openai.com/codex/rules\n"
        "#\n"
        "# Most-restrictive-wins (forbidden > prompt > allow), so any\n"
        "# stricter user rule in this project still takes precedence over\n"
        "# the entries below.\n\n"
    )
    blocks: list[str] = []

    # One prefix_rule per Archie Python script — driven by the install's
    # canonical script list, no manual enumeration.
    for script in _STANDALONE_SCRIPTS:
        if not script.endswith(".py"):
            continue
        script_path = f".archie/{script}"
        blocks.append(_prefix_rule_block(
            pattern=("python3", script_path),
            justification=f"Run Archie's {script}",
            match=[f"python3 {script_path}", f"python3 {script_path} \"$PROJECT_ROOT\""],
        ))

    # Catalogue entries — shell utilities, ad-hoc python, rm, read-only git.
    for rule in COMMAND_RULES:
        blocks.append(_prefix_rule_block(
            pattern=rule.codex_pattern,
            justification=rule.justification,
        ))

    return header + "\n\n".join(blocks) + "\n"


def _codex_entry_present(bucket: list, matcher: str | None, command: str) -> bool:
    for entry in bucket:
        if entry.get("matcher") != matcher:
            continue
        for h in entry.get("hooks", []):
            if h.get("command") == command:
                return True
    return False


def _codex_matcher(tool_match: str | None) -> str | None:
    if tool_match is None:
        return None
    return _MATCHER_NAME_CODEX.get(tool_match, tool_match)


def _toml_string(value: str) -> str:
    return value.replace("\\", "\\\\").replace('"', '\\"')


def _toml_serialize_value(value: object) -> str:
    if isinstance(value, bool):
        return "true" if value else "false"
    if isinstance(value, int):
        return str(value)
    if isinstance(value, float):
        return repr(value)
    if isinstance(value, str):
        return f'"{_toml_string(value)}"'
    if isinstance(value, list):
        parts = [_toml_serialize_value(v) for v in value]
        return "[" + ", ".join(parts) + "]"
    raise TypeError(f"Unsupported TOML value type: {type(value).__name__}")


_TOP_LEVEL_KEY_RE = re.compile(r"^([A-Za-z0-9_\-]+)\s*=\s*(.*)$")


def _toml_set_top_level(content: str, key: str, value: object) -> str:
    """Set a top-level TOML key in `content`.

    The Codex installer only patches two top-level keys, but users may already
    have the same key nested in another section or formatted as a multi-line
    array. This setter keeps the file idempotent by:
    - updating an existing top-level assignment when present
    - moving a misplaced section-level assignment to the top level
    - unioning string-array values without duplicating entries
    """
    entries = _find_toml_assignments(content, key)
    top_level = next((e for e in entries if e["scope"] == "top"), None)
    section_level = next((e for e in entries if e["scope"] == "section"), None)
    existing = top_level or section_level

    new_value = value
    if existing and isinstance(value, list):
        existing_items = _parse_inline_str_array(existing["raw_value"].strip())
        merged = list(existing_items)
        for item in value:
            if item not in merged:
                merged.append(item)
        new_value = merged

    new_assignment = f"{key} = {_toml_serialize_value(new_value)}"
    if not existing:
        return _insert_top_level_assignment(content, new_assignment)

    replacement = new_assignment
    if existing["end"] > existing["start"] and content[existing["end"] - 1: existing["end"]] == "\n":
        replacement += "\n"
    updated = content[: existing["start"]] + replacement + content[existing["end"] :]
    if existing["scope"] == "top":
        return updated

    without_section_assignment = content[: existing["start"]] + content[existing["end"] :]
    return _insert_top_level_assignment(without_section_assignment, new_assignment)


_STR_ITEM_RE = re.compile(r'"((?:[^"\\]|\\.)*)"')


def _parse_inline_str_array(raw: str) -> list[str]:
    normalized = raw.strip()
    if not normalized.startswith("[") or "]" not in normalized:
        return []
    inner = normalized[1: normalized.rindex("]")]
    return [m.group(1).replace('\\"', '"').replace("\\\\", "\\") for m in _STR_ITEM_RE.finditer(inner)]


def _toml_set_section_key(
    content: str, section: str, key: str, value: object, *, overwrite: bool = True,
) -> str:
    """Set a simple key inside a top-level TOML section.

    When `overwrite=False`, an existing assignment for `key` inside the section
    is left untouched — the function only writes the key if it isn't already
    present. Used by `patch_config` for `[agents]` defaults that should respect
    a user's prior customisation (`max_threads`, `max_depth`).
    """
    assignment = f"{key} = {_toml_serialize_value(value)}"
    lines = content.splitlines()
    section_header = f"[{section}]"
    header_idx: int | None = None
    next_header_idx = len(lines)

    for idx, line in enumerate(lines):
        stripped = line.strip()
        if stripped == section_header:
            header_idx = idx
            continue
        if header_idx is not None and idx > header_idx and stripped.startswith("[") and stripped.endswith("]"):
            next_header_idx = idx
            break

    if header_idx is None:
        prefix = content.rstrip()
        block = f"{section_header}\n{assignment}"
        if not prefix:
            return block + "\n"
        return prefix + "\n\n" + block + "\n"

    key_re = re.compile(rf"^(\s*){re.escape(key)}\s*=.*$")
    for idx in range(header_idx + 1, next_header_idx):
        if key_re.match(lines[idx]):
            if not overwrite:
                return content  # Respect user's existing value
            lines[idx] = assignment
            return "\n".join(lines) + "\n"

    lines.insert(next_header_idx, assignment)
    return "\n".join(lines) + "\n"


def _insert_top_level_assignment(content: str, assignment: str) -> str:
    section_match = re.search(r"^\[", content, flags=re.MULTILINE)
    insert_at = section_match.start() if section_match else len(content)
    head = content[:insert_at]
    tail = content[insert_at:]
    if head and not head.endswith("\n"):
        head += "\n"
    if head and not head.endswith("\n\n") and tail:
        head += "\n"
    return head + assignment + "\n" + tail


def _find_toml_assignments(content: str, key: str) -> list[dict[str, object]]:
    entries: list[dict[str, object]] = []
    pos = 0
    scope = "top"
    lines = content.splitlines(keepends=True)
    while pos < len(content) and lines:
        line = lines.pop(0)
        stripped = line.strip()
        if stripped.startswith("[") and stripped.endswith("]"):
            scope = "section"
            pos += len(line)
            continue

        match = _TOP_LEVEL_KEY_RE.match(line)
        if not match or match.group(1) != key:
            pos += len(line)
            continue

        start = pos
        raw_value = match.group(2)
        end = pos + len(line)
        if raw_value.lstrip().startswith("[") and "]" not in raw_value:
            while lines:
                next_line = lines.pop(0)
                raw_value += next_line
                end += len(next_line)
                if "]" in next_line:
                    break
        entries.append({
            "scope": scope,
            "start": start,
            "end": end,
            "raw_value": raw_value,
        })
        pos = end
    return entries
