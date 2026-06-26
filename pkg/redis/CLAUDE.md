# redis

<!-- archie:ai-start -->

> Redis client factory and configuration for OpenMeter, supporting standalone and Sentinel failover modes with optional TLS, plus OTel tracing/metrics instrumentation.

## Patterns

**Config struct with Validate + Configure(viper)** — Config holds Address/Database/auth plus nested Sentinel and TLS structs; Validate() aggregates errors via errors.Join; Configure(v, prefix) sets viper defaults under a prefix. (`func Configure(v *viper.Viper, prefix string) { v.SetDefault(fmt.Sprintf("%s.address", prefix), "127.0.0.1:6379") }`)
**Functional options for instrumentation** — NewClient(Options, ...Option) applies WithTracingProvider/WithMeterProvider options; nil providers are ignored. (`func WithTracingProvider(p trace.TracerProvider) Option { return func(o *Options) { if p != nil { o.TracingProvider = p } } }`)
**Sentinel vs standalone branch** — If Sentinel.Enabled, build redis.NewFailoverClient with MasterName + SentinelAddrs=[Address]; otherwise redis.NewClient with Addr=Address. (`if o.Sentinel.Enabled { client = redis.NewFailoverClient(&redis.FailoverOptions{MasterName: o.Sentinel.MasterName, SentinelAddrs: []string{o.Address}, ...}) }`)
**Always instrument with redisotel** — Both InstrumentTracing and InstrumentMetrics are always called (provider options only added when non-nil), errors wrapped with fmt.Errorf. (`if err := redisotel.InstrumentTracing(client, tracingOpts...); err != nil { return nil, fmt.Errorf(...) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `client.go` | Options struct (embeds Config + providers), Option functions, and NewClient that builds + instruments the *redis.Client. | TLS forces MinVersion TLS1.3; InsecureSkipVerify is wired from config. Sentinel uses single Address as the sentinel addr list. |
| `config.go` | Config struct, Validate(), Config.NewClient() convenience, and Configure(viper, prefix) defaults. | Configure sets an `expiration` default (24h) that is not a field on Config — it is read elsewhere. Validate requires Address and (when Sentinel enabled) MasterName. |

## Anti-Patterns

- Constructing redis.Client directly instead of via NewClient/Config.NewClient (skips OTel instrumentation).
- Adding config fields without a corresponding Configure() viper default and Validate() check.

## Decisions

- **Embed Config inside Options and layer functional Options for providers.** — Lets config-driven and DI-driven (tracer/meter) inputs combine without a wide constructor signature.

<!-- archie:ai-end -->
