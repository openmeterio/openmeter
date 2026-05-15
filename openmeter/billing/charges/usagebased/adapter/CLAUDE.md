# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing usagebased.Adapter for the usage-based charge lifecycle. Persists charges, realization runs, credit allocations, invoiced usage, payments, and detailed lines — all within context-propagated Ent transactions via entutils.TransactingRepo.

## Patterns

**TransactingRepo on every mutating method** — Every public method that writes to DB wraps its body with entutils.TransactingRepo (returns value) or entutils.TransactingRepoWithNoValue (void). Never call a.db.X directly in a method body — always go through the tx-rebound client inside the closure. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (usagebased.ChargeBase, error) { return MapChargeBaseFromDB(tx.db.ChargeUsageBased.UpdateOneID(id).Save(ctx)) })`)
**Validate inputs before opening a transaction** — Every public method calls input.Validate() (or charge.Validate(), runID.Validate(), etc.) before the TransactingRepo call. Validation errors escape without starting a DB transaction. (`if err := input.Validate(); err != nil { return nil, err }
return entutils.TransactingRepo(ctx, a, func(...) { ... })`)
**Tx/WithTx/Self triad for ctx-propagated transactions** — adapter implements Tx() (HijackTx + NewTxDriver), WithTx() (NewTxClientFromRawConfig), and Self(). All three must stay in sync with every field on the adapter struct — if a new field is added to adapter, WithTx must copy it to the tx-rebound copy. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { txDb := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txDb.Client(), logger: a.logger, metaAdapter: a.metaAdapter} }`)
**Compile-time interface assertions per file** — Each file declares a blank var assertion for the sub-interface it implements (e.g. var _ usagebased.ChargeAdapter = (*adapter)(nil)). Add one per file when implementing a new sub-interface. (`var _ usagebased.RealizationRunPaymentAdapter = (*adapter)(nil)`)
**Config struct + Validate() + New() constructor** — The adapter is always constructed via New(Config) which calls config.Validate() before returning. All dependencies live in Config. Never instantiate &adapter{} directly from outside the package. (`func New(config Config) (usagebased.Adapter, error) { if err := config.Validate(); err != nil { return nil, err }; return &adapter{...}, nil }`)
**Mapper functions in mapper.go, not inline** — All DB→domain conversions are delegated to named functions in mapper.go (MapChargeFromDB, MapChargeBaseFromDB, MapRealizationRunFromDB, MapRealizationRunBaseFromDB). Methods call these functions; they do not map inline. (`return MapChargeBaseFromDB(dbUpdatedChargeBase), nil`)
**Soft-delete via DeletedAt + namespace filter on all queries** — Deletions set DeletedAt rather than hard-deleting rows; active-record queries must filter DeletedAtIsNil(). Every Ent query also scopes by Namespace to enforce multi-tenancy. (`Where(dbchargeusagebasedrundetailedline.DeletedAtIsNil()).Where(dbchargeusagebasedrundetailedline.NamespaceEQ(ns))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Struct definition, Config/Validate/New constructor, and the Tx/WithTx/Self transaction plumbing — the core scaffolding every other file depends on. | Adding a field to the adapter struct without updating WithTx to copy it: the tx-rebound copy will silently lose the dependency and cause nil-pointer panics under concurrent charge advancement. |
| `charge.go` | CRUD for ChargeUsageBased rows: CreateCharges, GetByID, GetByIDs, UpdateCharge, DeleteCharge. | expandRealizations builds the WithRuns eager-load chain; if a new run-level sub-entity is added to the schema, add its WithX() edge here and its OrErr() check + mapping in mapper.go. |
| `mapper.go` | Pure DB→domain mapping functions with no Ent queries — the single source of mapping truth for this package. | MapRealizationRunFromDB uses OrErr() on every edge; if expandRealizations adds a new edge, this function must add the corresponding OrErr() check or it will silently return an error on every run expansion. |
| `detailedline.go` | UpsertRunDetailedLines (soft-delete existing lines then bulk-create with ON CONFLICT) and FetchDetailedLines. | The ON CONFLICT clause lists specific columns; if the schema gains new upsertable fields, add the corresponding UpdateX() call in the Exec chain. FetchDetailedLines uses the DetailedLinesPresent flag as the authority — never infer presence from row count alone. |
| `realizationrun.go` | CreateRealizationRun and UpdateRealizationRun for ChargeUsageBasedRuns rows. | UpdateRealizationRunInput uses mo.Option (IsPresent/OrEmpty) for partial updates — only set fields that are present to avoid overwriting with zero values; new fields must follow this pattern. |
| `payment.go` | CreateRunPayment and UpdateRunPayment for ChargeUsageBasedRunPayment rows. | Explicit namespace mismatch check between runID.Namespace and payment.InvoicedCreate.Namespace is a hard error before TransactingRepo — preserve this guard for any new methods. |
| `creditallocation.go` | CreateRunCreditRealization — bulk-creates credit allocation rows for a realization run using lo.Map + CreateBulk. | All rows share the runID's namespace; validate runID first and ensure new bulk builders inherit the same namespace. |
| `detailedline_test.go` | Integration test suite for UpsertRunDetailedLines upsert/soft-delete semantics, using testutils.InitPostgresDB and real Atlas migrations. | Tests bootstrap metaadapter.New directly (not app/common) to avoid import cycles — follow this pattern for any new test suites added here. |

## Anti-Patterns

- Calling a.db.X directly inside a method body without TransactingRepo — bypasses the ctx-carried Ent transaction and causes partial writes under concurrent charge advancement.
- Mapping DB rows inline inside query methods instead of delegating to named functions in mapper.go.
- Hard-deleting rows — this package uses soft-delete (SetDeletedAt) everywhere for audit trail and idempotent realization runs.
- Adding a field to the adapter struct without updating WithTx to propagate it to the tx-rebound copy.
- Omitting namespace filter on any Ent query — breaks multi-tenancy isolation between customers.

## Decisions

- **entutils.TransactingRepo wraps every write, even helpers that operate on a raw *entdb.Client** — Ent transactions propagate implicitly via ctx; bypassing rebinding causes partial writes in the multi-step AdvanceCharges / ApplyPatches flows (documented pitfall ctx-002).
- **Soft-delete with DeletedAt + metaAdapter.DeleteRegisteredCharge instead of hard DELETE** — Audit trail and idempotent realization reruns require historical charge state to remain queryable; hard deletes would break the realization run history.
- **UpsertRunDetailedLines uses ON CONFLICT keyed on (namespace, chargeID, runID, ChildUniqueReferenceID) with DeletedAt IS NULL partial index** — Rating reruns must replace existing detailed lines for the same logical line ID without creating duplicates, while selectively soft-deleting lines that no longer appear in the new set.

## Example: Add a new mutation method following TransactingRepo discipline

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
