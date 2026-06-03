# jobs

<!-- archie:ai-start -->

> Cobra CLI binary grouping administrative one-off jobs (billing advance/collect, entitlement recalculation, ledger backfill, LLM cost sync, DB migration, quickstart cron). The entire application is wired once via PersistentPreRunE before any sub-command runs; sub-commands source all services from internal.App globals. No business logic lives in cmd/jobs itself.

## Patterns

**PersistentPreRunE DI bootstrap** — rootCmd.PersistentPreRunE calls internal.InitializeApplication(ctx, configFileName) before every sub-command, calling internal.AppShutdown() and erroring on failure. Sub-commands must never initialise services themselves. (`PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return internal.InitializeApplication(ctx, configFileName) }`)
**internal.App as the single service registry** — All sub-commands access domain services exclusively via package-level globals internal.App and internal.Config — never by constructing services inline. (`internal.App.BillingService.AdvanceInvoices(cmd.Context(), billing.AdvanceInvoicesInput{...})`)
**cmd.Context() for context propagation** — Every RunE uses cmd.Context() (the signal-aware root context from signal.NotifyContext) — never context.Background() or context.TODO(). (`RunE: func(cmd *cobra.Command, args []string) error { return doWork(cmd.Context()) }`)
**Sub-command registration via init() Cmd var or RootCommand() factory** — Each sub-package exports either a package-level Cmd var or a RootCommand() factory; main.go wires them with rootCmd.AddCommand(...). billing/ledger/llmcost/quickstart expose Cmd; entitlement and migrate expose RootCommand(). (`rootCmd.AddCommand(billing.Cmd); rootCmd.AddCommand(entitlement.RootCommand())`)
**Nil-guard optional app services before use** — Optional features (e.g. ChargesAutoAdvancer) may be nil when disabled. Always nil-check before calling so the binary degrades gracefully instead of panicking. (`if internal.App.ChargesAutoAdvancer != nil { scheduler.NewJob(...) }`)
**Deferred AppShutdown in main** — main.go defers internal.AppShutdown() (nil-checked) to release resources on every exit path, including early-exit error paths inside PersistentPreRunE. (`defer func() { if internal.AppShutdown != nil { internal.AppShutdown() } }()`)
**Ledger backfill bypasses Wire DI for concrete adapters** — When credits.enabled=false Wire provides noop ledger resolvers. The ledger backfill sub-command constructs concrete adapters directly (including lockr.Locker) to write real rows, not the Wire-injected noops. (`// cmd/jobs/ledger/backfillaccounts/service builds lockr.NewLocker + concrete ledger adapters, not the Wire noops`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `main.go` | Binary entry point: signal.NotifyContext root ctx, Cobra root with PersistentPreRunE DI bootstrap, sub-command registration (version/entitlement/billing/ledger/llmcost/quickstart/migrate), deferred shutdown, log.PanicLogger. | PersistentPreRunE runs before every sub-command — side-effects here affect all commands. AppShutdown must be deferred or resources leak; main exits via os.Exit(1) on command error. |
| `version.go` | Minimal version sub-command (versionCommand()) printing internal.Version() (ldflags-injected build metadata). | internal.Version() reads ldflags variables — keep in sync with the build toolchain. |

## Anti-Patterns

- Constructing domain services (billing, charges, ledger) directly inside RunE instead of sourcing from internal.App
- Using context.Background() or context.TODO() in RunE — always use cmd.Context()
- Registering sub-command-specific flags on a parent aggregator Cmd — pollutes sibling flag sets
- Putting tools/migrate DDL or data transforms inline in a job command — those belong in tools/migrate/migrations/
- Using the quickstart gocron polling pattern in production worker paths — production billing flows use Kafka/Watermill

## Decisions

- **PersistentPreRunE wires the entire application once before any sub-command runs** — Centralises DI bootstrap in main.go so every sub-command automatically gets a fully-wired app without duplicating initialisation logic across sub-packages.
- **internal.App package-level globals rather than Cobra context injection** — Simpler sub-package access pattern; avoids type-assertion boilerplate across many independent sub-packages that each pull services from the context.
- **Ledger backfill bypasses Wire DI to construct concrete adapters** — When credits.enabled=false Wire provides noop ledger resolvers; the backfill job must write real rows, so it builds lockr.Locker and concrete adapters directly.

## Example: Add a new administrative sub-command that uses a domain service from internal.App

```
// cmd/jobs/myfeature/myfeature.go
package myfeature

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/openmeterio/openmeter/cmd/jobs/internal"
)

var Cmd = &cobra.Command{
	Use:   "myfeature",
	Short: "Run myfeature job",
	RunE: func(cmd *cobra.Command, args []string) error {
		if internal.App.MyService == nil {
			return fmt.Errorf("myfeature is disabled")
// ...
```

<!-- archie:ai-end -->
