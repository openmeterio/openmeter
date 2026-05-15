# redis

<!-- archie:ai-start -->

> Thin factory layer for go-redis/v9 client construction from a Viper-backed Config struct, with built-in OTel tracing and metrics instrumentation via redisotel. Supports standalone and Sentinel topologies; always enforces TLS 1.3 minimum when TLS is enabled.

## Patterns

**Validate before NewClient** — Config.Validate() uses errors.Join to accumulate field errors. Always call Validate() before constructing the client in Wire wiring or application startup. (`if err := cfg.Validate(); err != nil { return nil, err }`)
**Functional options for OTel providers** — TracerProvider and MeterProvider are injected via Option funcs (WithTracingProvider, WithMeterProvider). Nil providers are silently ignored; options are layered on top of the base Options struct. (`client, err := redis.NewClient(redis.Options{Config: cfg}, redis.WithTracingProvider(tp), redis.WithMeterProvider(mp))`)
**Configure sets Viper defaults with prefix** — Call Configure(v, prefix) to register all Redis defaults under a namespaced Viper prefix. Never call viper.SetDefault inline in app config for Redis settings. (`redis.Configure(v, "dedupe.redis")`)
**OTel instrumentation is always attempted** — redisotel.InstrumentTracing and InstrumentMetrics are called unconditionally; when no provider is set the opts slice is empty and the calls are effectively no-ops. Do not short-circuit these calls. (`if err := redisotel.InstrumentTracing(client, tracingOpts...); err != nil { return nil, err }`)
**TLS minimum version is TLS 1.3** — When TLS.Enabled is true, MinVersion is always set to tls.VersionTLS13. Do not override or relax this in callers. (`tlsConfig = &tls.Config{InsecureSkipVerify: o.TLS.InsecureSkipVerify, MinVersion: tls.VersionTLS13}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `client.go` | NewClient factory. Applies functional options, builds TLS config, constructs standalone or Sentinel client, instruments tracing and metrics. | redisotel.InstrumentTracing and InstrumentMetrics are called even when providers are nil (empty opts slice). Do not add nil-guard short-circuits — the unconditional call is intentional. |
| `config.go` | Config struct, Validate, Configure (Viper defaults), and Config.NewClient convenience method. | Config.NewClient() does not accept OTel options. Use NewClient(Options{Config: c}, WithTracingProvider(...)) directly when telemetry is needed. |

## Anti-Patterns

- Constructing go-redis client directly in application code — OTel instrumentation would be missing.
- Calling Configure with an empty prefix — all keys would collide at the root Viper namespace.
- Ignoring Config.Validate() — an empty Address produces a client that fails at first use, not at construction.
- Relaxing TLS MinVersion below tls.VersionTLS13 — this is a deliberate security invariant.

## Decisions

- **OTel instrumentation is always attempted, not gated on provider presence** — redisotel calls are no-ops when no provider is passed, so always calling them ensures new callers automatically get observability when they wire providers, without conditional branching.

## Example: Constructing a Redis client with OTel in Wire wiring

```
import (
    pkgredis "github.com/openmeterio/openmeter/pkg/redis"
    "go.opentelemetry.io/otel/metric"
    "go.opentelemetry.io/otel/trace"
)

func NewRedisClient(cfg pkgredis.Config, tp trace.TracerProvider, mp metric.MeterProvider) (*redis.Client, error) {
    if err := cfg.Validate(); err != nil {
        return nil, err
    }
    return pkgredis.NewClient(
        pkgredis.Options{Config: cfg},
        pkgredis.WithTracingProvider(tp),
        pkgredis.WithMeterProvider(mp),
    )
// ...
```

<!-- archie:ai-end -->
