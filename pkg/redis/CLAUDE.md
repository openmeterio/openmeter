# redis

<!-- archie:ai-start -->

> Thin factory layer for go-redis/v9 client construction from a Viper-backed Config struct, with built-in OTel tracing and metrics instrumentation via redisotel. Supports standalone and Sentinel topologies.

## Patterns

**Config.Validate before NewClient** — Config.Validate() uses errors.Join to accumulate field errors; always call Validate() before constructing the client in application wiring. (`if err := cfg.Validate(); err != nil { return err }`)
**Functional options for OTel providers** — OTel TracerProvider and MeterProvider are injected via Option funcs (WithTracingProvider, WithMeterProvider) layered on top of Options; nil providers are silently ignored. (`NewClient(Options{Config: cfg}, WithTracingProvider(tp), WithMeterProvider(mp))`)
**Configure sets Viper defaults with prefix** — Call Configure(v, prefix) to register all Redis defaults under a namespaced Viper prefix (e.g. "redis") — do not set defaults inline in app config. (`redis.Configure(v, "dedupe.redis")`)
**TLS minimum version is TLS 1.3** — When TLS.Enabled is true, the client always sets MinVersion: tls.VersionTLS13 — do not override this in callers. (`tlsConfig = &tls.Config{InsecureSkipVerify: o.TLS.InsecureSkipVerify, MinVersion: tls.VersionTLS13}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `client.go` | NewClient factory: applies options, builds TLS config, constructs standalone or Sentinel client, instruments tracing and metrics. | redisotel.InstrumentTracing and InstrumentMetrics are called even when providers are nil (opts slice is empty but call still happens) — do not short-circuit these calls. |
| `config.go` | Config struct, Validate, Configure (Viper defaults), and Config.NewClient convenience method. | Config.NewClient() does not accept OTel options — use NewClient(Options{Config: c}, WithTracingProvider(...)) directly when telemetry is needed. |

## Anti-Patterns

- Constructing go-redis client directly in application code instead of via this package — OTel instrumentation would be missing.
- Calling Configure with an empty prefix — all keys would collide at the root Viper namespace.
- Ignoring Config.Validate() — an empty Address will produce a client that fails at first use, not at construction.

## Decisions

- **OTel instrumentation is always attempted, not gated on provider presence** — redisotel.InstrumentTracing/Metrics are no-ops when no provider is passed, so always calling them avoids conditional branching and ensures new callers get observability automatically when they wire providers.

<!-- archie:ai-end -->
