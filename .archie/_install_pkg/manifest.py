"""Canonical types defining what Archie ships across all CLIs.

Manifest entries are CLI-agnostic. Each Connector reads them and emits the
appropriate per-CLI artifact (slash command / skill, hook config,
config patch). See docs/plans/2026-05-18-multi-agent-connector-architecture.md
for the full design.
"""
from dataclasses import dataclass
from typing import Literal, Optional

HookEvent = Literal[
    "pre-tool-use",
    "post-tool-use",
    "user-prompt-submit",
    "stop",
    "pre-commit",
]


@dataclass(frozen=True)
class CommandDef:
    name: str
    description: str
    body_path: str


@dataclass(frozen=True)
class HookDef:
    event: HookEvent
    tool_match: Optional[str]
    script_path: str
    blocking: bool


@dataclass(frozen=True)
class ConfigPatch:
    cli: str
    key: str
    value: object
    # Optional TOML section. None → top-level assignment (the default, used
    # for keys like `project_doc_max_bytes`). A string → write under
    # `[<section>]` (used for `[agents]` keys like `max_threads`/`max_depth`,
    # which the Codex docs locate in the user-home `~/.codex/config.toml`).
    section: str | None = None


@dataclass(frozen=True)
class CommandRule:
    """A shell command pattern the install pre-approves so the workflow runs
    prompt-free on both CLIs.

    `codex_pattern` is a tuple of exact-prefix argv tokens — the
    `prefix_rule.pattern` shape from the Codex execpolicy Rules schema
    (developers.openai.com/codex/rules). Codex's matcher treats the command
    as the argv list `execvp(3)` receives; the pattern must be an exact
    prefix of that list (each element is a literal string).

    `claude_glob` is the corresponding Claude permission entry (e.g.
    `Bash(python3 .archie/scanner.py *)`). The two shapes are kept
    side-by-side because Claude's allowlist uses shell-glob semantics
    while Codex's Rules use argv-prefix semantics — they're not
    derivable from each other for cases like `rm -f .archie/tmp/archie_*`
    where bash expands the glob to many literal argv entries.

    `justification` is the human-readable reason exposed in both the
    rendered Codex `.rules` file and (optionally) doc references.
    """
    name: str
    codex_pattern: tuple[str, ...]
    claude_glob: str
    justification: str
