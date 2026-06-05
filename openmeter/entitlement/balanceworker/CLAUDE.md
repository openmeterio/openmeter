# balanceworker

<!-- archie:ai-start -->

> Kafka/Watermill worker (package balanceworker, driven by cmd/balance-worker) that reacts to entitlement, grant, reset, and ingest events by recalculating an entitlement's value and emitting a snapshot.SnapshotEvent. Its primary constraint: every handler is idempotent and re-fetches live state, because rolled-back transactions also fire events and retries are expected.

## Patterns

**grouphandler event fan-in in eventHandler** — worker.eventHandler registers one grouphandler.NewGroupEventHandler per event type (EntitlementCreatedEventV2, grant.CreatedEvent/V2, grant.VoidedEvent/V2, meteredentitlement.EntitlementResetEvent/V3, ingestevents.EventBatchedIngest, events.RecalculateEvent) wired into a single grouphandler.NoPublishingHandler. (`grouphandler.NewGroupEventHandler(func(ctx, event *grant.CreatedEventV2) error { return w.opts.EventBus.Publish(ctx, events.RecalculateEvent{...}) })`)
**Trigger events normalize to RecalculateEvent** — Most upstream handlers do not recalculate inline; they re-publish an events.RecalculateEvent carrying Entitlement NamespacedID, OriginalEventSource, AsOf, SourceOperation. The RecalculateEvent handler then maps SourceOperation to a snapshot.ValueOperation and calls handleEntitlementEvent. (`w.opts.EventBus.Publish(ctx, events.RecalculateEvent{Entitlement: ..., AsOf: event.ResetAt, SourceOperation: events.OperationTypeMeteredEntitlementReset})`)
**Two-stage scope filtering before snapshotting** — handleEntitlementEvent first calls w.filters.IsNamespaceInScope, re-lists the entitlement via ListEntitlements(IncludeDeleted:true), then IsEntitlementInScope before producing a snapshot. Out-of-scope returns (nil,nil). (`inScope, err := w.filters.IsEntitlementInScope(ctx, filters.EntitlementFilterRequest{Entitlement: entitlementEntity, EventAt: opts.eventAt, Operation: ...})`)
**PublishIfNoError for handler->snapshot publishing** — Newer handlers (reset v3, RecalculateEvent) return w.opts.EventBus.WithContext(ctx).PublishIfNoError(w.handleEntitlementEvent(...)) so a nil event from filtering does not get published. (`return w.opts.EventBus.WithContext(ctx).PublishIfNoError(w.handleEntitlementEvent(ctx, id, options...))`)
**Snapshots built via snapshot.NewSnapshotEvent + marshaler.WithSource** — createSnapshotEvent/createDeletedSnapshotEvent resolve customer+subject (resolveCustomerAndSubject), feature (IncludeArchivedFeatureTrue), and entitlement value, then wrap with marshaler.WithSource(metadata.ComposeResourcePath(...)). (`marshaler.WithSource(metadata.ComposeResourcePath(namespace, metadata.EntityEntitlement, id), snap)`)
**RecordLastCalculation after each snapshot** — processEntitlementEntity / ProcessEntitlements call w.filters.RecordLastCalculation; failures are logged at WARN and swallowed (non-critical, worst case is redundant recalculation). (`w.filters.RecordLastCalculation(ctx, filters.RecordLastCalculationRequest{Entitlement: *entitlementEntity, CalculatedAt: calculatedAt})`)
**Reset operations snapshot at period start, not now** — When sourceOperation==ValueOperationReset, createSnapshotEvent is called with entitlementEntity.CurrentUsagePeriod.From (not calculatedAt) so the new period gets an initial-grant snapshot, and reset events skip RecordLastCalculation. (`snap, err := w.createSnapshotEvent(ctx, entitlementEntity, entitlementEntity.CurrentUsagePeriod.From, opts)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `worker.go` | Worker + WorkerOptions, New() router wiring (three AddConsumerHandler topics: system/ingest/balance-worker), eventHandler() event registration, AddHandler for extra batched-ingest handlers. | Adding an event type requires a new grouphandler.NewGroupEventHandler inside eventHandler(); WorkerOptions.Validate() must be extended for any new required dependency. |
| `entitlementhandler.go` | handleEntitlementEvent / processEntitlementEntity / createSnapshotEvent / createDeletedSnapshotEvent plus the functional-options (WithSource/WithEventAt/WithSourceOperation/WithRawIngestedEvents). | handleEntitlementEventOptions.Validate requires eventAt; entitlement-not-found returns router.NewWarningLogSeverityError to force a retry (not a hard fail). |
| `filters.go` | EntitlementFilters facade composing filters.NotificationsFilter + filters.HighWatermarkCache, with OTel counters; EntitlementFiltersConfig.Validate guard. | RecordLastCalculation only forwards to filters implementing filters.CalculationTimeRecorder; executeFilters is fail-fast (first out-of-scope short-circuits). |
| `recalculate.go` | Recalculator for full-namespace recalculation jobs (ListInScopeEntitlements paginates defaultPageSize=20k, ProcessEntitlements), with lrux TTL caches for feature and customer+subject. | DefaultIncludeDeletedDuration=24h governs which deleted entitlements get re-emitted; deleted/expired entitlements emit ValueOperationDelete with nil Value. |
| `ingesthandler.go` | handleBatchedIngestEvent: lists entitlements affected by meter slugs+subject via Repo, checks GetEntitlementActivityPeriod().Contains(eventTime), and publishes RecalculateEvent with OperationTypeIngest. | Empty MeterSlugs is a normal no-op; deleted entitlements are skipped (final delete event already sent). |
| `repository.go` | BalanceWorkerRepository interface (ListEntitlementsAffectedByIngestEvents), IngestEventQueryFilter.Validate, ListAffectedEntitlementsResponse.GetEntitlementActivityPeriod (StartBoundedPeriod from ActiveFrom/CreatedAt to min(ActiveTo,DeletedAt)). | GetEntitlementActivityPeriod prefers ActiveFrom over CreatedAt and the earliest of ActiveTo/DeletedAt as the end bound. |
| `subject_customer.go` | resolveCustomerAndSubject helper: customer is required, subject is optional and may be nil; missing subject row falls back to a synthetic subject carrying only the usage-attribution key. | Subjects are no longer persisted, so GenericNotFoundError on GetByKey is expected and must not propagate. |

## Anti-Patterns

- Recalculating inline in an upstream event handler instead of publishing events.RecalculateEvent and letting the RecalculateEvent handler converge to a snapshot.
- Producing a snapshot without first passing both IsNamespaceInScope and IsEntitlementInScope filter checks.
- Publishing a possibly-nil event directly instead of using EventBus.PublishIfNoError, which causes nil/filtered events to be emitted.
- Treating RecordLastCalculation failures as fatal - they are intentionally logged at WARN and swallowed.
- Snapshotting a reset at time.Now() instead of CurrentUsagePeriod.From, losing the initial-grant snapshot the notification pipeline needs.

## Decisions

- **Handlers re-fetch live state (entitlement, feature, value) on every event rather than trusting the event payload.** — Makes the worker resilient across event-schema versions and to rolled-back transactions that still fire events; retries converge.
- **A unified RecalculateEvent decouples the many trigger event types from the single recalculation/snapshot path.** — One idempotent recalculation routine serves grant, reset, create, delete, and ingest triggers, simplifying retries and dedup.
- **Scope filtering (notifications + high-watermark dedup) is a separate composable filters package injected into the worker.** — Lets the worker skip entitlements no notification rule cares about and dedup recently-calculated ones without bloating handler logic.

## Example: Re-publishing a trigger event as a RecalculateEvent inside eventHandler

```
grouphandler.NewGroupEventHandler(func(ctx context.Context, event *grant.CreatedEventV2) error {
  return w.opts.EventBus.Publish(ctx, events.RecalculateEvent{
    Entitlement:         pkgmodels.NamespacedID{Namespace: event.Namespace.ID, ID: event.Grant.OwnerID},
    OriginalEventSource: metadata.ComposeResourcePath(event.Namespace.ID, metadata.EntityEntitlement, event.Grant.OwnerID, metadata.EntityGrant, event.Grant.ID),
    AsOf:                event.Grant.ManagedModel.CreatedAt,
    SourceOperation:     events.OperationTypeGrantCreated,
  })
})
```

<!-- archie:ai-end -->
