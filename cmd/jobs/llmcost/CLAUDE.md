# llmcost

<!-- archie:ai-start -->

> Cobra sub-command group for LLM cost database operations; exposes a single 'sync' command that delegates to Wire-provided LLMCostSyncJob from internal.App — no local service construction.

## Patterns

**Package-level Cmd var with init() sub-command registration** — Cmd is declared as a package-level *cobra.Command; sub-commands are registered in init() via Cmd.AddCommand. This differs from the entitlement package's RootCommand() pattern — match the style of the file you are extending. (`var Cmd = &cobra.Command{Use: "llm-cost", ...}; func init() { Cmd.AddCommand(syncCmd()) }`)
**Delegate to internal.App service, never construct locally** — RunE must call internal.App.LLMCostSyncJob.Run(cmd.Context()) directly; never construct llmcost.SyncJob or its dependencies inside the command. (`return internal.App.LLMCostSyncJob.Run(cmd.Context())`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `llmcost.go` | Defines the 'llm-cost' parent command and 'sync' sub-command; canonical example of the init()-based sub-command registration pattern. | LLMCostSyncJob must exist on internal.App (wired via common.LLMCost in wire.go); removing it from Application breaks this command at compile time. |

## Anti-Patterns

- Constructing llmcost.SyncJob directly instead of using internal.App.LLMCostSyncJob
- Using context.Background() instead of cmd.Context()
- Adding business logic beyond delegating to the sync job

<!-- archie:ai-end -->
