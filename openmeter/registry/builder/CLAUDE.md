# builder

<!-- archie:ai-start -->

> Provides a single factory function GetEntitlementRegistry that wires together all entitlement-domain adapters, connectors, and services into a *registry.Entitlement struct. This is the non-Wire DI entry point used by balance-worker and test suites that need entitlement infrastructure without importing app/common.

## Patterns

**Options struct as constructor input** — All external dependencies are grouped into EntitlementOptions (DatabaseClient, StreamingConnector, Logger, MeterService, CustomerService, Publisher, Tracer, Locker, EntitlementsConfiguration). New parameters must be added to this struct, not as direct function arguments. (`opts EntitlementOptions passed to GetEntitlementRegistry`)
**Bottom-up adapter initialisation order** — DB adapters (featureDBAdapter, entitlementDBAdapter, grantDBAdapter, balanceSnapshotDBAdapter) are created first, then connectors that depend on them (featureConnector, creditConnector, meteredEntitlementConnector), then the top-level entitlement service. Do not create a connector before its adapter dependencies. (`featureDBAdapter -> featureConnector -> meteredEntitlementConnector -> entitlementConnector`)
**Register hooks after connector construction** — RegisterHooks is called immediately after the connector is constructed—not inline, not deferred. Both meteredEntitlementConnector and entitlementConnector each get their own RegisterHooks call. (`meteredEntitlementConnector.RegisterHooks(...); entitlementConnector.RegisterHooks(...)`)
**Return the registry struct by pointer** — GetEntitlementRegistry returns *registry.Entitlement with every field explicitly populated. Any new connector or adapter added to registry.Entitlement must be assigned here or callers will get nil fields. (`return &registry.Entitlement{Feature: featureConnector, GrantRepo: grantDBAdapter, ...}`)
**transactionManager via enttx.NewCreator** — The Ent transaction manager is created with enttx.NewCreator(opts.DatabaseClient) and passed into CreditConnectorConfig.TransactionManager. Do not pass the raw *db.Client as a transaction manager. (`transactionManager := enttx.NewCreator(opts.DatabaseClient)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `entitlement.go` | The only file in this package. Defines EntitlementOptions and GetEntitlementRegistry. Owns the full wiring sequence for entitlement-domain infrastructure. | Adding a new adapter/connector requires: (1) constructing it in the correct order, (2) passing it to downstream constructors, (3) assigning it to the returned registry struct. Skipping step 3 leaves callers with nil fields and runtime panics. |

## Anti-Patterns

- Importing app/common from this package — it must remain independent to avoid import cycles with test suites
- Skipping hook registration after connector construction — hooks like entitlementsubscriptionhook and credithook are mandatory for correct credit burn-down and subscription sync
- Passing opts.DatabaseClient directly as a TxCreator — always wrap with enttx.NewCreator
- Adding business logic or conditional wiring inside GetEntitlementRegistry — this is a pure factory; conditionals belong in the caller
- Returning a partially populated *registry.Entitlement — every field must be set or downstream nil-dereferences will occur

## Decisions

- **Single-function package instead of Wire provider set** — balance-worker and test suites need entitlement infrastructure without importing app/common Wire sets; a plain constructor avoids import cycles while still giving a fully wired registry
- **EntitlementOptions flat struct over variadic functional options** — All eight dependencies are mandatory; a flat struct makes missing fields a compile error (zero-value pointers panic early) and avoids the indirection of option functions

## Example: Constructing the entitlement registry for a worker or test

```
import (
	"github.com/openmeterio/openmeter/openmeter/registry/builder"
	"github.com/openmeterio/openmeter/openmeter/registry"
)

func buildRegistry(db *entdb.Client, ...) *registry.Entitlement {
	return registrybuilder.GetEntitlementRegistry(registrybuilder.EntitlementOptions{
		DatabaseClient:            db,
		EntitlementsConfiguration: cfg.Entitlements,
		StreamingConnector:        streamingConn,
		Logger:                    logger,
		MeterService:              meterSvc,
		CustomerService:           customerSvc,
		Publisher:                 publisher,
		Tracer:                    tracer,
// ...
```

<!-- archie:ai-end -->
