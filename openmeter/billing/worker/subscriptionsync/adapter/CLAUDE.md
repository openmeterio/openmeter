# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing subscriptionsync.Adapter and SyncStateAdapter — persists SubscriptionBillingSyncState records (upsert, get, invalidate) with full ctx-bound transaction support via entutils.TransactingRepo. Pure data access; no orchestration.

## Patterns

**TransactingRepo wrapping all DB writes** — Every DB-touching method calls entutils.TransactingRepo or TransactingRepoWithNoValue so the ctx-bound transaction is honored rather than falling off to the raw client. (`entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error { return tx.db.SubscriptionBillingSyncState.Delete()... })`)
**Config + Validate constructor** — New takes a Config, calls config.Validate() accumulating errors via errors.Join, and returns (Interface, error). (`func New(config Config) (subscriptionsync.Adapter, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**Tx / WithTx / Self triad** — Implement TxCreator: Tx() starts a hijacked tx, WithTx() rebinds the adapter to a TxDriver, Self() returns self; required for TransactingRepo. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { txDb := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txDb.Client()} }`)
**Interface compliance assertion** — Compile-time assertion at the top of each implementation file catches interface drift early. (`var _ subscriptionsync.SyncStateAdapter = (*adapter)(nil)`)
**UTC normalization on time fields** — All DB time values are normalized to UTC via .UTC() / lo.ToPtr(t.UTC()) to prevent timezone drift. (`nextSyncAfter = lo.ToPtr(nextSyncAfter.UTC())`)
**Upsert with OnConflictColumns** — Upsert uses Ent's OnConflictColumns(...).UpdateXxx() keyed on (subscription_id, namespace) to make UpsertSyncState idempotent. (`.OnConflictColumns(subscriptionbillingsyncstate.FieldSubscriptionID, subscriptionbillingsyncstate.FieldNamespace).UpdateHasBillables().UpdateSyncedAt().UpdateNextSyncAfter()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Config, adapter struct, New constructor, Tx/WithTx/Self triad — entry point for adapter construction. | Adding business logic here; all logic belongs in syncstate.go or domain files. |
| `syncstate.go` | Implements SyncStateAdapter: InvalidateSyncState, GetSyncStates, UpsertSyncState over the SubscriptionBillingSyncState entity. | Forgetting the TransactingRepo wrapper bypasses ctx-bound transactions; missing UTC() normalization on time fields. |

## Anti-Patterns

- Calling tx.db directly without wrapping in entutils.TransactingRepo/TransactingRepoWithNoValue.
- Storing or returning time values without .UTC() normalization.
- Adding domain or sync orchestration logic into the adapter (belongs in the service layer).
- Skipping the compile-time interface assertion var _ Interface = (*adapter)(nil).
- Constructing the adapter without a Config.Validate() call.

## Decisions

- **Adapter is scoped exclusively to SubscriptionBillingSyncState persistence; no subscription/invoice business logic.** — Separation of concerns: the service layer owns orchestration, the adapter owns data access, keeping it independently testable.
- **TransactingRepo is used even for single-statement operations.** — Callers may already have a transaction in ctx; rebinding ensures atomicity with surrounding operations and prevents partial writes.

## Example: Adding a new adapter method that writes to DB

```
import (
	"context"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)
func (a *adapter) SetFoo(ctx context.Context, input subscriptionsync.SetFooInput) error {
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		return tx.db.SubscriptionBillingSyncState.Update().SetFoo(input.Foo).Where(subscriptionbillingsyncstate.SubscriptionID(input.ID)).Exec(ctx)
	})
}
```

<!-- archie:ai-end -->
