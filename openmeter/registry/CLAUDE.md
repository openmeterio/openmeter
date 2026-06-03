# registry

<!-- archie:ai-start -->

> Organisational folder owning the entitlement registry: a registry.Entitlement struct (entitlement.go) plus a non-Wire factory in builder/ that wires every entitlement-domain adapter, connector, and service into one struct for callers (balance-worker, test suites) that must not import app/common.

## Patterns

**Registry struct + factory split** — entitlement.go defines the field-only Entitlement struct; builder/ holds the sole factory GetEntitlementRegistry that populates every field. Callers never build the struct manually. (`reg := builder.GetEntitlementRegistry(opts)`)
**Every struct field must be populated by the factory** — Adding a field to registry.Entitlement requires a matching assignment in builder/entitlement.go or downstream nil-derefs occur. (`Entitlement{Feature:..., Grant:..., MeteredEntitlement:..., Entitlement:...}`)
**Interface-typed fields only** — All fields are domain interface types (feature.FeatureConnector, credit.GrantConnector, entitlement.Service) — no concrete adapters leak through. (`EntitlementRepo entitlement.EntitlementRepo`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/registry/entitlement.go` | Field-only Entitlement registry struct holding all entitlement sub-service interfaces. | Adding a field without populating it in builder/entitlement.go causes runtime nil-dereferences. |

## Anti-Patterns

- Importing app/common from this folder — it must stay independent to avoid import cycles with test suites and balance-worker
- Putting wiring/factory logic in entitlement.go — wiring belongs only in builder/
- Returning a partially-populated *registry.Entitlement from the factory

## Decisions

- **Non-Wire single-function factory instead of a Wire provider set** — balance-worker and test suites need entitlement infrastructure without the full app/common import graph; a standalone factory avoids import cycles while centralising wiring.

<!-- archie:ai-end -->
