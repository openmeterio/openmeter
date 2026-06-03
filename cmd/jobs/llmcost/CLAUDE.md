# llmcost

<!-- archie:ai-start -->

> Cobra sub-command group for LLM cost database operations; exposes a single 'sync' command that delegates to the Wire-provided LLMCostSyncJob on internal.App, with no local service construction.

## Patterns

**Package-level Cmd var with init() sub-command registration** — Cmd is a package-level *cobra.Command; sub-commands are registered in init() via Cmd.AddCommand. This differs from the entitlement package's RootCommand() pattern — match the style of the file you are extending. (`var Cmd = &cobra.Command{Use: "llm-cost", ...}; func init() { Cmd.AddCommand(syncCmd()) }`)
**Delegate to internal.App service, never construct locally** — RunE must call internal.App.LLMCostSyncJob.Run(cmd.Context()) directly; never construct llmcost.SyncJob or its dependencies inside the command. (`return internal.App.LLMCostSyncJob.Run(cmd.Context())`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `llmcost.go` | Defines the 'llm-cost' parent command and its 'sync' sub-command; canonical example of the init()-based sub-command registration pattern. | LLMCostSyncJob must exist on internal.App (wired via common.LLMCost in wire.go); removing it from Application breaks this command at compile time. |

## Anti-Patterns

- Constructing llmcost.SyncJob directly instead of using internal.App.LLMCostSyncJob
- Using context.Background() instead of cmd.Context()
- Adding business logic beyond delegating to the sync job

## Example: Adding a sub-command that delegates to a Wire-provided job

```
func syncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Sync LLM cost prices from external sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			return internal.App.LLMCostSyncJob.Run(cmd.Context())
		},
	}
}
```

<!-- archie:ai-end -->
