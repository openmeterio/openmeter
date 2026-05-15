# common

<!-- archie:ai-start -->

> Houses all Google Wire provider sets and constructor functions that assemble domain services, Kafka clients, database clients, telemetry, and binary-specific run groups — zero business logic lives here. One file per domain concern; openmeter_*.go files define Wire sets per binary.

## Patterns

**One domain file per Wire set** — Each domain concern has exactly one file (billing.go, customer.go, ledger.go, etc.) exporting a PascalCase wire.NewSet var. Binary-specific composite sets live in openmeter_<binary>.go and compose domain sets. (`var Billing = wire.NewSet(BillingAdapter, NewBillingRatingService, NewBillingRegistry, NewBillingCustomerOverrideService)`)
**Registry structs for multi-service domains** — When a domain exposes multiple related services, group them in a <Domain>Registry struct (BillingRegistry, AppRegistry, ChargesRegistry). Callers depend on the registry. Nil-safe accessors like ChargesServiceOrNil() encapsulate optional sub-registries. (`type BillingRegistry struct { Billing billing.Service; Charges *ChargesRegistry }; func (r BillingRegistry) ChargesServiceOrNil() charges.Service { if r.Charges == nil { return nil }; return r.Charges.Service }`)
**Credits.Enabled guard in every ledger-touching provider** — Every provider that can trigger ledger writes must independently check creditsConfig.Enabled and return a noop when false. Four independent layers: ledger.go, customer.go, customerbalance.go, billing.go (newChargesRegistry skipped). A single guard is insufficient. (`func NewLedgerAccountService(creditsConfig config.CreditsConfiguration, repo ledgeraccount.Repo, locker *lockr.Locker) ledgeraccount.Service { if !creditsConfig.Enabled { return ledgernoop.AccountService{} }; return accountservice.New(repo, locker) }`)
**Hook and validator registration as Wire provider side-effects** — Cross-domain hooks (billing customer validator, subscription hook, ledger hook, entitlement validator) are registered inside app/common provider functions — not inside domain package constructors — to avoid circular imports. (`customerService.RegisterRequestValidator(validator); subscriptionServices.Service.RegisterHook(subscriptionValidator) // inside NewBillingRegistry`)
**Noop implementations for optional integrations** — Optional integrations (Svix, credits, portal) return compile-time-asserting noop structs when disabled. Type-assert against noop types (e.g. ledgernoop.AccountResolver) to conditionally skip handler registration. (`func NewLedgerNamespaceHandler(accountResolver ledger.AccountResolver) namespace.Handler { if _, ok := accountResolver.(ledgernoop.AccountResolver); ok { return ledgernoop.NamespaceHandler{} }; return resolvers.NewNamespaceHandler(accountResolver) }`)
**Closer function pattern for stateful resources** — Providers returning closeable resources (postgres driver, ClickHouse conn, Kafka consumer) return a func() closer as a second return value. Wire propagates these to the binary's run.Group or calls them on shutdown. (`func NewPostgresDriver(...) (*pgdriver.Driver, func(), error) { driver, err := pgdriver.NewPostgresDriver(...); return driver, func() { driver.Close() }, nil }`)
**config.go wire.FieldsOf for sub-struct injection** — config.go declares all wire.FieldsOf bindings decomposing config.Configuration into injectable sub-structs. Adding a new config sub-struct requires a wire.FieldsOf entry here before it can be injected downstream. (`wire.FieldsOf(new(config.Configuration), "Credits"), wire.FieldsOf(new(config.BillingConfiguration), "FeatureSwitches")`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `app/common/billing.go` | Wires the full billing + charges stack. newChargesRegistry is private and only invoked when creditsConfig.Enabled. Registers billing customer and subscription validators as side-effects inside NewBillingRegistry. | Any new ledger write path must be gated on creditsConfig.Enabled. ChargesRegistry is nil when credits disabled — always use BillingRegistry.ChargesServiceOrNil(). Never access BillingRegistry.Charges directly. |
| `app/common/charges.go` | Constructor functions for every charges sub-service (flatfee, usagebased, creditpurchase, lineage, meta) and their ledger handlers. Registers line engines with billingService.RegisterLineEngine as a side-effect. | Line engine registration order matters; each charge type engine must be registered before the first invoice advance. Called only from newChargesRegistry in billing.go — never instantiate directly from a binary. |
| `app/common/ledger.go` | Full LedgerStack Wire provider set. All functions independently check creditsConfig.Enabled and return noop types when false. NewLedgerNamespaceHandler skips real handler via type-assert against ledgernoop.AccountResolver. | NewLedgerHistoricalLedger intentionally creates a second accountservice.New with no Querier to avoid circular deps — do not remove the comment. Every new ledger provider must add its own Enabled guard. |
| `app/common/customer.go` | Creates customer.Service and registers the ledger hook (guarded by creditsConfig.Enabled), subject hook, and entitlement validator hook as Wire provider side-effects. | NewCustomerLedgerServiceHook must return ledgerresolvers.NoopCustomerLedgerHook{} when credits disabled. This is independent of ledger.go — both must be guarded. |
| `app/common/config.go` | Declares all wire.FieldsOf bindings decomposing config.Configuration into injectable sub-structs. Single source of truth for what config paths Wire knows about. | Adding a new config sub-struct requires a wire.FieldsOf entry here before it can be injected. Missing entries cause Wire compile failures in binary-specific graphs. |
| `app/common/openmeter_billingworker.go` | BillingWorker Wire set for cmd/billing-worker. Composes App, Customer, Secret, Lockr, Subscription, ProductCatalog, Entitlement, Billing, LedgerStack sets and wires Watermill subscriber and run.Group. | BillingWorkerGroup adds run.SignalHandler — do not add another one. EnsureBusinessAccounts must be in the startup sequence before app.Run(). |
| `app/common/watermill.go` | Watermill + Kafka publisher/subscriber wiring. WatermillNoPublisher set is shared with sink-worker which needs to control publisher close timing independently. | Sink-worker uses NewSinkWorkerPublisher (in openmeter_sinkworker.go) returning an empty closer — do not add publisher.Close() in FlushHandlerManager until drain is complete. |
| `app/common/telemetry.go` | Bootstraps OTel logger/meter/tracer providers, Prometheus handler, health checks, and telemetry HTTP server. GlobalInitializer.SetGlobals() must be called early in main. | Shutdown contexts use context.Background() with a 5s timeout — this is intentional (post-cancel graceful shutdown), not a bug. |

## Anti-Patterns

- Adding business logic (validation, computation, state mutation) inside app/common provider functions — providers must only construct and wire.
- Calling customerService.RegisterHooks / RegisterRequestValidator from domain packages — always do this inside app/common to avoid circular imports.
- Creating a new provider for ledger-backed features without a creditsConfig.Enabled guard returning a noop.
- Accessing BillingRegistry.Charges directly without nil-check — always use BillingRegistry.ChargesServiceOrNil().
- Importing app/common from any openmeter/* or pkg/* package — this is a leaf node depending on everything else, never the reverse.

## Decisions

- **All wiring concentrated in app/common with domain packages exposing plain constructors** — Compile-time Wire verification catches missing providers; seven binaries share provider sets without duplicating wiring; domain packages stay import-cycle-free because Wire wires through app/common outward only.
- **Credits feature guarded independently in each ledger-touching provider rather than a single choke point** — Credits cross-cuts ledger writes, customer hooks, namespace provisioning, and HTTP handlers across unrelated call graphs. No single injection point dominates all paths; each must independently return noops.
- **Registry structs (BillingRegistry, AppRegistry, SubscriptionServiceWithWorkflow) instead of individual service injection** — Groups logically cohesive services, reduces Wire graph complexity for callers (routers, workers), and lets ChargesServiceOrNil() encapsulate the credits-disabled nil case without scattered nil checks.

## Example: Adding a new ledger-backed provider that must be disabled when credits are off

```
// In app/common/newdomain.go
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
    return mydomainservice.New(mydomainservice.Config{
        Adapter: adapter,
// ...
```

<!-- archie:ai-end -->
