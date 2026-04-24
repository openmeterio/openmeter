# ledger

<!-- archie:ai-start -->

> Cobra parent command grouping ledger administrative sub-commands (currently only backfillaccounts). Owns the ledger namespace in the jobs CLI and coordinates sub-packages that bypass DI to access concrete ledger adapters when credits.enabled=false would otherwise produce noops.

## Patterns

**Aggregator-only parent command** — ledger.go registers sub-commands in init() and exposes Cmd — no RunE, no flags, no logic. (`Cmd.AddCommand(backfillaccounts.Cmd)`)
**Bypass DI for concrete ledger stack** — Sub-commands build concrete ledger adapters directly (not via app/common Wire outputs) because the Wire-provided resolver is a noop when credits.enabled=false. (`accountSvc := ledgeraccount.NewService(ledgeraccount.NewAdapter(entClient), lockr.NewLocker(...))`)
**Flags defined in init(), logic in RunE** — All flag bindings go in init(); RunE contains all execution logic including time flag parsing. (`func init() { Cmd.Flags().StringVar(&namespace, "namespace", "", "...") }`)
**RFC3339 time parsing for flag values** — Time-valued flags are parsed from RFC3339 strings inside RunE, normalized to UTC. (`t, err := time.Parse(time.RFC3339, flagVal); t = t.UTC()`)
**Print summary before returning error** — Backfill commands print a summary of processed/failed counts before returning any error so partial results remain visible. (`printSummary(result); return firstErr`)
**Interface-driven service in separate service package** — Backfill orchestration logic lives in a separate service sub-package (not in the Cobra command file) with interface-driven dependencies to allow unit testing. (`import "github.com/openmeterio/openmeter/cmd/jobs/ledger/backfillaccounts/service"`)
**Cursor pagination with MAX_SAFE_ITER guard** — All paginated loops in the service package use paginationv2.Cursor and enforce the MAX_SAFE_ITER limit to prevent infinite loops on large datasets. (`for iter := 0; cursor != nil && iter < paginationv2.MAX_SAFE_ITER; iter++ { ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `ledger.go` | Pure aggregator — registers backfillaccounts.Cmd in init(), exposes Cmd for the parent jobs command. | Never add RunE, flags, or business logic to ledger.go. |
| `backfillaccounts/backfillaccounts.go` | Cobra command file: flag binding in init(), concrete adapter construction and RunE execution. | Must build lockr.NewLocker — accountservice requires it for CreateCustomerAccounts. Do not use the Wire-provided ledger resolver (it is noop when credits.enabled=false). |
| `backfillaccounts/service/service.go` | Pure Go backfill orchestration with no Cobra or Ent imports; interface-driven for unit testability. | Use ValidationIssue error codes to detect missing-account condition — not nil checks. DryRun must only increment WouldProvision, never write. |
| `backfillaccounts/service/customer_lister_ent.go` | Ent-backed implementation of the CustomerLister interface used by the service. | Must use cursor pagination — not offset — to handle large customer sets safely. |
| `backfillaccounts/service/service_test.go` | Unit tests for the backfill service using mock/fake implementations. | Use t.Context() — not context.Background(). |

## Anti-Patterns

- Using the Wire-provided ledger account resolver from app/common — it is noop when credits.enabled=false
- Omitting lockr.NewLocker when constructing the account service — causes a panic at CreateCustomerAccounts
- Using offset pagination instead of cursor pagination in large customer loops
- Returning an error from RunE before calling printSummary — partial results become invisible
- Importing *entdb.Client directly in service.go — breaks unit testability

## Decisions

- **Build concrete ledger adapters directly instead of using DI outputs** — app/common wires ledger services to noops when credits.enabled=false; backfill jobs genuinely need real ledger writes, so they must construct the concrete stack themselves.
- **Separate service package from Cobra command** — Isolates orchestration logic from CLI plumbing, enabling unit tests that do not require a real database or Cobra context.
- **Detect missing-account condition via ValidationIssue error codes, not nil checks** — Ledger adapters return typed ValidationIssue errors for missing accounts; nil-checking Get results is unreliable and breaks idempotency.

## Example: Adding a new ledger sub-command that requires concrete adapter construction

```
// ledger.go init():
import "github.com/openmeterio/openmeter/cmd/jobs/ledger/mynewcmd"
func init() { Cmd.AddCommand(mynewcmd.Cmd) }

// ledger/mynewcmd/mynewcmd.go:
var Cmd = &cobra.Command{
    Use: "my-new-cmd",
    RunE: func(cmd *cobra.Command, args []string) error {
        app := internal.MustGetApp(cmd.Context())
        locker := lockr.NewLocker(app.DB)
        accountAdapter := ledgeraccountadapter.New(app.DB)
        accountSvc := ledgeraccountservice.New(accountAdapter, locker)
        result, err := service.Run(cmd.Context(), service.RunInput{...}, accountSvc)
        printSummary(result)
        return err
// ...
```

<!-- archie:ai-end -->
