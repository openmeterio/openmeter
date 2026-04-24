# pgxpoolobserver

<!-- archie:ai-start -->

> Single-function OTel metrics registration for pgxpool.Pool, exposing 13 pool statistics as observable gauges/counters. Call once per pool at startup; the registered callback polls pool.Stat() on each collection cycle.

## Patterns

**Register all observables in one RegisterCallback call** — All metric.Observable instances are collected into a slice and passed to meter.RegisterCallback in a single call — do not register separate callbacks per metric. (`_, err = meter.RegisterCallback(func(_ context.Context, o metric.Observer) error { ... }, allMetrics...)`)
**additionalAttributes variadic for label injection** — Callers pass extra attribute.KeyValue pairs (e.g. db name, instance ID) via the variadic additionalAttributes parameter; every ObserveInt64 call appends them. (`o.ObserveInt64(acquireCountMetric, acquireCount, metric.WithAttributes(additionalAttributes...))`)
**Guard derived metrics against division by zero** — avgAcquiredDurationMetric is only observed when acquireCount > 0 — replicate this guard for any derived ratio metrics. (`if acquireCount > 0 { o.ObserveInt64(avgAcquiredDurationMetric, acquireDurationMS/acquireCount, ...) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `observer.go` | Entire package; exposes only ObservePoolMetrics(meter, pool, ...attributes). | Error from each meter.Int64ObservableCounter/Gauge registration is checked individually and returned immediately — all 13 registrations must succeed or the function returns an error. |

## Anti-Patterns

- Calling ObservePoolMetrics more than once for the same pool/meter pair — duplicate metric registration will error.
- Adding new metrics without appending to allMetrics — they will not be observed in the callback.

## Decisions

- **Observable push model (RegisterCallback) instead of explicit Record calls** — Pool stats are instantaneous gauge values polled by the metrics collector; push-on-collection avoids background goroutines and unnecessary polling when no collector is active.

<!-- archie:ai-end -->
