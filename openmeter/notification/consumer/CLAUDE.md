# consumer

<!-- archie:ai-start -->

> Kafka/Watermill consumer (driven by cmd/notification-service) that subscribes to system events and turns entitlement balance.snapshot, entitlement reset, and standard-invoice created/updated events into notification.Events via the notification.Service. The deduplication logic here is the package's most subtle code.

## Patterns

**grouphandler fan-in over typed events** — Consumer.New registers one consumer handler on SystemEventsTopic built from grouphandler.NewNoPublishingHandler with NewGroupEventHandler closures per typed event (snapshot.SnapshotEvent, billing.StandardInvoiceCreatedEvent, billing.StandardInvoiceUpdatedEvent). Each closure nil-checks then delegates to a struct handler. (`grouphandler.NewGroupEventHandler(func(ctx, event *billing.StandardInvoiceCreatedEvent) error { if event==nil {return nil}; return consumer.invoiceHandler.Handle(ctx, event.EventStandardInvoice, notification.EventTypeInvoiceCreated) })`)
**Per-event-type handler structs holding Notification+Logger** — EntitlementSnapshotHandler and InvoiceEventHandler each hold a notification.Service and a *slog.Logger (group-scoped via opts.Logger.WithGroup). Add new event kinds as new handler structs, not new methods on Consumer. (`type InvoiceEventHandler struct { Notification notification.Service; Logger *slog.Logger }`)
**Rule fan-out + feature filtering** — Handlers ListRules for the relevant EventType+namespace then lo.Filter by rule.Config.<X>.Features against event FeatureID/FeatureKey (empty Features => match all). Errors per-rule are collected with errors.Join, not returned early. (`slices.Contains(rule.Config.BalanceThreshold.Features, event.Entitlement.FeatureID) || slices.Contains(..., event.Entitlement.FeatureKey)`)
**Dedupe via BalanceEventDedupHash (V1 sha256 / V2 xxh3)** — Before creating a balance-threshold event the handler computes NewBalanceEventDedupHash and ListEvents over the current usage period with DeduplicationHashes [V1,V2]; only creates when no prior event exists or the last event used a different threshold. The dedupe hash is also stored as an annotation. (`DeduplicationHashes: []string{dedupHash.V1(), dedupHash.V2()}`)
**Snapshot eligibility gating** — isBalanceThresholdEvent requires EntitlementTypeMetered, Update/Reset operation, IsActive(clock.Now()), and non-nil Value.Balance/Usage. isEntitlementResetEvent additionally requires ValueOperationReset. EntitlementSnapshotHandler.Handle runs both branches. (`if event.Entitlement.EntitlementType != entitlement.EntitlementTypeMetered { return false }`)
**Payload built from driver mappers, not raw structs** — Event payloads are assembled by mapping domain to API via entitlementdriver.Parser.ToMetered, productcatalogdriver.MapFeatureToResponse, subjecthttphandler.FromSubject, customerhttphandler.CustomerToAPI, billinghttp.MapEventInvoiceToAPI before calling Notification.CreateEvent. (`apiInvoice, err := billinghttp.MapEventInvoiceToAPI(event)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `consumer.go` | Options{Validate}, Consumer struct, New() wires router + group handlers, Run/Close | Gathering invoices are skipped downstream (invoice.go), not here; every group closure must nil-check the event pointer. |
| `entitlementsnapshot.go` | EntitlementSnapshotHandler dispatcher routing snapshot events to balance-threshold and reset branches | Both branches may fire for one event; keep them independent and wrap each error with context. |
| `entitlementbalancethreshold.go` | Threshold activation math, getActiveThresholdsWithHighestPriority, dedupe hash, createEvent | totalGrants==0 => ErrNoBalanceAvailable (skip, don't error); balance and overage cannot both be >absoluteZero; V2 hash uses thresholdKind so usage/balance kinds dedupe independently. |
| `entitlementreset.go` | Reset-event handling; one event per usage period (skips if any prior event in period) | createResetEvent reuses the balance event version (TODO OM-1508) — keep the Customer.ID guard before CustomerToAPI. |
| `invoice.go` | InvoiceEventHandler: skip gathering invoices, fan out to active non-disabled rules, annotate with invoice id/number | Returns nil (not error) when no rules or status==Gathering; payload Version must be EventPayloadVersionCurrent. |

## Anti-Patterns

- Creating a balance-threshold event without checking the dedupe hash against the current usage period
- Returning early on the first rule error instead of collecting via errors.Join
- Treating totalGrants==0 as a real error instead of skipping (ErrNoBalanceAvailable)
- Building API payloads by hand instead of via the *driver mapping helpers
- Emitting notification events for gathering-status invoices

## Decisions

- **Two dedupe hash versions (V1 sha256, V2 xxh3 including thresholdKind)** — V2 separates usage vs balance threshold kinds so both can fire in one period; querying both hashes keeps backward compatibility with events created under V1.
- **Reset events are one-per-usage-period** — A reset is a single boundary event; re-emitting on every subsequent snapshot in the same period would spam channels.

## Example: Fan-out + feature-filter + dedupe before creating a notification event

```
rules, err := b.Notification.ListRules(ctx, notification.ListRulesInput{
	Namespaces: []string{event.Namespace.ID},
	Types:      []notification.EventType{notification.EventTypeBalanceThreshold},
})
affected := lo.Filter(rules.Items, func(r notification.Rule, _ int) bool {
	if len(r.Config.BalanceThreshold.Features) == 0 { return true }
	return slices.Contains(r.Config.BalanceThreshold.Features, event.Entitlement.FeatureID)
})
for _, rule := range affected {
	if err := b.handleRule(ctx, event, rule); err != nil { errs = append(errs, err) }
}
return errors.Join(errs...)
```

<!-- archie:ai-end -->
