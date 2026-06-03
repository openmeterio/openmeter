# app

<!-- archie:ai-start -->

> Organizational root for application assembly, split into app/common (Google Wire DI provider sets that assemble every domain service into binary-specific run graphs) and app/config (the single shared Viper config.Configuration struct used by all seven binaries). Contains zero business logic; nothing in openmeter/* or pkg/* may import either child.

## Patterns

**Leaf-node import direction** — app/common depends on all openmeter/* and pkg/* packages; the reverse is forbidden and causes import cycles. app/config is imported only by app/common and cmd/*. (`// OK: app/common/billing.go imports openmeter/billing — NEVER: openmeter/billing imports app/common`)
**One child per concern** — app/common owns runtime wiring (provider sets, hook/validator registration, noop fallbacks); app/config owns pure data structs with Validate() and SetViperDefaults. The two concerns must not bleed together. (`// app/config/billing.go: type BillingConfig struct{...}; func (c BillingConfig) Validate() error`)
**Credits guard at every ledger-touching provider** — Any app/common provider wiring ledger-backed features must independently check creditsConfig.Enabled and return a noop when false; a single centralized guard is insufficient because credits cross-cut many call graphs. (`if !cfg.Enabled { return ledgernoop.AccountService{} }; return ledgeraccount.New(db)`)
**Registry structs for multi-service domains** — Cohesive services group into typed registry structs (BillingRegistry, AppRegistry, ChargesRegistry) with nil-safe accessors (ChargesServiceOrNil()) instead of individual injection into router.Config. (`svc := registry.ChargesServiceOrNil(); if svc == nil { return nil /* credits disabled */ }`)
**Hook/validator registration as provider side-effects** — Cross-domain hooks (billing→customer, ledger→customer) and request validators register inside app/common provider functions as side-effects to avoid circular imports between domain packages. (`svc := customer.New(adapter); svc.RegisterHooks(ledgerHook); return svc`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `common/billing.go` | Wire provider set for billing; constructs billing.Service and BillingRegistry including charges adapters. | Adding business logic in providers; accessing BillingRegistry.Charges directly instead of ChargesServiceOrNil(). |
| `common/ledger.go` | Wires ledger services; returns noop implementations when credits.enabled=false at every provider independently. | Any new ledger-touching provider lacking a creditsConfig.Enabled guard returning a noop type. |
| `common/customer.go` | Registers customer service hooks and request validators from billing/ledger as Wire provider side-effects. | Calling RegisterHooks/RegisterRequestValidator from domain packages instead of here — causes circular imports. |
| `common/charges.go` | Registers charge-type line engines via billing.Service.RegisterLineEngine() as provider side-effects. | Calling RegisterLineEngine from domain packages or cmd/* — must happen here to avoid circular imports. |
| `common/openmeter_billingworker.go` | Defines the BillingWorker Wire set composing the provider sets cmd/billing-worker needs. | Missing hook providers register silently — Wire sees only types, not side-effects; omitting one drops the hook with no compile error. |
| `config/config.go` | Root config.Configuration struct for all seven binaries; SetViperDefaults is the single Viper registration point. | Adding config fields without a matching Validate() check and Configure* default registration. |

## Anti-Patterns

- Importing app/common from any openmeter/* or pkg/* package — reverses the leaf-node import direction and creates cycles.
- Adding business logic (validation, computation, state mutation, panic/os.Exit) inside provider functions — providers may only construct and wire.
- Creating a ledger-backed provider without a creditsConfig.Enabled guard returning a noop.
- Calling RegisterHooks/RegisterRequestValidator from domain constructors instead of app/common providers.
- Accessing BillingRegistry.Charges directly without ChargesServiceOrNil() — panics when credits are disabled.

## Decisions

- **All DI wiring concentrated in app/common with plain constructors in domain packages.** — Wire produces compile-time-verified dependency graphs; a single edit point adds a provider to any binary without duplicating constructor chains per cmd/*.
- **Credits guarded independently in each ledger-touching provider rather than one choke point.** — Credits cross-cut HTTP handlers, customer hooks, namespace provisioning, and charge creation — no single injection point dominates all paths reaching ledger writes.
- **Registry structs instead of individual service injection into router.Config.** — Nil-safe accessors (ChargesServiceOrNil) encapsulate the credits-disabled nil case; individual field access would scatter nil checks across callers.

## Example: Adding a new ledger-backed provider that respects the credits flag

```
// app/common/ledger.go
import (
    ledgernoop "github.com/openmeterio/openmeter/openmeter/ledger/noop"
    "github.com/openmeterio/openmeter/app/config"
    entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
)

func NewLedgerTransactionService(creditsConfig config.CreditsConfiguration, db *entdb.Client) ledger.TransactionService {
    if !creditsConfig.Enabled {
        return ledgernoop.TransactionService{}
    }
    return ledgertx.New(db)
}
```

<!-- archie:ai-end -->
