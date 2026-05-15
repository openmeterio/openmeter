# registry

<!-- archie:ai-start -->

> Organisational folder owning the entitlement registry factory — a non-Wire DI entry point that wires all entitlement-domain adapters, connectors, and services into a single *registry.Entitlement struct for use by the balance-worker and test suites that need entitlement infrastructure without importing app/common.

## Patterns

**Single-function factory package** — One exported function GetEntitlementRegistry in builder/ produces a fully-populated *registry.Entitlement; callers never construct the struct manually. (`reg := registry.GetEntitlementRegistry(opts)`)
**Options struct as constructor input** — All dependencies (DatabaseClient, StreamingConnector, etc.) are collected in a flat EntitlementOptions struct, not variadic functional options. (`opts := registry.EntitlementOptions{DatabaseClient: db, StreamingConnector: ch}`)
**Bottom-up adapter initialisation order** — Adapters are constructed before connectors, connectors before services; no circular dependency at runtime. (`featureRepo -> featureConnector -> grantRepo -> creditConnector -> entitlementService`)
**Register hooks after connector construction** — entitlementsubscriptionhook and credithook must be registered after the connector is built but before the struct is returned. (`connector.RegisterHook(credithook.New(...))`)
**transactionManager via enttx.NewCreator** — Always wrap opts.DatabaseClient with enttx.NewCreator — never pass the raw *entdb.Client as the TxCreator. (`txCreator := enttx.NewCreator(opts.DatabaseClient)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/registry/builder/entitlement.go` | The sole factory function GetEntitlementRegistry that wires every entitlement sub-service. | Missing hook registrations after connector construction; partial population of the returned struct causing nil-dereferences downstream. |
| `openmeter/registry/entitlement.go` | Defines the Entitlement registry struct holding all entitlement sub-service fields (Feature, FeatureRepo, EntitlementOwner, CreditBalance, Grant, GrantRepo, MeteredEntitlement, Entitlement, EntitlementRepo). | Adding a new service field here without populating it in builder/entitlement.go causes nil-dereferences at runtime. |

## Anti-Patterns

- Importing app/common from this package — it must remain independent to avoid import cycles with test suites
- Skipping hook registration after connector construction — credithook and entitlementsubscriptionhook are mandatory for correct credit burn-down and subscription sync
- Passing opts.DatabaseClient directly as a TxCreator — always wrap with enttx.NewCreator
- Adding business logic or conditional wiring inside GetEntitlementRegistry — conditionals belong in the caller
- Returning a partially populated *registry.Entitlement — every field must be set or downstream nil-dereferences will occur

## Decisions

- **Single-function package instead of Wire provider set** — balance-worker and test suites need entitlement infrastructure without the full app/common import graph; a standalone factory avoids import cycles while keeping wiring centralised.
- **EntitlementOptions flat struct over variadic functional options** — Flat struct makes all required dependencies explicit and compile-time checkable without the indirection of functional option closures.

<!-- archie:ai-end -->
