# eventhandler

<!-- archie:ai-start -->

> Implements notification.EventHandler: the asynchronous dispatch + reconciliation loop that drives notification events to delivery via the webhook (Svix) provider. It owns the delivery-status state machine (PENDING/SENDING/RESENDING -> SUCCESS/FAILED) and a pglock-leader-elected periodic reconciler; the noop/ child supplies the disabled-mode implementation.

## Patterns

**Compile-time interface assertion** — Handler must satisfy notification.EventHandler, asserted at package scope so a drifting method set fails to compile. (`var _ notification.EventHandler = (*Handler)(nil)`)
**Config + Validate + New constructor** — New(config Config) calls config.Validate() (collecting errs into models.NewNillableGenericValidationError) and defaults zero-value durations/worker counts from notification.Default* constants. Never construct Handler directly. (`if err := config.Validate(); err != nil { return nil, err }; if config.ReconcileInterval == 0 { config.ReconcileInterval = notification.DefaultReconcileInterval }`)
**Fire-and-forget Dispatch with detached context** — Dispatch returns nil immediately and runs reconcileEvent in a goroutine under context.WithTimeout(context.Background(), DefaultDispatchTimeout) with a recover() guard. The caller's ctx is captured only as a trace link (trace.LinkFromContext), not propagated, so the async work survives the originating request. (`go func(){ defer recover...; ctx, cancel := context.WithTimeout(context.Background(), notification.DefaultDispatchTimeout); defer cancel(); tracex.StartWithNoValue(ctx, h.tracer, "event_handler.dispatch", trace.WithNewRoot(), trace.WithLinks(spanLink)).Wrap(fn) }()`)
**Leader-elected reconcile loop** — Start() loops while running.Load(), acquiring reconcilerLeaderLockKey via lockClient.Do; on pglock.ErrNotAcquired it just continues (another node is leader). A ticker fires Reconcile every reconcileInterval. Close() flips the atomic and closes stopCh via sync.OnceFunc. (`h.lockClient.Do(ctx, reconcilerLeaderLockKey, func(rCtx, _){ ticker := time.NewTicker(h.reconcileInterval); for { select { case <-ticker.C: h.Reconcile(rCtx) } } })`)
**Bounded worker-pool pagination in Reconcile** — Reconcile pages repo.ListEvents (PageSize 50) filtering for PENDING/SENDING/RESENDING with NextAttemptBefore = now - nextAttemptDelay (10s jitter), and processes each event through a semaphore.Weighted(workerPoolSize) + wg.Go worker with per-worker recover(). (`workerPool := semaphore.NewWeighted(h.workerPoolSize); workerPool.Acquire(ctx, 1); wg.Go(func(){ defer workerPool.Release(1); defer recover...; h.reconcileEvent(ctx, &event) })`)
**Delivery-status state machine via UpdateEventDeliveryStatusInput** — reconcileWebhookEvent computes a single *notification.UpdateEventDeliveryStatusInput per active status in a switch on status.State, then persists via repo.UpdateEventDeliveryStatus only when input != nil. Active statuses are first filtered (filterActiveDeliveryStatusesByChannelType) and priority-sorted (sortDeliveryStatusStateByPriority); SUCCESS/FAILED are terminal and skipped. (`switch status.State { case EventDeliveryStatusStatePending: input = &notification.UpdateEventDeliveryStatusInput{...} }; if input != nil { h.repo.UpdateEventDeliveryStatus(ctx, *input) }`)
**Typed webhook error classification** — Provider errors are classified with webhook.IsNotFoundError / IsMessageAlreadyExistsError / IsUnrecoverableError / IsValidationError and lo.ErrorsAs[webhook.RetryableError] to choose retry vs terminal-fail, recording the decision via package Err* sentinels (ErrSystemRecoverableError, ErrSystemUnrecoverableError, ErrUserSendAttemptsExhausted, etc.) as the status Reason. (`rErr, ok := lo.ErrorsAs[webhook.RetryableError](err); if ok { retryAfter = rErr.RetryAfter() }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler struct, Config/Validate, New constructor, Start/Close leader-loop lifecycle. | running is atomic.Bool; Start() refuses to run twice (CompareAndSwap). Close uses stopChClose (sync.OnceFunc) so closing twice is safe. LockClient (*pglock.Client) is required by Validate. |
| `dispatch.go` | Dispatch: async entry point invoked on event creation; spawns reconcileEvent in a detached goroutine. | Always returns nil even though work may fail later (logged only). Uses context.Background()+timeout intentionally; do not 'fix' this to propagate the request ctx or the dispatch dies when the HTTP request ends. |
| `reconcile.go` | Reconcile (periodic batch) and reconcileEvent (per-event channel fan-out by ChannelType). | nextAttemptDelay (10s) jitter is deliberate so downstream provider state settles before re-reconcile. Only ChannelTypeWebhook is handled; other types just log an error. Each worker has its own recover(). |
| `webhook.go` | reconcileWebhookEvent state machine plus getWebhookMessage/sendWebhookMessage/resendWebhookMessage helpers and eventAsPayload serializer. | Largest file (~600 lines). The per-state switch must always end up with input==nil (skip) or a fully-populated UpdateEventDeliveryStatusInput. pendingTimeout/sendingTimeout govern give-up. eventAsPayload must add a case for every new notification.EventType (delegates to httpdriver.FromEventAs*Payload) or it returns 'unknown event type'. |
| `deliverystatus.go` | Pure helpers: filterActiveDeliveryStatusesByChannelType and sortDeliveryStatusStateByPriority. | Filtering drops SUCCESS/FAILED and any status whose ChannelID isn't a webhook channel on event.Rule.Channels. Priority map ties FAILED and SUCCESS at 3; changing it reorders reconcile processing. |
| `noop/handler.go` | No-op notification.EventHandler for disabled/unwired mode. | Must stay side-effect free and never return a non-nil error; mirrors the real constructor signature only so DI can swap it in. |

## Anti-Patterns

- Propagating the caller's request context into Dispatch's goroutine instead of the detached context.Background()+timeout - the async delivery would be cancelled when the originating request finishes.
- Leaving a switch branch in reconcileWebhookEvent that neither sets input nor falls into the terminal SUCCESS/FAILED/no-op path - appends an 'unhandled reconciling state' error and silently stalls the delivery.
- Adding a new notification.EventType without a matching case in eventAsPayload (webhook.go) - sending fails with 'unknown event type' at delivery time.
- Constructing Handler directly or skipping config.Validate() - bypasses required-dependency checks (Repository, Webhook, Logger, Tracer, LockClient) and duration/worker defaulting.
- Spawning goroutines (Dispatch, Reconcile workers) without a recover() guard - a single panic would crash the worker process; every async block here defers recover() + debug.Stack() logging.

## Decisions

- **Reconciliation is gated behind a single pglock leader lock (reconcilerLeaderLockKey) with ErrNotAcquired treated as a benign skip.** — Multiple notification-service replicas can run, but only one should drive the periodic delivery-status reconcile to avoid duplicate provider calls.
- **Dispatch is fire-and-forget and reconciliation re-derives state from the webhook provider rather than trusting the in-flight send.** — Svix delivery is asynchronous; the reconcile loop polls provider message/delivery state with jitter (nextAttemptDelay) so transient/late provider updates eventually converge the local delivery status.
- **All async work is wrapped in tracex.Start/StartWithNoValue spans with explicit span attributes and AddEvent calls.** — The pipeline is detached from the request, so trace links + span events are the primary observability for why a delivery succeeded, retried, or failed.

## Example: Per-event reconcile fans out by channel type, then the webhook handler drives the delivery-status state machine.

```
func (h *Handler) reconcileEvent(ctx context.Context, event *notification.Event) error {
	fn := func(ctx context.Context) error {
		channelTypes := lo.UniqMap(event.Rule.Channels, func(c notification.Channel, _ int) notification.ChannelType { return c.Type })
		var errs []error
		for _, channelType := range channelTypes {
			switch channelType {
			case notification.ChannelTypeWebhook:
				if err := h.reconcileWebhookEvent(ctx, event); err != nil {
					errs = append(errs, err)
				}
			default:
				h.logger.ErrorContext(ctx, "unsupported channel type", "type", channelType)
			}
		}
		return errors.Join(errs...)
// ...
```

<!-- archie:ai-end -->
