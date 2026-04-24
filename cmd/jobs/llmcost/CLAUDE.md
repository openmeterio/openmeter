# llmcost

<!-- archie:ai-start -->

> Cobra sub-command group for LLM cost database operations; exposes a single 'sync' command that delegates to the Wire-provided LLMCostSyncJob from internal.App.

## Patterns

**Package-level Cmd var with init() sub-command registration** — Cmd is declared as a package-level *cobra.Command; sub-commands are registered in init() via Cmd.AddCommand, unlike the entitlement package which uses RootCommand(). Both styles exist — match the file you're extending. (`var Cmd = &cobra.Command{Use: "llm-cost", ...}; func init() { Cmd.AddCommand(syncCmd()) }`)
**Delegate to internal.App service, never construct** — RunE calls internal.App.LLMCostSyncJob.Run(cmd.Context()) directly; no local service construction. (`return internal.App.LLMCostSyncJob.Run(cmd.Context())`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `llmcost.go` | Defines the 'llm-cost' parent command and the 'sync' sub-command; canonical example of the init()-based sub-command registration pattern. | LLMCostSyncJob must exist on internal.App (it's wired via common.LLMCost in wire.go); removing it from Application would break this command at compile time. |

## Anti-Patterns

- Constructing llmcost.SyncJob directly instead of using internal.App.LLMCostSyncJob
- Using context.Background() instead of cmd.Context()

<!-- archie:ai-end -->
