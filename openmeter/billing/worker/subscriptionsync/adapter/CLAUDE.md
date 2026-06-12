# adapter

<!-- archie:ai-start -->

> Ent-backed persistence adapter for the subscription→billing sync bridge; stores and reads per-subscription SubscriptionBillingSyncState (last sync time, billables flag, next-sync cooldown) used by the reconciler to skip up-to-date subscriptions. Implements subscriptionsync.Adapter and subscriptionsync.SyncStateAdapter.

## Patterns

**Config + Validate + New constructor** — Adapter is constructed via New(Config) returning the interface type; Config.Validate() collects errors into []error and returns errors.Join(errs...), requiring Client non-nil before constructing. (`func New(config Config) (subscriptionsync.Adapter, error) { if err := config.Validate(); err != nil { return nil, err }; return &adapter{db: config.Client}, nil }`)
**Transaction hijacking via Tx/WithTx/Self** — adapter implements the transaction.Repo-style contract: Tx hijacks via db.HijackTx wrapping the eDriver in entutils.NewTxDriver; WithTx rebinds db with entdb.NewTxClientFromRawConfig; Self returns the receiver. (`txCtx, rawConfig, eDriver, err := a.db.HijackTx(ctx, &sql.TxOptions{ReadOnly: false})`)
**TransactingRepo wrapping for all queries** — Every SyncStateAdapter method body is wrapped in entutils.TransactingRepo / TransactingRepoWithNoValue so it rebinds to the tx in ctx; the closure receives a tx-scoped *adapter and uses tx.db, never a.db. (`return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error { ... tx.db.SubscriptionBillingSyncState... })`)
**Compile-time interface assertion** — syncstate.go asserts `var _ subscriptionsync.SyncStateAdapter = (*adapter)(nil)` so missing methods fail compilation. (`var _ subscriptionsync.SyncStateAdapter = (*adapter)(nil)`)
**FromDB mapping normalizes to UTC** — mapSyncStateFromDB converts every *entdb.SubscriptionBillingSyncState timestamp to .UTC() (SyncedAt and nillable NextSyncAfter) before returning the domain SyncState. (`SyncedAt: state.SyncedAt.UTC(), NextSyncAfter: lo.ToPtr(nextSyncAfter.UTC())`)
**Upsert via OnConflictColumns on (subscription_id, namespace)** — UpsertSyncState validates input, then Creates with OnConflictColumns(FieldSubscriptionID, FieldNamespace) + UpdateHasBillables/UpdateSyncedAt/UpdateNextSyncAfter, normalizing all times to UTC. (`OnConflictColumns(subscriptionbillingsyncstate.FieldSubscriptionID, subscriptionbillingsyncstate.FieldNamespace).UpdateHasBillables()...`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Config/New constructor and the Tx/WithTx/Self transaction-hijacking plumbing over *entdb.Client. | Tx uses ReadOnly:false — do not assume read-only; WithTx must rebind db from the raw tx config, not reuse a.db. |
| `syncstate.go` | SyncStateAdapter implementation: InvalidateSyncState (delete), GetSyncStates (batch OR-filter by namespaced IDs), UpsertSyncState (conflict upsert), and mapSyncStateFromDB. | GetSyncStates builds an OR over per-ID (SubscriptionID AND Namespace) predicates via lo.Map — always scope by both fields; always normalize times to UTC on read and write. |

## Anti-Patterns

- Querying with a.db directly inside a method instead of using the tx-scoped tx.db from TransactingRepo — breaks transaction correctness.
- Filtering SubscriptionBillingSyncState by SubscriptionID without also constraining Namespace — leaks across tenants.
- Returning non-UTC timestamps from FromDB mapping — downstream cooldown comparisons assume UTC.
- Adding a new SyncStateAdapter method without preserving the `var _ subscriptionsync.SyncStateAdapter = (*adapter)(nil)` assertion.

## Decisions

- **Persist sync state in a dedicated SubscriptionBillingSyncState table keyed by (subscription_id, namespace) with HasBillables and NextSyncAfter.** — Lets the reconciler cheaply skip subscriptions with no billables or still in cooldown rather than re-running full sync for every active subscription.
- **Use HijackTx + TransactingRepo rather than passing a tx client through arguments.** — Matches the repo-wide billing adapter convention so the adapter rebinds to a tx already carried in ctx, keeping helpers transaction-aware.

## Example: Upsert sync state with conflict handling and UTC normalization

```
return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
  nextSyncAfter := input.NextSyncAfter
  if nextSyncAfter != nil { nextSyncAfter = lo.ToPtr(nextSyncAfter.UTC()) }
  return tx.db.SubscriptionBillingSyncState.Create().
    SetHasBillables(input.HasBillables).
    SetSyncedAt(input.SyncedAt.UTC()).
    SetNillableNextSyncAfter(nextSyncAfter).
    SetSubscriptionID(input.SubscriptionID.ID).
    SetNamespace(input.SubscriptionID.Namespace).
    OnConflictColumns(subscriptionbillingsyncstate.FieldSubscriptionID, subscriptionbillingsyncstate.FieldNamespace).
    UpdateHasBillables().UpdateSyncedAt().UpdateNextSyncAfter().Exec(ctx)
})
```

<!-- archie:ai-end -->
