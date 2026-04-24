# jobs

<!-- archie:ai-start -->

> Cobra CLI binary grouping administrative one-off jobs (billing advance/collect, entitlement recalculation, ledger backfill, LLM cost sync, DB migration, quickstart cron). Each sub-command lives in its own sub-package under cmd/jobs/; shared wiring is in cmd/jobs/internal.

## Patterns

**PersistentPreRunE initialises the app once** — main.go's rootCmd.PersistentPreRunE calls internal.InitializeApplication(ctx, configFileName) before every sub-command executes. Sub-commands must never initialise services themselves. (`PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return internal.InitializeApplication(ctx, configFileName) }`)
**internal.App as the single service registry** — All sub-commands access domain services exclusively via package-level globals internal.App and internal.Config — never by constructing services inline. (`internal.App.BillingService.AdvanceInvoices(cmd.Context(), ...)`)
**cmd.Context() for context propagation** — Every RunE uses cmd.Context() (derived from the signal-aware root context) rather than context.Background() or context.TODO(). (`RunE: func(cmd *cobra.Command, args []string) error { return doWork(cmd.Context()) }`)
**Sub-command registered in sub-package init() or RootCommand()** — Each sub-package exports either a package-level Cmd var (registered in init()) or a RootCommand() factory. main.go adds them with rootCmd.AddCommand(...). (`rootCmd.AddCommand(billing.Cmd); rootCmd.AddCommand(entitlement.RootCommand())`)
**Nil-guard optional app services before use** — Optional features (e.g. ChargesAutoAdvancer) may be nil when the feature is disabled. Always nil-check before calling. (`if internal.App.ChargesAutoAdvancer != nil { ... }`)
**Deferred AppShutdown in main** — main.go defers internal.AppShutdown() to release resources regardless of execution path, including early-exit error paths. (`defer func() { if internal.AppShutdown != nil { internal.AppShutdown() } }()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `main.go` | Binary entry point: signal-aware context, Cobra root, PersistentPreRunE DI bootstrap, sub-command registration, and deferred shutdown. | PersistentPreRunE runs before every sub-command — side-effects here affect all commands. AppShutdown must be deferred or resources leak. |
| `version.go` | Minimal version sub-command; delegates to internal.Version() for build metadata. | Calls internal.Version() which reads ldflags-injected variables — keep in sync with build toolchain. |
| `cmd/jobs/internal/wire.go` | Wire provider sets and initializeApplication declaration; source of truth for DI graph. | Adding a new service field to Application requires a matching provider in wire.Build. Never edit wire_gen.go directly. |
| `cmd/jobs/internal/globals.go` | Package-level App, AppShutdown, Config globals consumed by all sub-packages. | These are intentionally global (nolint:gochecknoglobals). Do not introduce a second App construction path. |
| `cmd/jobs/billing/billing.go` | Pure aggregator — registers advance, advancecharges, collect, subscriptionsync sub-commands with no logic. | No business logic belongs here; keep it as a pure AddCommand aggregator. |
| `cmd/jobs/ledger/backfillaccounts/service/service.go` | Concrete ledger adapter construction that bypasses DI noops when credits.enabled=false. | Must build lockr.Locker and concrete ledger adapters directly — using Wire-provided noop resolvers silently skips writes. |
| `cmd/jobs/quickstart/cronjobs.go` | gocron-based polling loops for quickstart/demo environments only. | Not for production workers — production workers use Kafka/Watermill. Nil-check all optional services before scheduling. |

## Anti-Patterns

- Constructing domain services (billing, charges, ledger) directly inside RunE instead of using internal.App
- Using context.Background() or context.TODO() in RunE — always use cmd.Context()
- Registering sub-command flags on a parent aggregator Cmd (pollutes sibling flag sets)
- Calling tools/migrate DDL or data transforms inline in migrate.go — those belong in tools/migrate/migrations/
- Using quickstart cron patterns (gocron polling) in production workers

## Decisions

- **PersistentPreRunE wires the entire application once before any sub-command runs** — Centralises DI bootstrap in main.go so every sub-command automatically gets a fully-wired app without duplicating initialisation logic.
- **internal.App package-level globals rather than cobra context injection** — Simpler sub-package access pattern; avoids cobra context type-assertion boilerplate across many independent sub-packages.
- **Ledger backfill bypasses Wire DI to construct concrete adapters** — When credits.enabled=false, Wire provides noop ledger resolvers; the backfill job must write real rows so it builds concrete adapters directly.

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
		return internal.App.MyService.DoWork(cmd.Context())
// ...
```

<!-- archie:ai-end -->
