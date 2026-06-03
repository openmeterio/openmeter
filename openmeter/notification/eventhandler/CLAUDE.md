# eventhandler

<!-- archie:ai-start -->

> Implements notification.EventHandler: Dispatch() fires domain events to Svix webhook channels in a detached goroutine, and a pg-advisory-lock-gated periodic Reconcile() loop drives stuck delivery statuses (Pending/Sending/Resending) to terminal states via a per-status state machine in webhook.go. The noop/ child provides a safe zero-value handler when the subsystem is disabled.

## Patterns

**Compile-time interface assertion** — handler.go declares the assertion so any new notification.EventHandler method without an implementation fails the build. (`var _ notification.EventHandler = (*Handler)(nil)`)
**Config + Validate() before construction** — New(config Config) calls config.Validate() first, returning models.NewNillableGenericValidationError; Repository, Webhook, Logger, Tracer, and LockClient are mandatory. Zero-valued intervals/timeouts default to package constants. (`func New(config Config) (*Handler, error) { if err := config.Validate(); err != nil { return nil, err }; if config.ReconcileInterval == 0 { config.ReconcileInterval = notification.DefaultReconcileInterval } ... }`)
**Dispatch fires a detached goroutine and always returns nil** — Dispatch() captures trace.LinkFromContext(ctx), launches a goroutine with context.WithTimeout(context.Background(), notification.DefaultDispatchTimeout) and a fresh root span (trace.WithNewRoot + WithLinks), recovers panics, and only logs errors. Callers must not rely on the return value for delivery confirmation. (`spanLink := trace.LinkFromContext(ctx); go func() { defer recover(); ctx, cancel := context.WithTimeout(context.Background(), notification.DefaultDispatchTimeout); defer cancel(); tracex.StartWithNoValue(ctx, h.tracer, "event_handler.dispatch", trace.WithNewRoot(), trace.WithLinks(spanLink)).Wrap(fn) }(); return nil`)
**pglock.Client.Do leader-election in Start()** — Start() loops calling h.lockClient.Do(ctx, reconcilerLeaderLockKey, ...) so only one replica runs the reconcile ticker; pglock.ErrNotAcquired is treated as non-error (continue and retry). running atomic.Bool guards double-Start. (`err := h.lockClient.Do(ctx, reconcilerLeaderLockKey, func(rCtx context.Context, _ *pglock.Lock) error { ticker := time.NewTicker(h.reconcileInterval); ... }); if errors.Is(err, pglock.ErrNotAcquired) { continue }`)
**Paginated reconcile scan with nextAttemptDelay jitter** — Reconcile pages delivery statuses (PageSize 50) filtered to Pending/Sending/Resending and NextAttemptBefore = clock.Now()-10s, fanning out to reconcileEvent via a semaphore-bounded worker pool. The 10s jitter lets Svix async state settle. (`nextAttemptBefore := clock.Now().Add(-1 * nextAttemptDelay); out, err := h.repo.ListEvents(ctx, notification.ListEventsInput{Page: page, DeliveryStatusStates: [...], NextAttemptBefore: nextAttemptBefore})`)
**Delivery-status state machine with sentinel failure reasons** — reconcileWebhookEvent switches on status.State (Pending/Resending/Sending) and stores one of the ErrUser*/ErrSystem* sentinel error vars verbatim as the DB Reason. Never introduce terminal reason strings outside webhook.go. (`switch status.State { case notification.EventDeliveryStatusStatePending: ...; case notification.EventDeliveryStatusStateSending: input = &notification.UpdateEventDeliveryStatusInput{State: notification.EventDeliveryStatusStateFailed, Reason: ErrUserSendAttemptsExhausted.Error(), ...} }`)
**sync.OnceFunc for idempotent Close()** — stopChClose wraps close(stopCh) in sync.OnceFunc so Close() is safe to call repeatedly; Close() flips running false, cancels ctx, and closes stopCh. (`stopChClose := sync.OnceFunc(func() { close(stopCh) })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler struct, Config + Validate(), New() constructor, Start()/Close() lifecycle (pglock.Do loop calling Reconcile on a ticker). | ReconcileInterval/PendingTimeout/SendingTimeout/ReconcilerWorkers default to package constants when zero — never pass zero expecting a no-op. atomic.Bool running guards double-Start. Start/Close are pointer receivers because stopCh/stopChClose are pointer state. |
| `dispatch.go` | Dispatch() goroutine launcher; captures the OTel span link, opens a fresh root span inside the goroutine, recovers panics. | Always returns nil; never check the return value for delivery success — the reconcile loop is the recovery path. |
| `reconcile.go` | Reconcile() paginates pending events and fans out via a semaphore worker pool; reconcileEvent dispatches per channel type to reconcileWebhookEvent. Defines nextAttemptDelay (10s). | Each worker has its own panic recovery; the 10s nextAttemptDelay must stay to avoid racing Svix async updates. Use the received ctx, not context.Background(). |
| `webhook.go` | reconcileWebhookEvent drives the per-delivery-status state machine against Svix; defines the six ErrUser*/ErrSystem* sentinel error vars stored verbatim as DB failure reasons. | The sentinels are the canonical human-readable reasons — do not add new terminal strings elsewhere. RetryableError.RetryAfter() controls the transient retry delay; IsNotFoundError/IsMessageAlreadyExistsError/IsUnrecoverableError gate transitions. |
| `deliverystatus.go` | Pure helpers: filterActiveDeliveryStatusesByChannelType and sortDeliveryStatusStateByPriority (no side effects). | Keep these pure — no logger/tracer/DB. Priority order Pending=0, Resending=1, Sending=2 determines reconcile order within one event. |

## Anti-Patterns

- Calling reconcileEvent synchronously from Dispatch — breaks the fire-and-forget contract; Watermill consumers must not block on webhook delivery
- Using context.Background() inside reconcileEvent or reconcileWebhookEvent instead of the received ctx — loses OTel spans and transaction bindings
- Adding terminal delivery-status reason strings outside webhook.go's sentinel error vars
- Skipping pglock.Do leader-election in Start() — multiple pods double-reconcile and produce duplicate Svix sends
- Making Start/Close value receivers — stopCh/stopChClose are pointer-receiver state and mixing receivers races

## Decisions

- **Dispatch fires a goroutine and always returns nil** — Watermill consumers must not block on webhook delivery; failures are non-fatal to the consumer pipeline and recovered by the reconcile loop.
- **Start() uses pglock.Client.Do for leader-election, not lockr advisory locks** — The reconcile loop must survive transaction boundaries; pglock holds a connection-scoped lock across many ticks, whereas pg_advisory_xact_lock releases on each transaction commit.
- **10s nextAttemptDelay before re-querying Svix state** — Svix processes messages asynchronously; querying immediately yields stale not-found states and causes false-positive failure transitions.

## Example: Add a new channel-type branch in reconcileEvent

```
// reconcile.go
for _, channelType := range channelTypes {
	switch channelType {
	case notification.ChannelTypeWebhook:
		if err := h.reconcileWebhookEvent(ctx, event); err != nil { errs = append(errs, err) }
	case notification.ChannelTypeSlack: // new type
		if err := h.reconcileSlackEvent(ctx, event); err != nil { errs = append(errs, err) }
	default:
		h.logger.ErrorContext(ctx, "unsupported channel type", "type", channelType)
	}
}
```

<!-- archie:ai-end -->
