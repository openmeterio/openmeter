# clickhouseotel

<!-- archie:ai-start -->

> OTel instrumentation wrappers for ClickHouse connections: a tracing decorator (ClickHouseTracer) wrapping clickhouse.Conn for Query/QueryRow/Exec/AsyncInsert spans, and a ConnPoolMetrics poller emitting pool gauges and ping histograms via OTel metric.Meter.

## Patterns

**Decorator enforced by compile-time assertion** — ClickHouseTracer embeds clickhouse.Conn and overrides selected methods. var _ clickhouse.Conn = (*ClickHouseTracer)(nil) ensures any new upstream method is caught at compile time. (`var _ clickhouse.Conn = (*ClickHouseTracer)(nil)`)
**Config struct with Validate() before construction** — ConnPoolMetricsConfig and ClickHouseTracerConfig carry Validate() returning errors.Join(errs...); constructors call it first. Skipping Validate() means nil Conn/Meter/Tracer panics on first use. (`if c.Conn == nil { errs = append(errs, errors.New("conn is required")) }; return errors.Join(errs...)`)
**Start/Shutdown lifecycle with atomic guard** — ConnPoolMetrics.Start() uses started.Swap(true) to prevent double-start; Shutdown() calls stopClose (sync.OnceFunc) and blocks on <-doneChan. Always call Shutdown() to release the goroutine. (`if m.started.Swap(true) { return errors.New("conn pool metrics already started") }`)
**Span per ClickHouse call with error recording** — Each overridden method opens a tracer span, defers span.End(), calls the inner Conn method, and on failure calls span.RecordError + span.SetStatus(codes.Error). (`ctx, span := c.Tracer.Start(ctx, "clickhouse.Query", ...); defer span.End(); rows, err = c.Conn.Query(ctx, query, args...)`)
**Ping timeout capped at min(pollInterval, 5s)** — ping() derives a context timeout from pollInterval but caps it at 5s so a slow ClickHouse ping never blocks the poll cycle or Shutdown wait. (`if pingTimeout > 5*time.Second { pingTimeout = 5 * time.Second }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `otel.go` | ClickHouseTracer decorator; wraps Query, QueryRow, Exec, AsyncInsert with OTel spans. | Adding a new clickhouse.Conn method without a matching override here silently bypasses tracing for that call. |
| `connpool.go` | Background goroutine polling clickhouse.Conn.Stats() and emitting OTel gauge/histogram metrics. | Start() must not be called twice (atomic guard returns an error callers may ignore); always call Shutdown() to drain doneChan and avoid goroutine leaks. |

## Anti-Patterns

- Calling Start() more than once — the atomic guard returns an error callers may silently ignore.
- Skipping Validate() before NewConnPoolMetrics/NewClickHouseTracer — nil Conn/Meter/Tracer panics at first use.
- Using context.Background() inside record()/ping() instead of the ctx passed to Start() — breaks cancellation and span propagation.
- Adding a new clickhouse.Conn method without a corresponding override in ClickHouseTracer — calls bypass tracing silently.

## Decisions

- **Decorator pattern over forking the ClickHouse client.** — clickhouse.Conn is an interface; embedding and overriding selected methods adds instrumentation without maintaining a fork of the upstream client.
- **Poll-based metrics rather than callback hooks.** — The ClickHouse Go client exposes pool stats only via Stats(); OTel's push model requires explicit Record() calls, so a ticker goroutine is the only viable approach.

## Example: Wrap a ClickHouse connection with tracing and start pool metrics

```
import "github.com/openmeterio/openmeter/pkg/framework/clickhouseotel"

tracer, _ := clickhouseotel.NewClickHouseTracer(clickhouseotel.ClickHouseTracerConfig{Tracer: tracer, Conn: rawConn})
metrics, _ := clickhouseotel.NewConnPoolMetrics(clickhouseotel.ConnPoolMetricsConfig{Conn: rawConn, Meter: meter, Logger: logger, PollInterval: 15 * time.Second})
_ = metrics.Start(ctx)
defer metrics.Shutdown()
```

<!-- archie:ai-end -->
