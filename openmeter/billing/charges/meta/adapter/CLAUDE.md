# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing meta.Adapter for the charges meta-registry — persists the Charge cross-reference join table (linking charge IDs to their typed sub-entities: flatfee, usagebased, creditpurchase) and manages soft-delete lifecycle. All writes are transaction-aware via entutils.TransactingRepoWithNoValue.

## Patterns

**TransactingRepoWithNoValue on every write** — Every mutating method wraps its body in entutils.TransactingRepoWithNoValue(ctx, a, func(ctx, tx *adapter) error{...}), rebinding to the caller's ctx-propagated transaction or starting a new one. Using raw a.db bypasses the active tx and causes partial writes in multi-step AdvanceCharges flows. (`return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error { return tx.db.Charge.CreateBulk(creates...).Save(ctx) })`)
**Tx / WithTx / Self triad** — adapter implements all three TxUser methods: Tx(ctx) via HijackTx+NewTxDriver, WithTx(ctx, tx) via entdb.NewTxClientFromRawConfig, and Self(). All three must exist for TransactingRepoWithNoValue to rebind correctly. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { txDb := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txDb.Client(), logger: a.logger} }`)
**Config.Validate() before New()** — Config holds Client *entdb.Client and Logger *slog.Logger. New() calls config.Validate() and returns an error before constructing — never returns a broken adapter. (`func New(config Config) (meta.Adapter, error) { if err := config.Validate(); err != nil { return nil, err }; return &adapter{db: config.Client, logger: config.Logger}, nil }`)
**Compile-time interface assertion** — var _ meta.Adapter = (*adapter)(nil) at package level ensures the adapter always satisfies meta.Adapter at compile time. (`var _ meta.Adapter = (*adapter)(nil)`)
**Soft-delete with DeletedAtIsNil guard** — DeleteRegisteredCharge sets deleted_at via SetDeletedAt(clock.Now()) under a chargedb.DeletedAtIsNil() predicate — never a hard DELETE, and double-deletion becomes a no-op. (`tx.db.Charge.UpdateOneID(in.ID).Where(chargedb.DeletedAtIsNil(), chargedb.Namespace(in.Namespace)).SetDeletedAt(clock.Now()).Exec(ctx)`)
**Input Validate() first line** — Every exported method calls in.Validate() before touching the DB; input types own their own validation. (`func (a *adapter) RegisterCharges(ctx context.Context, in meta.RegisterChargesInput) error { if err := in.Validate(); err != nil { return err }; ... }`)
**Type-switched FK assignment on charge creation** — RegisterCharges switches on in.Type and calls the matching SetCharge*ID builder so exactly one FK column is populated per charge type; an unknown type returns an error. (`switch in.Type { case meta.ChargeTypeFlatFee: create = create.SetChargeFlatFeeID(charge.ID); case meta.ChargeTypeUsageBased: create = create.SetChargeUsageBasedID(charge.ID); case meta.ChargeTypeCreditPurchase: create = create.SetChargeCreditPurchaseID(charge.ID) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Defines Config, New, and the adapter struct with Tx/WithTx/Self. The only place *entdb.Client is stored; all DB access elsewhere uses tx.db obtained from WithTx. | Never store or pass a bare *entdb.Client outside Tx/WithTx/Self — it bypasses the ctx-bound transaction. |
| `charges.go` | Implements RegisterCharges (bulk-create with type-switched FK assignment via slicesx.MapWithErr) and DeleteRegisteredCharge (soft-delete). Both wrap Ent calls in TransactingRepoWithNoValue. | Any new write method must also use TransactingRepoWithNoValue; missing it drops the write out of the caller's transaction. |

## Anti-Patterns

- Calling tx.db.Charge... directly in an exported method body without TransactingRepoWithNoValue
- Hard-deleting rows (DeleteOneID) instead of soft-deleting via SetDeletedAt
- Putting business logic or validation in this adapter — logic belongs in the meta service layer
- Adding a Config field without a matching nil-check in Config.Validate()
- Removing the var _ meta.Adapter = (*adapter)(nil) compile-time assertion

## Decisions

- **TransactingRepoWithNoValue instead of direct Ent calls** — Charge advancement mixes writes across adapters; an un-rebound helper produces partial writes under concurrency. Explicit wrapping is the only compiler-enforceable guard.
- **Soft-delete with DeletedAtIsNil predicate guard** — Billing charge records are audit-critical; hard deletes lose history, and the IsNil predicate makes double-soft-delete a no-op rather than an inconsistent state.

## Example: Add a new write method that bulk-updates charge status

```
import (
	"context"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	chargedb "github.com/openmeterio/openmeter/openmeter/ent/db/charge"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (a *adapter) MarkChargesProcessed(ctx context.Context, in meta.MarkChargesProcessedInput) error {
	if err := in.Validate(); err != nil {
		return err
	}
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		_, err := tx.db.Charge.Update().
			Where(chargedb.IDIn(in.IDs...), chargedb.Namespace(in.Namespace)).
			Save(ctx)
// ...
```

<!-- archie:ai-end -->
