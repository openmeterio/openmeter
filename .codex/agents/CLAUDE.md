# agents

<!-- archie:ai-start -->

> Codex CLI agent definitions for the Archie workflow. Holds archie-analysis.toml, the worker-agent profile used for Archie scan, deep-scan, and intent-layer subagent tasks. Config-only — no executable source.

## Patterns

**Single worker-agent TOML profile** — An agent is one .toml file declaring name, description, model_reasoning_effort, sandbox_mode, and a developer_instructions block. The archie_analysis worker runs with medium reasoning effort and workspace-write sandbox. (`name = "archie_analysis"
model_reasoning_effort = "medium"
sandbox_mode = "workspace-write"`)
**Constrained worker contract in developer_instructions** — developer_instructions pin the worker to the parent prompt: stay within the assigned task, write the requested artifact to the requested path, do not modify unrelated files, and do not paste large artifacts back into the conversation. (`developer_instructions = """You are an Archie workflow worker. Follow the parent prompt exactly... Do not paste large artifacts..."""`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `archie-analysis.toml` | Defines the archie_analysis worker agent used by Archie scan/deep-scan/intent-layer subagent tasks. | sandbox_mode is workspace-write — the worker may write artifacts but must not touch unrelated source; keep developer_instructions aligned with the Archie subagent contract. |

## Anti-Patterns

- Loosening sandbox_mode beyond workspace-write for the analysis worker
- Adding business or repo-specific logic into the agent profile (it only configures a worker)
- Editing developer_instructions to allow modifying unrelated source files or pasting large artifacts

<!-- archie:ai-end -->
