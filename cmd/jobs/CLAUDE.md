# jobs

<!-- archie:ai-start -->

> Entrypoint package for the `jobs` CLI binary — a Cobra root command that runs one-off operational jobs (billing advancement/collection/subscriptionsync, ledger account backfill, entitlement snapshot recalculation, llmcost, quickstart, migrate) against a single Wire-built `internal.Application`. main.go owns process lifecycle and command registration only; all job logic lives in subcommand packages.

## Patterns

**Cobra root aggregates subcommand packages** — main.go's rootCmd registers each domain subcommand via rootCmd.AddCommand(...), pulling each from its own package (billing.Cmd, ledger.Cmd, llmcost.Cmd, quickstart.Cmd, entitlement.RootCommand(), migrate.RootCommand()). New jobs are added by creating a subpackage and registering it here — never by adding RunE logic in main.go. (`rootCmd.AddCommand(billing.Cmd); rootCmd.AddCommand(ledger.Cmd)`)
**App initialized once in PersistentPreRunE** — The Wire application is built exactly once via internal.InitializeApplication(ctx, configFileName) inside rootCmd.PersistentPreRunE before any subcommand RunE runs. Subcommands read the shared internal.App / internal.Config globals instead of constructing their own DI graph. (`PersistentPreRunE: func(...) error { return internal.InitializeApplication(ctx, configFileName) }`)
**Signal-aware root context** — main() builds ctx via signal.NotifyContext(context.Background(), SIGINT, SIGHUP, SIGTERM) and runs rootCmd.ExecuteContext(ctx); subcommands must propagate this context (cmd.Context()) for cancellation on shutdown. (`ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM)`)
**Deferred shutdown + panic logging** — main() defers log.PanicLogger(log.WithExit) first, then a guarded internal.AppShutdown() call (nil-checked because init may fail before it is set). PersistentPreRunE also invokes AppShutdown on init failure. (`defer log.PanicLogger(log.WithExit); defer func(){ if internal.AppShutdown != nil { internal.AppShutdown() } }()`)
**Config via Viper bound to --config flag** — A persistent --config flag (default config.yaml) is bound with viper.BindPFlag and consumed by internal.loadConfig (config.go), which unmarshals into internal.Config and calls Config.Validate(). Job logic never re-reads config files. (`viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `main.go` | Process entrypoint: builds signal context, defines rootCmd, wires PersistentPreRunE -> internal.InitializeApplication, registers all subcommand trees, runs ExecuteContext. | No RunE/business logic here. SilenceUsage/SilenceErrors are true, so errors are logged manually and os.Exit(1) is called explicitly — keep that contract. |
| `version.go` | Defines `version` subcommand via versionCommand(); prints internal.Version() (version, revision, revisionDate). | Version data comes from internal.Version(), not ldflags read here directly. |
| `internal/` | Holds the Wire application graph (wire.go/wire_gen.go), package-level globals (App, AppShutdown, Config, ConfigFile in globals.go), config loading (config.go), and InitializeApplication (app.go). | App/AppShutdown/Config are nolint:gochecknoglobals mutable package globals; safe to read only after PersistentPreRunE has run. Subcommands must guard nil AppShutdown. |

## Anti-Patterns

- Adding job business logic, flags, or service construction in main.go instead of a dedicated subcommand package.
- Building a second Wire application or constructing services directly in a subcommand instead of reading internal.App.
- Forgetting rootCmd.AddCommand(...) for a new subpackage — the command becomes unreachable from the CLI.
- Introducing context.Background() in a subcommand instead of propagating the signal-aware ctx threaded from ExecuteContext.
- Skipping the nil-check on internal.AppShutdown, which can be unset if initialization fails early.

## Decisions

- **Initialize the full application once in PersistentPreRunE rather than per-subcommand.** — All operational jobs share the same heavy DI graph (DB, billing, ledger, streaming); building it once at the root avoids duplicated wiring and config parsing across subcommands.
- **Keep main.go a thin Cobra aggregator with no RunE logic.** — Each operational concern (billing, ledger, entitlement, migrate) evolves independently in its own package; the root only owns lifecycle, signals, config flag, and registration.

## Example: Adding a new operational subcommand to the jobs binary

```
import (
	"github.com/spf13/cobra"
	"github.com/openmeterio/openmeter/cmd/jobs/internal"
)

// in your subpackage:
var Cmd = &cobra.Command{
	Use: "myjob",
	RunE: func(cmd *cobra.Command, args []string) error {
		// internal.App is already built by the root's PersistentPreRunE
		return internal.App.SomeService.DoWork(cmd.Context())
	},
}

// in cmd/jobs/main.go:
// ...
```

<!-- archie:ai-end -->
