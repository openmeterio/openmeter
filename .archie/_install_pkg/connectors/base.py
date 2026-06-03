"""Connector interface. Each supported CLI implements this contract.

A Connector's only job is to translate canonical manifest entries
(CommandDef, HookDef, ConfigPatch) into files in the CLI's
native idiom. All bodies, scripts, and prompts live in archie/assets/ —
connectors emit thin shims that reference those canonical files at
runtime.

Capability vocabulary (subset of):
    commands                       — install_command supported
    hooks:pre-tool-use             — install_hook for that event supported
    hooks:post-tool-use
    hooks:user-prompt-submit
    hooks:stop
    hooks:pre-commit               — git pre-commit (universal — every connector declares this)
    parallel-agents                — CLI runtime supports parallel sub-agent fan-out
    config-patch                   — patch_config supported (Codex only)

See docs/plans/2026-05-18-multi-agent-connector-architecture.md for the
full design and feature parity matrix.
"""
from abc import ABC, abstractmethod
from pathlib import Path

from ..manifest import CommandDef, ConfigPatch, HookDef


class Connector(ABC):
    name: str
    capabilities: frozenset[str]

    # Render map for the templated canonical workflow (archie/assets/workflow/).
    # The install loop renders every workflow file through these before writing
    # them into <project>/.archie/workflow/<cli>/. `render_tokens` substitutes
    # inline `{{TOKEN}}` slots; `render_partials` substitutes multi-line
    # `{{>partial}}` block slots with the connector's native phrasing.
    # See HANDOFF_codex_command_parity.md §4 for the locked slot vocabulary.
    render_tokens: dict[str, str] = {}
    render_partials: dict[str, str] = {}

    @abstractmethod
    def home_dir(self) -> Path:
        ...

    @abstractmethod
    def install_command(self, project_root: Path, cmd: CommandDef) -> None:
        ...

    def install_hook(self, project_root: Path, hook: HookDef) -> None:
        raise NotImplementedError(f"{self.name} does not support hooks")

    def patch_config(self, patches: list[ConfigPatch]) -> None:
        return

    def finalize(self, project_root: Path) -> None:
        return

    def detect(self) -> bool:
        return self.home_dir().exists()

    def supports_event(self, event: str) -> bool:
        return f"hooks:{event}" in self.capabilities
