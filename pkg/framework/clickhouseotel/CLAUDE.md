# clickhouseotel

<!-- archie:ai-start -->

> Provides OTel instrumentation wrappers for ClickHouse connections: a tracing decorator (ClickHouseTracer) that wraps clickhouse.Conn for Query/QueryRow/Exec/AsyncInsert spans, and a ConnPoolMetrics poller that emits connection-pool gauges and ping histograms via OTel metric.Meter.

## Patterns

**Decorator wraps interface** — ClickHouseTracer embeds clickhouse.Conn and overrides specific methods; compile-time assertion var _ clickhouse.Conn = (*ClickHouseTracer)(nil) enforces the contract. (`var _ clickhouse.Conn = (*ClickHouseTracer)(nil)`)
**Config struct with Validate()** — Both ConnPoolMetricsConfig and ClickHouseTracerConfig carry a Validate() method that returns errors.Join(errs...) before the constructor proceeds. (`func (c ConnPoolMetricsConfig) Validate() error { ... }`)
**Start/Shutdown lifecycle** — ConnPoolMetrics uses atomic.Bool to guard single Start(); Shutdown() signals stopChan and blocks on doneChan. sync.OnceFunc ensures channels close exactly once. (`stopClose := sync.OnceFunc(func() { close(stopChan) })`)
**Span per ClickHouse call** — Every overridden method opens a tracer span, defers span.End(), calls the inner Conn method, and calls span.RecordError + span.SetStatus on failure. (`ctx, span := c.Tracer.Start(ctx, "clickhouse.Query", ...); defer span.End()`)
**Ping timeout capped at pollInterval or 5s** — ping() uses context.WithTimeout capped at min(pollInterval, 5s) so ping can never block the full poll cycle. (`if pingTimeout > 5*time.Second { pingTimeout = 5*time.Second }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `otel.go` | ClickHouseTracer decorator; wraps Query, QueryRow, Exec, AsyncInsert with OTel spans. | Adding new clickhouse.Conn methods requires adding a matching override here; otherwise calls bypass tracing. |
| `connpool.go` | Background goroutine polling clickhouse.Conn.Stats() and emitting OTel gauge/histogram metrics. | Start() must not be called twice (atomic guard); always call Shutdown() to drain doneChan. |

## Anti-Patterns

- Calling Start() more than once — the atomic guard returns an error but callers may ignore it
- Skipping Validate() before NewConnPoolMetrics/NewClickHouseTracer — nil Conn/Meter/Tracer will panic at first use
- Using context.Background() inside record()/ping() instead of the ctx passed to Start()

## Decisions

- **Decorator pattern over subclassing** — clickhouse.Conn is an interface; embedding it and overriding selected methods adds instrumentation without forking the ClickHouse client.
- **Poll-based metrics rather than callback hooks** — ClickHouse Go client exposes pool stats via Stats() sync call; OTel push model requires explicit Record() calls, so a ticker goroutine is the only option.

## Example: Wrap a ClickHouse connection with tracing

```
import "github.com/openmeterio/openmeter/pkg/framework/clickhouseotel"

conn, err := clickhouseotel.NewClickHouseTracer(clickhouseotel.ClickHouseTracerConfig{
    Tracer: tracer,
    Conn:   rawConn,
})
```

<!-- archie:ai-end -->
