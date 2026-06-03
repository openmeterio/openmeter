# entitlement

<!-- archie:ai-start -->

> Cobra sub-command group for entitlement administrative jobs; exposes recalculate-balance-snapshots, which drives balanceworker.Recalculator to recompute metered entitlement balance snapshots and publish results to the event bus. All dependencies come exclusively from internal.App.

## Patterns

**Source all services from internal.App** — Commands never construct domain services locally; all dependencies (EntitlementRegistry, EventPublisher, Meter, NotificationService, Customer, Subject, Logger) are read from the package-level internal.App global. (`balanceworker.NewRecalculator(balanceworker.RecalculatorOptions{Entitlement: internal.App.EntitlementRegistry, EventBus: internal.App.EventPublisher, ...})`)
**Root + sub-command via cmd.AddCommand in RootCommand()** — root.go defines RootCommand() returning the 'entitlement' parent cobra.Command; each job is registered through cmd.AddCommand(NewXxxCommand()) inside RootCommand(). (`cmd.AddCommand(NewRecalculateBalanceSnapshotsCommand())`)
**cmd.Context() for context propagation** — RunE closures pass cmd.Context() to all domain calls, never context.Background(). (`return recalculator.Recalculate(cmd.Context(), "default", time.Now())`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `root.go` | Defines the 'entitlement' parent Cobra command and registers all sub-commands via cmd.AddCommand. | New jobs must be added via cmd.AddCommand here to be reachable from the CLI. |
| `recalculatesnapshots.go` | Canonical job: builds balanceworker.Recalculator from internal.App fields (HighWatermarkCacheSize=100_000) and calls Recalculate(cmd.Context(), "default", time.Now()). | All RecalculatorOptions fields come from internal.App — adding a dependency requires updating cmd/jobs/internal/wire.go Application struct first and regenerating. |

## Anti-Patterns

- Constructing domain services (Recalculator dependencies) inside the command instead of sourcing from internal.App
- Using context.Background() instead of cmd.Context()
- Adding business logic to RunE beyond orchestrating internal.App services
- Registering sub-commands outside of RootCommand()

## Decisions

- **Dependencies come exclusively from the internal.App global rather than local construction** — Wire-generated initializeApplication in cmd/jobs/internal produces a fully wired Application; repeating wiring in job commands would duplicate provider graphs and risk inconsistency.

## Example: Add a new entitlement job sub-command

```
// newjob.go
package entitlement
import (
  "github.com/spf13/cobra"
  "github.com/openmeterio/openmeter/cmd/jobs/internal"
)
func NewMyEntitlementJobCommand() *cobra.Command {
  return &cobra.Command{
    Use:   "my-job",
    Short: "Does something with entitlements",
    RunE: func(cmd *cobra.Command, args []string) error {
      return internal.App.EntitlementRegistry.SomeOp(cmd.Context())
    },
  }
}
// ...
```

<!-- archie:ai-end -->
