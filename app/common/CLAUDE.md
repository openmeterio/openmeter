# common

<!-- archie:ai-start -->

> Houses all Google Wire provider sets and constructor functions that assemble domain services, Kafka/DB clients, telemetry, and per-binary run groups — zero business logic. One file per domain concern; openmeter_<binary>.go files define the composite Wire set per binary.

## Patterns

**One domain file per Wire set** — Each domain concern has exactly one file (billing.go, customer.go, ledger.go, ...) exporting a PascalCase wire.NewSet var. Binary-specific composite sets live in openmeter_<binary>.go and compose the domain sets. (`var Billing = wire.NewSet(BillingAdapter, NewBillingRatingService, NewBillingRegistry, NewBillingCustomerOverrideService)`)
**Registry structs for multi-service domains** — Group related services in a <Domain>Registry (BillingRegistry, AppRegistry, ChargesRegistry); callers depend on the registry. Nil-safe accessors encapsulate optional sub-registries. (`type BillingRegistry struct { Billing billing.Service; Charges *ChargesRegistry }; func (r BillingRegistry) ChargesServiceOrNil() charges.Service { if r.Charges == nil { return nil }; return r.Charges.Service }`)
**creditsConfig.Enabled guard in every ledger-touching provider** — Every provider that can trigger ledger writes independently checks creditsConfig.Enabled and returns a noop when false. Four independent layers: ledger.go, customer.go, customerbalance.go, billing.go (newChargesRegistry skipped). A single guard is insufficient. (`func NewLedgerAccountService(creditsConfig config.CreditsConfiguration, ...) ledgeraccount.Service { if !creditsConfig.Enabled { return ledgernoop.AccountService{} }; return accountservice.New(...) }`)
**Hook/validator registration as provider side-effects** — Cross-domain hooks (billing customer validator, subscription hook, ledger hook, entitlement validator) are registered inside app/common provider functions — never inside domain constructors — to avoid circular imports. (`customerService.RegisterRequestValidator(validator); subscriptionServices.Service.RegisterHook(subscriptionValidator) // inside NewBillingRegistry`)
**Noop (never nil) for optional integrations** — Optional integrations (Svix, credits, portal) return compile-time-asserting noop structs when disabled. Type-assert against the noop type to conditionally skip handler registration. (`func NewLedgerNamespaceHandler(ar ledger.AccountResolver) namespace.Handler { if _, ok := ar.(ledgernoop.AccountResolver); ok { return ledgernoop.NamespaceHandler{} }; return resolvers.NewNamespaceHandler(ar) }`)
**Closer function pattern for stateful resources** — Providers returning closeable resources (postgres driver, ClickHouse conn, Kafka consumer) return a func() closer as a second value; Wire propagates it to the binary's run.Group / shutdown. (`func NewPostgresDriver(...) (*pgdriver.Driver, func(), error) { d, err := pgdriver.NewPostgresDriver(...); return d, func() { d.Close() }, nil }`)
**config.go wire.FieldsOf for sub-struct injection** — config.go declares all wire.FieldsOf bindings decomposing config.Configuration into injectable sub-structs. A new injectable config sub-struct needs a FieldsOf entry here first. (`wire.FieldsOf(new(config.Configuration), "Credits"), wire.FieldsOf(new(config.BillingConfiguration), "FeatureSwitches")`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `billing.go` | Wires the full billing + charges stack; private newChargesRegistry runs only when creditsConfig.Enabled. Registers billing customer and subscription validators as side-effects inside NewBillingRegistry. | ChargesRegistry is nil when credits disabled — use BillingRegistry.ChargesServiceOrNil(), never access .Charges directly. Any new ledger write path must be gated on creditsConfig.Enabled. |
| `charges.go` | Constructors for every charges sub-service (flatfee, usagebased, creditpurchase, lineage, meta) plus their ledger handlers. Registers line engines via billingService.RegisterLineEngine as a side-effect. | Each charge type engine must be registered before the first invoice advance. Called only from newChargesRegistry in billing.go — never instantiate directly from a binary. |
| `ledger.go` | Full LedgerStack provider set. Every function independently checks creditsConfig.Enabled and returns noop types when false; NewLedgerNamespaceHandler skips the real handler via type-assert against ledgernoop.AccountResolver. | NewLedgerHistoricalLedger intentionally creates a second accountservice.New with no Querier to avoid circular deps — keep the comment. Every new ledger provider must add its own Enabled guard. |
| `customer.go` | Creates customer.Service and registers the ledger hook (creditsConfig.Enabled-guarded), subject hook, and entitlement validator hook as provider side-effects. | NewCustomerLedgerServiceHook must return ledgerresolvers.NoopCustomerLedgerHook{} when credits disabled — this is independent of ledger.go; both must be guarded. |
| `config.go` | Declares all wire.FieldsOf bindings decomposing config.Configuration into injectable sub-structs. Single source of truth for which config paths Wire knows about. | A new injectable config sub-struct needs a wire.FieldsOf entry here; a missing entry causes Wire compile failures in binary-specific graphs. |
| `openmeter_billingworker.go` | BillingWorker composite set for cmd/billing-worker, composing App, Customer, Secret, Lockr, Subscription, ProductCatalog, Entitlement, Billing, LedgerStack and wiring the Watermill subscriber and run.Group. | BillingWorkerGroup adds run.SignalHandler — do not add another. EnsureBusinessAccounts must run in the startup sequence before app.Run(). |
| `watermill.go` | Watermill + Kafka publisher/subscriber wiring. WatermillNoPublisher set is shared with sink-worker, which controls publisher close timing independently. | Sink-worker uses NewSinkWorkerPublisher (in openmeter_sinkworker.go) returning an empty closer — do not add publisher.Close() in FlushHandlerManager until drain is complete. |
| `telemetry.go` | Bootstraps OTel logger/meter/tracer providers, Prometheus handler, health checks, and the telemetry HTTP server. GlobalInitializer.SetGlobals() must be called early in main. | Shutdown contexts use context.Background() with a 5s timeout intentionally (post-cancel graceful shutdown), not a bug. |

## Anti-Patterns

- Adding business logic (validation, computation, state mutation) inside provider functions — they may only construct/wire plus side-effect hook registration
- Calling customerService.RegisterHooks / RegisterRequestValidator from domain packages instead of here, reintroducing circular imports
- Creating a new ledger-backed provider without a creditsConfig.Enabled guard returning a noop
- Accessing BillingRegistry.Charges directly without nil-check instead of ChargesServiceOrNil()
- Importing app/common from any openmeter/* or pkg/* package — this is a leaf that depends on everything else, never the reverse

## Decisions

- **Concentrate all wiring in app/common while domain packages expose plain constructors** — Compile-time Wire verification catches missing providers; seven binaries share provider sets without duplicating wiring; domain packages stay import-cycle-free because wiring flows outward only.
- **Guard the credits feature independently in each ledger-touching provider rather than at one choke point** — Credits cross-cuts ledger writes, customer hooks, namespace provisioning, and HTTP handlers across unrelated call graphs — no single injection point dominates all paths, so each must independently return noops.
- **Registry structs (BillingRegistry, AppRegistry, SubscriptionServiceWithWorkflow) instead of injecting individual services** — Groups cohesive services, reduces Wire graph complexity for callers, and lets ChargesServiceOrNil() encapsulate the credits-disabled nil case without scattered checks.

## Example: Add a new ledger-backed provider that must be disabled when credits are off

```
// app/common/newdomain.go
func NewMyDomainService(
    creditsConfig config.CreditsConfiguration,
    ledgerService ledger.Ledger,
    db *entdb.Client,
) (mydomain.Service, error) {
    if !creditsConfig.Enabled {
        return mydomainnoop.NewService(), nil
    }
    adapter, err := mydomainadapter.New(mydomainadapter.Config{Client: db})
    if err != nil {
        return nil, fmt.Errorf("init mydomain adapter: %w", err)
    }
    return mydomainservice.New(mydomainservice.Config{Adapter: adapter})
}
```

<!-- archie:ai-end -->
