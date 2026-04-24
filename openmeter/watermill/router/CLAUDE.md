# router

<!-- archie:ai-start -->

> Constructs the standard Watermill *message.Router with a fixed middleware stack: PoisonQueue (DLQ), DLQ telemetry with OTel tracing, CorrelationID, Recoverer, exponential-backoff Retry, optional ProcessingTimeout with context restore, and per-handler metrics. All workers use NewDefaultRouter as their router factory.

## Patterns

**Fixed middleware order enforced by NewDefaultRouter** — Middleware is added in a specific order: PoisonQueue → DLQTelemetry → CorrelationID → Recoverer → Retry → (RestoreContext+Timeout if configured) → HandlerMetrics. New workers must use NewDefaultRouter rather than constructing their own router. (`router, err := router.NewDefaultRouter(opts)`)
**RestoreContext wraps Timeout to fix watermill#467** — Timeout middleware overrides msg context permanently; RestoreContext saves and restores the original context after the handler returns. Always use RestoreContext immediately before Timeout — never add Timeout standalone. (`router.AddMiddleware(RestoreContext, middleware.Timeout(opts.Config.ProcessingTimeout))`)
**MaxRetries off-by-one correction** — Watermill's Retry executes MaxRetries+1 times; NewDefaultRouter subtracts 1 from opts.Config.Retry.MaxRetries before passing to middleware. config.RetryConfiguration.MaxRetries is the true total attempt count. (`if maxRetries > 0 { maxRetries = maxRetries - 1 }`)
**WarningLogSeverityError for non-fatal consumer errors** — Wrap errors with router.NewWarningLogSeverityError to log them as warnings instead of errors in DLQ telemetry. Used for expected transient failures (e.g. race between DB commit and Kafka message arrival). (`return router.NewWarningLogSeverityError(err)`)
**Options.Validate before router construction** — Options.Validate() checks Subscriber, Publisher, Logger, MetricMeter, Tracer, and Config.Validate(). Always validate options; construction will not succeed without all required fields. (`if err := opts.Validate(); err != nil { return nil, err }`)
**HandlerMetrics and DLQ telemetry require metric.Meter and trace.Tracer** — All OTel instruments are created at router construction time. Pass real sdkmetric/sdktrace providers in tests (NewMeterProvider().Meter(...)) — nil values are rejected by Validate. (`options.MetricMeter = sdkmetric.NewMeterProvider().Meter("test")
options.Tracer = sdktrace.NewTracerProvider().Tracer("test")`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `router.go` | NewDefaultRouter: constructs *message.Router with full middleware stack from Options. Config is sourced from config.ConsumerConfiguration (DLQ topic, retry params, processing timeout). | DLQ.Topic and ConsumerGroupName must be set in Config before passing to NewDefaultRouter; they are not validated by Options.Validate but missing values will cause Watermill panics at runtime. |
| `metrics.go` | HandlerMetrics (per-handler attempt metrics) and NewDLQTelemetryMiddleware (full message lifecycle with OTel span). Both are consumed internally by NewDefaultRouter. | NewDLQTelemetryMiddleware wraps the entire retry+DLQ span in a single OTel trace span — do not add another span wrapper around the router handler or traces will nest incorrectly. |
| `context.go` | RestoreContext middleware: saves msg.Context() before handler, defers restore. Must be applied immediately before Timeout. | If Timeout is used without RestoreContext, subsequent retries share a cancelled context and will immediately fail. |
| `errors.go` | WarningLogSeverityError sentinel type. DLQTelemetry checks for this type via errors.As to downgrade log level. | Only affects log severity — the message still goes to DLQ. Do not use as a way to suppress DLQ routing. |
| `logger.go` | warningOnlyLogger: Watermill logger adapter that downgrades Error→Warn for retry noise. Used exclusively in the Retry middleware within NewDefaultRouter. | warningOnlyLogger.With() returns a new warningOnlyLogger preserving the slog.Logger instance — ensure the returned adapter is used, not discarded. |
| `router_test.go` | Table-driven integration tests for DLQ, retry, and timeout scenarios using gochannel in-memory pubsub. Reference for valid Options construction in tests. | Tests use DoneCondition channel to await async handler completion — avoid time.Sleep; use done.Wait(120s) pattern. |

## Anti-Patterns

- Constructing a *message.Router directly instead of NewDefaultRouter — middleware order, DLQ wiring, and OTel tracing will be missing.
- Adding Timeout middleware without RestoreContext immediately before it — cancelled contexts will break retry logic.
- Setting MaxRetries in config to 0 intending 'no retries' — the subtraction logic leaves it at 0 which Watermill treats as 1 attempt; set to 1 for a single attempt with no retry.
- Returning non-WarningLogSeverityError for expected transient failures — they log at ERROR level and may alert unnecessarily.
- Skipping Options.Validate — missing Tracer/MetricMeter causes nil-pointer panics inside HandlerMetrics construction.

## Decisions

- **PoisonQueue as outermost middleware, HandlerMetrics as innermost** — PoisonQueue must catch all errors including those from Recoverer; HandlerMetrics must measure only the actual handler execution time, not the retry/DLQ overhead.
- **RestoreContext workaround for watermill#467** — Watermill's Timeout middleware permanently replaces msg context; without RestoreContext the next retry attempt starts with an already-cancelled context, causing immediate failure regardless of ProcessingTimeout.
- **warningOnlyLogger for Retry middleware** — Retry errors are expected transient failures; logging them as errors would pollute alert channels. Only final DLQ disposition should be an error log.

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
