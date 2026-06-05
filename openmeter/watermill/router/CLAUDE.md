# router

<!-- archie:ai-start -->

> Builds the standard Watermill message.Router (NewDefaultRouter) with the project's middleware stack — poison-queue/DLQ, OTel metrics+tracing, correlation, recover, retry, timeout — driving every Kafka consumer (balance/billing/notification workers).

## Patterns

**Ordered middleware stack in NewDefaultRouter** — Order is load-bearing: PoisonQueueWithFilter (outermost) -> DLQ telemetry -> CorrelationID -> Recoverer -> Retry -> (optional RestoreContext+Timeout) -> HandlerMetrics. Don't reorder; comments explain why (e.g. Timeout after Retry, RestoreContext before Timeout). (`router.AddMiddleware(poisionQueue); router.AddMiddleware(dlqMetrics); router.AddMiddleware(middleware.CorrelationID, middleware.Recoverer)`)
**Options.Validate() gates all required deps** — NewDefaultRouter requires Subscriber, Publisher, Logger, MetricMeter, Tracer and a valid config.ConsumerConfiguration; nil any of them and construction errors out. (`if err := opts.Validate(); err != nil { return nil, err }`)
**MaxRetries off-by-one correction** — Watermill's Retry runs MaxRetries+1 times, so the code decrements opts.Config.Retry.MaxRetries by one before configuring the middleware. (`if maxRetries > 0 { maxRetries = maxRetries - 1 }`)
**Context restore around Timeout** — RestoreContext middleware saves/restores msg context (watermill issue 467) so the Timeout middleware's cancelled context doesn't leak into retries. (`router.AddMiddleware(RestoreContext, middleware.Timeout(opts.Config.ProcessingTimeout))`)
**WarningLogSeverityError downgrades DLQ log level** — DLQ telemetry middleware logs ErrorContext by default but switches to WarnContext when the error unwraps to *WarningLogSeverityError; wrap expected-failure errors with NewWarningLogSeverityError to avoid error-level noise. (`if _, ok := lo.ErrorsAs[*WarningLogSeverityError](err); ok { logger = opts.Logger.WarnContext }`)
**Metrics keyed by ce_type** — Both HandlerMetrics and DLQ telemetry tag metrics/spans with message.event_type from the ce_type header via metricAttributeTypeFromMessage, defaulting to 'UNKNOWN'. (`ce_type := msg.Metadata.Get(marshaler.CloudEventsHeaderType); if ce_type == "" { ce_type = unkonwnEventType }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `router.go` | Options/Validate and NewDefaultRouter — the canonical middleware assembly. | Middleware order is intentional and commented; PoisonQueueWithFilter skips DLQ when router.IsClosed() so in-flight messages are NAcked/retried instead of dead-lettered on shutdown. |
| `metrics.go` | HandlerMetrics (per-try) and NewDLQTelemetryMiddleware (full processing incl. retries) plus metricAttributeTypeFromMessage. | Handler failures are logged as Warn ('produced later than DB commit'); DLQ middleware sets msg context to a span context and restores it via defer. NewDLQTelemetryOptions.Validate requires MetricMeter/Logger/Router/Tracer. |
| `context.go` | RestoreContext middleware (watermill issue 467 workaround). | Must wrap the Timeout middleware, not replace it. |
| `errors.go` | WarningLogSeverityError + NewWarningLogSeverityError for log-severity control; implements Unwrap. | Only affects DLQ log level, not retry/DLQ behavior itself. |
| `logger.go` | warningOnlyLogger adapter that downgrades watermill Error() calls to slog Warn (used for the Retry middleware logger). | Watermill only has Error/Info levels; this is the workaround to keep retry noise at warn. |
| `router_test.go` | Table-driven gochannel-backed integration test covering happy/failed/retry/timeout/DLQ paths. | Uses an in-memory gochannel pubsub and a DoneCondition signal; good template for testing consumer wiring. |

## Anti-Patterns

- Reordering or removing middleware in NewDefaultRouter (DLQ/timeout/retry/context-restore interplay breaks).
- Constructing message.Router directly for a consumer instead of NewDefaultRouter, losing DLQ/metrics/retry.
- Logging expected/transient failures at error level instead of wrapping with NewWarningLogSeverityError.
- Adding the Timeout middleware without RestoreContext (cancelled context leaks into retries — issue 467).
- Forgetting the MaxRetries-1 correction, causing one extra attempt than configured.

## Decisions

- **Push to DLQ as the outermost step but skip it when router.IsClosed().** — On graceful shutdown Close() cancels contexts immediately; NAcking instead of dead-lettering lets Kafka redeliver rather than losing/duplicating to DLQ.
- **Single NewDefaultRouter with a fixed stack, escapable by building your own router.** — Standardizes observability and failure handling across all workers; the doc comment explicitly allows custom routers for special cases.

## Example: Build the standard consumer router

```
router, err := router.NewDefaultRouter(router.Options{
    Subscriber:  sub,
    Publisher:   pub,
    Logger:      logger,
    MetricMeter: metricMeter,
    Tracer:      tracer,
    Config:      consumerCfg, // config.ConsumerConfiguration
})
if err != nil { return err }
router.AddNoPublisherHandler("name", topic, sub, handler.Handle)
```

<!-- archie:ai-end -->
