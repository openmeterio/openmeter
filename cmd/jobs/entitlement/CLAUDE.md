# entitlement

<!-- archie:ai-start -->

> Cobra sub-command group for entitlement administrative jobs; currently owns one job (recalculate-balance-snapshots) that recalculates metered entitlement balance snapshots and publishes results to the event bus via balanceworker.Recalculator.

## Patterns

**Access app state via internal.App global** — All job commands read dependencies (EntitlementRegistry, EventPublisher, NotificationService, Customer, Subject, Meter, Logger) from the package-level internal.App struct, never constructing services themselves. (`balanceworker.NewRecalculator(balanceworker.RecalculatorOptions{Entitlement: internal.App.EntitlementRegistry, EventBus: internal.App.EventPublisher, ...})`)
**Root + sub-command structure** — root.go defines RootCommand() returning a parent cobra.Command with Use='entitlement', and each job file adds a sub-command via cmd.AddCommand in RootCommand. (`cmd.AddCommand(NewRecalculateBalanceSnapshotsCommand())`)
**Use cmd.Context() for context propagation** — RunE closures pass cmd.Context() to domain calls, never context.Background(). (`recalculator.Recalculate(cmd.Context(), "default", time.Now())`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `root.go` | Defines the 'entitlement' parent Cobra command and registers all sub-commands. | New jobs must be added via cmd.AddCommand here to be reachable. |
| `recalculatesnapshots.go` | Implements recalculate-balance-snapshots job using balanceworker.NewRecalculator; canonical example of job structure. | All RecalculatorOptions fields come from internal.App — adding a dependency requires updating internal/wire.go Application struct first. |

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
