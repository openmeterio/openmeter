# worker

<!-- archie:ai-start -->

> Top-level Watermill-based billing worker that fans out subscription and invoice events from the system Kafka topic to typed handlers via a single NoPublishingHandler multiplexer. Owns the Worker struct, WorkerOptions config, and AddHandler extension point; child packages (advance, collect, asyncadvance, subscriptionsync) own their specific logic independently.

## Patterns

**WorkerOptions.Validate() constructor guard** — All required fields are checked in Validate() before any goroutine or router is started; New() returns error if Validate() fails. ChargesService is explicitly optional (nil allowed) so credits-disabled deployments still work. (`if err := opts.Validate(); err != nil { return nil, err }`)
**grouphandler.NewGroupEventHandler per event type in eventHandler()** — Each Watermill event type gets its own NewGroupEventHandler closure registered in eventHandler(). The closure receives a typed event pointer and delegates immediately to a service method with no business logic. (`grouphandler.NewGroupEventHandler(func(ctx context.Context, event *billing.AdvanceStandardInvoiceEvent) error {
    return w.asyncAdvanceHandler.Handle(ctx, event)
})`)
**LockdownNamespaces guard before every service call** — Every handler closure checks slices.Contains(w.lockdownNamespaces, event.<Namespace>) and returns nil early for locked namespaces. Must be present in every new handler closure. (`if event != nil && slices.Contains(w.lockdownNamespaces, event.Invoice.Namespace) { return nil }`)
**Optional ChargesService nil-guard pattern** — ChargesService is optional; its handler is only constructed and registered when the field is non-nil. This is the canonical pattern for optional sub-handlers in WorkerOptions. (`if opts.ChargesService != nil { asyncAdvanceChargesHandler, err = chargesasyncadvance.New(...) }`)
**AddHandler for post-construction extension** — Post-construction callers use Worker.AddHandler(handler) to append additional GroupEventHandler implementations; handlers run after built-ins and must be idempotent since events can be retried. (`worker.AddHandler(myExtraHandler) // from app/common or cmd layer`)
**Single Kafka consumer for all event types** — All event types flow through one router.AddConsumerHandler subscription on SystemEventsTopic; the NoPublishingHandler multiplexes by ce_type header internally. Unknown event types are silently dropped. (`router.AddConsumerHandler("billing_worker_system_events", opts.SystemEventsTopic, opts.Router.Subscriber, worker.nonPublishingHandler.Handle)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `worker.go` | Defines WorkerOptions, Worker struct, New() constructor, eventHandler() registration loop, AddHandler(), Run(), Close(). Single authoritative place for handler registration. | Adding new event types requires a new NewGroupEventHandler entry in eventHandler() with a lockdownNamespaces guard; forgetting the guard or making ChargesService required breaks credits-disabled deployments. |

## Anti-Patterns

- Returning ErrInvoiceCannotAdvance from a handler closure — Watermill nacks and requeues; catch it inside the handler and return nil instead.
- Calling billing.Adapter directly from worker.go — all DB access must go through billing.Service or injected sub-services.
- Adding business logic inside worker.go event closures — delegate immediately to the appropriate sub-service (asyncAdvanceHandler, subscriptionSync, etc.).
- Omitting the lockdownNamespaces guard in a new event handler closure — causes the handler to run for locked namespaces that should be skipped.
- Using context.Background() inside handler closures — always use the ctx passed by the Watermill closure parameter.

## Decisions

- **Single NoPublishingHandler multiplexing all event types over one Kafka subscription.** — Simplifies router topology; unknown event types are silently dropped, making the worker tolerant of schema evolution without redeployment.
- **ChargesService is optional (nil-guarded) in WorkerOptions.** — Allows deployments where charges are disabled to run the billing worker without wiring the charges subsystem.
- **AddHandler() extension point instead of subclassing or re-wiring.** — Lets app/common and cmd layer attach domain-specific handlers after construction without modifying the core worker package.

## Example: Adding a new subscription event type handler with lockdown guard

```
grouphandler.NewGroupEventHandler(func(ctx context.Context, event *subscription.SomeNewEvent) error {
    if event != nil && slices.Contains(w.lockdownNamespaces, event.Subscription.Namespace) {
        return nil
    }
    return w.subscriptionSync.SynchronizeSubscriptionAndInvoiceCustomer(ctx, event.SubscriptionView, time.Now())
}),
```

<!-- archie:ai-end -->
