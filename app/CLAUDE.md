# app

<!-- archie:ai-start -->

> Organizational root for application-layer code split into two children: app/common (Google Wire DI wiring for all binaries) and app/config (shared Viper configuration structs). Contains no business logic; its sole constraint is that nothing in openmeter/* or pkg/* may import it.

## Patterns

**Leaf-node import direction** — app/common depends on all openmeter/* and pkg/* packages, never the reverse. app/config is imported by app/common and cmd/* only. (`openmeter/billing imports nothing from app/; app/common/billing.go imports openmeter/billing`)
**One child per concern** — app/common owns runtime wiring (Wire provider sets, hook/validator registration, noop fallbacks); app/config owns pure data structs with Validate() and SetViperDefaults. (`app/config/billing.go defines BillingConfig.Validate(); app/common/billing.go calls billing.New(adapter, ...) inside a Wire provider`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `app/common/billing.go` | Wire provider set for billing domain; constructs billing.Service, BillingRegistry, charges adapters | Adding business logic or state mutation inside provider functions; forgetting BillingRegistry.ChargesServiceOrNil() nil-check pattern |
| `app/common/ledger.go` | Wires ledger services; returns noop implementations when credits.enabled=false | Any new ledger-touching provider that lacks a creditsConfig.Enabled guard returning a noop |
| `app/common/customer.go` | Registers customer service hooks and request validators from billing and ledger packages | Calling RegisterHooks or RegisterRequestValidator from domain packages instead of here (causes circular imports) |
| `app/config/config.go` | Root config.Configuration struct used by all binaries; assembles all sub-configs | Adding fields without a corresponding Validate() check and Configure* default |

## Anti-Patterns

- Importing app/common from any openmeter/* or pkg/* package
- Adding business logic (validation, computation, state mutation) inside app/common provider functions
- Creating a ledger-backed provider without a credits.Enabled guard returning a noop
- Calling customerService.RegisterHooks or RegisterRequestValidator from domain packages instead of app/common
- Adding config fields to app/config structs without Validate() coverage and SetViperDefaults registration

## Decisions

- **All DI wiring concentrated in app/common with plain constructors in domain packages** — Compile-time Wire verification; single edit point to add a provider to any binary without duplicating graphs
- **Credits feature guarded independently in each ledger-touching provider in app/common** — Credits cross-cuts multiple unrelated call graphs (HTTP handlers, customer hooks, namespace provisioning); no single choke point can gate all of them

<!-- archie:ai-end -->
