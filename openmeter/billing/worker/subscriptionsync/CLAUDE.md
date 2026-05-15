# subscriptionsync

<!-- archie:ai-start -->

> Bridge package that synchronizes live subscription state into billing invoice lines and charges. Defines the Service, Adapter, and SyncState contract types that the three sub-packages (adapter/, service/, reconciler/) implement; callers interact only with these interfaces, never with sub-package internals.

## Patterns

**Composite Service interface from three sub-interfaces** — Service interface is composed of EventHandler (Kafka event adapters), SyncService (sync orchestration), and SyncStateService (state reads). Callers depend only on the sub-interface they need, not the full surface. (`type Service interface { EventHandler; SyncService; SyncStateService }`)
**Adapter split: SyncStateAdapter + TxCreator** — Adapter embeds SyncStateAdapter (domain operations: InvalidateSyncState, GetSyncStates, UpsertSyncState) and entutils.TxCreator for transaction management. No business logic lives in the adapter interface. (`type Adapter interface { SyncStateAdapter; entutils.TxCreator }`)
**Input type aliases with Validate()** — Adapter input types are aliases (UpsertSyncStateInput = SyncState, InvalidateSyncStateInput = models.NamespacedID). UpsertSyncStateInput.Validate() enforces non-zero SyncedAt and required NextSyncAfter when HasBillables=true. Constructors must call Validate() before proceeding. (`func (i UpsertSyncStateInput) Validate() error { var errs []error; if i.SyncedAt.IsZero() { errs = append(errs, errors.New("synced at is required")) }; return errors.Join(errs...) }`)
**Functional option for optional behavior** — SynchronizeSubscription accepts variadic SynchronizeSubscriptionOption functions (e.g. EnableDryRun()) rather than boolean parameters to keep the method signature stable across callers. (`func EnableDryRun() SynchronizeSubscriptionOption { return func(o *SynchronizeSubscriptionOptions) { o.DryRun = true } }`)
**models.NamespacedID as identity type** — SyncState.SubscriptionID uses models.NamespacedID; input type aliases use pkg/models types consistently. Call SubscriptionID.Validate() inside UpsertSyncStateInput.Validate(). (`type InvalidateSyncStateInput = models.NamespacedID`)
**TransactingRepo on every adapter method body** — Sub-package adapter/ wraps every DB write with entutils.TransactingRepo or TransactingRepoWithNoValue so the ctx-bound Ent transaction is honored. All time values must be normalized with .UTC(). (`return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, repo *adapter) error { ... repo.db.BillingSubscriptionBillingSyncState.Create().SetSyncedAt(input.SyncedAt.UTC()) ... })`)
**Thin EventHandler methods** — HandleCancelledEvent, HandleSubscriptionSyncEvent, and HandleInvoiceCreation are Kafka-event adapters that fetch the subscription view and delegate to SynchronizeSubscription. No orchestration or state-diffing logic belongs here. (`func (s *service) HandleSubscriptionSyncEvent(ctx context.Context, event *subscription.SubscriptionSyncEvent) error { view, err := s.subscriptionService.GetView(ctx, event.SubscriptionID); ...; return s.SynchronizeSubscription(ctx, view, event.At) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Defines Adapter and SyncStateAdapter interfaces plus SyncState domain struct and UpsertSyncStateInput.Validate(). This is the contract between service layer and persistence layer. | SyncState.SyncedAt must never be zero; NextSyncAfter must be set when HasBillables=true — Validate() enforces both. Do not add business or sync orchestration logic here. |
| `service.go` | Defines the Service interface composed of EventHandler, SyncService, and SyncStateService. Declares SynchronizeSubscriptionOptions and EnableDryRun functional option. | EventHandler methods must remain thin Kafka-event adapters. Adding reconciliation logic (plan/apply, state diffing) here couples the contract to the implementation. |

## Anti-Patterns

- Adding sync orchestration logic (plan/apply, state diffing) to adapter.go or service.go interfaces — that belongs in service/ sub-package
- Calling UpsertSyncState without invoking Validate() first — zero SyncedAt or missing NextSyncAfter silently corrupts reconciler scheduling
- Expanding EventHandler methods beyond fetching the subscription view and delegating to SynchronizeSubscription
- Using context.Background() anywhere in this package — all operations must propagate the caller ctx for OTel tracing and Ent transaction reuse
- Treating GetSyncStatesInput as a single ID — it is a slice of models.NamespacedID for bulk fetch

## Decisions

- **Service is split into three sub-interfaces (EventHandler, SyncService, SyncStateService) rather than one flat interface.** — Callers (reconciler, billing worker, Kafka handlers) each need a different surface; composing sub-interfaces avoids forcing unrelated dependencies on each caller and makes mocking in tests cheaper.
- **SyncState carries HasBillables + NextSyncAfter to drive scheduler skipping in the reconciler.** — Avoids polling every subscription on every reconciler tick; subscriptions with no billables or a future NextSyncAfter are skipped without a full subscription load, reducing DB pressure.
- **Input types for adapter methods are defined as type aliases (= models.NamespacedID, = SyncState) rather than wrapper structs.** — Keeps the API surface minimal and avoids boilerplate conversion at call sites while still providing named types for clarity and IDE discoverability.

## Example: Implementing a new SyncStateAdapter method in the Ent adapter with full TransactingRepo wrapping and UTC normalization

```
import (
	"context"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (a *adapter) UpsertSyncState(ctx context.Context, input subscriptionsync.UpsertSyncStateInput) error {
	if err := input.Validate(); err != nil {
		return err
	}
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, repo *adapter) error {
		_, err := repo.db.BillingSubscriptionBillingSyncState.Create().
			SetSubscriptionID(input.SubscriptionID.ID).
			SetSyncedAt(input.SyncedAt.UTC()).
			SetHasBillables(input.HasBillables).
// ...
```

<!-- archie:ai-end -->
