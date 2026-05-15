# app

<!-- archie:ai-start -->

> Organizational root split into two children: app/common (Google Wire DI provider sets assembling all domain services into binary-specific run graphs) and app/config (single shared Viper configuration struct used by all seven binaries). Contains zero business logic; nothing in openmeter/* or pkg/* may import either child.

## Patterns

**Leaf-node import direction** — app/common depends on all openmeter/* and pkg/* packages; the reverse import direction is forbidden and causes import cycles. app/config is imported only by app/common and cmd/*. (`// OK: app/common/billing.go imports openmeter/billing
// NEVER: openmeter/billing imports app/common`)
**One child per concern** — app/common owns runtime wiring (Wire provider sets, hook/validator registration, noop fallbacks for optional integrations); app/config owns pure data structs with Validate() and SetViperDefaults. The two concerns must not bleed into each other. (`// app/config/billing.go: type BillingConfig struct { ... }; func (c BillingConfig) Validate() error
// app/common/billing.go: func NewBillingService(adapter billing.Adapter, ...) billing.Service { return billing.New(adapter) }`)
**Credits feature guard at every ledger-touching provider** — Any provider in app/common that wires ledger-backed features must independently check creditsConfig.Enabled and return a noop implementation when false. A single centralized guard is insufficient because credits cross-cuts multiple independent call graphs. (`func NewLedgerAccountService(cfg config.CreditsConfiguration, db *entdb.Client) ledger.AccountService {
    if !cfg.Enabled {
        return ledgernoop.AccountService{}
    }
    return ledgeraccount.New(db)
}`)
**Registry structs for multi-service domains** — Cohesive services are grouped into typed registry structs (BillingRegistry, AppRegistry, ChargesRegistry) with nil-safe accessor methods (ChargesServiceOrNil()) rather than individual injection into router.Config. (`svc := registry.ChargesServiceOrNil()
if svc == nil {
    return nil // credits disabled
}`)
**Hook and validator registration as Wire provider side-effects** — Cross-domain hooks (billing→customer, ledger→customer) and request validators are registered inside app/common provider functions as side-effects to avoid circular imports between domain packages. (`func NewCustomerService(adapter customer.Adapter, ledgerHook CustomerLedgerHook) customer.Service {
    svc := customer.New(adapter)
    svc.RegisterHooks(ledgerHook) // side-effect — not in domain constructor
    return svc
}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `app/common/billing.go` | Wire provider set for billing domain; constructs billing.Service and BillingRegistry including charges adapters | Adding business logic or state mutation inside providers; accessing BillingRegistry.Charges directly instead of ChargesServiceOrNil() |
| `app/common/ledger.go` | Wires ledger services; returns noop implementations when credits.enabled=false at every provider independently | Any new ledger-touching provider that lacks a creditsConfig.Enabled guard returning a noop type |
| `app/common/customer.go` | Registers customer service hooks and request validators from billing and ledger packages as Wire provider side-effects | Calling RegisterHooks or RegisterRequestValidator from domain packages instead of here — causes circular imports |
| `app/config/config.go` | Root config.Configuration struct used by all seven binaries; assembles all sub-configs; SetViperDefaults is the single Viper registration point | Adding config fields without a corresponding Validate() check and Configure* default registration |
| `app/common/charges.go` | Registers charge type line engines with billing.Service.RegisterLineEngine() as Wire provider side-effects | Calling RegisterLineEngine from domain packages or cmd/* — must happen here to avoid circular imports |
| `app/common/openmeter_billingworker.go` | Defines the BillingWorker Wire set composing provider sets needed by cmd/billing-worker | Missing hook providers that register silently — Wire sees only types, not side-effects; omitting a hook provider drops that hook with no compile error |

## Anti-Patterns

- Importing app/common from any openmeter/* or pkg/* package — reverses the leaf-node import direction and creates import cycles
- Adding business logic (validation, computation, state mutation) inside app/common provider functions — providers must only construct and wire
- Creating a ledger-backed provider without a creditsConfig.Enabled guard returning a noop — credits cross-cuts multiple call graphs
- Calling customerService.RegisterHooks or RegisterRequestValidator from domain package constructors instead of app/common providers
- Accessing BillingRegistry.Charges directly without ChargesServiceOrNil() — panics at runtime when credits are disabled

## Decisions

- **All DI wiring concentrated in app/common with plain constructors in domain packages** — Wire produces compile-time-verified dependency graphs; single edit point to add a provider to any binary without duplicating constructor chains in each cmd/*
- **Credits feature guarded independently in each ledger-touching provider in app/common rather than a single choke point** — Credits cross-cuts HTTP handlers, customer hooks, namespace provisioning, and charge creation — no single injection point dominates all call graphs that can reach ledger writes
- **Registry structs (BillingRegistry, AppRegistry) instead of individual service injection into router.Config** — Nil-safe accessor methods (ChargesServiceOrNil) encapsulate the credits-disabled nil case; individual field access would require nil checks scattered throughout callers

## Example: Adding a new ledger-backed provider in app/common that respects the credits feature flag

```
// app/common/ledger.go
import (
    ledgernoop "github.com/openmeterio/openmeter/openmeter/ledger/noop"
    "github.com/openmeterio/openmeter/app/config"
    entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
)

func NewLedgerTransactionService(
    creditsConfig config.CreditsConfiguration,
    db *entdb.Client,
) ledger.TransactionService {
    if !creditsConfig.Enabled {
        return ledgernoop.TransactionService{} // noop when credits disabled
    }
    return ledgertx.New(db)
// ...
```

<!-- archie:ai-end -->
