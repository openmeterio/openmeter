# pgxpoolobserver

<!-- archie:ai-start -->

> Single-function OTel metrics registration for pgxpool.Pool, exposing 13 pool statistics as observable gauges/counters via a single RegisterCallback call. Call ObservePoolMetrics once per pool at startup; the registered callback polls pool.Stat() on each OTel collection cycle.

## Patterns

**Single RegisterCallback for all metrics** — Collect all metric.Observable instances into allMetrics and pass them to meter.RegisterCallback in one call. Never register separate callbacks per metric — OTel requires all observables used inside a callback to be declared in the same RegisterCallback call. (`_, err = meter.RegisterCallback(func(_ context.Context, o metric.Observer) error { ... }, allMetrics...)`)
**Append-then-register pattern** — Each metric is created with meter.Int64ObservableCounter/Gauge, error-checked immediately (return on error), then appended to allMetrics before RegisterCallback. New metrics must follow this exact sequence. (`m, err := meter.Int64ObservableGauge("pgxpool.idle_conns", ...)
if err != nil { return err }
allMetrics = append(allMetrics, m)`)
**additionalAttributes variadic for label injection** — Callers pass extra attribute.KeyValue pairs via the variadic additionalAttributes parameter; every ObserveInt64 call appends them via metric.WithAttributes(additionalAttributes...). (`o.ObserveInt64(idleConnsMetric, int64(stat.IdleConns()), metric.WithAttributes(additionalAttributes...))`)
**Guard derived metrics against division by zero** — avgAcquiredDurationMetric is only observed when acquireCount > 0. Replicate this guard for any future ratio or average metrics. (`if acquireCount > 0 { o.ObserveInt64(avgAcquiredDurationMetric, acquireDurationMS/acquireCount, ...) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `observer.go` | Entire package. Exposes only ObservePoolMetrics(meter, pool, ...attributes). 13 metrics registered in a single function. | Each of the 13 meter.Int64Observable* calls is checked for error and returns immediately on failure. Adding a new metric requires: create → check error → append to allMetrics → observe inside callback. |

## Anti-Patterns

- Calling ObservePoolMetrics more than once for the same pool/meter pair — duplicate metric registration will error.
- Adding a new metric without appending it to allMetrics — it will never be observed in the callback.
- Omitting the division-by-zero guard on derived ratio metrics — produces NaN or panic in the callback.

## Decisions

- **Observable push model (RegisterCallback) instead of explicit Record calls** — Pool stats are instantaneous gauge values; polling them only when the OTel collector pulls avoids background goroutines and unnecessary polling when no collector is active.

<!-- archie:ai-end -->
