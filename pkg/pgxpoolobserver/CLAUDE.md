# pgxpoolobserver

<!-- archie:ai-start -->

> OpenTelemetry instrumentation helper that registers observable metrics for a pgxpool.Pool. Single-function package consumed by pkg/framework/pgdriver to expose connection-pool stats.

## Patterns

**Register-then-callback metric registration** — Each metric is created via meter.Int64ObservableCounter/Gauge, appended to allMetrics, then a single RegisterCallback reads pool.Stat() and ObserveInt64s every value. (`_, err = meter.RegisterCallback(func(_ context.Context, o metric.Observer) error { stat := pool.Stat(); o.ObserveInt64(acquireCountMetric, stat.AcquireCount(), ...); return nil }, allMetrics...)`)
**Fail-fast on metric creation error** — Every meter.Int64Observable* call is immediately error-checked and returns on failure before registering the callback. (`acquireCountMetric, err := meter.Int64ObservableCounter("pgxpool.acquire_count", ...); if err != nil { return err }`)
**Attribute pass-through** — additionalAttributes variadic is forwarded to every ObserveInt64 via metric.WithAttributes so callers can tag pool metrics. (`o.ObserveInt64(idleConnsMetric, int64(stat.IdleConns()), metric.WithAttributes(additionalAttributes...))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `observer.go` | Single exported func ObservePoolMetrics(meter, pool, ...attrs) that registers ~13 `pgxpool.*` metrics (acquire_count, idle_conns, total_conns, etc.) from pgxpool.Stat(). | Avg acquire duration is only observed when acquireCount > 0 to avoid divide-by-zero. Implementation is adapted from cmackenzie1/pgxpool-prometheus; keep metric names `pgxpool.*` stable for dashboards. |

## Anti-Patterns

- Registering multiple callbacks instead of one callback over allMetrics.
- Computing avg acquire duration without guarding acquireCount > 0.

## Decisions

- **One RegisterCallback reading pool.Stat() once per collection cycle.** — pool.Stat() is a consistent snapshot; reading it once per observe cycle keeps all metrics coherent and cheap.

<!-- archie:ai-end -->
