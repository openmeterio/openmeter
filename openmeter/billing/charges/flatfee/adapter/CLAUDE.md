# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing the flatfee.Adapter interface for flat-fee charge persistence. All reads and writes go through this package; it is the only place that touches the ChargeFlatFee, ChargeFlatFeeCreditAllocations, ChargeFlatFeeDetailedLine, ChargeFlatFeePayment, and ChargeFlatFeeInvoicedUsage Ent entities.

## Patterns

**TransactingRepo wrapping on every write method** — Every mutating method (CreateCharges, UpdateCharge, DeleteCharge, UpsertDetailedLines, CreateCreditAllocations, CreatePayment, UpdatePayment, CreateInvoicedUsage) wraps its body with entutils.TransactingRepo or entutils.TransactingRepoWithNoValue so the method rebinds to the ctx-carried transaction. Using tx.db (not a.db) inside the closure is mandatory. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (flatfee.Charge, error) { ... tx.db.ChargeFlatFee... })`)
**Config struct + Validate() + New() constructor** — Construction follows the pattern: define a Config struct, implement Config.Validate() returning an error for each missing field, then New(Config) returns the interface and error. The var _ flatfee.Adapter = (*adapter)(nil) compile-time check must be present. (`func New(config Config) (flatfee.Adapter, error) { if err := config.Validate(); err != nil { return nil, err }; return &adapter{...}, nil }`)
**Tx / WithTx / Self triad for transaction propagation** — adapter must implement Tx(ctx) to hijack a new transaction, WithTx(ctx, *TxDriver) to re-create itself with a tx-bound *entdb.Client, and Self() returning itself. These three methods are the contract required by entutils.TransactingRepo. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { txDb := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txDb.Client(), ...} }`)
**Input Validate() before any DB access** — Every exported adapter method calls input.Validate() (or charge.Validate()) as the first statement and returns its error immediately, before opening a transaction. (`func (a *adapter) CreateCharges(ctx context.Context, in flatfee.CreateChargesInput) ([]flatfee.Charge, error) { if err := in.Validate(); err != nil { return nil, err }; ... }`)
**Mapper functions isolated in mapper.go** — All DB-to-domain and domain-to-DB conversions live in mapper.go (MapChargeFlatFeeFromDB, MapChargeBaseFromDB, mapDetailedLineFromDB, proRatingConfigFromDB, proRatingConfigToDB). No conversion logic inside charge.go / credits.go / payment.go / usage.go. (`return MapChargeFlatFeeFromDB(entity, input.Expands)`)
**Soft-delete via DeletedAt field, not hard DELETE** — Records are never hard-deleted. DeleteCharge sets clock.Now() on DeletedAt and updates Status. UpsertDetailedLines marks superseded rows deleted via a bulk Update().SetDeletedAt(now) before re-inserting replacements. (`charge.DeletedAt = lo.ToPtr(clock.Now()); charge.Status = flatfee.StatusDeleted; update.SetStatusDetailed(charge.Status)`)
**Expand flags gate edge loading** — Edges (CreditAllocations, Payment, InvoicedUsage, DetailedLines) are loaded only when the corresponding meta.Expands flag is present. expandRealizations() helper adds WithCreditAllocations/WithInvoicedUsage/WithPayment to the query. (`if input.Expands.Has(meta.ExpandRealizations) { query = expandRealizations(query) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Declares Config, New constructor, and the Tx/WithTx/Self transaction triad. The only file that touches entdb.Client directly for transaction setup. | Never store *entdb.Tx here; always reconstruct via entdb.NewTxClientFromRawConfig so the client is tx-scoped. |
| `charge.go` | CRUD for ChargeFlatFee rows. CreateCharges also calls metaAdapter.RegisterCharges to register IDs in the meta registry within the same transaction. | metaAdapter.RegisterCharges must be called inside the TransactingRepo closure so a failure rolls back both the charge rows and the meta registration. |
| `detailedline.go` | Upsert (soft-delete old + bulk insert new) for ChargeFlatFeeDetailedLine using ON CONFLICT on (namespace, charge_id, child_unique_reference_id) where deleted_at IS NULL. | The conflict resolution explicitly ignores created_at and id fields; omitting SetIgnore for these would overwrite immutable fields on upsert. |
| `mapper.go` | All DB↔domain type conversions. MapChargeFlatFeeFromDB gates edge mapping behind expands flags; accessing an unloaded edge returns NotLoadedError which is treated as a programming error. | When adding a new Ent edge, add a corresponding expand flag and gate mapping behind it — returning empty data silently is worse than a clear error. |
| `credits.go` | Creates ChargeFlatFeeCreditAllocations rows in bulk via creditrealization.Create helper. | chargeID is passed separately from creditAllocation inputs; the DB entity only stores chargeID.ID (not the namespace separately) — namespace enforcement is on the parent charge row. |
| `payment.go` | Creates and updates ChargeFlatFeePayment (one payment row per charge). Uses payment.CreateInvoiced and payment.UpdateInvoiced helpers from the models/payment sub-package. | UpdatePayment filters by namespace via chargeflatfeepayment.Namespace predicate — omitting this allows cross-namespace updates. |
| `usage.go` | Creates a single ChargeFlatFeeInvoicedUsage row per charge via invoicedusage.Create helper. | No update path — invoiced usage is append-only; adding an Update method here would break charge finality invariants. |
| `detailedline_test.go` | Integration test using testutils.InitPostgresDB + real migrations. Tests UpsertDetailedLines upsert/soft-delete semantics end-to-end. | Uses t.Context() (not context.Background()) throughout. Tests verify the DB row directly after the upsert to confirm soft-delete rather than trusting the return value alone. |

## Anti-Patterns

- Using a.db directly inside a method body instead of tx.db inside a TransactingRepo closure — breaks transaction propagation from ctx.
- Hard-deleting rows with Ent .Delete() / .DeleteOne() — all deletes must set DeletedAt.
- Accessing Ent edge fields (e.g. entity.Edges.Payment) without loading them via WithPayment() — causes a panic or NotLoadedError.
- Adding conversion logic (DB↔domain mapping) inside charge.go / credits.go / payment.go / usage.go instead of mapper.go.
- Calling metaAdapter methods outside the TransactingRepo closure — the meta registration would not roll back with the charge row on failure.

## Decisions

- **All write helpers wrap with entutils.TransactingRepo even when the caller already holds a transaction.** — Ent transactions propagate via ctx; TransactingRepo rebinds to the existing tx if present, so nesting is safe and each helper is independently callable without leaking tx concerns to callers.
- **Upsert of DetailedLines uses soft-delete + ON CONFLICT bulk insert rather than diff-and-patch.** — The caller provides a full replacement set keyed by ChildUniqueReferenceID; a full-replace upsert is simpler and idempotent — re-running the same input produces the same DB state.
- **Edge loading (realizations, detailed lines) is gated behind explicit meta.Expands flags rather than always eager-loading.** — Many call sites (e.g. state machine transitions) only need ChargeBase fields; eager-loading all edges would issue unnecessary JOINs on every read.

## Example: Adding a new write method to the adapter (e.g. patch a charge field)

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
