# subscriptionsync

<!-- archie:ai-start -->

> Bridge package that synchronizes live subscription state into billing invoice lines and charges. This root defines the Service, Adapter, and SyncState contract types that the three sub-packages (adapter/, service/, reconciler/) implement; callers interact only with these interfaces, never with sub-package internals.

## Patterns

**Composite Service interface from three sub-interfaces** — Service is composed of EventHandler (thin Kafka-event adapters), SyncService (sync orchestration), and SyncStateService (state reads). Callers depend only on the sub-interface they need. (`type Service interface { EventHandler; SyncService; SyncStateService }`)
**Adapter split: SyncStateAdapter + TxCreator** — Adapter embeds SyncStateAdapter (InvalidateSyncState, GetSyncStates, UpsertSyncState) and entutils.TxCreator. No business logic lives in the adapter interface. (`type Adapter interface { SyncStateAdapter; entutils.TxCreator }`)
**Input type aliases with Validate()** — Adapter inputs are aliases (UpsertSyncStateInput = SyncState, InvalidateSyncStateInput = models.NamespacedID, GetSyncStatesInput = []models.NamespacedID). UpsertSyncStateInput.Validate() enforces non-zero SyncedAt and required NextSyncAfter when HasBillables=true. Constructors call Validate() first. (`if i.HasBillables && i.NextSyncAfter == nil { errs = append(errs, errors.New("next sync after is required when the subscription has billables")) }`)
**Functional option for optional behavior** — SyncByView/SyncByID accept variadic SynchronizeSubscriptionOption (e.g. EnableDryRun()) instead of boolean params to keep signatures stable. (`func EnableDryRun() SynchronizeSubscriptionOption { return func(o *SynchronizeSubscriptionOptions) { o.DryRun = true } }`)
**Thin EventHandler methods** — HandleCancelledEvent, HandleDeletedEvent, HandleSubscriptionSyncEvent, HandleInvoiceCreation fetch the subscription view and delegate to SyncByView/SyncByID. No orchestration or state-diffing here. (`view, err := s.subscriptionService.GetView(ctx, event.SubscriptionID); ...; return s.SyncByView(ctx, view, event.At)`)
**TransactingRepo + UTC on every adapter body** — Sub-package adapter/ wraps every DB write with entutils.TransactingRepo / TransactingRepoWithNoValue so the ctx-bound Ent transaction is honored; all time values are normalized with .UTC(). (`return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, repo *adapter) error { repo.db.BillingSubscriptionBillingSyncState.Create().SetSyncedAt(input.SyncedAt.UTC()) })`)
**Per-customer serialized mutation in service/** — The service sub-package runs a three-phase pipeline (load persisted state → compute target → plan+apply reconciler diff) and wraps all mutations in billing.Service.WithLock, delegating DB writes through billing.Service and charges.Service rather than adapters. (`// service/ orchestrates via billing.Service.WithLock; reconciler/ re-drives missed syncs via service interfaces only`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Defines Adapter and SyncStateAdapter interfaces, the SyncState domain struct, the input type aliases, and UpsertSyncStateInput.Validate() — the contract between service and persistence layers. | SyncState.SyncedAt must never be zero; NextSyncAfter must be set when HasBillables=true — Validate() enforces both. Do not add orchestration logic here. |
| `service.go` | Defines the Service interface composed of EventHandler, SyncService, SyncStateService; declares SynchronizeSubscriptionOptions and the EnableDryRun functional option. | EventHandler methods must remain thin Kafka adapters; adding reconciliation/plan-apply logic here couples the contract to the implementation. |

## Anti-Patterns

- Adding sync orchestration (plan/apply, state diffing) to adapter.go or service.go interfaces — that belongs in the service/ sub-package.
- Calling UpsertSyncState without invoking Validate() first — zero SyncedAt or missing NextSyncAfter silently corrupts reconciler scheduling.
- Expanding EventHandler methods beyond fetching the subscription view and delegating to SyncByView/SyncByID.
- Using context.Background() anywhere — all operations must propagate the caller ctx for OTel tracing and Ent transaction reuse.
- Treating GetSyncStatesInput as a single ID — it is a slice of models.NamespacedID for bulk fetch.

## Decisions

- **Service is split into three sub-interfaces (EventHandler, SyncService, SyncStateService) rather than one flat interface.** — Callers (reconciler, billing worker, Kafka handlers) each need a different surface; composing sub-interfaces avoids forcing unrelated dependencies and makes mocking cheaper.
- **SyncState carries HasBillables + NextSyncAfter to drive scheduler skipping in the reconciler.** — Subscriptions with no billables or a future NextSyncAfter are skipped without a full subscription load, avoiding polling every subscription on every reconciler tick.
- **Adapter input types are type aliases (= models.NamespacedID, = SyncState) rather than wrapper structs.** — Keeps the API surface minimal and avoids boilerplate conversion at call sites while still providing named types for clarity and IDE discoverability.

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
            SetHasBillables(input.HasBillables).Save(ctx)
// ...
```

<!-- archie:ai-end -->
