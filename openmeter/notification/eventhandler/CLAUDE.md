# eventhandler

<!-- archie:ai-start -->

> Implements notification.EventHandler: dispatches domain events asynchronously to Svix webhook channels (fire-and-forget goroutine) and runs a pg-advisory-lock-gated periodic reconciliation loop that drives stuck delivery statuses (Pending/Sending/Resending) to terminal states via a per-status state machine in webhook.go.

## Patterns

**Compile-time interface assertion** — handler.go declares `var _ notification.EventHandler = (*Handler)(nil)` to catch interface drift at compile time. Any new method added to notification.EventHandler without a matching implementation here fails the build. (`var _ notification.EventHandler = (*Handler)(nil)`)
**Config struct with Validate() before construction** — All constructor dependencies are collected in Config. Config.Validate() is called first in New() and returns models.NewNillableGenericValidationError so missing fields surface before any goroutine is started. (`func New(config Config) (*Handler, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**Dispatch fires a detached goroutine with panic recovery and fresh root span** — Dispatch() captures the OTel span link from the caller's context, launches a goroutine with context.WithTimeout(context.Background(), notification.DefaultDispatchTimeout), and always returns nil. Errors are only logged — callers must not rely on the return value for delivery confirmation. (`go func() { defer recover(); ctx, cancel := context.WithTimeout(context.Background(), notification.DefaultDispatchTimeout); defer cancel(); tracex.StartWithNoValue(ctx, h.tracer, "event_handler.dispatch", trace.WithNewRoot(), trace.WithLinks(spanLink)).Wrap(fn) }()`)
**pg advisory lock via pglock.Client.Do for leader-election in Start()** — Start() calls h.lockClient.Do(ctx, reconcilerLeaderLockKey, ...) so only one replica runs the reconcile ticker. pglock.ErrNotAcquired is treated as a non-error — the pod silently waits and retries the lock acquisition in a loop. (`err := h.lockClient.Do(ctx, reconcilerLeaderLockKey, func(rCtx context.Context, _ *pglock.Lock) error { ticker := time.NewTicker(h.reconcileInterval); ... })`)
**Paginated scan in Reconcile with nextAttemptDelay filter** — Reconcile paginates delivery statuses (PageSize 50, incrementing PageNumber) filtered to Pending/Sending/Resending states and NextAttemptBefore = clock.Now()-10s to let Svix async state settle before re-querying. Never remove the 10s jitter. (`nextAttemptBefore := clock.Now().Add(-1 * nextAttemptDelay); for { out, err := h.repo.ListEvents(ctx, notification.ListEventsInput{Page: page, DeliveryStatusStates: [...], NextAttemptBefore: nextAttemptBefore}); ...; page.PageNumber++ }`)
**Delivery status state machine in reconcileWebhookEvent** — webhook.go drives Pending→send/timeout, Resending→resend+Sending, Sending→read-provider-state/fail-on-timeout transitions. All terminal failure reasons are defined as sentinel error vars in webhook.go — never introduce new terminal reason strings outside that file. (`switch status.State { case notification.EventDeliveryStatusStatePending: ...; case notification.EventDeliveryStatusStateResending: ...; case notification.EventDeliveryStatusStateSending: ... }`)
**sync.OnceFunc for idempotent Close()** — stopCh is a plain chan struct{}; stopChClose is wrapped with sync.OnceFunc so Close() is safe to call multiple times and Start() stops reliably on signal. (`stopChClose := sync.OnceFunc(func() { close(stopCh) })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler struct definition, Config validation, New() constructor, Start()/Close() lifecycle. Start() runs pglock.Do loop calling Reconcile() on a ticker. | ReconcileInterval, PendingTimeout, and SendingTimeout default to package-level constants if zero — never pass zero expecting a no-op. atomic.Bool running guards against double-Start. |
| `dispatch.go` | Dispatch() goroutine launcher. Captures OTel span link before launching, then opens a fresh root span inside the goroutine with context.WithTimeout. | Dispatch always returns nil; real errors are only logged. Do NOT check the return value for dispatch success — the reconcile loop is the recovery path. |
| `reconcile.go` | Reconcile() paginates pending events and fans out to reconcileEvent per item using a semaphore-bounded worker pool (workerPoolSize). reconcileEvent dispatches to reconcileWebhookEvent by channel type. | nextAttemptDelay (10s) must remain to avoid racing Svix async state updates. Worker goroutines each have their own panic recovery. |
| `webhook.go` | reconcileWebhookEvent drives the per-delivery-status state machine against the Svix webhook provider. Defines all six sentinel error vars (ErrUser*, ErrSystem*) that are stored verbatim as DB failure reasons. | The six sentinel errors are the canonical human-readable failure reasons. Do not introduce new terminal-state strings outside this file. RetryableError.RetryAfter() controls the next-attempt delay for transient errors. |
| `deliverystatus.go` | Pure stateless helpers: filterActiveDeliveryStatusesByChannelType and sortDeliveryStatusStateByPriority. No side effects. | Keep these functions pure — no logger, tracer, or DB calls. Priority map order (Pending=0, Resending=1, Sending=2) determines which status is reconciled first within an event. |

## Anti-Patterns

- Calling reconcileEvent synchronously from Dispatch — breaks the fire-and-forget contract; Watermill consumers must not block on webhook delivery.
- Using context.Background() inside reconcileEvent or reconcileWebhookEvent instead of the received ctx — OTel spans and transaction bindings would be lost.
- Adding new terminal delivery status reason strings outside webhook.go sentinel error vars — creates inconsistent DB state across reconcile iterations.
- Skipping the pglock.Do leader-election in Start() — multiple pods would double-reconcile events and produce duplicate Svix sends.
- Making Handler methods value receivers for Start/Close — stopCh and stopChClose are pointer-receiver state; mixing receivers causes data races.

## Decisions

- **Dispatch fires a goroutine and always returns nil** — Callers (Watermill consumers) must not block on webhook delivery; failures are non-fatal to the consumer pipeline and are recovered by the reconcile loop.
- **Start() uses pglock.Client.Do for leader-election rather than application-level lockr advisory locks** — The reconcile loop must survive transaction boundaries — pglock holds a connection-scoped lock across multiple reconcile ticks, whereas pg_advisory_xact_lock releases on each transaction commit.
- **nextAttemptDelay jitter (10s) before re-querying Svix state** — Svix processes messages asynchronously; querying immediately after dispatch yields stale 'not found' states and causes unnecessary retries and false-positive failure transitions.

## Example: Adding a new channel type handler inside reconcileEvent (reconcile.go)

```
// In reconcile.go, extend the switch in reconcileEvent:
for _, channelType := range channelTypes {
    switch channelType {
    case notification.ChannelTypeWebhook:
        if err := h.reconcileWebhookEvent(ctx, event); err != nil {
            errs = append(errs, err)
        }
    case notification.ChannelTypeSlack: // new type
        if err := h.reconcileSlackEvent(ctx, event); err != nil {
            errs = append(errs, err)
        }
    default:
        h.logger.ErrorContext(ctx, "unsupported channel type", "type", channelType)
    }
}
```

<!-- archie:ai-end -->
