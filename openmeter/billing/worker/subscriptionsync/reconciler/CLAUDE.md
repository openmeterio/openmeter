# reconciler

<!-- archie:ai-start -->

> Periodic reconciliation component that re-syncs subscription state into billing, compensating for the event-driven invoice pipeline when a processing error caused a subscription to stop syncing. Drives subscriptionsync.Service.SyncByID across active and deleted subscriptions.

## Patterns

**ReconcilerConfig + Validate + NewReconciler constructor** — Reconciler is built from ReconcilerConfig via NewReconciler; Validate() requires SubscriptionSync, SubscriptionService, CustomerService, and Logger all non-nil (returns first error). Logger is injected, never slog.Default(). (`func NewReconciler(config ReconcilerConfig) (*Reconciler, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**Input struct with Validate per public method** — ReconcilerListSubscriptionsInput.Validate() enforces Lookback > 0; ListSubscriptions calls it before paging. (`if i.Lookback <= 0 { return errors.New("lookback must be greater than 0") }`)
**Two-pass paginated listing (active + deleted)** — ListSubscriptions pages with PageSize=defaultWindowSize(10_000) twice: first ActiveInPeriod over the lookback window, then a second pass with DeletedAt+IncludeDeleted to reconcile deleted subscriptions; each loop breaks when Items is empty. (`ActiveInPeriod: &timeutil.StartBoundedPeriod{From: clock.Now().Add(-in.Lookback), To: lo.ToPtr(clock.Now())}`)
**Join subscriptions with sync state via SliceToMap** — mapToSubscriptionWithSyncState fetches sync states by namespaced IDs, indexes them with lo.SliceToMap keyed on NamespacedID, and embeds an optional *SyncState pointer in SubscriptionWithSyncState (nil when no state exists). (`syncStatesBySubscriptionID := lo.SliceToMap(syncStates, func(s subscriptionsync.SyncState) (models.NamespacedID, subscriptionsync.SyncState) {...})`)
**Cooldown/skip gating in All, overridable by Force** — All() skips reconciliation when SyncState shows no billables, no NextSyncAfter, or NextSyncAfter in the future — unless ReconcilerAllInput.Force is set; per-subscription errors are accumulated with errors.Join and do not abort the loop. (`if subscription.SyncState.NextSyncAfter.After(clock.Now()) { ...skip... }`)
**clock package for time, not time.Now in loops** — Window boundaries and cooldown comparisons use pkg/clock (clock.Now()) so tests can freeze time; note ReconcileSubscription passes time.Now() to SyncByID. (`From: clock.Now().Add(-in.Lookback)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `reconciler.go` | Defines Reconciler + ReconcilerConfig, SubscriptionWithSyncState, ListSubscriptions (two-pass paging), mapToSubscriptionWithSyncState, ReconcileSubscription (delegates to SyncByID), and All (gated batch reconcile). | The deleted-subscription second pass exists because of a delete bug (comment dated 2026-05-19) and currently reconciles ALL deleted subscriptions ignoring lookback; All accumulates errors with errors.Join rather than failing fast; Force bypasses every skip guard. |

## Anti-Patterns

- Aborting the All() loop on the first per-subscription error instead of joining into outErr and continuing.
- Using time.Now() for window/cooldown logic instead of clock.Now(), which breaks frozen-time tests.
- Calling slog.Default() instead of the injected Logger.
- Removing or ignoring the SyncState skip guards (HasBillables / NextSyncAfter) without honoring the Force flag — would re-sync everything every run.
- Listing without paging to empty (PageSize defaultWindowSize loop) — would silently process only the first page.

## Decisions

- **Add a reconciliation loop on top of the event-driven invoice pipeline.** — Invoice creation is purely event driven; a processing error can leave a subscription stuck, so periodic reconciliation guarantees eventual convergence (see package doc comment on Reconciler).
- **Gate reconciliation on persisted SyncState (HasBillables + NextSyncAfter) unless Force.** — Avoids re-running expensive SyncByID for subscriptions with no billables or still within their cooldown window, keeping batch runs cheap.
- **Reconcile deleted subscriptions in a separate full pass.** — A historical delete bug left deleted subscriptions un-synced, so they must all be reconciled rather than only those active in the lookback window.

## Example: Skip-gated batch reconciliation accumulating errors

```
for _, subscription := range subscriptions {
  if !in.Force && subscription.SyncState != nil {
    if !subscription.SyncState.HasBillables { continue }
    if subscription.SyncState.NextSyncAfter == nil { continue }
    if subscription.SyncState.NextSyncAfter.After(clock.Now()) { continue }
  }
  if err := r.ReconcileSubscription(ctx, subscription.NamespacedID); err != nil {
    r.logger.ErrorContext(ctx, "failed to reconcile subscription", "error", err)
    outErr = errors.Join(outErr, fmt.Errorf("failed to reconcile subscription: %w", err))
  }
}
```

<!-- archie:ai-end -->
