# reconciler

<!-- archie:ai-start -->

> Crash-recovery reconciler that periodically re-drives subscription-to-billing sync for subscriptions that may have missed event-driven processing. Iterates active subscriptions in paginated windows and calls SynchronizeSubscription for each eligible entry. Service-interface only; no DB access.

## Patterns

**Config struct with Validate + named constructor** — ReconcilerConfig holds all dependencies; Validate() checks each for nil; NewReconciler validates before constructing and returns (*Reconciler, error). (`func NewReconciler(config ReconcilerConfig) (*Reconciler, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**Paginated window scan with defaultWindowSize** — ListSubscriptions iterates pages of 10,000 via pagination.Page until an empty page returns, preventing unbounded memory usage. (`for { subs, _ := r.subscriptionService.List(ctx, subscription.ListSubscriptionsInput{Page: pagination.Page{PageNumber: pageIndex, PageSize: defaultWindowSize}}); if len(subs.Items) == 0 { break }; pageIndex++ }`)
**SyncState-gated skipping logic** — All() skips subscriptions where SyncState.HasBillables is false or NextSyncAfter is nil/in the future, avoiding unnecessary re-sync when the state machine is quiescent. (`if !in.Force && subscription.SyncState != nil { if !subscription.SyncState.HasBillables { continue } ... }`)
**errors.Join for non-fatal per-item errors** — All() accumulates per-subscription errors via errors.Join, logging each but continuing so one failure does not abort the batch. (`outErr = errors.Join(outErr, fmt.Errorf("failed to reconcile subscription: %w", err))`)
**clock.Now() for testable time** — Use clock.Now() (not time.Now()) when computing lookback windows and comparing NextSyncAfter to enable clock injection in tests. (`ActiveInPeriod: &timeutil.StartBoundedPeriod{From: clock.Now().Add(-in.Lookback), To: lo.ToPtr(clock.Now())}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `reconciler.go` | Entire reconciler: config, ListSubscriptions (paginated), ReconcileSubscription (single), All (batch with skip logic). Delegates to subscriptionsync.Service and subscription.Service; no DB access. | Calling time.Now() directly instead of clock.Now(); adding DB access (route through services); missing deleted-customer guard in ReconcileSubscription. |

## Anti-Patterns

- Using time.Now() instead of clock.Now() — breaks time-based tests.
- Loading all subscriptions into memory without pagination (respect defaultWindowSize).
- Aborting the entire batch on a single error instead of accumulating with errors.Join.
- Adding Ent/DB imports — this layer must remain service-interface-only.
- Removing the deleted-customer guard in ReconcileSubscription without equivalent replacement.

## Decisions

- **Reconciler owns no DB adapter; it orchestrates exclusively through service interfaces.** — Keeps the reconciler testable with mocks and decoupled from persistence; transactionality is managed by the service layer.
- **Force flag bypasses SyncState-based skipping for manual/admin full re-sync.** — Administrators need to force reconciliation during incident recovery without clearing sync state first.

## Example: A reconciler method listing and processing subscriptions in a paginated batch

```
import (
	"errors"; "fmt"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/pagination"
)
func (r *Reconciler) ProcessBatch(ctx context.Context, in ProcessBatchInput) error {
	pageIndex := 1
	var outErr error
	for {
		subs, err := r.subscriptionService.List(ctx, subscription.ListSubscriptionsInput{Page: pagination.Page{PageNumber: pageIndex, PageSize: defaultWindowSize}})
		if err != nil { return err }
		if len(subs.Items) == 0 { break }
		pageIndex++
	}
	return outErr
// ...
```

<!-- archie:ai-end -->
