# builder

<!-- archie:ai-start -->

> Single-function package providing GetEntitlementRegistry, a non-Wire factory that wires all entitlement-domain adapters, connectors, and services into *registry.Entitlement. Used by balance-worker and test suites that need entitlement infrastructure without importing app/common.

## Patterns

**Options struct as constructor input** — All external dependencies are grouped into EntitlementOptions (DatabaseClient, StreamingConnector, Logger, MeterService, CustomerService, Publisher, Tracer, Locker, EntitlementsConfiguration). New parameters must be added to this struct, not as direct function arguments. (`GetEntitlementRegistry(EntitlementOptions{DatabaseClient: db, StreamingConnector: conn, ...})`)
**Bottom-up adapter initialisation order** — DB adapters (featureDBAdapter, entitlementDBAdapter, grantDBAdapter, balanceSnapshotDBAdapter) are created first, then connectors that depend on them, then the top-level entitlement service. Never create a connector before its adapter dependencies. (`featureDBAdapter -> featureConnector -> meteredEntitlementConnector -> entitlementConnector`)
**Register hooks immediately after connector construction** — RegisterHooks is called immediately after each connector is constructed — not inline, not deferred. Both meteredEntitlementConnector and entitlementConnector each get their own RegisterHooks call. (`meteredEntitlementConnector.RegisterHooks(meteredentitlement.ConvertHook(entitlementsubscriptionhook.NewEntitlementSubscriptionHook(...)))`)
**Return fully populated registry struct by pointer** — GetEntitlementRegistry returns *registry.Entitlement with every field explicitly populated. Any new connector or adapter added to registry.Entitlement must be assigned here or callers will get nil fields and runtime panics. (`return &registry.Entitlement{Feature: featureConnector, GrantRepo: grantDBAdapter, MeteredEntitlement: meteredEntitlementConnector, Entitlement: entitlementConnector, ...}`)
**transactionManager via enttx.NewCreator** — The Ent transaction manager is created with enttx.NewCreator(opts.DatabaseClient) and passed into CreditConnectorConfig.TransactionManager. Never pass the raw *db.Client as a transaction manager. (`transactionManager := enttx.NewCreator(opts.DatabaseClient)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `entitlement.go` | The only file in this package. Defines EntitlementOptions and GetEntitlementRegistry. Owns the full wiring sequence for entitlement-domain infrastructure. | Adding a new adapter/connector requires: (1) constructing it in the correct bottom-up order, (2) passing it to downstream constructors that depend on it, (3) assigning it to the returned registry struct. Skipping step 3 leaves callers with nil fields and runtime panics. |

## Anti-Patterns

- Importing app/common from this package — it must remain independent to avoid import cycles with test suites
- Skipping hook registration after connector construction — entitlementsubscriptionhook and credithook are mandatory for correct credit burn-down and subscription sync
- Passing opts.DatabaseClient directly as a TxCreator — always wrap with enttx.NewCreator
- Adding business logic or conditional wiring inside GetEntitlementRegistry — this is a pure factory; conditionals belong in the caller
- Returning a partially populated *registry.Entitlement — every field must be set or downstream nil-dereferences will occur at runtime

## Decisions

- **Single-function package instead of Wire provider set** — balance-worker and test suites need entitlement infrastructure without importing app/common Wire sets; a plain constructor avoids import cycles while still giving a fully wired registry
- **EntitlementOptions flat struct over variadic functional options** — All dependencies are mandatory; a flat struct makes missing fields a compile error (zero-value pointers panic early) and avoids the indirection of option functions

## Example: Constructing the entitlement registry for a worker or test

```
import (
	registrybuilder "github.com/openmeterio/openmeter/openmeter/registry/builder"
	"github.com/openmeterio/openmeter/openmeter/registry"
)

func buildRegistry(db *entdb.Client, streamingConn streaming.Connector, meterSvc meter.Service, customerSvc customer.Service, publisher eventbus.Publisher, tracer trace.Tracer, locker *lockr.Locker, cfg config.EntitlementsConfiguration, logger *slog.Logger) *registry.Entitlement {
	return registrybuilder.GetEntitlementRegistry(registrybuilder.EntitlementOptions{
		DatabaseClient:            db,
		EntitlementsConfiguration: cfg,
		StreamingConnector:        streamingConn,
		Logger:                    logger,
		MeterService:              meterSvc,
		CustomerService:           customerSvc,
		Publisher:                 publisher,
		Tracer:                    tracer,
// ...
```

<!-- archie:ai-end -->
