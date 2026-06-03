"""Connector registry.

Per-CLI connectors register here as they're implemented. The install loop
(archie/install.py) iterates this list.

See docs/plans/2026-05-18-multi-agent-connector-architecture.md §18
for ownership per connector:

    Stage 2 — Claude agent: ClaudeConnector  (done)
    Stage 3 — Codex agent:  CodexConnector   (pending — see docs/plans/HANDOFF_CODEX.md)
"""
from .base import Connector
from .claude import ClaudeConnector
from .codex import CodexConnector

ALL_CONNECTORS: list[Connector] = [
    ClaudeConnector(),
    CodexConnector(),
]
