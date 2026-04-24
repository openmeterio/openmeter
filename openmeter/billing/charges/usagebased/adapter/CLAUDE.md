# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing usagebased.Adapter for the usage-based charge lifecycle. It persists charges, realization runs, credit allocations, invoiced usage, payments, and detailed lines — all within context-propagated transactions.

## Patterns

**TransactingRepo wrapping on every write method** — Every mutating adapter method wraps its body with entutils.TransactingRepo (or TransactingRepoWithNoValue for void returns). This rebinds the Ent client to any transaction already in ctx, or starts a new one. Never use a.db directly inside a method body. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (usagebased.ChargeBase, error) { ... tx.db.ChargeUsageBased.UpdateOneID(...).Save(ctx) })`)
**Interface compliance assertions at compile time** — Every adapter sub-interface is asserted with a blank var declaration at the top of each file (e.g. var _ usagebased.ChargeAdapter = (*adapter)(nil)). Add one per file when implementing a new sub-interface. (`var _ usagebased.RealizationRunPaymentAdapter = (*adapter)(nil)`)
**Config struct + Validate() + New() constructor** — The adapter is constructed via New(Config) which calls config.Validate() before returning. All dependencies (Client, Logger, MetaAdapter) are in Config. Never expose a raw struct literal constructor. (`func New(config Config) (usagebased.Adapter, error) { if err := config.Validate(); err != nil { return nil, err }; return &adapter{...}, nil }`)
**WithTx + Self() for entutils transaction rebinding** — The adapter implements WithTx(ctx, *TxDriver) *adapter and Self() *adapter so entutils.TransactingRepo can rebind the Ent client to the ctx-carried transaction. Both must stay in sync with every field on the adapter struct. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { txDb := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txDb.Client(), logger: a.logger, metaAdapter: a.metaAdapter} }`)
**Input validation before entering TransactingRepo** — Every public method calls input.Validate() (or charge.Validate(), etc.) before calling TransactingRepo. Validation errors escape without opening a transaction. (`if err := input.Validate(); err != nil { return nil, err }`)
**Mapper functions in mapper.go, not inline in methods** — All DB→domain conversions are delegated to named MapX functions in mapper.go (MapChargeFromDB, MapChargeBaseFromDB, MapRealizationRunFromDB, MapRealizationRunBaseFromDB). Methods call these; they do not map inline. (`return MapChargeBaseFromDB(dbUpdatedChargeBase), nil`)
**Soft-delete via DeletedAt + namespace filter on all queries** — Deletions set DeletedAt on the row (and call metaAdapter.DeleteRegisteredCharge) rather than hard-deleting. Queries must filter DeletedAtIsNil() for active records. Every query also scopes by Namespace to enforce multi-tenancy. (`Where(dbchargeusagebasedrundetailedline.DeletedAtIsNil()).Where(dbchargeusagebasedrundetailedline.NamespaceEQ(...))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Struct definition, Config/Validate/New constructor, Tx/WithTx/Self transaction plumbing | If you add a field to adapter struct, you MUST update WithTx to copy it — otherwise the tx-rebound copy loses the dependency |
| `charge.go` | CRUD for ChargeUsageBased rows: CreateCharges, GetByID, GetByIDs, UpdateCharge, DeleteCharge | expandRealizations helper builds Ent WithRuns eager-load chain; add new edges here when new run-level sub-entities are added |
| `mapper.go` | Pure DB→domain mapping functions; no Ent queries here | MapRealizationRunFromDB asserts edges loaded via OrErr — if you add a new edge to expandRealizations, add the corresponding OrErr check and map it here |
| `detailedline.go` | UpsertRunDetailedLines (soft-delete + bulk create with ON CONFLICT) and FetchDetailedLines | The ON CONFLICT clause lists specific columns; if schema changes add new upsertable fields, add the corresponding UpdateX() call in the Exec chain |
| `realizationrun.go` | CreateRealizationRun and UpdateRealizationRun for ChargeUsageBasedRuns rows | Uses mo.Option pattern (IsPresent/OrEmpty) for partial updates — only set fields that are present to avoid overwriting with zero values |
| `detailedline_test.go` | Integration test suite for UpsertRunDetailedLines upsert/soft-delete semantics; uses testutils.InitPostgresDB + real migrations | Tests bootstrap metaadapter.New directly (not app/common) to avoid import cycles; follow this pattern for new test suites here |
| `creditallocation.go` | CreateRunCreditRealization — bulk-creates credit allocation rows for a realization run | Uses lo.Map + CreateBulk — all rows share the same runID namespace; validate runID first |
| `payment.go` | CreateRunPayment and UpdateRunPayment for ChargeUsageBasedRunPayment rows | Namespace mismatch between runID and payment.InvoicedCreate is a hard error — check is explicit before TransactingRepo |

## Anti-Patterns

- Calling a.db.X directly inside a method body without TransactingRepo — bypasses ctx-carried transaction and causes partial writes
- Mapping DB rows inline inside query methods instead of delegating to mapper.go functions
- Hard-deleting rows — this package uses soft-delete (SetDeletedAt) everywhere for audit trail
- Omitting namespace filter on any Ent query — breaks multi-tenancy isolation
- Adding fields to adapter struct without updating WithTx to propagate them to the tx-rebound copy

## Decisions

- **entutils.TransactingRepo wraps every write, even helpers that receive a raw *entdb.Client** — Ent transactions propagate implicitly via ctx; bypassing rebinding causes partial writes under concurrent charge advancement (see AGENTS.md pitfalls)
- **Soft-delete with DeletedAt + metaAdapter.DeleteRegisteredCharge instead of hard DELETE** — Audit trail and idempotent realization runs require historical charge state to remain queryable
- **UpsertRunDetailedLines uses ON CONFLICT with ChildUniqueReferenceID as the idempotency key** — Rating reruns must replace existing detailed lines for the same logical line ID without creating duplicates, while preserving lines not in the new set via selective soft-delete

## Example: Add a new mutation method to the adapter following TransactingRepo discipline

```
import (
	"context"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (a *adapter) SetRunResult(ctx context.Context, runID usagebased.RealizationRunID, result usagebased.RunResult) error {
	if err := runID.Validate(); err != nil {
		return err
	}
	if err := result.Validate(); err != nil {
		return err
	}
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		_, err := tx.db.ChargeUsageBasedRuns.UpdateOneID(runID.ID).
// ...
```

<!-- archie:ai-end -->
