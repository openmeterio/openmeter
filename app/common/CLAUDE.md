# common

<!-- archie:ai-start -->

> Houses all Google Wire provider sets and constructor functions that wire domain services, adapters, Kafka clients, database clients, telemetry, and configuration into fully-assembled application structs. One file per domain area; openmeter_*.go files define binary-specific Wire sets. Contains zero business logic.

## Patterns

**One domain file per Wire set** — Each domain has exactly one file (billing.go, customer.go, subscription.go, etc.) exporting a PascalCase wire.NewSet var. Binary-specific sets (openmeter_server.go, openmeter_billingworker.go, etc.) compose domain sets. (`var Billing = wire.NewSet(BillingAdapter, NewBillingRatingService, NewBillingRegistry, NewBillingCustomerOverrideService)`)
**Registry structs for multi-service domains** — When a domain exposes multiple related services, group them in a <Domain>Registry struct (BillingRegistry, AppRegistry, SubscriptionServiceWithWorkflow) and pass the registry instead of individual services. (`type BillingRegistry struct { Billing billing.Service; Charges *ChargesRegistry }; func (r BillingRegistry) ChargesServiceOrNil() charges.Service`)
**Credits.Enabled guard at provider level** — Every provider that touches ledger writes checks creditsConfig.Enabled and returns a noop when false. This must be done independently in ledger.go, customer.go, customerbalance.go, and any new provider that uses ledger. (`if !creditsConfig.Enabled { return ledgernoop.AccountService{} }; return accountservice.New(...)`)
**Hook and validator registration inside wiring** — Cross-domain hooks (billing customer validator, subscription validator, entitlement validator, ledger hook) are registered inside provider functions in app/common, not in domain packages, to avoid circular imports. (`customerService.RegisterRequestValidator(validator); subscriptionServices.Service.RegisterHook(subscriptionValidator)`)
**Noop providers for optional features** — Optional integrations (Svix, credits, portal, progressmanager) return noop implementations when disabled. Type-assert against noop types (e.g. ledgernoop.AccountResolver) to conditionally skip handler registration. (`if _, ok := accountResolver.(ledgernoop.AccountResolver); ok { return ledgernoop.NamespaceHandler{} }`)
**Closer function pattern for stateful resources** — Providers returning closeable resources return a func() closer as a second return value. The wire graph propagates these closers to the binary's run.Group or calls them on shutdown. (`func NewPostgresDriver(...) (*pgdriver.Driver, func(), error)`)
**config.go wire.FieldsOf for sub-struct injection** — config.go declares all Wire FieldsOf bindings so downstream providers receive strongly-typed sub-config structs (e.g. config.BillingConfiguration, config.CreditsConfiguration) without importing the top-level Configuration. (`wire.FieldsOf(new(config.Configuration), "Credits"), wire.FieldsOf(new(config.BillingConfiguration), "FeatureSwitches")`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `app/common/billing.go` | Wires the full billing + charges stack. newChargesRegistry is private and only invoked when creditsConfig.Enabled. Registers billing customer and subscription validators as side-effects. | Any new ledger write path must be gated on creditsConfig.Enabled here. ChargesRegistry is nil when credits disabled — always use BillingRegistry.ChargesServiceOrNil(). |
| `app/common/charges.go` | Constructor functions for every charges sub-service (flatfee, usagebased, creditpurchase, lineage, meta) and their ledger handlers. Also registers line engines with billingService.RegisterLineEngine. | Line engine registration order matters; each charge type engine must be registered before the first invoice advance. Called only from newChargesRegistry in billing.go. |
| `app/common/ledger.go` | Full LedgerStack Wire provider set. All functions return noop types when credits disabled. NewLedgerNamespaceHandler skips real handler registration via type-assert against ledgernoop.AccountResolver. | NewLedgerHistoricalLedger contains a known hack: it creates a second accountservice.New with no Querier to avoid circular deps — do not remove the comment. |
| `app/common/customer.go` | Creates customer.Service and registers the ledger hook (guarded by creditsConfig.Enabled), subject hook, and entitlement validator hook. | NewCustomerLedgerServiceHook must return ledgerresolvers.NoopCustomerLedgerHook{} when credits disabled — bypassing this causes ledger writes when credits are off. |
| `app/common/config.go` | Declares all wire.FieldsOf bindings decomposing config.Configuration into injectable sub-structs. This is the single source of truth for what config paths Wire knows about. | Adding a new config sub-struct requires a wire.FieldsOf entry here before it can be injected. |
| `app/common/openmeter_billingworker.go` | BillingWorker Wire set for cmd/billing-worker. Composes App, Customer, Secret, Lockr, FFX, Subscription, ProductCatalog, Entitlement, Billing, LedgerStack sets and wires the Watermill subscriber and run.Group. | BillingWorkerGroup adds run.SignalHandler — do not add another one. |
| `app/common/telemetry.go` | Bootstraps OTel logger/meter/tracer providers, Prometheus handler, health checks, and telemetry HTTP server. GlobalInitializer.SetGlobals() must be called early in main. | Shutdown contexts use context.Background() with a 5s timeout — this is intentional to run after the parent ctx is cancelled. |
| `app/common/watermill.go` | Watermill + Kafka publisher/subscriber wiring. WatermillNoPublisher set is shared with sink-worker which needs to control publisher close timing independently. | Sink-worker uses NewSinkWorkerPublisher (in openmeter_sinkworker.go) which returns an empty closer — do not add publisher.Close() in FlushHandlerManager until drain is complete. |

## Anti-Patterns

- Adding business logic (validation, computation, state mutation) inside app/common provider functions — providers must only construct and wire.
- Calling customerService.RegisterHooks / RegisterRequestValidator from domain packages — always do this inside app/common to avoid circular imports.
- Creating a new provider for ledger-backed features without a creditsConfig.Enabled guard returning a noop.
- Depending on BillingRegistry.Charges directly without nil-check — always use BillingRegistry.ChargesServiceOrNil().
- Importing app/common from any openmeter/* or pkg/* package — this package is a leaf node that depends on everything else, never the reverse.

## Decisions

- **All wiring concentrated in app/common with domain packages exposing plain constructors** — Compile-time Wire verification catches missing providers; seven binaries share provider sets without duplicating wiring code; domain packages stay import-cycle-free.
- **Credits feature guarded independently in each ledger-touching provider** — Credits cross-cuts ledger, customer hooks, namespace provisioning, and HTTP handlers — no single choke point can gate all paths, so each must independently return noops.
- **Registry structs (BillingRegistry, AppRegistry, SubscriptionServiceWithWorkflow) instead of individual service injection** — Groups logically cohesive services, reduces Wire graph complexity for callers (routers, workers), and lets ChargesServiceOrNil() encapsulate the credits-disabled nil case.

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
