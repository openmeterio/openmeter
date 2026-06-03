# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing the flatfee.Adapter interface for flat-fee charge persistence. Sole owner of the ChargeFlatFee, ChargeFlatFeeRun, ChargeFlatFeeRunDetailedLine, ChargeFlatFeeRunCreditAllocations, ChargeFlatFeeRunPayment, and ChargeFlatFeeRunInvoicedUsage Ent entities; all reads and writes must go through this package.

## Patterns

**TransactingRepo wrapping on every method body** — Every exported method wraps its body with entutils.TransactingRepo (value) or TransactingRepoWithNoValue (void). Inside the closure use tx.db — never a.db — so the method rebinds to the ctx-carried transaction. Applies even to read methods. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (flatfee.Charge, error) { row, err := tx.db.ChargeFlatFee.Query()... })`)
**Tx / WithTx / Self triad** — adapter implements all three methods required by entutils.TransactingRepo: Tx(ctx) hijacks via HijackTx + NewTxDriver, WithTx(ctx, tx) reconstructs via entdb.NewTxClientFromRawConfig, Self() returns itself. Never store *entdb.Tx as a field. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { txDb := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txDb.Client(), logger: a.logger, metaAdapter: a.metaAdapter} }`)
**Config + Validate() + New() with compile-time assertion** — Constructed via a Config struct with Validate() returning errors for each missing field, then New(Config) returning the interface and error. var _ flatfee.Adapter = (*adapter)(nil) must be present in adapter.go. (`func New(config Config) (flatfee.Adapter, error) { if err := config.Validate(); err != nil { return nil, err }; return &adapter{db: config.Client, logger: config.Logger, metaAdapter: config.MetaAdapter}, nil }`)
**Input.Validate() as first statement before any DB access** — Every exported method calls input.Validate() (or charge.Validate()/charge.ManagedModel.Validate()) first and returns the error immediately, before opening a TransactingRepo block. (`if err := in.Validate(); err != nil { return nil, err }; return entutils.TransactingRepo(ctx, a, func(...) { ... })`)
**All DB↔domain conversions isolated in mapper.go** — MapChargeFlatFeeFromDB, MapChargeBaseFromDB, mapRunDetailedLineFromDB, mapRealizationsFromDB, proRatingConfigFromDB/ToDB, etc. live exclusively in mapper.go. No conversion logic in charge.go/credits.go/payment.go/usage.go/realizationrun.go. (`charge, err := MapChargeFlatFeeFromDB(entity, input.Expands)`)
**Soft-delete via DeletedAt — never hard DELETE** — Records are never removed with .Delete()/.DeleteOne(). DeleteCharge sets clock.Now() on DeletedAt and Status to StatusDeleted; UpsertDetailedLines bulk-marks superseded rows deleted with SetDeletedAt before re-inserting. (`charge.DeletedAt = lo.ToPtr(clock.Now()); charge.Status = flatfee.StatusDeleted`)
**Expand flags gate edge eager-loading** — Edges (CreditAllocations, Payment, InvoicedUsage, DetailedLines) load only when the matching meta.Expands flag is present; expandRealizations() adds WithCreditAllocations/WithInvoicedUsage/WithPayment. Accessing an unloaded edge is a programming error (NotLoadedError). (`if input.Expands.Has(meta.ExpandRealizations) { query = expandRealizations(query) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Declares Config, New constructor, and the Tx/WithTx/Self transaction triad. Only file that touches entdb.Client directly for transaction setup. | Never store *entdb.Tx as a struct field; always reconstruct the client via entdb.NewTxClientFromRawConfig so it is tx-scoped. |
| `charge.go` | CRUD for ChargeFlatFee rows. CreateCharges also calls metaAdapter.RegisterCharges within the same TransactingRepo closure to register IDs in the meta registry. | metaAdapter.RegisterCharges must be inside the TransactingRepo closure so a failure rolls back both charge rows and meta registration atomically. |
| `detailedline.go` | Upsert (soft-delete superseded rows + ON CONFLICT bulk insert) for ChargeFlatFeeRunDetailedLine, keyed on (namespace, run_id, child_unique_reference_id) WHERE deleted_at IS NULL. | ON CONFLICT must SetIgnore(FieldCreatedAt) and SetIgnore(FieldID) — omitting these overwrites immutable fields on conflict resolution. |
| `mapper.go` | All DB↔domain conversions. MapChargeFlatFeeFromDB gates edge mapping behind expands flags. | When adding a new Ent edge, add a matching meta.Expands constant and gate the mapping behind it — silently returning empty data is worse than a clear error. |
| `realizationrun.go` | Creates/updates ChargeFlatFeeRun rows (CreateCurrentRun, UpdateRealizationRun, DetachCurrentRun). | CreateCurrentRun issues a ForUpdate() lock on the charge row before creating the run to prevent duplicate current runs under concurrency. |
| `credits.go` | Bulk-creates ChargeFlatFeeRunCreditAllocations rows within the run's namespace via the creditrealization.Create helper. | Namespace comes from the parent run row; chargeID.Namespace is passed explicitly to creditrealization.Create — do not derive it from other fields. |
| `payment.go` | Creates/updates ChargeFlatFeeRunPayment (one payment row per run). UpdatePayment filters by namespace via the chargeflatfeerunpayment.Namespace predicate. | The namespace predicate on UpdatePayment is mandatory — omitting it allows cross-namespace updates. |
| `usage.go` | Creates a single ChargeFlatFeeRunInvoicedUsage row per run; also updates the run's LineID and InvoiceID in the same transaction. | No update path exists — invoiced usage is append-only. Adding an Update method would break charge finality invariants. |

## Anti-Patterns

- Using a.db directly inside a method body instead of tx.db inside a TransactingRepo closure — breaks transaction propagation from ctx.
- Hard-deleting rows with Ent .Delete()/.DeleteOne() — all deletes must set DeletedAt to preserve soft-delete semantics.
- Accessing Ent edge fields (e.g. entity.Edges.Payment) without loading them via WithPayment() — causes NotLoadedError treated as a bug.
- Adding DB↔domain conversion logic inside charge.go/credits.go/payment.go/usage.go/realizationrun.go instead of mapper.go.
- Calling metaAdapter.RegisterCharges or DeleteRegisteredCharge outside the TransactingRepo closure — breaks atomicity with the charge row mutation.

## Decisions

- **Every method wraps with entutils.TransactingRepo even if the caller already holds a transaction.** — TransactingRepo rebinds to the existing ctx-carried transaction if present and falls back to Self() otherwise, making helpers independently callable and safely nestable without leaking transaction concerns upward.
- **UpsertDetailedLines uses soft-delete of superseded rows + ON CONFLICT bulk insert rather than diff-and-patch.** — The caller provides a full replacement set keyed by ChildUniqueReferenceID; full-replace upsert is simpler and idempotent — re-running the same input yields the same DB state, safe under charge realization retries.
- **Edge loading is gated behind explicit meta.Expands flags.** — Many call sites (state machine transitions, status checks) need only ChargeBase fields; eager-loading all edges would issue unnecessary JOINs on every read, degrading the hot AdvanceCharges path.

## Example: Adding a new write method that patches a single charge field

```
import (
	"context"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	dbchargeflatfee "github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfee"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (a *adapter) PatchChargeFoo(ctx context.Context, id flatfee.ChargeID, value string) error {
	if err := id.Validate(); err != nil { return err }
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		_, err := tx.db.ChargeFlatFee.UpdateOneID(id.ID).
			Where(dbchargeflatfee.NamespaceEQ(id.Namespace)).
			SetFoo(value).Save(ctx)
		return err
	})
// ...
```

<!-- archie:ai-end -->
