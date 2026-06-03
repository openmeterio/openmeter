# balanceworker

<!-- archie:ai-start -->

> Kafka-driven worker that recalculates entitlement balances on lifecycle events (grant created/voided, entitlement created/deleted/reset, batched ingest) and publishes snapshot events downstream. Primary constraint: every recalculation path must pass the filter chain before any DB/ClickHouse access, and lifecycle handlers must publish a RecalculateEvent rather than calculating inline.

## Patterns

**Two-stage recalculation via RecalculateEvent** — Direct lifecycle handlers in worker.go eventHandler() publish events.RecalculateEvent to the balance-worker topic; a separate RecalculateEvent handler calls handleEntitlementEvent. Decouples fan-out retry from calculation retry. (`w.opts.EventBus.Publish(ctx, events.RecalculateEvent{Entitlement: pkgmodels.NamespacedID{...}, SourceOperation: events.OperationTypeEntitlementCreated})`)
**Filter-before-calculate** — handleEntitlementEvent calls w.filters.IsNamespaceInScope then IsEntitlementInScope before fetching; filtered events return (nil, nil), not an error. (`inScope, err := w.filters.IsNamespaceInScope(ctx, entitlementID.Namespace); if !inScope { return nil, nil }`)
**handleOption functional options with mandatory WithEventAt** — handleEntitlementEventOptions built via WithSource/WithEventAt/WithSourceOperation/WithRawIngestedEvents; opts.Validate() errors when eventAt.IsZero(). (`w.handleEntitlementEvent(ctx, id, WithSource(src), WithEventAt(event.ResetAt), WithSourceOperation(snapshot.ValueOperationReset))`)
**PublishIfNoError for conditional publish** — Handlers that may return a nil event without error use w.opts.EventBus.WithContext(ctx).PublishIfNoError so a filtered nil-event is neither published nor retried. (`return w.opts.EventBus.WithContext(ctx).PublishIfNoError(w.handleEntitlementEvent(ctx, id, options...))`)
**Options structs Validate() first in every constructor** — WorkerOptions, RecalculatorOptions, EntitlementFiltersConfig implement Validate() via errors.Join; New/NewRecalculator call it first and return early so missing deps surface as startup errors. (`if err := opts.Validate(); err != nil { return nil, fmt.Errorf("failed to validate worker options: %w", err) }`)
**RecordLastCalculation after each snapshot (non-fatal)** — After createSnapshotEvent, call w.filters.RecordLastCalculation; failure only logs a warning since it merely affects high-watermark dedup. (`if err := w.filters.RecordLastCalculation(...); err != nil { w.opts.Logger.WarnContext(ctx, ...) }`)
**AddHandler for post-construction extensibility** — w.AddHandler appends extra batched-ingest GroupEventHandlers after construction (used by billing-worker); handlers must be idempotent since errors trigger retry. (`worker.AddHandler(grouphandler.NewGroupEventHandler(func(ctx, *ingestevents.EventBatchedIngest) error { ... }))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `worker.go` | Entry point: WorkerOptions, Worker, New() (subscribes 3 Kafka topics), eventHandler() registering every NoPublishingHandler closure. | All new event-type handlers must be registered inside grouphandler.NewNoPublishingHandler in eventHandler(); AddHandler is the only post-construction extension point. |
| `entitlementhandler.go` | Core calculation: handleEntitlementEvent (filter→fetch→filter→processEntitlementEntity), createSnapshotEvent/createDeletedSnapshotEvent, snapshotToEvent. | Reset uses CurrentUsagePeriod.From as calculatedAt, not time.Now(). Deleted/expired entitlements emit a nil-Value delete snapshot, not an error. Always call in.Validate() in snapshotToEvent. |
| `recalculate.go` | Recalculator struct for batch jobs: ListInScopeEntitlements (paginated 20k/page), ProcessEntitlements, sendEntitlementEvent; LRU+TTL caches for feature & customerSubject. | DefaultIncludeDeletedDuration=24h keeps deleted entitlements in scope; the paginated loop is intentional (Ent IN-subqueries break on large sets). |
| `ingesthandler.go` | handleBatchedIngestEvent: queries Repo for affected entitlements, checks activity-period overlap, publishes a RecalculateEvent per affected entitlement. | Deleted entitlements are explicitly skipped; activity period uses min(ActiveTo, DeletedAt) as upper bound via GetEntitlementActivityPeriod. |
| `filters.go` | EntitlementFilters wraps []NamedFilter (NotificationsFilter + HighWatermarkCache) with OTel counters; RecordLastCalculation type-asserts CalculationTimeRecorder. | Filter chain short-circuits on first false; only filters implementing CalculationTimeRecorder receive RecordLastCalculation. |
| `repository.go` | BalanceWorkerRepository interface + ListAffectedEntitlementsResponse; GetEntitlementActivityPeriod computes the valid event window. | ActiveFrom wins over CreatedAt; when both ActiveTo and DeletedAt present, the earlier wins as upper bound (lo.MinBy). |
| `subject_customer.go` | resolveCustomerAndSubject helper shared by Worker and Recalculator; Subject may be nil for customers without usage attribution. | Nil UsageAttribution or no subject keys returns (customer, nil, nil); a missing subject row falls back to a synthetic Subject with just the key. |

## Anti-Patterns

- Publishing a snapshot event directly from a lifecycle handler instead of going through RecalculateEvent — breaks two-stage dedup and the filter pipeline
- Calling handleEntitlementEvent without WithEventAt — eventAt.IsZero() is a hard validation error
- Adding DB or external-service calls to filter implementations without LRU+TTL caching — IsNamespaceInScope/IsEntitlementInScope run per-event and saturate downstream services
- Registering new event-type handlers outside the grouphandler.NewNoPublishingHandler block in eventHandler() — they are never called
- Using time.Now() as calculatedAt for reset operations — reset snapshots must use CurrentUsagePeriod.From

## Decisions

- **Three-topic subscription (system + ingest + balance-worker) with a RecalculateEvent intermediary on the balance-worker topic** — Separates high-volume ingest fan-out (one event → many entitlements) from the actual ClickHouse balance calculation, enabling independent retry and high-watermark deduplication.
- **Recalculator as a struct separate from Worker** — Batch recalculation jobs (cmd/jobs) reuse the calculation logic without constructing a Kafka-connected Worker.

<!-- archie:ai-end -->
