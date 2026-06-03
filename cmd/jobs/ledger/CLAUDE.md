# ledger

<!-- archie:ai-start -->

> Cobra parent command grouping ledger administrative sub-commands (currently only backfillaccounts). Owns the ledger namespace in the jobs CLI and coordinates sub-packages that intentionally bypass Wire DI to construct concrete ledger adapters, because the wired account resolver is a noop when credits.enabled=false.

## Patterns

**Aggregator-only parent command** — ledger.go registers sub-commands in init() and exposes Cmd — no RunE, no flags, no logic. (`func init() { Cmd.AddCommand(backfillaccounts.Cmd) }`)
**Bypass DI for the concrete ledger stack** — Sub-commands build concrete ledger adapters directly (not via app/common Wire outputs) because the Wire-provided resolver is a noop when credits.enabled=false. (`accountSvc := ledgeraccountservice.New(ledgeraccountadapter.New(app.DB), lockr.NewLocker(app.DB))`)
**Flags in init(), logic in RunE** — All flag bindings go in init(); RunE contains all execution logic including time-flag parsing. (`func init() { Cmd.Flags().StringVar(&namespace, "namespace", "", "...") }`)
**RFC3339 time parsing with UTC normalization** — Time-valued flags are parsed from RFC3339 strings inside RunE and explicitly normalized to UTC. (`t, err := time.Parse(time.RFC3339, flagVal); t = t.UTC()`)
**Print summary before returning error** — Backfill commands print a result summary (processed/failed counts) before returning any error so partial results stay visible to operators. (`printSummary(result); return firstErr`)
**Interface-driven service in a separate package** — Backfill orchestration lives in a separate service sub-package with interface-driven deps, enabling unit testing without a real DB or Cobra context. (`import "github.com/openmeterio/openmeter/cmd/jobs/ledger/backfillaccounts/service"`)
**Cursor pagination with MAX_SAFE_ITER guard** — Paginated loops in the service package use paginationv2.Cursor and enforce MAX_SAFE_ITER to prevent infinite loops on large customer sets. (`for iter := 0; cursor != nil && iter < paginationv2.MAX_SAFE_ITER; iter++ { ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `ledger.go` | Pure aggregator — registers backfillaccounts.Cmd in init() and exposes Cmd. | Never add RunE, flags, or business logic to ledger.go. |
| `backfillaccounts/backfillaccounts.go` | Cobra command: flag binding in init(), concrete adapter construction, and RunE execution. | Must build lockr.NewLocker — accountservice.New requires a Locker for CreateCustomerAccounts. Do not use the Wire-provided ledger resolver (noop when credits.enabled=false). |
| `backfillaccounts/service/service.go` | Pure Go backfill orchestration with no Cobra/Ent imports; interface-driven for unit testability. | Detect missing-account via ValidationIssue error codes, not nil checks. DryRun must only increment WouldProvision, never write. |
| `backfillaccounts/service/customer_lister_ent.go` | Ent-backed CustomerLister implementation used by the service. | Must use cursor pagination, not offset, to handle large customer sets safely. |
| `backfillaccounts/service/service_test.go` | Unit tests using mock CustomerLister and account service. | Use t.Context(), not context.Background(), so cancellation is tied to the test harness. |

## Anti-Patterns

- Using the Wire-provided ledger account resolver from app/common — it is a noop when credits.enabled=false
- Omitting lockr.NewLocker when constructing the account service — panics at CreateCustomerAccounts
- Using offset pagination instead of cursor pagination in large customer loops
- Returning an error from RunE before calling printSummary — partial results become invisible to operators
- Importing *entdb.Client directly in service.go — breaks unit testability

## Decisions

- **Build concrete ledger adapters directly instead of using DI outputs** — app/common wires ledger services to noops when credits.enabled=false; backfill jobs need real ledger writes, so they construct the concrete stack themselves.
- **Separate service package from the Cobra command** — Isolates orchestration from CLI plumbing, enabling unit tests without a real DB or Cobra context.
- **Detect missing-account via ValidationIssue error codes, not nil checks** — Ledger adapters return typed ValidationIssue errors for missing accounts; nil-checking Get results is unreliable and breaks backfill idempotency.

<!-- archie:ai-end -->
