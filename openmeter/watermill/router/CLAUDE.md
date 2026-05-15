# router

<!-- archie:ai-start -->

> Constructs the standard Watermill *message.Router with a fixed, ordered middleware stack: PoisonQueue (DLQ), DLQ telemetry with OTel tracing, CorrelationID, Recoverer, exponential-backoff Retry, optional ProcessingTimeout+RestoreContext, and per-handler HandlerMetrics. All worker binaries must use NewDefaultRouter as their router factory.

## Patterns

**Fixed middleware order via NewDefaultRouter** — Middleware is added in a specific order: PoisonQueue → DLQTelemetry → CorrelationID → Recoverer → Retry → (RestoreContext+Timeout if configured) → HandlerMetrics. New workers must call NewDefaultRouter rather than constructing their own router. (`watermillRouter, err := router.NewDefaultRouter(router.Options{...})`)
**RestoreContext immediately before Timeout — never Timeout standalone** — Timeout middleware permanently replaces msg.Context() after returning; RestoreContext saves and restores the original context so subsequent retries start with a fresh context. Always pair them in this order. (`router.AddMiddleware(RestoreContext, middleware.Timeout(opts.Config.ProcessingTimeout))`)
**MaxRetries off-by-one correction** — Watermill's Retry executes MaxRetries+1 times total; NewDefaultRouter subtracts 1 before passing to the middleware. config.RetryConfiguration.MaxRetries is the true total attempt count. MaxRetries=0 means one attempt, no retry. (`if maxRetries > 0 { maxRetries = maxRetries - 1 }`)
**WarningLogSeverityError for non-fatal transient errors** — Wrap expected transient errors (e.g. race between DB commit and Kafka message arrival) with router.NewWarningLogSeverityError. DLQTelemetry downgrades these to WARN log level; the message still goes to DLQ. (`return router.NewWarningLogSeverityError(err)`)
**Options.Validate before construction** — Options.Validate() checks Subscriber, Publisher, Logger, MetricMeter, Tracer, and Config.Validate(). Missing MetricMeter or Tracer causes nil-pointer panics inside HandlerMetrics construction — always validate. (`if err := opts.Validate(); err != nil { return nil, err }`)
**Real OTel providers required in tests** — All OTel instruments are created at router construction time. Pass sdkmetric.NewMeterProvider().Meter(...) and sdktrace.NewTracerProvider().Tracer(...) in tests — nil values are rejected by Validate. (`options.MetricMeter = sdkmetric.NewMeterProvider().Meter("test")
options.Tracer = sdktrace.NewTracerProvider().Tracer("test")`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `router.go` | NewDefaultRouter: constructs *message.Router with full middleware stack from Options. Config sourced from config.ConsumerConfiguration. | DLQ.Topic and ConsumerGroupName must be set in Config before passing to NewDefaultRouter; they are not validated by Options.Validate but missing values will cause Watermill panics at runtime. |
| `metrics.go` | HandlerMetrics (per-handler attempt metrics) and NewDLQTelemetryMiddleware (full message lifecycle with OTel span). Both consumed internally by NewDefaultRouter. | NewDLQTelemetryMiddleware wraps the entire retry+DLQ span in a single OTel trace span — do not add another span wrapper around the router handler or traces will nest incorrectly. |
| `context.go` | RestoreContext middleware: saves msg.Context() before handler, defers restore after. Must be applied immediately before Timeout. | If Timeout is added without RestoreContext, subsequent retries share a cancelled context and will immediately fail regardless of ProcessingTimeout value. |
| `errors.go` | WarningLogSeverityError sentinel type. DLQTelemetry checks via errors.As to downgrade log level to WARN. | Only affects log severity — the message still goes to DLQ. Do not use as a way to suppress DLQ routing. |
| `logger.go` | warningOnlyLogger: Watermill logger adapter that downgrades Error→Warn for retry noise. Used exclusively in the Retry middleware within NewDefaultRouter. | warningOnlyLogger.With() returns a new warningOnlyLogger — ensure the returned adapter is used, not the original. |
| `router_test.go` | Table-driven integration tests for DLQ, retry, and timeout scenarios using gochannel in-memory pubsub. Reference for valid Options construction in tests. | Tests use DoneCondition channel to await async handler completion — use done.Wait(120s) pattern, never time.Sleep. |

## Anti-Patterns

- Constructing a *message.Router directly instead of NewDefaultRouter — middleware order, DLQ wiring, and OTel tracing will be absent.
- Adding Timeout middleware without RestoreContext immediately before it — cancelled contexts will break retry logic and cause immediate timeouts on all retry attempts.
- Setting MaxRetries=0 expecting 'no retries with no DLQ' — MaxRetries=0 means one attempt then DLQ; set MaxRetries=1 for one attempt plus one retry.
- Returning non-WarningLogSeverityError for expected transient failures — they log at ERROR level and may trigger unnecessary alerts.
- Skipping Options.Validate — missing Tracer/MetricMeter causes nil-pointer panics inside HandlerMetrics construction.

## Decisions

- **PoisonQueue outermost, HandlerMetrics innermost** — PoisonQueue must catch all errors including those from Recoverer; HandlerMetrics must measure only actual handler execution time, not the retry/DLQ overhead surrounding it.
- **RestoreContext workaround for watermill#467** — Watermill's Timeout middleware permanently replaces msg.Context() after the handler returns; without RestoreContext the next retry attempt starts with an already-cancelled context, causing immediate failure regardless of ProcessingTimeout.
- **warningOnlyLogger for Retry middleware only** — Retry errors are expected transient failures; logging them at ERROR would pollute alert channels. Only final DLQ disposition (surfaced by DLQTelemetry) should be an error log.

## Example: Constructing the default router in a worker binary

```
import (
    "github.com/openmeterio/openmeter/openmeter/watermill/router"
    "github.com/openmeterio/openmeter/app/config"
)

watermillRouter, err := router.NewDefaultRouter(router.Options{
    Subscriber:  kafkaSubscriber,
    Publisher:   kafkaPublisher, // for DLQ
    Logger:      logger,
    MetricMeter: metricMeter,
    Tracer:      tracer,
    Config: config.ConsumerConfiguration{
        ConsumerGroupName: "billing-worker",
        DLQ: config.DLQConfiguration{Topic: "billing-worker-dlq"},
        Retry: config.RetryConfiguration{
// ...
```

<!-- archie:ai-end -->
