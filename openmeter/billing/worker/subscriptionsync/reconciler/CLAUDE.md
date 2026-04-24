# reconciler

<!-- archie:ai-start -->

> Crash-recovery reconciler that periodically re-drives subscription-to-billing sync for subscriptions that may have missed event-driven processing. Iterates active subscriptions in paginated windows and calls SynchronizeSubscription for each eligible entry.

## Patterns

**Config struct with Validate + named constructor** — ReconcilerConfig holds all dependencies; Validate() checks each for nil; NewReconciler calls Validate before constructing and returns (*Reconciler, error). (`func NewReconciler(config ReconcilerConfig) (*Reconciler, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**Paginated window scan with defaultWindowSize** — ListSubscriptions iterates pages of 10,000 subscriptions using pagination.Page{PageNumber, PageSize} until an empty page is returned, preventing unbounded memory usage. (`for { subscriptions, err := r.subscriptionService.List(ctx, subscription.ListSubscriptionsInput{Page: pagination.Page{PageNumber: pageIndex, PageSize: defaultWindowSize}}); if len(subscriptions.Items) == 0 { break }; pageIndex++ }`)
**SyncState-gated skipping logic** — All.ReconcileSubscription skips subscriptions where SyncState.HasBillables is false or NextSyncAfter is nil or in the future, avoiding unnecessary re-sync when the state machine is quiescent. (`if !in.Force && subscription.SyncState != nil { if !subscription.SyncState.HasBillables { continue } ... }`)
**Errors.Join for non-fatal per-item errors** — All() uses errors.Join to accumulate per-subscription reconciliation errors, logging each but continuing to process remaining subscriptions so one failure does not abort the batch. (`outErr = errors.Join(outErr, fmt.Errorf("failed to reconcile subscription: %w", err))`)
**clock.Now() for testable time references** — Use clock.Now() instead of time.Now() when computing lookback windows and comparing NextSyncAfter to enable clock injection in tests. (`ActiveInPeriod: &timeutil.StartBoundedPeriod{From: clock.Now().Add(-in.Lookback), To: lo.ToPtr(clock.Now())}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `reconciler.go` | Entire reconciler: config, ListSubscriptions (paginated), ReconcileSubscription (single), All (batch with skip logic). No DB access — delegates to subscriptionsync.Service and subscription.Service. | Calling time.Now() directly instead of clock.Now(); adding DB access (route through services); missing deleted-customer guard in ReconcileSubscription. |

## Anti-Patterns

- Using time.Now() instead of clock.Now() — breaks time-based tests
- Loading all subscriptions into memory without pagination (defaultWindowSize must be respected)
- Aborting the entire batch on a single reconciliation error instead of accumulating with errors.Join
- Adding Ent/DB imports — this layer must remain service-interface-only with no direct persistence calls
- Removing the deleted-customer guard in ReconcileSubscription without replacing with equivalent behaviour

## Decisions

- **Reconciler does not own a DB adapter; it orchestrates exclusively through service interfaces (subscriptionsync.Service, subscription.Service, customer.Service).** — Keeps the reconciler testable with mocks and decoupled from persistence; transactionality is managed by the service layer.
- **Force flag bypasses SyncState-based skipping to support manual or admin-triggered full re-sync without altering stored state.** — Operational requirement: administrators need to force reconciliation during incident recovery without clearing sync state first.

<!-- archie:ai-end -->
