# consumer

<!-- archie:ai-start -->

> Watermill-based Kafka consumer for the notification service, routing system-topic events (entitlement snapshots, billing invoice created/updated) to typed handlers that call notification.Service.CreateEvent. Primary constraint: each new event type requires a new grouphandler.NewGroupEventHandler registration in consumer.go.

## Patterns

**grouphandler.NewNoPublishingHandler with typed GroupEventHandlers** — consumer.go wires all event types in a single NewNoPublishingHandler call, each wrapped in grouphandler.NewGroupEventHandler[T]. Unknown event types are silently dropped. (`grouphandler.NewNoPublishingHandler(opts.Marshaler, opts.Router.MetricMeter, grouphandler.NewGroupEventHandler(func(ctx context.Context, event *snapshot.SnapshotEvent) error { return consumer.entitlementSnapshotHandler.Handle(ctx, *event) }), ...)`)
**EntitlementSnapshotHandler dispatches both threshold and reset** — entitlementsnapshot.go routes to handleAsSnapshotEvent (balance threshold) and handleAsEntitlementResetEvent (reset) based on isBalanceThresholdEvent / isEntitlementResetEvent predicates. Both paths call notification.Service.CreateEvent. (`func (b *EntitlementSnapshotHandler) Handle(ctx context.Context, event snapshot.SnapshotEvent) error { if b.isBalanceThresholdEvent(event) { ... }; if b.isEntitlementResetEvent(event) { ... } }`)
**Dual V1+V2 dedup hash query for balance threshold events** — Balance threshold events are deduplicated by querying ListEvents with DeduplicationHashes containing both V1 (sha256) and V2 (xxh3 with ThresholdKind prefix). New events are only created when no matching hash exists in the current usage period. (`dedupHash, err := NewBalanceEventDedupHash(balSnapshot, rule.ID, *threshold); lastEvents, _ := b.Notification.ListEvents(ctx, notification.ListEventsInput{DeduplicationHashes: []string{dedupHash.V1(), dedupHash.V2()}})`)
**EventPayloadVersionCurrent required for invoice payloads** — Invoice notification.EventPayload must always include EventPayloadMeta{Type: eventType, Version: notification.EventPayloadVersionCurrent}. Missing Version causes adapter deserialization to reject the payload. (`payload := notification.EventPayload{EventPayloadMeta: notification.EventPayloadMeta{Type: eventType, Version: notification.EventPayloadVersionCurrent}, Invoice: &notification.InvoicePayload{Invoice: apiInvoice}}`)
**Annotation-based event metadata** — All created events carry annotations (AnnotationEventSubjectKey, AnnotationEventFeatureKey, AnnotationEventCustomerID, AnnotationBalanceEventDedupeHash) set from the snapshot/invoice payload before calling CreateEvent. (`annotations := models.Annotations{notification.AnnotationEventSubjectKey: in.Snapshot.Subject.Key, notification.AnnotationBalanceEventDedupeHash: in.DedupeHash}`)
**Invoice handler skips gathering invoices** — InvoiceEventHandler.Handle checks event.Invoice.Status == billing.StandardInvoiceStatusGathering and returns nil early — gathering invoices do not produce notification events. (`if event.Invoice.Status == billing.StandardInvoiceStatusGathering { return nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `consumer.go` | Wires the Watermill router, instantiates handler structs, registers all event type handlers. Run(ctx) starts the router; Close() stops it. | Adding a new event type requires adding a new grouphandler.NewGroupEventHandler here; forgetting it means the event is silently ignored. |
| `entitlementsnapshot.go` | Dispatcher: calls handleAsSnapshotEvent (balance threshold) and handleAsEntitlementResetEvent (reset). Both can trigger on the same SnapshotEvent. | isBalanceThresholdEvent gates processing — non-metered entitlements and delete operations are skipped; event.Value must not be nil. |
| `entitlementbalancethreshold.go` | Core balance threshold logic: rule filtering, dedup hash computation (V1/V2), last-event comparison, event creation. Contains BalanceEventDedupHash, getNumericThreshold, getActiveThresholdsWithHighestPriority. | Percentage thresholds require TotalAvailableGrantAmount > 0; if grants are zero, ErrNoBalanceAvailable is returned and event creation is skipped. ThresholdKind is part of the V2 hash — changing the kind mapping is a breaking dedup change. |
| `entitlementreset.go` | Handles entitlement reset events: lists rules of type EventTypeEntitlementReset, deduplicates by checking for existing events in the current period, creates EntitlementReset payloads. | Dedup here checks for any event in the current usage period — simpler than balance threshold dedup, but still scoped to the period. |
| `invoice.go` | Handles billing.StandardInvoiceCreatedEvent and StandardInvoiceUpdatedEvent. Calls billinghttp.MapEventInvoiceToAPI to get the API invoice, then creates one notification event per active rule. | Gathering invoices are skipped explicitly. Rules with Disabled=true are also skipped with a warning log. |

## Anti-Patterns

- Adding a new event type handler without registering it in consumer.go's NewNoPublishingHandler call — event is silently dropped
- Setting Version field to anything other than notification.EventPayloadVersionCurrent in invoice EventPayloadMeta
- Changing BalanceEventDedupHash field composition without migrating V1/V2 hash constants — breaks dedup for existing events
- Calling notification.Service methods without propagating the Watermill message context (msg.Context())
- Triggering notification events for gathering invoices (Status == StandardInvoiceStatusGathering)

## Decisions

- **Dual V1+V2 dedup hash query for balance threshold events** — V2 adds ThresholdKind to the hash to prevent usage and balance threshold events from aliasing each other; V1 is queried alongside for backward compatibility with events created before V2 existed.
- **activeThresholds struct separates Usage and Balance thresholds** — Usage (value/percent) and balance (remaining balance) thresholds are semantically independent; the highest active threshold from each kind is selected separately so both types can fire concurrently.

## Example: Add a new system-event handler to the consumer

```
// In consumer.go, inside grouphandler.NewNoPublishingHandler(...) call:
grouphandler.NewGroupEventHandler(func(ctx context.Context, event *billing.SomeNewEvent) error {
	if event == nil {
		return nil
	}
	return consumer.someNewHandler.Handle(ctx, *event)
}),
// Then add SomeNewHandler struct in a new file and call notification.Service.CreateEvent inside it.
```

<!-- archie:ai-end -->
