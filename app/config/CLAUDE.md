# config

<!-- archie:ai-start -->

> Defines the single shared config.Configuration struct (Viper-based) used by all binaries, with one file per concern. Provides Configure* functions that set Viper defaults and pflag bindings, plus Validate() methods on every sub-struct.

## Patterns

**One config struct per concern file** — Each domain concern has its own file (billing.go, kafka.go, aggregation.go, etc.) with a typed struct, a Validate() error method, and a Configure*(v *viper.Viper) function setting defaults. (`type BillingConfiguration struct { AdvancementStrategy billing.AdvancementStrategy; Worker BillingWorkerConfiguration }; func (c BillingConfiguration) Validate() error`)
**Validate() returns errors.Join** — All Validate() methods accumulate errors using var errs []error + errs = append(errs, ...) + return errors.Join(errs...) to surface all failures at once. (`func (c BalanceWorkerConfiguration) Validate() error { var errs []error; ...; return errors.Join(errs...) }`)
**SetViperDefaults as the single registration point** — config.go's SetViperDefaults calls every Configure* function in order. New config concerns must add a Configure* call here to be picked up. (`func SetViperDefaults(v *viper.Viper, flags *pflag.FlagSet) { ...; ConfigureBilling(v, flags); ConfigureProductCatalog(v); ... }`)
**Sub-config helper methods for derived values** — Config structs expose helper methods (AsURL(), GetClientOptions(), AsConsumerConfig()) that transform config fields into types expected by third-party clients, keeping that translation out of app/common. (`func (c ClickHouseAggregationConfiguration) GetClientOptions() *clickhouse.Options`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `app/config/config.go` | Root Configuration struct aggregating all sub-configs. SetViperDefaults registers all defaults. Validate() fans out to all sub-config validators and validates meters. | credits is validated twice (both a Credits and credit prefix) — known duplication. New sub-configs must be added to both the struct and Validate(). |
| `app/config/billing.go` | BillingConfiguration and BillingFeatureSwitchesConfiguration. FeatureSwitches.NamespaceLockdown is a []string allowlist that gates billing ops per namespace. | AdvancementStrategy references openmeter/billing domain type directly — config package imports the billing domain package. |
| `app/config/aggregation.go` | ClickHouse connection config including TLS, retry, pool metrics. GetClientOptions() produces *clickhouse.Options for the client. | All numeric fields default to >0; Validate() rejects 0-valued fields — always set defaults in ConfigureAggregation. |

## Anti-Patterns

- Adding business logic or state to config structs — they are pure data containers with validation.
- Calling SetViperDefaults from tests without also calling each Configure* function — tests must call SetViperDefaults to get defaults.
- Adding config fields without corresponding Validate() check and Configure* default — silent zero-value misconfigurations result.

## Decisions

- **Single shared config.Configuration type for all binaries** — Wire FieldsOf in app/common/config.go decomposes it into typed sub-structs per domain; each binary only injects the sub-structs it needs, but all binaries load the same config file shape.

<!-- archie:ai-end -->
