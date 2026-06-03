# worker

<!-- archie:ai-start -->

> Top-level Watermill billing worker that fans out subscription and invoice events from the SystemEventsTopic to typed handlers via a single NoPublishingHandler multiplexer. Owns the Worker struct, WorkerOptions config and AddHandler extension point; child packages (advance, collect, asyncadvance, subscriptionsync) own their specific batch/event logic independently and are reached only through their service interfaces.

## Patterns

**WorkerOptions.Validate() constructor guard** — All required fields (SystemEventsTopic, Router, EventBus, Logger, BillingService, SubscriptionService, BillingSubscriptionSync) are checked in Validate() and New() returns an error if it fails. ChargesService is explicitly optional (nil allowed) for credits-disabled deployments. (`if err := opts.Validate(); err != nil { return nil, err }`)
**grouphandler.NewGroupEventHandler per event type in eventHandler()** — Each Watermill event type gets its own NewGroupEventHandler closure that receives a typed event pointer and delegates immediately to a service method with no business logic in worker.go. (`grouphandler.NewGroupEventHandler(func(ctx context.Context, event *billing.AdvanceStandardInvoiceEvent) error { return w.asyncAdvanceHandler.Handle(ctx, event) })`)
**LockdownNamespaces guard before every service call** — Every handler closure checks slices.Contains(w.lockdownNamespaces, event.<Namespace>) and returns nil early for locked namespaces. Must be present in every new closure. (`if event != nil && slices.Contains(w.lockdownNamespaces, event.Invoice.Namespace) { return nil }`)
**Optional ChargesService nil-guard pattern** — ChargesService is optional; the chargesasyncadvance handler is only constructed when opts.ChargesService != nil, and its closure returns nil when asyncAdvanceChargesHandler == nil. (`if opts.ChargesService != nil { asyncAdvanceChargesHandler, err = chargesasyncadvance.New(...) }`)
**AddHandler for post-construction extension** — Callers append additional GroupEventHandlers via Worker.AddHandler; they run after built-ins and must be idempotent because events are retried on any handler error. (`worker.AddHandler(myExtraHandler)`)
**Single Kafka consumer for all event types** — All event types flow through one router.AddConsumerHandler subscription on SystemEventsTopic; the NoPublishingHandler multiplexes by ce_type header and silently drops unknown types. (`router.AddConsumerHandler("billing_worker_system_events", opts.SystemEventsTopic, opts.Router.Subscriber, worker.nonPublishingHandler.Handle)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `worker.go` | Defines WorkerOptions + Validate(), the Worker struct, New() constructor, the eventHandler() registration loop, AddHandler(), Run(ctx), and Close(). Single authoritative place for handler registration. | Adding an event type requires a new NewGroupEventHandler entry in eventHandler() with a lockdownNamespaces guard; forgetting the guard, or making ChargesService required, breaks credits-disabled deployments. |

## Anti-Patterns

- Returning ErrInvoiceCannotAdvance from a handler closure — Watermill nacks and requeues; catch it inside the handler and return nil.
- Calling billing.Adapter directly from worker.go — all DB access must go through billing.Service or injected sub-services.
- Adding business logic inside worker.go event closures — delegate immediately to asyncAdvanceHandler, subscriptionSync, or asyncAdvanceChargesHandler.
- Omitting the lockdownNamespaces guard in a new event closure — runs the handler for namespaces that should be skipped.
- Using context.Background() inside handler closures — always use the ctx passed by the Watermill closure parameter.

## Decisions

- **A single NoPublishingHandler multiplexes all event types over one Kafka subscription.** — Simplifies router topology; unknown event types are silently dropped, making the worker tolerant of schema evolution without redeployment.
- **ChargesService is optional (nil-guarded) in WorkerOptions.** — Lets deployments with charges disabled run the billing worker without wiring the charges subsystem.
- **AddHandler() extension point instead of subclassing or re-wiring.** — Lets app/common and the cmd layer attach domain-specific handlers after construction without modifying the core worker package.

## Example: Registering a new subscription event handler with the lockdown guard

```
grouphandler.NewGroupEventHandler(func(ctx context.Context, event *subscription.UpdatedEvent) error {
	if event != nil && slices.Contains(w.lockdownNamespaces, event.UpdatedView.Subscription.Namespace) {
		return nil
	}
	return w.subscriptionSync.SyncByViewAndInvoiceCustomer(ctx, event.UpdatedView, time.Now())
})
```

<!-- archie:ai-end -->
