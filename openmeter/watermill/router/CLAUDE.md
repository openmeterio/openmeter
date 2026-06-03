# router

<!-- archie:ai-start -->

> Constructs the standard Watermill *message.Router with a fixed, ordered middleware stack: PoisonQueue (DLQ), DLQ telemetry with OTel tracing, CorrelationID, Recoverer, exponential-backoff Retry, optional ProcessingTimeout+RestoreContext, and per-handler HandlerMetrics. All worker binaries must use NewDefaultRouter.

## Patterns

**Fixed middleware order via NewDefaultRouter** — Middleware order: PoisonQueue -> DLQTelemetry -> CorrelationID -> Recoverer -> Retry -> (RestoreContext+Timeout if configured) -> HandlerMetrics. New workers call NewDefaultRouter, not their own router. (`watermillRouter, err := router.NewDefaultRouter(router.Options{...})`)
**RestoreContext immediately before Timeout** — Timeout middleware permanently replaces msg.Context(); RestoreContext saves and restores the original context so retries start fresh. Always pair them in this order. (`router.AddMiddleware(RestoreContext, middleware.Timeout(opts.Config.ProcessingTimeout))`)
**MaxRetries off-by-one correction** — Watermill's Retry runs MaxRetries+1 times; NewDefaultRouter subtracts 1. config.RetryConfiguration.MaxRetries is the true total attempt count; MaxRetries=0 means one attempt, no retry. (`if maxRetries > 0 { maxRetries = maxRetries - 1 }`)
**WarningLogSeverityError for transient errors** — Wrap expected transient errors with router.NewWarningLogSeverityError; DLQTelemetry downgrades these to WARN but the message still goes to DLQ. (`return router.NewWarningLogSeverityError(err)`)
**Options.Validate before construction** — Options.Validate() checks Subscriber, Publisher, Logger, MetricMeter, Tracer, and Config.Validate(); missing MetricMeter/Tracer causes nil-pointer panics in HandlerMetrics. (`if err := opts.Validate(); err != nil { return nil, err }`)
**Real OTel providers required in tests** — All OTel instruments are created at router construction; pass real sdkmetric/sdktrace providers in tests — nil values are rejected by Validate. (`options.MetricMeter = sdkmetric.NewMeterProvider().Meter("test")`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `router.go` | NewDefaultRouter: builds *message.Router with the full middleware stack from Options; config from config.ConsumerConfiguration. | DLQ.Topic and ConsumerGroupName must be set in Config; they are not validated by Options.Validate but missing values panic at runtime. |
| `metrics.go` | HandlerMetrics (per-handler attempt metrics) and NewDLQTelemetryMiddleware (full lifecycle OTel span); both used internally. | NewDLQTelemetryMiddleware wraps the whole retry+DLQ span in one OTel span — do not add another span wrapper or traces nest incorrectly. |
| `context.go` | RestoreContext middleware: saves msg.Context() before the handler, restores after. Must precede Timeout. | Timeout without RestoreContext means retries share a cancelled context and fail immediately regardless of ProcessingTimeout. |
| `errors.go` | WarningLogSeverityError sentinel; DLQTelemetry checks via errors.As to downgrade log level. | Only affects log severity — the message still goes to DLQ; not a way to suppress DLQ routing. |
| `logger.go` | warningOnlyLogger downgrading Error->Warn for retry noise, used only in the Retry middleware. | warningOnlyLogger.With() returns a new logger — use the returned adapter, not the original. |
| `router_test.go` | Table-driven DLQ/retry/timeout tests using gochannel; reference for valid Options in tests. | Use the DoneCondition done.Wait(120s) pattern, never time.Sleep. |

## Anti-Patterns

- Constructing a *message.Router directly instead of NewDefaultRouter — middleware order, DLQ wiring, and OTel tracing will be absent.
- Adding Timeout middleware without RestoreContext immediately before it — cancelled contexts break retries.
- Setting MaxRetries=0 expecting 'no retries with no DLQ' — that means one attempt then DLQ; use MaxRetries=1 for one retry.
- Returning non-WarningLogSeverityError for expected transient failures — they log at ERROR and may trigger alerts.
- Skipping Options.Validate — missing Tracer/MetricMeter causes nil-pointer panics in HandlerMetrics.

## Decisions

- **PoisonQueue outermost, HandlerMetrics innermost.** — PoisonQueue must catch all errors including Recoverer's; HandlerMetrics must measure only actual handler time, not retry/DLQ overhead.
- **RestoreContext workaround for watermill#467.** — Timeout permanently replaces msg.Context(); without RestoreContext the next retry starts with a cancelled context, failing immediately.
- **warningOnlyLogger for the Retry middleware only.** — Retry errors are expected transient failures; logging at ERROR would pollute alerts — only final DLQ disposition is an error log.

## Example: Constructing the default router in a worker binary

```
import (
	"github.com/openmeterio/openmeter/openmeter/watermill/router"
	"github.com/openmeterio/openmeter/app/config"
)
watermillRouter, err := router.NewDefaultRouter(router.Options{
	Subscriber: kafkaSubscriber, Publisher: kafkaPublisher, Logger: logger,
	MetricMeter: metricMeter, Tracer: tracer,
	Config: config.ConsumerConfiguration{ConsumerGroupName: "billing-worker", DLQ: config.DLQConfiguration{Topic: "billing-worker-dlq"}},
})
```

<!-- archie:ai-end -->
