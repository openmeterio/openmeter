# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing subscriptionsync.Adapter and SyncStateAdapter interfaces. Manages persistence of SubscriptionBillingSyncState records (upsert, get, invalidate) with full ctx-bound transaction support via entutils.TransactingRepo.

## Patterns

**TransactingRepo wrapping all DB writes** — Every method that touches the DB must call entutils.TransactingRepo or TransactingRepoWithNoValue so the ctx-bound transaction is honored rather than falling off to the raw client. (`entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error { return tx.db.SubscriptionBillingSyncState.Delete()... })`)
**Config + Validate constructor pattern** — Constructor takes a Config struct, calls config.Validate() which accumulates errors with errors.Join, and returns (Interface, error). (`func New(config Config) (subscriptionsync.Adapter, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**Tx / WithTx / Self triad** — adapter implements TxCreator by providing Tx() (starts a new hijacked tx), WithTx() (rebinds adapter to a given TxDriver), and Self() (returns self for non-tx case). This triad is required for entutils.TransactingRepo to work. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { txDb := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txDb.Client()} }`)
**Interface compliance assertion** — Compile-time assertion var _ subscriptionsync.SyncStateAdapter = (*adapter)(nil) placed at the top of each implementation file. (`var _ subscriptionsync.SyncStateAdapter = (*adapter)(nil)`)
**UTC normalization on time fields** — All time values read from or written to the DB are normalized to UTC via .UTC() or lo.ToPtr(t.UTC()) to prevent timezone drift in stored timestamps. (`nextSyncAfter = lo.ToPtr(nextSyncAfter.UTC())`)
**Upsert with OnConflictColumns** — Upsert operations use Ent's OnConflictColumns(...).UpdateXxx() pattern keyed on (subscription_id, namespace) to make UpsertSyncState idempotent. (`.OnConflictColumns(subscriptionbillingsyncstate.FieldSubscriptionID, subscriptionbillingsyncstate.FieldNamespace).UpdateHasBillables().UpdateSyncedAt().UpdateNextSyncAfter()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Defines Config, adapter struct, New constructor, and the Tx/WithTx/Self transaction triad. Entry point for all adapter construction. | Adding business logic here; all logic belongs in syncstate.go or equivalent domain files. |
| `syncstate.go` | Implements SyncStateAdapter: InvalidateSyncState, GetSyncStates, UpsertSyncState. Uses Ent SubscriptionBillingSyncState entity. | Forgetting TransactingRepo wrapper — raw tx.db calls outside a TransactingRepo will bypass ctx-bound transactions. Missing UTC() normalization on time fields. |

## Anti-Patterns

- Calling tx.db directly in a method body without wrapping in entutils.TransactingRepo/TransactingRepoWithNoValue
- Storing or returning time values without .UTC() normalization
- Adding domain or sync orchestration logic into the adapter (belongs in subscriptionsync service layer)
- Skipping the compile-time interface assertion var _ Interface = (*adapter)(nil)
- Constructing adapter without Config.Validate() call

## Decisions

- **Adapter is scoped exclusively to SubscriptionBillingSyncState persistence; no subscription or invoice business logic lives here.** — Separation of concerns: the service layer in subscriptionsync owns orchestration; the adapter owns only data access, keeping it independently testable.
- **TransactingRepo is used even for single-statement operations.** — Callers may already have a transaction in ctx; rebinding ensures atomicity with surrounding operations and prevents partial writes.

## Example: Adding a new adapter method that writes to DB

```
import (
	"context"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (a *adapter) SetFoo(ctx context.Context, input subscriptionsync.SetFooInput) error {
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		return tx.db.SubscriptionBillingSyncState.Update().
			SetFoo(input.Foo).
			Where(subscriptionbillingsyncstate.SubscriptionID(input.ID)).
			Exec(ctx)
	})
}
```

<!-- archie:ai-end -->
