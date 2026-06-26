# subscriptionsync

<!-- archie:ai-start -->

> The subscription→billing sync bridge package: defines the contracts that reconcile a subscription's billable target state into billing artifacts (invoice lines + charges) and persist per-subscription sync bookkeeping. The root holds only the public interfaces (service.go, adapter.go); service/ orchestrates, adapter/ persists, and reconciler/ drives the periodic re-sync loop. Primary constraint: this is a thin interface-defining package — concrete behavior lives in the child packages, all of which must honor the per-customer billing lock and the Plan-is-pure / Apply-is-the-only-writer split.

## Patterns

**Composed Service interface from three role interfaces** — Service embeds EventHandler + SyncService + SyncStateService; new public capabilities go onto the matching sub-interface, not a flat method list. EventHandler holds the event entrypoints (HandleCancelledEvent, HandleDeletedEvent, HandleSubscriptionSyncEvent, HandleInvoiceCreation), SyncService the imperative Sync* calls, SyncStateService the read-only GetSyncStates. (`type Service interface { EventHandler; SyncService; SyncStateService }`)
**Adapter = SyncStateAdapter + entutils.TxCreator** — The persistence contract is only the three sync-state ops (InvalidateSyncState, GetSyncStates, UpsertSyncState) plus the transaction-creator mixin; the adapter/ child implements it. Invoice/charge persistence never belongs on this Adapter — that flows through billing.Service. (`type Adapter interface { SyncStateAdapter; entutils.TxCreator }`)
**Input type aliases over wrapper structs** — Inputs reuse existing types via aliases (InvalidateSyncStateInput = models.NamespacedID, GetSyncStatesInput = []models.NamespacedID, UpsertSyncStateInput = SyncState) instead of new structs; Validate() lives on the aliased target (SyncState). (`type UpsertSyncStateInput = SyncState`)
**Validate() collects errors via errors.Join** — UpsertSyncStateInput.Validate appends each field issue to var errs []error and returns errors.Join(errs...) rather than returning on first failure — follow this when adding new input validators. (`var errs []error; if err := i.SubscriptionID.Validate(); err != nil { errs = append(errs, err) }; return errors.Join(errs...)`)
**DryRun via functional option** — Sync entrypoints take ...SynchronizeSubscriptionOption; EnableDryRun() sets SynchronizeSubscriptionOptions.DryRun. New sync knobs follow the same option-func shape, not extra method params. (`func EnableDryRun() SynchronizeSubscriptionOption { return func(o *SynchronizeSubscriptionOptions) { o.DryRun = true } }`)
**SyncState HasBillables⇒NextSyncAfter coupling invariant** — When HasBillables is true, NextSyncAfter must be non-nil; if set, NextSyncAfter must be non-zero. The reconciler relies on this pairing for cooldown gating — preserve it in any writer of SyncState. (`if i.HasBillables && i.NextSyncAfter == nil { errs = append(errs, errors.New("next sync after is required when the subscription has billables")) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Declares the package-public Service interface (composed of EventHandler, SyncService, SyncStateService) and the SynchronizeSubscriptionOption/EnableDryRun option type. Pure contract — no implementation. | Both SyncByView/SyncByID (option-aware) and the *AndInvoiceCustomer variants exist; the invoice-customer variants intentionally take no options — don't collapse them. Add new capability to the correct sub-interface, not a flat dump on Service. |
| `adapter.go` | Declares Adapter (SyncStateAdapter + entutils.TxCreator), the SyncState struct, the three input aliases, and UpsertSyncStateInput.Validate. | Keep scoped to sync-state bookkeeping only; invoice/line/charge persistence goes through billing.Service/Adapter. The HasBillables⇒NextSyncAfter invariant in Validate is load-bearing for reconciler cooldown. |

## Anti-Patterns

- Adding invoice-line or charge persistence methods to the Adapter interface — its only job is SubscriptionBillingSyncState bookkeeping; billing writes go through billing.Service.
- Flattening the composed Service interface or dropping a sub-interface (EventHandler/SyncService/SyncStateService) — callers and the event consumer depend on these roles.
- Writing SyncState with HasBillables=true but NextSyncAfter=nil (or zero) — violates Validate and silently breaks the reconciler's cooldown skip.
- Returning on the first validation error instead of collecting into errs and errors.Join — diverges from repo convention and hides field issues.
- Introducing a new input struct where a models.NamespacedID / SyncState alias already expresses the shape.

## Decisions

- **Keep the root package as interfaces + the SyncState DTO only; push orchestration to service/, persistence to adapter/, and the periodic loop to reconciler/.** — Lets the event-driven path and the reconciliation loop both depend on the same narrow contracts without import cycles, and keeps pure planning cleanly separable from writing.
- **Model sync bookkeeping as a dedicated SyncState (SubscriptionID, HasBillables, SyncedAt, NextSyncAfter) rather than deriving freshness from billing artifacts.** — The reconciler can cheaply skip up-to-date subscriptions via persisted HasBillables + NextSyncAfter cooldown instead of recomputing target state for every subscription each run.

## Example: Defining the persistence contract and its validated input

```
type Adapter interface {
	SyncStateAdapter
	entutils.TxCreator
}

type SyncState struct {
	SubscriptionID models.NamespacedID
	HasBillables   bool
	SyncedAt       time.Time
	NextSyncAfter  *time.Time
}

func (i UpsertSyncStateInput) Validate() error {
	var errs []error
	if err := i.SubscriptionID.Validate(); err != nil {
// ...
```

<!-- archie:ai-end -->
