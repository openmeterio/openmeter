# balanceworker

<!-- archie:ai-start -->

> Kafka-driven worker that recalculates entitlement balances on lifecycle events (grant created/voided, entitlement created/deleted/reset, batched ingest). Subscribes to three Kafka topics (system, ingest, balance-worker) and publishes snapshot events downstream; the two-stage RecalculateEvent pattern decouples fan-out from calculation.

## Patterns

**Two-stage recalculation via RecalculateEvent** — Direct lifecycle handlers (grant, entitlement, ingest) publish a RecalculateEvent to the balance-worker topic rather than calling handleEntitlementEvent directly. A second handler for RecalculateEvent performs the actual calculation. Keeps fan-out retry independent from calculation retry. (`w.opts.EventBus.Publish(ctx, events.RecalculateEvent{Entitlement: pkgmodels.NamespacedID{Namespace: event.Namespace.ID, ID: event.Entitlement.ID}, OriginalEventSource: metadata.ComposeResourcePath(...), AsOf: event.Entitlement.ManagedModel.CreatedAt, SourceOperation: events.OperationTypeEntitlementCreated})`)
**Filter-before-calculate discipline** — Every recalculation path calls w.filters.IsNamespaceInScope then w.filters.IsEntitlementInScope before any DB or ClickHouse access. Filtered events return (nil, nil) — not an error. The filter chain short-circuits on first false. (`inScope, err := w.filters.IsNamespaceInScope(ctx, entitlementID.Namespace); if !inScope { return nil, nil }`)
**handleOption functional options with mandatory WithEventAt** — handleEntitlementEventOptions is built via functional options: WithSource, WithEventAt, WithSourceOperation, WithRawIngestedEvents. opts.Validate() is always called first and returns an error when eventAt.IsZero(). (`w.opts.EventBus.WithContext(ctx).PublishIfNoError(w.handleEntitlementEvent(ctx, id, WithSource(src), WithEventAt(event.ResetAt), WithSourceOperation(snapshot.ValueOperationReset)))`)
**PublishIfNoError for conditional publish** — Handlers that may legitimately return nil event (filtered) without error use w.opts.EventBus.WithContext(ctx).PublishIfNoError(...) so nil-event is not published but also does not trigger a retry. (`return w.opts.EventBus.WithContext(ctx).PublishIfNoError(w.handleEntitlementEvent(ctx, pkgmodels.NamespacedID{...}, options...))`)
**Options structs with Validate() called in every constructor** — WorkerOptions, RecalculatorOptions, EntitlementFiltersConfig all implement Validate() via errors.Join. Constructors call Validate() first and return early. Missing dependencies surface as startup errors, not runtime panics. (`if err := opts.Validate(); err != nil { return nil, fmt.Errorf("failed to validate worker options: %w", err) }`)
**RecordLastCalculation after every successful snapshot (non-fatal)** — After creating a snapshot event, always call w.filters.RecordLastCalculation with the entitlement and calculatedAt. Failure is non-fatal — log a warning and continue, because it only affects high-watermark deduplication. (`err = w.filters.RecordLastCalculation(ctx, filters.RecordLastCalculationRequest{Entitlement: *entitlementEntity, CalculatedAt: calculatedAt}); if err != nil { w.opts.Logger.WarnContext(ctx, "failed to record last calculation", "error", err) }`)
**AddHandler for post-construction extensibility** — w.AddHandler(grouphandler.GroupEventHandler) appends additional batched-ingest handlers after construction, called in order. Used by billing-worker to hook into ingest events without modifying balance-worker internals. Handlers must be idempotent. (`worker.AddHandler(grouphandler.NewGroupEventHandler(func(ctx context.Context, event *ingestevents.EventBatchedIngest) error { return billingService.HandleIngestEvent(ctx, event) }))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `worker.go` | Entry point: WorkerOptions, Worker struct, New() constructor, router setup subscribing to 3 Kafka topics, eventHandler() registering all NoPublishingHandler event type closures. | All new event type handlers must be registered inside grouphandler.NewNoPublishingHandler in eventHandler(). w.nonPublishingHandler is set after construction; AddHandler is the only safe extension point post-construction. |
| `entitlementhandler.go` | Core calculation: handleEntitlementEvent (filter → fetch entitlement → filter → processEntitlementEntity), processEntitlementEntity (deleted/reset/normal branching), createSnapshotEvent, createDeletedSnapshotEvent, snapshotToEvent. | Reset operations use entitlementEntity.CurrentUsagePeriod.From as calculatedAt, not time.Now(). Deleted/expired entitlements emit a delete snapshot with nil Value — not an error. Always call in.Validate() in snapshotToEvent. |
| `recalculate.go` | Recalculator struct for batch recalculation jobs: ListInScopeEntitlements (paginated at 20k/page to avoid Ent IN-subquery limit), ProcessEntitlements, sendEntitlementEvent. Uses LRU+TTL caches for feature and customerSubject lookups. | DefaultIncludeDeletedDuration=24h governs how long deleted entitlements stay in-scope for batch jobs. The paginated loop is intentional — Ent IN-subqueries break on large sets. |
| `ingesthandler.go` | handleBatchedIngestEvent: queries BalanceWorkerRepository for affected entitlements, checks activity period overlap with event timestamps, publishes RecalculateEvent for each affected entitlement. | Deleted entitlements are explicitly skipped — their final snapshot was published at deletion. Activity period check (GetEntitlementActivityPeriod) uses min(ActiveTo, DeletedAt) as upper bound. |
| `filters.go` | EntitlementFilters wraps a []NamedFilter chain (NotificationsFilter + HighWatermarkCache) with OTel counters per scope/filter/name. | Filter chain short-circuits on first false result. RecordLastCalculation only calls filters that implement CalculationTimeRecorder (optional interface checked via type assertion). |
| `repository.go` | BalanceWorkerRepository interface + ListAffectedEntitlementsResponse DTO. GetEntitlementActivityPeriod computes the valid event-reception window. | Activity period uses ActiveFrom over CreatedAt when set. When both ActiveTo and DeletedAt are present, the earlier one wins as the upper bound (min via lo.MinBy). |
| `subject_customer.go` | resolveCustomerAndSubject package-level helper used by both Worker and Recalculator. Subject may be nil when customer has no usage attribution. | If cus.UsageAttribution == nil or GetFirstSubjectKey returns no keys, returns (customer, nil, nil) — not an error. Missing subject entity falls back to a synthetic Subject with just the key. |

## Anti-Patterns

- Publishing a snapshot event directly from a lifecycle event handler instead of going through RecalculateEvent — breaks two-stage deduplication and filter pipeline
- Calling handleEntitlementEvent without WithEventAt — eventAt.IsZero() causes a hard validation error
- Adding DB or external service calls to filter implementations without LRU+TTL caching — IsNamespaceInScope/IsEntitlementInScope are called per-event and will saturate downstream services
- Registering new event type handlers outside the grouphandler.NewNoPublishingHandler block in eventHandler() — they will never be called
- Using time.Now() as calculatedAt for reset operations — reset snapshots must use CurrentUsagePeriod.From to capture initial grant state

## Decisions

- **Three-topic subscription (system + ingest + balance-worker) with RecalculateEvent intermediary on the balance-worker topic** — Separates high-volume ingest fan-out (one event → many entitlements → many RecalculateEvents) from the actual ClickHouse balance calculation, allowing independent retry and deduplication via high-watermark cache
- **Filter chain before every recalculation (NotificationsFilter + HighWatermarkCache)** — Prevents unnecessary ClickHouse queries for entitlements with no active notification rules or that were recently calculated; critical for cost at scale since every ingest event can affect multiple entitlements
- **Recalculator as separate struct from Worker** — Batch recalculation jobs (cmd/jobs) need the same calculation logic without constructing a full Kafka-connected Worker; extracting Recalculator enables reuse without Kafka dependency

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
