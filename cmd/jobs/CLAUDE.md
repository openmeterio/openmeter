# jobs

<!-- archie:ai-start -->

> Cobra CLI binary grouping administrative one-off jobs (billing advance/collect, entitlement recalculation, ledger backfill, LLM cost sync, DB migration, quickstart cron). The entire application is wired once via PersistentPreRunE before any sub-command runs; sub-commands source all services from internal.App globals.

## Patterns

**PersistentPreRunE DI bootstrap** — rootCmd.PersistentPreRunE calls internal.InitializeApplication(ctx, configFileName) before every sub-command. Sub-commands must never initialise services themselves. (`PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return internal.InitializeApplication(ctx, configFileName) }`)
**internal.App as the single service registry** — All sub-commands access domain services exclusively via package-level globals internal.App and internal.Config — never by constructing services inline. (`internal.App.BillingService.AdvanceInvoices(cmd.Context(), billing.AdvanceInvoicesInput{...})`)
**cmd.Context() for context propagation** — Every RunE uses cmd.Context() (signal-aware root context) — never context.Background() or context.TODO(). (`RunE: func(cmd *cobra.Command, args []string) error { return doWork(cmd.Context()) }`)
**Sub-command registration via init() or RootCommand()** — Each sub-package exports either a package-level Cmd var registered in init() or a RootCommand() factory. main.go adds them with rootCmd.AddCommand(...). (`rootCmd.AddCommand(billing.Cmd); rootCmd.AddCommand(entitlement.RootCommand())`)
**Nil-guard optional app services before use** — Optional features (e.g. ChargesAutoAdvancer) may be nil when the feature is disabled. Always nil-check before calling. (`if internal.App.ChargesAutoAdvancer != nil { scheduler.NewJob(...) }`)
**Deferred AppShutdown in main** — main.go defers internal.AppShutdown() to release resources regardless of execution path, including early-exit error paths. (`defer func() { if internal.AppShutdown != nil { internal.AppShutdown() } }()`)
**Ledger backfill bypasses Wire DI for concrete adapters** — When credits.enabled=false Wire provides noop ledger resolvers. The ledger backfill sub-command must construct concrete adapters directly (including lockr.Locker) to write real rows. (`// cmd/jobs/ledger/backfillaccounts/service/service.go builds lockr.NewLocker and concrete ledger adapters, not the Wire-injected noops`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `main.go` | Binary entry point: signal-aware context, Cobra root, PersistentPreRunE DI bootstrap, sub-command registration, deferred shutdown. | PersistentPreRunE runs before every sub-command — side-effects here affect all commands. AppShutdown must be deferred or resources leak. |
| `version.go` | Minimal version sub-command; delegates to internal.Version() for ldflags-injected build metadata. | internal.Version() reads ldflags variables — keep in sync with the build toolchain. |
| `cmd/jobs/internal/wire.go` | Wire provider sets and initializeApplication declaration; source of truth for the DI graph. | Adding a new Application field requires a matching provider in wire.Build. Never edit wire_gen.go directly — regenerate with 'make generate'. |
| `cmd/jobs/internal/globals.go` | Package-level App, AppShutdown, Config globals consumed by all sub-packages. | Intentionally global (nolint:gochecknoglobals). Do not introduce a second App construction path. |
| `cmd/jobs/billing/billing.go` | Pure aggregator — registers advance, advancecharges, collect, subscriptionsync sub-commands with no logic. | No business logic belongs here; keep it as a pure AddCommand aggregator. Each sub-command lives in its own package. |
| `cmd/jobs/ledger/backfillaccounts/service/service.go` | Concrete ledger adapter construction that bypasses DI noops when credits.enabled=false; uses cursor pagination with MAX_SAFE_ITER guard. | Must build lockr.Locker and concrete ledger adapters directly. Using Wire-provided noop resolvers silently skips all writes. |
| `cmd/jobs/quickstart/cronjobs.go` | gocron-based polling loops for quickstart/demo environments only. | Not for production workers — production uses Kafka/Watermill. Nil-check all optional services before scheduling. |

## Anti-Patterns

- Constructing domain services (billing, charges, ledger) directly inside RunE instead of sourcing from internal.App
- Using context.Background() or context.TODO() in RunE — always use cmd.Context()
- Registering sub-command-specific flags on a parent aggregator Cmd (pollutes sibling flag sets)
- Calling tools/migrate DDL or data transforms inline in migrate.go — those belong in tools/migrate/migrations/
- Using quickstart gocron polling patterns in production workers — production billing flows use Kafka/Watermill

## Decisions

- **PersistentPreRunE wires the entire application once before any sub-command runs** — Centralises DI bootstrap in main.go so every sub-command automatically gets a fully-wired app without duplicating initialisation logic across sub-packages.
- **internal.App package-level globals rather than Cobra context injection** — Simpler sub-package access pattern; avoids type-assertion boilerplate across many independent sub-packages that each need to pull services from the context.
- **Ledger backfill bypasses Wire DI to construct concrete adapters** — When credits.enabled=false, Wire provides noop ledger resolvers; the backfill job must write real rows so it builds lockr.Locker and concrete adapters directly.

## Example: Adding a new administrative sub-command that uses a domain service from internal.App

```
// cmd/jobs/myfeature/myfeature.go
package myfeature

import (
	"github.com/spf13/cobra"
	"github.com/openmeterio/openmeter/cmd/jobs/internal"
)

var Cmd = &cobra.Command{
	Use:   "myfeature",
	Short: "Run myfeature job",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Always use cmd.Context(), never context.Background()
		// Always source services from internal.App
		if internal.App.MyService == nil {
// ...
```

<!-- archie:ai-end -->
