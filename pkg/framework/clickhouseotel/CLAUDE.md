# clickhouseotel

<!-- archie:ai-start -->

> OpenTelemetry instrumentation wrappers around a ClickHouse connection: ClickHouseTracer wraps clickhouse.Conn to span every Query/QueryRow/Exec/AsyncInsert, and ConnPoolMetrics polls Conn.Stats() to emit pool gauges and ping health metrics.

## Patterns

**Config struct + Validate() + New constructor** — Each type has a paired XxxConfig with a Validate() error that collects nil-dependency checks into []error and returns errors.Join(errs...); the New constructor calls cfg.Validate() first and returns (T, error). (`func NewConnPoolMetrics(cfg ConnPoolMetricsConfig) (*ConnPoolMetrics, error) { if err := cfg.Validate(); err != nil { return nil, err } ... }`)
**Embedded clickhouse.Conn delegation** — ClickHouseTracer embeds clickhouse.Conn so all un-overridden methods pass through; only the query-executing methods are overridden to add spans. Compile-time assertion var _ clickhouse.Conn = (*ClickHouseTracer)(nil) guards the interface. (`type ClickHouseTracer struct { clickhouse.Conn; Tracer trace.Tracer }`)
**Span-per-query with error recording** — Every wrapped method opens a span with query+args attributes, defers span.End(), and on error calls span.RecordError(err) + span.SetStatus(codes.Error, err.Error()) before returning the original error unchanged. (`ctx, span := c.Tracer.Start(ctx, "clickhouse.Query", trace.WithAttributes(attribute.String("query", query)))`)
**Idempotent start/shutdown via sync.OnceFunc + atomic** — ConnPoolMetrics guards Start with started atomic.Bool (Swap detects double-start) and closes stopChan/doneChan exactly once with sync.OnceFunc; Shutdown signals stopClose then waits on doneChan only if started. (`if m.started.Swap(true) { return errors.New("conn pool metrics already started") }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `otel.go` | ClickHouseTracer wrapping clickhouse.Conn; one span per Query/QueryRow/Exec/AsyncInsert; anyToStrings(args...) stringifies args for the span attribute. | args are serialized into span attributes via fmt.Sprintf("%v") — avoid putting secrets in query args. Errors are recorded on the span but returned unmodified; do not swallow them. |
| `connpool.go` | Background poller emitting clickhouse.pool.* gauges (open/idle counts + pct of max) and clickhouse.ping_time_ms / ping_failures_total via a ticker loop in run(). | ping timeout is clamped to <=5s; Shutdown blocks on doneChan, so a hung ping could delay shutdown by up to that timeout. record() is also called once before the first tick so the series exists. |

## Anti-Patterns

- Mutating or wrapping the returned error inside the tracer methods — callers expect the underlying clickhouse error semantics; only record it on the span.
- Calling Start twice on ConnPoolMetrics or closing stopChan/doneChan directly instead of via the OnceFunc closers.
- Constructing either type without going through its New constructor (skipping Validate and metric registration).

## Decisions

- **ConnPoolMetrics records once immediately before the ticker loop** — Ensures the metric series exists from the moment Start runs rather than only after the first poll interval elapses.
- **Ping timeout is bounded independently of poll interval** — A long poll interval must not let a single ping block shutdown indefinitely, since Shutdown waits for the run loop to drain.

## Example: Wrap a ClickHouse conn with tracing

```
tracer, err := clickhouseotel.NewClickHouseTracer(clickhouseotel.ClickHouseTracerConfig{Tracer: tp.Tracer("clickhouse"), Conn: conn})
if err != nil { return err }
// tracer is a clickhouse.Conn that spans every query
```

<!-- archie:ai-end -->
