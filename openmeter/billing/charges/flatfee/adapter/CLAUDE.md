# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing the flatfee.Adapter interface for flat-fee charge persistence. It is the sole owner of ChargeFlatFee, ChargeFlatFeeRun, ChargeFlatFeeRunDetailedLine, ChargeFlatFeeRunCreditAllocations, ChargeFlatFeeRunPayment, and ChargeFlatFeeRunInvoicedUsage Ent entities; all reads and writes must go through this package.

## Patterns

**TransactingRepo wrapping on every method body** — Every exported method wraps its body with entutils.TransactingRepo (returning value) or entutils.TransactingRepoWithNoValue (void). Inside the closure, use tx.db — never a.db — so the method rebinds to the ctx-carried transaction. This applies even to read methods that do not open a new transaction. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (flatfee.Charge, error) { row, err := tx.db.ChargeFlatFee.Query()... })`)
**Tx / WithTx / Self triad** — adapter must implement all three methods required by entutils.TransactingRepo: Tx(ctx) hijacks via HijackTx + NewTxDriver, WithTx(ctx, tx) reconstructs adapter with entdb.NewTxClientFromRawConfig, Self() returns itself. Never store *entdb.Tx as a field. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { txDb := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txDb.Client(), logger: a.logger, metaAdapter: a.metaAdapter} }`)
**Config struct + Validate() + New() constructor with compile-time assertion** — Every adapter is constructed via a Config struct with a Validate() method that returns errors for each missing field, followed by New(Config) returning the interface and error. The var _ flatfee.Adapter = (*adapter)(nil) compile-time assertion must be present in adapter.go. (`func New(config Config) (flatfee.Adapter, error) { if err := config.Validate(); err != nil { return nil, err }; return &adapter{db: config.Client, logger: config.Logger, metaAdapter: config.MetaAdapter}, nil }`)
**Input.Validate() as the first statement before any DB access** — Every exported method calls input.Validate() (or charge.Validate()) as the first statement and returns the error immediately, before opening a TransactingRepo block. This prevents invalid inputs from reaching the DB. (`func (a *adapter) CreateCharges(ctx context.Context, in flatfee.CreateChargesInput) ([]flatfee.Charge, error) { if err := in.Validate(); err != nil { return nil, err }; return entutils.TransactingRepo(ctx, a, func(...) { ... }) }`)
**All DB↔domain conversions isolated in mapper.go** — MapChargeFlatFeeFromDB, MapChargeBaseFromDB, mapRunDetailedLineFromDB, mapRealizationsFromDB, proRatingConfigFromDB/ToDB, and all other conversion helpers live exclusively in mapper.go. No conversion logic is permitted inside charge.go, credits.go, payment.go, usage.go, or realizationrun.go. (`charge, err := MapChargeFlatFeeFromDB(entity, input.Expands)`)
**Soft-delete via DeletedAt field — never hard DELETE** — Records are never removed with Ent .Delete() or .DeleteOne(). DeleteCharge sets clock.Now() on DeletedAt and updates Status to StatusDeleted. UpsertDetailedLines bulk-marks superseded rows deleted with SetDeletedAt before re-inserting replacements. (`charge.DeletedAt = lo.ToPtr(clock.Now()); charge.Status = flatfee.StatusDeleted; update = update.SetStatusDetailed(charge.Status)`)
**Expand flags gate edge eager-loading** — Edges (CreditAllocations, Payment, InvoicedUsage, DetailedLines) are loaded only when the corresponding meta.Expands flag is present. The expandRealizations() helper adds WithCreditAllocations/WithInvoicedUsage/WithPayment. Accessing an unloaded edge returns NotLoadedError which is treated as a programming error. (`if input.Expands.Has(meta.ExpandRealizations) { query = expandRealizations(query) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Declares Config, New constructor, and the Tx/WithTx/Self transaction triad. Only file that touches entdb.Client directly for transaction setup. | Never store *entdb.Tx as a struct field; always reconstruct the client via entdb.NewTxClientFromRawConfig so it is tx-scoped. |
| `charge.go` | CRUD for ChargeFlatFee rows. CreateCharges also calls metaAdapter.RegisterCharges within the same TransactingRepo closure to register IDs in the meta registry. | metaAdapter.RegisterCharges must be inside the TransactingRepo closure so a failure rolls back both charge rows and meta registration atomically. |
| `detailedline.go` | Upsert (soft-delete superseded rows + ON CONFLICT bulk insert) for ChargeFlatFeeRunDetailedLine, keyed on (namespace, run_id, child_unique_reference_id) WHERE deleted_at IS NULL. | ON CONFLICT must SetIgnore(FieldCreatedAt) and SetIgnore(FieldID) — omitting these would overwrite immutable fields on conflict resolution. |
| `mapper.go` | All DB↔domain type conversions. MapChargeFlatFeeFromDB gates edge mapping behind expands flags; edges accessed without loading cause NotLoadedError. | When adding a new Ent edge, add a corresponding meta.Expands constant and gate the mapping behind it — silently returning empty data is worse than a clear error. |
| `realizationrun.go` | Creates and updates ChargeFlatFeeRun rows (current run lifecycle: CreateCurrentRun, UpdateRealizationRun, DetachCurrentRun). | CreateCurrentRun issues a ForUpdate() lock on the charge row before creating the run to prevent duplicate current runs under concurrency. |
| `credits.go` | Bulk-creates ChargeFlatFeeRunCreditAllocations rows within the run's namespace using creditrealization.Create helper. | Namespace enforcement comes from the parent run row; chargeID.Namespace is passed explicitly to creditrealization.Create — do not derive it from other fields. |
| `payment.go` | Creates and updates ChargeFlatFeeRunPayment (one payment row per run). UpdatePayment filters by namespace via chargeflatfeerunpayment.Namespace predicate. | The namespace predicate on UpdatePayment is mandatory — omitting it allows cross-namespace updates. |
| `usage.go` | Creates a single ChargeFlatFeeRunInvoicedUsage row per run. Also updates the run's LineID and InvoiceID in the same transaction. | No update path exists — invoiced usage is append-only. Adding an Update method would break charge finality invariants. |

## Anti-Patterns

- Using a.db directly inside a method body instead of tx.db inside a TransactingRepo closure — breaks transaction propagation from ctx.
- Hard-deleting rows with Ent .Delete() / .DeleteOne() — all deletes must set DeletedAt to preserve soft-delete semantics.
- Accessing Ent edge fields (e.g. entity.Edges.Payment) without loading them via WithPayment() — causes NotLoadedError treated as a programming bug.
- Adding DB↔domain conversion logic inside charge.go / credits.go / payment.go / usage.go / realizationrun.go instead of mapper.go.
- Calling metaAdapter.RegisterCharges or metaAdapter.DeleteRegisteredCharge outside the TransactingRepo closure — breaks atomicity with the charge row mutation.

## Decisions

- **Every method wraps with entutils.TransactingRepo even if the caller already holds a transaction.** — TransactingRepo rebinds to the existing ctx-carried transaction if one is present and falls back to Self() otherwise, making helpers independently callable and safely nestable without leaking transaction concerns upward.
- **UpsertDetailedLines uses soft-delete of superseded rows + ON CONFLICT bulk insert rather than diff-and-patch.** — The caller provides a full replacement set keyed by ChildUniqueReferenceID; a full-replace upsert is simpler and idempotent — re-running the same input produces the same DB state, which is safe under the charge realization retry pattern.
- **Edge loading (realizations, detailed lines) is gated behind explicit meta.Expands flags.** — Many call sites (state machine transitions, status checks) only need ChargeBase fields; eager-loading all edges would issue unnecessary JOINs on every read, degrading performance on the hot AdvanceCharges path.

## Example: Adding a new write method that patches a single charge field

```
import (
	"context"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	dbchargeflatfee "github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfee"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (a *adapter) PatchChargeFoo(ctx context.Context, id flatfee.ChargeID, value string) error {
	if err := id.Validate(); err != nil {
		return err
	}
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		_, err := tx.db.ChargeFlatFee.UpdateOneID(id.ID).
			Where(dbchargeflatfee.NamespaceEQ(id.Namespace)).
			SetFoo(value).
// ...
```

<!-- archie:ai-end -->
