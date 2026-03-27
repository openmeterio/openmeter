# Subscription → Billing Sync Algorithm

Primary location: `openmeter/billing/worker/subscriptionsync/`

## Entry Points

The sync is triggered by the billing worker (`worker/worker.go`) on these events:

| Event | Handler |
|---|---|
| `subscription.Created/Updated/Continued` | `SynchronizeSubscriptionAndInvoiceCustomer` |
| `subscription.Cancelled` | `HandleCancelledEvent` |
| `subscription.SubscriptionSyncEvent` | `HandleSubscriptionSyncEvent` (self-loop) |
| `billing.StandardInvoiceCreatedEvent` | `HandleInvoiceCreation` → re-syncs all subscriptions in the new invoice's lines |

The self-loop mechanism (`SubscriptionSyncEvent`) ensures the gathering invoice is refilled with the next period's lines immediately after an invoice is issued, without waiting for the cron.

## Main Algorithm (`service/sync.go`)

```
SynchronizeSubscriptionAndInvoiceCustomer(ctx, SubscriptionView, asOf)
  → SynchronizeSubscription
    → billingService.WithLock(customerID)  ← advisory lock, serializes per customer
      → persistedstate.Loader.LoadForSubscription   ← fetch current DB state
      → targetstate.Builder.Build                   ← compute desired state
      → reconciler.Plan                             ← diff: new / delete / upsert
      → reconciler.Apply                            ← write patches to DB
      → updateSyncState                             ← upsert SubscriptionBillingSyncState row
  → invoicePendingLines  ← InvoicePendingLines(ProgressiveBillingOverride=false)
```

## Target State Builder (`service/targetstate/`)

`Builder.Build` iterates each phase and calls `PhaseIterator.Generate(ctx, generationLimit)`.

**Generation limit**: the end of the current aligned billing period (`spec.GetAlignedBillingPeriodAt(asOf)`). Lines are never generated past the current period boundary, except for in-advance items whose `InvoiceAt` equals the period start — these are pre-generated so the gathering invoice is immediately ready.

**Phase iterator loop** (`phaseiterator.go`): for each item in the phase, loops by `BillingCadence` until `InvoiceAt > iterationEnd`. Each iteration produces a `SubscriptionItemWithPeriods` with:
- `ServicePeriod`: the actual usage window for this item cadence period
- `BillingPeriod`: the subscription-aligned outer billing period

### `GetInvoiceAt()` — When Does a Line Appear on an Invoice?

```go
// Flat-fee in-advance: bill at the start of the billing period
if price.Type() == FlatPrice && paymentTerm == InAdvance {
    return BillingPeriod.Start
}
// Everything else (in-arrears, usage-based): bill at end of service or billing period
return max(ServicePeriod.End, BillingPeriod.End)
```

This is defined in `phaseiterator.go` and is the canonical source of billing timing truth.

## Reconciler (`service/reconciler/`)

### Plan (`reconciler.go`)

Computes symmetric difference on `ChildUniqueReferenceID` keys:
- **Only in persisted**: `LinesToDelete`
- **Only in target**: `NewSubscriptionItems`
- **In both**: `LinesToUpsert`

### ChildUniqueReferenceID Format

```
{subscriptionID}/{phaseKey}/{itemKey}/v[{version}]/period[{periodIndex}]
```

One-time (non-recurring) items omit `/period[N]`. Version increments when the rate card changes within a phase.

### Apply (`apply.go` + `invoiceupdate.go`)

Translates the plan into `Patch` objects and calls `InvoiceUpdater.ApplyPatches`. The updater handles three invoice states:

| Invoice state | Action |
|---|---|
| Gathering invoice | Direct edit via `UpdateGatheringInvoice` |
| Mutable standard invoice | `UpdateStandardInvoice` + `SnapshotLineQuantity` |
| Immutable standard invoice | Emit `ValidationIssue` (no actual mutation — detects drift) |

## Persisted State (`service/persistedstate/`)

`Loader.LoadForSubscription` calls `billingService.GetLinesForSubscription`, which queries `BillingInvoiceLine` rows where:
- `subscription_id = ?`
- `parent_line_id IS NULL` (top-level lines only; split children handled by mapper)
- Either `deleted_at IS NULL` OR (`deleted_at IS NOT NULL` AND `managed_by = manually_managed`)

Returns `[]billing.LineOrHierarchy` — gathering lines and standard lines/split hierarchies mixed.

## SubscriptionBillingSyncState Table

Managed in `adapter/syncstate.go`. Upserted on `(subscription_id, namespace)` after each sync:

```go
.OnConflictColumns(FieldSubscriptionID, FieldNamespace).
    UpdateHasBillables().UpdateSyncedAt().UpdateNextSyncAfter().Exec(ctx)
```

`NextSyncAfter` is set to `SubscriptionMaxGenerationTimeLimit` (the end of the last generated billing period). The worker uses this to know when to next sync without processing every event repeatedly.

## Billing Cadence and Alignment

`SubscriptionSpec.BillingCadence` (ISO duration, e.g. `P1M`) is the master cadence.
`BillingAnchor` is the fixed time anchor for period alignment.

`GetAlignedBillingPeriodAt(at)` in `subscription/subscriptionspec.go`:
1. Finds the active phase at `at`
2. Builds `timeutil.Recurrence` from `(BillingCadence, BillingAnchor)`
3. Calls `recurrence.GetPeriodAt(at)` to find the enclosing period
4. Clips boundaries to phase start/end

## Subscription-Level Timing Resolution

`Timing.ResolveForSpec` in `subscription/timing.go` resolves `next_billing_cycle` to `GetAlignedBillingPeriodAt(now).To`. Subscription edits with `next_billing_cycle` timing are deferred to the next billing period boundary.

## Subscription Validator (`validators/subscription/validator.go`)

Implements `subscription.SubscriptionCommandHook`. On `AfterCreate` and `AfterUpdate`, validates:
- Customer has a `CustomerOverride` pointing to a billing profile
- Profile apps have the required capabilities: `CapabilityTypeCalculateTax`, `CapabilityTypeInvoiceCustomers`, `CapabilityTypeCollectPayments`

If the subscription has no priced rate cards, validation is skipped entirely.

## Annotations

On `GatheringLine` and `StandardLine`:

- `AnnotationSubscriptionSyncIgnore` — sync algorithm skips this line (managed externally)
- `AnnotationSubscriptionSyncForceContinuousLines` — forces progressive billing to treat lines as continuous even if the pricing type would normally disallow it (use with care)

Both are defined in `billing/annotations.go`.
