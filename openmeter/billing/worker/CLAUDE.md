# worker

<!-- archie:ai-start -->

> Top-level Watermill-based billing worker that fans out subscription and invoice events from the system Kafka topic to typed handlers. Owns the Worker struct, WorkerOptions config, and the goroutine-safe AddHandler extension point; child packages own advance, collect, asyncadvance, and subscriptionsync logic independently.

## Patterns

**WorkerOptions.Validate() constructor guard** — All required fields are checked in Validate() before any goroutine or router is started; return error from New() if Validate() fails. (`if err := opts.Validate(); err != nil { return nil, err }`)
**grouphandler.NewGroupEventHandler for each event type** — Each Watermill event type gets its own NewGroupEventHandler closure registered in eventHandler(). The closure receives a typed event pointer and delegates to a service method. (`grouphandler.NewGroupEventHandler(func(ctx context.Context, event *billing.AdvanceStandardInvoiceEvent) error { return w.asyncAdvanceHandler.Handle(ctx, event) })`)
**LockdownNamespaces guard before service call** — Every handler closure checks slices.Contains(w.lockdownNamespaces, event.<Namespace>) and returns nil early to skip locked namespaces. (`if event != nil && slices.Contains(w.lockdownNamespaces, event.Invoice.Namespace) { return nil }`)
**Optional ChargesService nil-guard** — ChargesService is optional; its handler is only constructed and registered when the field is non-nil. Pattern for optional sub-handlers. (`if opts.ChargesService != nil { asyncAdvanceChargesHandler, err = chargesasyncadvance.New(...) }`)
**AddHandler for external extension without recompilation** — Post-construction callers (e.g. billing-worker cmd) use Worker.AddHandler(handler) to append handlers; handlers run after built-ins and must be idempotent. (`worker.AddHandler(myExtraHandler) // called from app/common or cmd layer`)
**Single Kafka consumer handler per worker** — All event types flow through one router.AddConsumerHandler subscription on SystemEventsTopic; the NoPublishingHandler multiplexes by event type internally. (`router.AddConsumerHandler("billing_worker_system_events", opts.SystemEventsTopic, opts.Router.Subscriber, worker.nonPublishingHandler.Handle)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `worker.go` | Defines WorkerOptions, Worker struct, New() constructor, eventHandler() registration loop, AddHandler(), Run(), and Close(). The single authoritative place for handler registration order. | Adding new event types requires a new NewGroupEventHandler entry in eventHandler(); forgetting the lockdownNamespaces guard; making ChargesService required instead of optional breaks deployments without charges. |

## Anti-Patterns

- Returning ErrInvoiceCannotAdvance from a handler closure — Watermill will nack and requeue; catch it inside the handler and return nil
- Calling billing.Adapter directly from worker.go — all DB access must go through billing.Service or the injected sub-services
- Adding business logic inside worker.go event closures — delegate immediately to the appropriate sub-service (asyncAdvanceHandler, subscriptionSync, etc.)
- Registering a handler after New() returns without using AddHandler() — router is already built at that point
- Using context.Background() inside handler closures — always use the ctx passed by Watermill via the closure parameter

## Decisions

- **Single NoPublishingHandler multiplexing all event types over one Kafka subscription** — Simplifies router topology; unknown event types are silently dropped, making the worker tolerant of schema evolution without redeployment.
- **ChargesService is optional (nil-guarded) in WorkerOptions** — Allows deployments where charges are disabled to run the billing worker without wiring the charges subsystem.
- **AddHandler() extension point instead of subclassing or re-wiring** — Lets app/common and cmd layer attach domain-specific handlers (e.g. reconciler) after construction without modifying the core worker package.

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
