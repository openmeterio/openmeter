# eventhandler

<!-- archie:ai-start -->

> Implements the notification.EventHandler interface: dispatches events asynchronously to webhook channels via Svix and runs a periodic reconciliation loop that re-drives any in-flight or stuck delivery statuses (Pending/Sending/Resending) to terminal states.

## Patterns

**Compile-time interface assertion** — handler.go declares `var _ notification.EventHandler = (*Handler)(nil)` to catch interface drift at compile time. (`var _ notification.EventHandler = (*Handler)(nil)`)
**Config struct with Validate()** — All constructor dependencies are collected in a `Config` struct. `Config.Validate()` is called first in `New()` and returns `models.NewNillableGenericValidationError` so missing fields surface before any goroutine is started. (`func New(config Config) (*Handler, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**Dispatch fires a detached goroutine with panic recovery** — Dispatch() always returns nil immediately; the actual work runs in a goroutine that captures the OTel span link from the caller's context, creates a fresh context.WithTimeout(context.Background(), notification.DefaultDispatchTimeout), and wraps panics with recover() + stack logging. (`go func() { defer recover(); ctx, cancel := context.WithTimeout(context.Background(), notification.DefaultDispatchTimeout); defer cancel(); tracex.StartWithNoValue(ctx, h.tracer, "event_handler.dispatch", ...).Wrap(fn) }()`)
**Reconcile loop uses pg advisory lock via lockr** — Reconcile() calls transaction.RunWithNoValue then h.lockr.LockForTXWithScopes(ctx, reconcileLockKey) so only one pod reconciles at a time. ErrLockTimeout is treated as a non-error (skip silently). (`if err := h.lockr.LockForTXWithScopes(ctx, reconcileLockKey); err != nil { if errors.Is(err, lockr.ErrLockTimeout) { return nil } }`)
**Paginated scan in Reconcile with NextAttemptBefore filter** — Reconcile scans delivery statuses with a page loop (PageSize 50, incrementing PageNumber) filtering on states Pending/Sending/Resending and `NextAttemptBefore = clock.Now()-nextAttemptDelay` to give downstream providers time to settle. (`for { out, err := h.repo.ListEvents(ctx, notification.ListEventsInput{Page: page, DeliveryStatusStates: [...], NextAttemptBefore: clock.Now().Add(-nextAttemptDelay)}); ...; page.PageNumber++ }`)
**Delivery status state machine in reconcileWebhookEvent** — webhook.go drives per-status state transitions: Pending→send or timeout, Resending→resend+set Sending, Sending→read provider state or fail on timeout. Errors are classified as unrecoverable (fail), retryable (reschedule via RetryableError.RetryAfter()), or transient (use h.reconcileInterval). (`switch status.State { case Pending: ...; case Resending: ...; case Sending: ... }`)
**Stop channel closed via sync.OnceFunc** — stopCh is a plain chan struct{}; stopChClose is wrapped with sync.OnceFunc so Close() is idempotent and Start() safely stops on signal. (`stopChClose := sync.OnceFunc(func() { close(stopCh) })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler struct definition, Config validation, New() constructor, Start()/Close() lifecycle. Start() runs a ticker loop calling Reconcile(); panic in Start() closes the stopCh. | ReconcileInterval, PendingTimeout, and SendingTimeout default to package-level constants if zero — never pass zero intentionally expecting a no-op. |
| `dispatch.go` | Dispatch() goroutine launcher. Captures OTel span link before the goroutine, then opens a fresh root span inside the goroutine. | Dispatch always returns nil; real errors are only logged. Do not check the return value for dispatch success. |
| `reconcile.go` | Reconcile() acquires advisory lock, paginates pending events, and calls reconcileEvent per item. reconcileEvent fans out to reconcileWebhookEvent by channel type. | nextAttemptDelay jitter (10s) must remain to avoid racing Svix async state updates; removing it causes spurious missing-status false-positives. |
| `webhook.go` | reconcileWebhookEvent drives the per-delivery-status state machine against the Svix webhook provider. Defines all sentinel error vars (ErrUser*, ErrSystem*). | The six sentinel errors are the canonical human-readable failure reasons stored in the DB. Do not introduce new terminal-state strings outside this file. |
| `deliverystatus.go` | Pure helpers: filterActiveDeliveryStatusesByChannelType removes terminal and channel-type-mismatched statuses; sortDeliveryStatusStateByPriority orders by priority map. | These are stateless functions with no side effects — keep them that way. |

## Anti-Patterns

- Calling reconcileEvent synchronously from Dispatch (breaks the fire-and-forget contract; callers must not block).
- Using context.Background() inside reconcileEvent or reconcileWebhookEvent instead of the received ctx — OTel spans and transaction bindings would be lost.
- Adding new terminal delivery status reason strings outside webhook.go sentinel error vars — creates inconsistent DB state.
- Skipping lockr.LockForTXWithScopes in Reconcile — multiple pods would double-reconcile events and produce duplicate Svix sends.
- Making Handler a value receiver for Start/Close — stopCh and stopChClose are pointer-receiver state; mixing receivers causes data races.

## Decisions

- **Dispatch fires a goroutine and always returns nil** — Callers (Watermill consumers) must not block on webhook delivery; failures are non-fatal to the consumer pipeline and are recovered by the reconcile loop.
- **Reconcile holds a pg advisory lock for its entire transaction** — Prevents multiple notification-service replicas from sending duplicate Svix messages when the same pending event is visible to all of them.
- **nextAttemptDelay jitter before re-querying Svix state** — Svix processes messages asynchronously; querying immediately after dispatch yields stale 'not found' states and causes unnecessary retries.

## Example: Adding a new channel type handler inside reconcileEvent

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
