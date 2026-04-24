# balanceworker

<!-- archie:ai-start -->

> Kafka-driven worker that recalculates entitlement balances on lifecycle events (grant created/voided, entitlement created/deleted/reset, batched ingest). Subscribes to three topics (system, ingest, balance-worker) and publishes snapshot events downstream to the notification service.

## Patterns

**Two-stage recalculation via RecalculateEvent** — Direct lifecycle events (grant, entitlement, ingest) are converted into RecalculateEvent and published back to the balance-worker topic; a second handler consumes RecalculateEvent and calls handleEntitlementEvent. This decouples event fanout from actual calculation. (`opts.EventBus.Publish(ctx, events.RecalculateEvent{Entitlement: ..., SourceOperation: events.OperationTypeGrantCreated})`)
**Filter-before-calculate discipline** — Every recalculation path calls w.filters.IsNamespaceInScope then w.filters.IsEntitlementInScope before any DB or ClickHouse access. Skipped events return (nil, nil) — not an error. (`inScope, err := w.filters.IsNamespaceInScope(ctx, ns); if !inScope { return nil, nil }`)
**handleOption functional options for event context** — WithSource, WithEventAt, WithSourceOperation, WithRawIngestedEvents build handleEntitlementEventOptions. Always pass WithEventAt; zero eventAt fails Validate(). (`w.handleEntitlementEvent(ctx, id, WithSource(src), WithEventAt(t), WithSourceOperation(snapshot.ValueOperationReset))`)
**PublishIfNoError for conditional publish** — w.opts.EventBus.WithContext(ctx).PublishIfNoError(w.handleEntitlementEvent(...)) is the canonical pattern for handlers that might return nil event (filtered) without error. (`return w.opts.EventBus.WithContext(ctx).PublishIfNoError(w.handleEntitlementEvent(ctx, id, ...))`)
**Options structs with Validate() on all constructors** — WorkerOptions, RecalculatorOptions, and EntitlementFiltersConfig all implement Validate() using errors.Join. Constructors call Validate() first and return early on error. (`if err := opts.Validate(); err != nil { return nil, fmt.Errorf("failed to validate worker options: %w", err) }`)
**RecordLastCalculation after every successful snapshot** — After creating a snapshot event, call w.filters.RecordLastCalculation with the entitlement and calculatedAt. Failure is non-fatal (warn + continue) because it only affects deduplication. (`err = w.filters.RecordLastCalculation(ctx, filters.RecordLastCalculationRequest{Entitlement: *e, CalculatedAt: calculatedAt})`)
**AddHandler for extensibility after construction** — w.AddHandler(grouphandler.GroupEventHandler) appends additional batched-ingest handlers post-construction. Used by billing-worker to hook into ingest events without modifying balance-worker internals. (`worker.AddHandler(grouphandler.NewGroupEventHandler(func(ctx context.Context, event *ingestevents.EventBatchedIngest) error { ... }))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `worker.go` | Entry point: WorkerOptions, Worker struct, New() constructor, router setup subscribing to 3 Kafka topics, eventHandler() registering all NoPublishingHandler event type closures. | All new event type handlers must be registered inside the grouphandler.NewNoPublishingHandler call in eventHandler(). Never add handlers outside this call — w.nonPublishingHandler is set after construction and AddHandler is the extension point. |
| `entitlementhandler.go` | Core calculation: handleEntitlementEvent (filter → fetch entitlement → filter → processEntitlementEntity), processEntitlementEntity (deleted/reset/normal branching), createSnapshotEvent, createDeletedSnapshotEvent, snapshotToEvent. | Reset operations use entitlementEntity.CurrentUsagePeriod.From as calculatedAt, not time.Now(). Deleted/expired entitlements emit a delete snapshot with nil Value, not an error. |
| `recalculate.go` | Recalculator struct for batch recalculation jobs: ListInScopeEntitlements (paginated, includes recently deleted), ProcessEntitlements, sendEntitlementEvent. Uses LRU caches for feature and customerSubject lookups. | Uses defaultPageSize=20_000 and paginates explicitly because Ent IN-subqueries break on large sets. DefaultIncludeDeletedDuration=24h governs how long deleted entitlements stay in scope. |
| `ingesthandler.go` | handleBatchedIngestEvent: queries BalanceWorkerRepository for affected entitlements, checks activity period overlap with event timestamps, publishes RecalculateEvent for each affected entitlement. | Entitlement activity period check (GetEntitlementActivityPeriod) gates whether an ingest event can affect the entitlement. Skip deleted entitlements silently — their final snapshot was already published. |
| `filters.go` | EntitlementFilters wraps a []NamedFilter chain (NotificationsFilter + HighWatermarkCache) with OTel counters per scope/filter/name. | Filter chain short-circuits on first false result. RecordLastCalculation calls only filters implementing CalculationTimeRecorder (optional interface). All metric counters must be initialized via WithMetrics before use. |
| `repository.go` | BalanceWorkerRepository interface + ListAffectedEntitlementsResponse DTO. GetEntitlementActivityPeriod computes the valid event-reception window (uses min of ActiveTo/DeletedAt as upper bound). | Activity period uses ActiveFrom over CreatedAt when set. When both ActiveTo and DeletedAt are present, the earlier one wins as the upper bound. |
| `subject_customer.go` | resolveCustomerAndSubject helper: gets customer, optionally resolves first subject key from usage attribution. Subject may be nil (no usage attribution) — callers must handle nil Subject. | If cus.UsageAttribution == nil, returns (customer, nil, nil) — not an error. Used in both Worker and Recalculator; shared via package-level function, not a method. |

## Anti-Patterns

- Publishing a snapshot event directly from a lifecycle event handler instead of going through RecalculateEvent — breaks the two-stage deduplication and filter pipeline
- Calling handleEntitlementEvent without WithEventAt — eventAt.IsZero() causes a hard error in Validate()
- Adding DB or external service calls to filter implementations without LRU+TTL caching — IsNamespaceInScope/IsEntitlementInScope are called per-event
- Registering new event type handlers outside the grouphandler.NewNoPublishingHandler block in eventHandler() — they will never be called
- Using time.Now() as the snapshot calculatedAt for reset operations — reset snapshots must use CurrentUsagePeriod.From to capture initial grant state

## Decisions

- **Three-topic subscription (system + ingest + balance-worker) with RecalculateEvent intermediary** — Separates high-volume ingest fan-out (one event → many entitlements → many RecalculateEvents) from the actual balance calculation, allowing independent retry and deduplication via high-watermark cache
- **Filter chain before every recalculation** — Prevents unnecessary ClickHouse queries for entitlements with no active notification rules or that were recently calculated, critical for cost at scale
- **Recalculator as separate struct from Worker** — Batch recalculation jobs (cmd/jobs) need to reuse the same calculation logic without constructing a full Kafka-connected Worker, so the core logic is extracted into Recalculator

## Example: Registering a new lifecycle event type that triggers balance recalculation

```
// In worker.go eventHandler(), inside grouphandler.NewNoPublishingHandler:
grouphandler.NewGroupEventHandler(func(ctx context.Context, event *somepkg.SomeEvent) error {
    return w.opts.EventBus.Publish(ctx, events.RecalculateEvent{
        Entitlement:         pkgmodels.NamespacedID{Namespace: event.Namespace.ID, ID: event.EntitlementID},
        OriginalEventSource: metadata.ComposeResourcePath(event.Namespace.ID, metadata.EntityEntitlement, event.EntitlementID),
        AsOf:                event.OccurredAt,
        SourceOperation:     events.OperationTypeEntitlementCreated, // pick closest matching
    })
}),
```

<!-- archie:ai-end -->
