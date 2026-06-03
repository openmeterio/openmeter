# consumer

<!-- archie:ai-start -->

> Watermill-based Kafka consumer for the notification service, routing system-topic events (entitlement snapshots, billing invoice created/updated) to typed handlers that call notification.Service.CreateEvent. Each new event type requires a new grouphandler registration in consumer.go.

## Patterns

**grouphandler.NewNoPublishingHandler with typed GroupEventHandlers** — consumer.go wires all event types in a single NewNoPublishingHandler call, each wrapped in grouphandler.NewGroupEventHandler[T]. Unknown event types are silently dropped. (`grouphandler.NewNoPublishingHandler(opts.Marshaler, opts.Router.MetricMeter, grouphandler.NewGroupEventHandler(func(ctx context.Context, event *snapshot.SnapshotEvent) error { return consumer.entitlementSnapshotHandler.Handle(ctx, *event) }), ...)`)
**EntitlementSnapshotHandler dispatches both threshold and reset** — entitlementsnapshot.go routes to handleAsSnapshotEvent (balance threshold) and handleAsEntitlementResetEvent (reset) based on predicates; both call notification.Service.CreateEvent. (`if b.isBalanceThresholdEvent(event) { ... }; if b.isEntitlementResetEvent(event) { ... }`)
**Dual V1+V2 dedup hash query for balance threshold events** — Balance threshold events dedup by querying ListEvents with DeduplicationHashes containing both V1 (sha256) and V2 (xxh3 with ThresholdKind prefix). New events created only when no matching hash exists in the current usage period. (`dedupHash, _ := NewBalanceEventDedupHash(balSnapshot, rule.ID, *threshold); b.Notification.ListEvents(ctx, notification.ListEventsInput{DeduplicationHashes: []string{dedupHash.V1(), dedupHash.V2()}})`)
**EventPayloadVersionCurrent required for invoice payloads** — Invoice EventPayload must include EventPayloadMeta{Type: eventType, Version: notification.EventPayloadVersionCurrent}. Missing Version causes adapter deserialization to reject the payload. (`payload := notification.EventPayload{EventPayloadMeta: notification.EventPayloadMeta{Type: eventType, Version: notification.EventPayloadVersionCurrent}, Invoice: &notification.InvoicePayload{Invoice: apiInvoice}}`)
**Annotation-based event metadata** — Created events carry annotations (AnnotationEventSubjectKey, AnnotationEventFeatureKey, AnnotationEventCustomerID, AnnotationBalanceEventDedupeHash) set from the snapshot/invoice before CreateEvent. (`annotations := models.Annotations{notification.AnnotationEventSubjectKey: in.Snapshot.Subject.Key, notification.AnnotationBalanceEventDedupeHash: in.DedupeHash}`)
**Invoice handler skips gathering invoices** — InvoiceEventHandler.Handle returns nil early when event.Invoice.Status == billing.StandardInvoiceStatusGathering — gathering invoices produce no notification events. (`if event.Invoice.Status == billing.StandardInvoiceStatusGathering { return nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `consumer.go` | Wires the Watermill router, instantiates handler structs, registers all event handlers. Run(ctx) starts the router; Close() stops it. | Adding a new event type requires a new grouphandler.NewGroupEventHandler here; forgetting it means the event is silently ignored. |
| `entitlementsnapshot.go` | Dispatcher calling handleAsSnapshotEvent (threshold) and handleAsEntitlementResetEvent (reset); both can fire on the same SnapshotEvent. | isBalanceThresholdEvent gates processing — non-metered entitlements and deletes are skipped; event.Value must not be nil. |
| `entitlementbalancethreshold.go` | Core balance threshold logic: rule filtering, dedup hash V1/V2, last-event comparison, event creation. Contains BalanceEventDedupHash, getActiveThresholdsWithHighestPriority. | Percentage thresholds require TotalAvailableGrantAmount > 0 (else ErrNoBalanceAvailable). ThresholdKind is part of V2 hash — changing the mapping is a breaking dedup change. |
| `entitlementreset.go` | Handles reset events: lists rules of type EventTypeEntitlementReset, dedups by existing event in current period, creates EntitlementReset payloads. | Dedup checks for any event in the current usage period — simpler than threshold dedup but still period-scoped. |
| `invoice.go` | Handles StandardInvoiceCreatedEvent/UpdatedEvent; maps via billinghttp.MapEventInvoiceToAPI, creates one event per active rule. | Gathering invoices skipped explicitly; rules with Disabled=true skipped with a warning log. |

## Anti-Patterns

- Adding a new event type handler without registering it in consumer.go's NewNoPublishingHandler — event is silently dropped
- Setting Version to anything other than notification.EventPayloadVersionCurrent in invoice EventPayloadMeta
- Changing BalanceEventDedupHash field composition without migrating V1/V2 hash constants — breaks dedup
- Calling notification.Service methods without propagating msg.Context()
- Triggering notification events for gathering invoices (Status == StandardInvoiceStatusGathering)

## Decisions

- **Dual V1+V2 dedup hash query for balance threshold events** — V2 adds ThresholdKind to prevent usage and balance threshold events from aliasing; V1 is queried alongside for backward compatibility with pre-V2 events.
- **activeThresholds struct separates Usage and Balance thresholds** — Usage (value/percent) and balance thresholds are independent; the highest active of each kind is selected separately so both can fire concurrently.

## Example: Add a new system-event handler to the consumer

```
// In consumer.go, inside grouphandler.NewNoPublishingHandler(...):
grouphandler.NewGroupEventHandler(func(ctx context.Context, event *billing.SomeNewEvent) error {
	if event == nil {
		return nil
	}
	return consumer.someNewHandler.Handle(ctx, *event)
}),
// Then add SomeNewHandler in a new file and call notification.Service.CreateEvent inside it.
```

<!-- archie:ai-end -->
