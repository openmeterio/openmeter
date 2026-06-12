# registry

<!-- archie:ai-start -->

> In-domain composition layer that bundles the entitlement/credit/grant/feature connectors into a single registry.Entitlement aggregate struct, so downstream wiring (app/common DI, tests) consumes one struct instead of nine separate connectors. The domain-side counterpart to app/common's Wire registries.

## Patterns

**Aggregate-struct registry** — registry.Entitlement is a plain struct of interface-typed fields, no methods; it only groups connectors built elsewhere. (`Entitlement{ Feature, FeatureRepo, EntitlementOwner, CreditBalance, Grant, GrantRepo, MeteredEntitlement, Entitlement, EntitlementRepo }`)
**Interface-typed fields only** — Every field is an interface (feature.FeatureConnector, credit.BalanceConnector, grant.OwnerConnector, entitlement.Service, ...) — no concrete adapters stored here. (`CreditBalance credit.BalanceConnector`)
**Assembly lives in builder/** — The struct is declared here; the only assembly function (GetEntitlementRegistry) lives in registry/builder and is the single entry point that populates all fields. (`registry/builder/entitlement.go`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `entitlement.go` | Declares the registry.Entitlement aggregate struct grouping feature/credit/grant/entitlement connectors. | Adding a field here means builder/ must populate it; a nil field returned from the builder breaks downstream consumers. |

## Anti-Patterns

- Adding business logic, queries, or methods to registry.Entitlement — it is a pure container.
- Storing concrete adapter types instead of the connector/service interfaces.
- Building the registry anywhere other than registry/builder.GetEntitlementRegistry.

## Decisions

- **Keep entitlement-stack wiring in an in-domain builder separate from app/common Wire registries.** — Tests need a fully-wired entitlement system without the full Wire graph.

## Example: The aggregate struct exposed to downstream wiring

```
type Entitlement struct {
	Feature            feature.FeatureConnector
	FeatureRepo        feature.FeatureRepo
	EntitlementOwner   grant.OwnerConnector
	CreditBalance      credit.BalanceConnector
	Grant              credit.GrantConnector
	GrantRepo          grant.Repo
	MeteredEntitlement meteredentitlement.Connector
	Entitlement        entitlement.Service
	EntitlementRepo    entitlement.EntitlementRepo
}
```

<!-- archie:ai-end -->
