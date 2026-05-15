# config

<!-- archie:ai-start -->

> Defines the single shared config.Configuration struct (Viper-based) used by all seven binaries, with one file per concern. Provides Configure* functions that set Viper defaults and pflag bindings, plus Validate() methods on every sub-struct that accumulate all errors via errors.Join.

## Patterns

**One config struct + Validate() + Configure*() per concern file** — Each domain concern has its own file with a typed struct, a Validate() error method accumulating errors via errors.Join, and a Configure*(v *viper.Viper) function setting all defaults. No global state, no side-effects. (`type BillingConfiguration struct { AdvancementStrategy billing.AdvancementStrategy; Worker BillingWorkerConfiguration }; func (c BillingConfiguration) Validate() error { var errs []error; ...; return errors.Join(errs...) }`)
**SetViperDefaults as the single registration point** — config.go's SetViperDefaults calls every Configure* function in order. New config concerns must add a Configure* call here before they will be loaded. Duplicate calls for Credits are a known bug (called twice). (`func SetViperDefaults(v *viper.Viper, flags *pflag.FlagSet) { ...; ConfigureBilling(v, flags); ConfigureProductCatalog(v); ConfigureApps(v, flags); ... }`)
**Sub-config helper methods for derived values** — Config structs expose helper methods (AsURL(), GetClientOptions(), AsConsumerConfig(), AsConfigMap()) that transform raw fields into types expected by third-party clients, keeping translation out of app/common. (`func (c ClickHouseAggregationConfiguration) GetClientOptions() *clickhouse.Options { return &clickhouse.Options{Addr: []string{c.Address}, Auth: clickhouse.Auth{Database: c.Database, ...}} }`)
**Validate() accumulates all errors with errors.Join** — All Validate() methods use var errs []error + errs = append(errs, ...) + return errors.Join(errs...) to surface all failures at once. Sub-struct errors are prefixed with errorsx.WithPrefix for context. (`func (c BalanceWorkerConfiguration) Validate() error { var errs []error; if err := c.ConsumerConfiguration.Validate(); err != nil { errs = append(errs, errorsx.WithPrefix(err, "consumer")) }; return errors.Join(errs...) }`)
**Consumer config squash embedding for Kafka worker configs** — Binary-specific worker configs (BalanceWorkerConfiguration, BillingWorkerConfiguration) embed ConsumerConfiguration with mapstructure:",squash" to inherit Kafka consumer settings without field duplication. (`type BalanceWorkerConfiguration struct { ConsumerConfiguration `mapstructure:",squash"`; StateStorage BalanceWorkerStateStorageConfiguration }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `app/config/config.go` | Root Configuration struct aggregating all sub-configs. SetViperDefaults registers all defaults. Validate() fans out to all sub-config validators and also validates meter definitions. | Credits is validated twice (both 'credits' and 'credit' prefix) — known duplication. New sub-configs must be added to both the struct and Validate(). New Configure* functions must be added to SetViperDefaults. |
| `app/config/billing.go` | BillingConfiguration and BillingFeatureSwitchesConfiguration. FeatureSwitches.NamespaceLockdown is a []string allowlist gating billing operations per namespace. | AdvancementStrategy references openmeter/billing domain type directly — this config file imports the billing domain package, which is an intentional exception to the usual dependency direction. |
| `app/config/aggregation.go` | ClickHouse connection config including TLS, retry, pool metrics. GetClientOptions() produces *clickhouse.Options for the ClickHouse client. | All numeric fields default to >0 and Validate() rejects 0-valued fields — always set positive defaults in ConfigureAggregation or tests will fail on zero-value configs. |
| `app/config/kafka.go` | KafkaConfiguration and KafkaIngestConfiguration with AsConfigMap()/CreateKafkaConfig() helpers. ConsumerConfig produced here is consumed by common.kafka.go. | KafkaIngestConfiguration.TopicProvisioner is decomposed via wire.FieldsOf in app/common/config.go — changing field names requires updating the FieldsOf binding. |
| `app/config/credits.go` | CreditsConfiguration with Enabled bool and EnableCreditThenInvoice bool. Injected into every provider that must be credits-guarded. | This struct is the type-safe flag checked in four independent wiring layers — do not add default-true fields without updating all ledger-guarded providers. |

## Anti-Patterns

- Adding business logic or state to config structs — they are pure data containers with validation only.
- Calling viper.SetDefault directly from cmd/* binaries instead of adding a Configure* function in this package.
- Adding config fields without a corresponding Validate() check and Configure* default — silent zero-value misconfigurations result.
- Importing openmeter/* domain packages beyond openmeter/meter and openmeter/billing — this package should have minimal domain imports.

## Decisions

- **Single shared config.Configuration type for all binaries** — Wire FieldsOf in app/common/config.go decomposes it into typed sub-structs per domain; each binary injects only what it needs, but all binaries load the same config file shape, preventing config drift between binaries.
- **Configure* functions set Viper defaults, not init() or package-level vars** — Explicit function calls in SetViperDefaults make the default registration order visible, allow per-test selective invocation, and avoid global state races between parallel test processes.
- **errors.Join accumulation in Validate() rather than fail-fast** — Surfacing all config errors at startup is more actionable for operators than stopping at the first invalid field, especially when multiple misconfigurations stem from a single environment variable change.

## Example: Adding a new config concern for a hypothetical domain

```
// app/config/newdomain.go
type NewDomainConfiguration struct {
    Enabled  bool
    Endpoint string
}

func (c NewDomainConfiguration) Validate() error {
    var errs []error
    if c.Enabled && c.Endpoint == "" {
        errs = append(errs, errors.New("endpoint is required when enabled"))
    }
    return errors.Join(errs...)
}

func ConfigureNewDomain(v *viper.Viper) {
// ...
```

<!-- archie:ai-end -->
