# subscriptionsync

<!-- archie:ai-start -->

> Bridge package that synchronizes live subscription state into billing invoice lines and charges. Defines the Service, Adapter, and SyncState types that the three sub-packages (adapter/, service/, reconciler/) implement against.

## Patterns

**Composite Service interface** — Service interface is composed of three focused sub-interfaces: EventHandler (Kafka event adapters), SyncService (sync orchestration), SyncStateService (state reads). Callers depend only on the sub-interface they need. (`type Service interface { EventHandler; SyncService; SyncStateService }`)
**Adapter split: SyncStateAdapter + TxCreator** — Adapter embeds SyncStateAdapter (domain operations: InvalidateSyncState, GetSyncStates, UpsertSyncState) plus entutils.TxCreator for transaction management. No business logic lives in the adapter interface. (`type Adapter interface { SyncStateAdapter; entutils.TxCreator }`)
**Input type with Validate()** — Input structs (e.g. UpsertSyncStateInput = SyncState) carry a Validate() method that guards against zero-value or logically inconsistent fields. Constructors must call Validate() before proceeding. (`func (i UpsertSyncStateInput) Validate() error { ... errors.Join(errs...) }`)
**Functional option for SynchronizeSubscription** — Optional behavior (DryRun) is passed via SynchronizeSubscriptionOption functions rather than boolean parameters, keeping the method signature stable across callers. (`func EnableDryRun() SynchronizeSubscriptionOption { return func(o *SynchronizeSubscriptionOptions) { o.DryRun = true } }`)
**models.NamespacedID as identity type** — SyncState.SubscriptionID is models.NamespacedID; input aliases (InvalidateSyncStateInput, GetSyncStatesInput) use standard pkg/models types to stay consistent with the rest of the billing domain. (`type InvalidateSyncStateInput = models.NamespacedID`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Defines Adapter and SyncStateAdapter interfaces plus SyncState domain struct and UpsertSyncStateInput.Validate(). This is the contract between service layer and persistence layer. | SyncState.SyncedAt must never be zero; NextSyncAfter must be set when HasBillables=true — Validate() enforces both. Do not add business logic here. |
| `service.go` | Defines the Service interface composed of EventHandler, SyncService, and SyncStateService. Also declares SynchronizeSubscriptionOptions and EnableDryRun functional option. | EventHandler methods (HandleCancelledEvent, HandleSubscriptionSyncEvent, HandleInvoiceCreation) must stay thin — they are Kafka-event adapters that delegate to SynchronizeSubscription, not orchestration logic. |

## Anti-Patterns

- Adding sync orchestration logic (plan/apply, state diffing) to the adapter.go interface — that belongs in service/
- Calling UpsertSyncState without invoking Validate() first — zero SyncedAt or missing NextSyncAfter will silently corrupt reconciler scheduling
- Expanding EventHandler methods beyond fetching the subscription view and delegating to SynchronizeSubscription
- Using context.Background() anywhere in this package — all operations must propagate the caller ctx for OTel tracing
- Treating GetSyncStatesInput as a single ID — it is a slice of NamespacedIDs for bulk fetch

## Decisions

- **Service is split into three sub-interfaces (EventHandler, SyncService, SyncStateService) rather than one flat interface.** — Callers (reconciler, billing worker, Kafka handlers) each need a different surface; composing sub-interfaces avoids forcing unrelated dependencies on each caller.
- **SyncState carries HasBillables + NextSyncAfter to drive scheduler skipping in the reconciler.** — Avoids polling every subscription on every reconciler tick; subscriptions with no billables or a future NextSyncAfter are skipped without a full subscription load.
- **Input types for adapter methods are defined as type aliases (= models.NamespacedID, = []models.NamespacedID, = SyncState) rather than wrapper structs.** — Keeps the API surface minimal and avoids boilerplate conversion at call sites while still providing named types for clarity.

## Example: Implementing a new SyncStateAdapter method in the Ent adapter — full transacting pattern

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
