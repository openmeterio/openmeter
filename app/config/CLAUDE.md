# config

<!-- archie:ai-start -->

> Defines the single shared Viper-based config.Configuration struct used by all seven binaries, one file per concern. Each file pairs a typed struct, a Validate() that accumulates errors via errors.Join, and a Configure*(v) that sets Viper defaults / pflag bindings.

## Patterns

**One struct + Validate() + Configure*() per concern file** — Each concern has a typed struct, a Validate() error accumulating via errors.Join, and a Configure*(v *viper.Viper) setting defaults. No global state, no side-effects beyond viper defaults. (`type BillingConfiguration struct { AdvancementStrategy billing.AdvancementStrategy; Worker BillingWorkerConfiguration }; func (c BillingConfiguration) Validate() error { var errs []error; /* ... */ return errors.Join(errs...) }`)
**SetViperDefaults as the single registration point** — config.go SetViperDefaults calls every Configure* function in order; a new concern must add its Configure* call here before it is loaded. ConfigureCredits is currently called twice (known duplication). (`func SetViperDefaults(v *viper.Viper, flags *pflag.FlagSet) { ConfigureBilling(v, flags); ConfigureProductCatalog(v); ConfigureApps(v, flags); /* ... */ ConfigureCredits(v, "credits") }`)
**Helper methods for derived client types** — Config structs expose helpers (AsURL(), GetClientOptions(), AsConsumerConfig(), AsConfigMap()) that translate raw fields into third-party client types, keeping translation out of app/common. (`func (c ClickHouseAggregationConfiguration) GetClientOptions() *clickhouse.Options { return &clickhouse.Options{Addr: []string{c.Address}, Auth: clickhouse.Auth{Database: c.Database, Username: c.Username, Password: c.Password}, ...} }`)
**Validate() accumulates all errors with errors.Join** — Validate() methods build var errs []error, append each failure (sub-struct errors prefixed with errorsx.WithPrefix or fmt.Errorf), and return errors.Join(errs...) so all misconfigs surface at once. (`if err := c.ConsumerConfiguration.Validate(); err != nil { errs = append(errs, errorsx.WithPrefix(err, "consumer")) }`)
**Consumer config squash embedding for Kafka worker configs** — Worker configs embed ConsumerConfiguration with mapstructure:",squash" to inherit Kafka consumer settings without field duplication. (`type BalanceWorkerConfiguration struct { ConsumerConfiguration `mapstructure:",squash"`; StateStorage BalanceWorkerStateStorageConfiguration }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `config.go` | Root Configuration struct aggregating all sub-configs; SetViperDefaults registers all defaults; Validate() fans out to every sub-validator and validates meter definitions (setting ManagedResource per meter). | Credits is validated twice (prefixes 'credits' and 'credit') and ConfigureCredits(v, "credits") is called twice in SetViperDefaults — known duplication. New sub-configs must be added to the struct, Validate(), and a Configure* call. |
| `billing.go` | BillingConfiguration + BillingFeatureSwitchesConfiguration; FeatureSwitches.NamespaceLockdown is a []string allowlist gating billing operations per namespace. | AdvancementStrategy references the openmeter/billing domain type directly — this is the intentional exception where a config file imports a domain package. |
| `aggregation.go` | ClickHouse connection config (TLS, retry, pool metrics); GetClientOptions() builds *clickhouse.Options for the ClickHouse client. | Numeric fields (dialTimeout, maxOpenConns, maxIdleConns, connMaxLifetime, blockBufferSize) must be > 0 — Validate() rejects zero values, so always set positive defaults in ConfigureAggregation or zero-value configs fail. |
| `kafka.go` | KafkaConfiguration / KafkaIngestConfiguration with AsConfigMap()/CreateKafkaConfig() helpers; the ConsumerConfig produced here is consumed by app/common/kafka.go. | KafkaIngestConfiguration.TopicProvisioner is decomposed via wire.FieldsOf in app/common/config.go — renaming a field requires updating the FieldsOf binding. |
| `credits.go` | CreditsConfiguration with Enabled and EnableCreditThenInvoice flags; injected into every provider that must be credits-guarded. | This is the type-safe flag checked in four independent wiring layers — do not add default-true fields without updating all ledger-guarded providers in app/common. |

## Anti-Patterns

- Adding business logic or mutable state to config structs — they are pure data containers with validation only
- Calling viper.SetDefault directly from cmd/* binaries instead of adding a Configure* function here
- Adding a config field without a corresponding Validate() check and Configure* default, producing silent zero-value misconfiguration
- Importing openmeter/* domain packages beyond openmeter/meter and openmeter/billing — keep domain imports minimal

## Decisions

- **Single shared config.Configuration type for all binaries** — wire.FieldsOf in app/common decomposes it into typed sub-structs per domain so each binary injects only what it needs, while all binaries load the same config shape — preventing config drift across binaries.
- **Configure* functions set Viper defaults instead of init()/package-level vars** — Explicit calls in SetViperDefaults make registration order visible, allow per-test selective invocation, and avoid global-state races between parallel test processes.
- **errors.Join accumulation in Validate() rather than fail-fast** — Surfacing all config errors at startup is more actionable for operators, especially when one env-var change produces several invalid fields.

## Example: Add a new config concern for a hypothetical domain

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
