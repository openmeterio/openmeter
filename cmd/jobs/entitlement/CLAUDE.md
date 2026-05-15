# entitlement

<!-- archie:ai-start -->

> Cobra sub-command group for entitlement administrative jobs; exposes recalculate-balance-snapshots which drives balanceworker.Recalculator to recompute metered entitlement balance snapshots and publish results to the event bus. All dependencies come exclusively from internal.App.

## Patterns

**Source all services from internal.App** — Commands must never construct domain services locally. All dependencies (EntitlementRegistry, EventPublisher, NotificationService, Customer, Subject, Meter, Logger) are read from the package-level internal.App global. (`balanceworker.NewRecalculator(balanceworker.RecalculatorOptions{Entitlement: internal.App.EntitlementRegistry, EventBus: internal.App.EventPublisher, ...})`)
**Root + sub-command via cmd.AddCommand in RootCommand()** — root.go defines RootCommand() returning a parent cobra.Command with Use='entitlement'; each job file registers itself via cmd.AddCommand(NewXxxCommand()) inside RootCommand(). (`cmd.AddCommand(NewRecalculateBalanceSnapshotsCommand())`)
**Use cmd.Context() for context propagation** — RunE closures must pass cmd.Context() to all domain calls, never context.Background(). (`return recalculator.Recalculate(cmd.Context(), "default", time.Now())`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `root.go` | Defines the 'entitlement' parent Cobra command and registers all sub-commands via cmd.AddCommand. | New jobs must be added via cmd.AddCommand here to be reachable from the CLI. |
| `recalculatesnapshots.go` | Canonical job implementation: constructs balanceworker.Recalculator from internal.App fields and invokes Recalculate(cmd.Context(), ...). | All RecalculatorOptions fields come from internal.App — adding a dependency requires updating cmd/jobs/internal/wire.go Application struct first and regenerating. |

## Anti-Patterns

- Constructing domain services (e.g. balanceworker.Recalculator dependencies) inside the command instead of sourcing from internal.App
- Using context.Background() instead of cmd.Context()
- Adding business logic to RunE beyond orchestrating internal.App services
- Registering sub-commands outside of RootCommand()

## Decisions

- **Dependencies come exclusively from internal.App global rather than local construction.** — Wire-generated initializeApplication in cmd/jobs/internal produces a fully wired Application; repeating wiring in job commands would duplicate provider graphs and risk inconsistency.

## Example: Adding a new entitlement job sub-command

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
// ...
```

<!-- archie:ai-end -->
