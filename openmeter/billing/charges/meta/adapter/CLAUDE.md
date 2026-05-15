# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing meta.Adapter for the charges meta-registry — persists charge cross-reference records (the Charge join table linking charge IDs to their typed sub-entities) and manages their soft-delete lifecycle. All DB writes are transaction-aware via entutils.TransactingRepoWithNoValue.

## Patterns

**TransactingRepoWithNoValue on every write** — Every mutating method calls entutils.TransactingRepoWithNoValue(ctx, a, func(ctx, tx *adapter) error{...}). This rebinds to the caller's ctx-propagated transaction or starts a new one. Using the raw a.db directly bypasses the active tx. (`return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error { return tx.db.Charge.Create()...Save(ctx) })`)
**Tx / WithTx / Self triad** — adapter implements all three methods required by entutils.TxUser[T]: Tx(ctx) via HijackTx+NewTxDriver, WithTx(ctx, tx) via NewTxClientFromRawConfig, and Self(). All three must be present for TransactingRepoWithNoValue to rebind correctly. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { txDb := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txDb.Client(), logger: a.logger} }`)
**Config struct with Validate() before New()** — Config holds Client *entdb.Client and Logger *slog.Logger. New() calls config.Validate() and returns error before constructing — never returns a broken adapter. (`func New(config Config) (meta.Adapter, error) { if err := config.Validate(); err != nil { return nil, err }; return &adapter{db: config.Client, logger: config.Logger}, nil }`)
**Compile-time interface assertion** — var _ meta.Adapter = (*adapter)(nil) at package level ensures adapter always satisfies meta.Adapter at compile time. (`var _ meta.Adapter = (*adapter)(nil)`)
**Soft-delete with DeletedAtIsNil guard** — DeleteRegisteredCharge sets deleted_at via SetDeletedAt(clock.Now()) with a chargedb.DeletedAtIsNil() predicate — never issues a hard DELETE and prevents double-deletion. (`tx.db.Charge.UpdateOneID(in.ID).Where(chargedb.DeletedAtIsNil(), chargedb.Namespace(in.Namespace)).SetDeletedAt(clock.Now()).Exec(ctx)`)
**Input Validate() before any Ent call** — Every exported method calls in.Validate() as the first line, returning its error before touching the DB. Input types own their own validation. (`func (a *adapter) RegisterCharges(ctx context.Context, in meta.RegisterChargesInput) error { if err := in.Validate(); err != nil { return err }; ... }`)
**Type-switched FK assignment on charge creation** — RegisterCharges switches on in.Type (FlatFee/UsageBased/CreditPurchase) and calls the matching SetCharge*ID builder method, ensuring exactly one FK column is populated per charge type. (`switch in.Type { case meta.ChargeTypeFlatFee: create = create.SetChargeFlatFeeID(charge.ID); case meta.ChargeTypeUsageBased: create = create.SetChargeUsageBasedID(charge.ID); ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Defines Config, New constructor, and the adapter struct with Tx/WithTx/Self. The only place where *entdb.Client is stored; all DB access elsewhere goes through tx.db obtained from WithTx. | Never store or pass a bare *entdb.Client outside Tx/WithTx/Self — doing so bypasses the ctx-bound transaction. |
| `charges.go` | Implements RegisterCharges (bulk-create with type-switched FK assignment) and DeleteRegisteredCharge (soft-delete). Both wrap Ent calls in TransactingRepoWithNoValue. | Any new write method must also use TransactingRepoWithNoValue. Missing this wrapping causes writes to fall outside the caller's transaction and produces partial writes in multi-step AdvanceCharges flows. |

## Anti-Patterns

- Calling tx.db.Charge... directly in an exported method body without TransactingRepoWithNoValue — bypasses the ctx transaction
- Hard-deleting rows (tx.db.Charge.DeleteOneID) instead of soft-deleting via SetDeletedAt
- Storing business logic or validation in this adapter — logic belongs in the meta service layer
- Adding new fields to Config without a corresponding nil-check in Config.Validate()
- Removing or bypassing the var _ meta.Adapter = (*adapter)(nil) compile-time assertion

## Decisions

- **TransactingRepoWithNoValue instead of direct Ent calls** — Charge advancement mixes multiple writes across adapters; if any helper uses the raw client instead of rebinding to ctx's transaction, partial writes occur under concurrency. Explicit wrapping is the only compiler-enforceable guard.
- **Separate Config struct with Validate() before construction** — Fails fast with a descriptive error rather than a nil-pointer panic at first DB call, which could be deep inside a billing transaction.
- **Soft-delete with DeletedAtIsNil predicate guard** — Billing charge records are audit-critical; hard deletes lose history. The IsNil predicate makes double-soft-delete a no-op rather than an error-prone inconsistent state.

## Example: Add a new write method that bulk-updates charge status

```
import (
	"context"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	chargedb "github.com/openmeterio/openmeter/openmeter/ent/db/charge"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (a *adapter) MarkChargesProcessed(ctx context.Context, in meta.MarkChargesProcessedInput) error {
	if err := in.Validate(); err != nil {
		return err
	}
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		_, err := tx.db.Charge.Update().
			Where(chargedb.IDIn(in.IDs...), chargedb.Namespace(in.Namespace)).
// ...
```

<!-- archie:ai-end -->
