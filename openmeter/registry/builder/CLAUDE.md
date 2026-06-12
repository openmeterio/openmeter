# builder

<!-- archie:ai-start -->

> In-domain composition root that wires the entitlement/credit/grant/feature stack into a single registry.Entitlement aggregate. GetEntitlementRegistry is the one assembly function consumed by app/common DI and by tests that need a fully-wired entitlement system without the full Wire graph.

## Patterns

**Single options-in, registry-out builder** — Wiring is exposed as one exported func GetEntitlementRegistry(opts EntitlementOptions) *registry.Entitlement; all external dependencies arrive via the EntitlementOptions struct, not package globals or slog.Default(). (`func GetEntitlementRegistry(opts EntitlementOptions) *registry.Entitlement { ... return &registry.Entitlement{...} }`)
**Adapters before connectors before services** — Construction follows a strict bottom-up order: Postgres repo adapters (NewPostgresFeatureRepo, NewPostgresEntitlementRepo, NewPostgresGrantRepo, NewPostgresBalanceSnapshotRepo) first, then connectors (feature, owner, credit, metered), then the entitlement service last. (`featureDBAdapter := productcatalogpgadapter.NewPostgresFeatureRepo(opts.DatabaseClient, opts.Logger)`)
**Config-struct constructors** — Connectors/services that take many deps are built with a typed config literal (balance.SnapshotServiceConfig, credit.CreditConnectorConfig, entitlementservice.ServiceConfig) rather than long positional arg lists. (`creditConnector := credit.NewCreditConnector(credit.CreditConnectorConfig{GrantRepo: grantDBAdapter, ...})`)
**Shared creditConnector aliasing** — One credit.NewCreditConnector instance is aliased to both creditBalanceConnector and grantConnector — the same object satisfies credit.BalanceConnector and credit.GrantConnector; do not build two separate connectors. (`creditBalanceConnector := creditConnector; grantConnector := creditConnector`)
**Hook registration at wire time** — Subscription and credit hooks are attached here via RegisterHooks on the metered and entitlement connectors, not inside the connector constructors. (`entitlementConnector.RegisterHooks(entitlementsubscriptionhook.NewEntitlementSubscriptionHook(...), credithook.NewEntitlementHook(grantDBAdapter))`)
**All nine registry fields populated** — The returned *registry.Entitlement must set every field (Feature, FeatureRepo, EntitlementOwner, CreditBalance, Grant, GrantRepo, MeteredEntitlement, Entitlement, EntitlementRepo) so downstream consumers can rely on non-nil connectors. (`return &registry.Entitlement{Feature: featureConnector, FeatureRepo: featureDBAdapter, ...}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `entitlement.go` | Defines EntitlementOptions and GetEntitlementRegistry — the sole builder that assembles feature, credit/grant, balance snapshot, metered/static/boolean entitlement connectors plus the entitlement service into registry.Entitlement. | enttx.NewCreator(opts.DatabaseClient) supplies TransactionManager to the credit connector; Granularity is hardcoded to time.Minute and SnapshotGracePeriod comes from opts.EntitlementsConfiguration.GetGracePeriod(); the static/boolean connectors are constructed inline with no args. |

## Anti-Patterns

- Building two separate credit connectors instead of aliasing one creditConnector to both CreditBalance and Grant.
- Using slog.Default() or package-level globals instead of taking Logger/Tracer/Publisher through EntitlementOptions.
- Registering hooks inside connector constructors rather than via RegisterHooks here at wire time.
- Returning a *registry.Entitlement with any field left nil.
- Adding business logic or queries here — this folder is composition only; logic belongs in the connector/service packages it wires.

## Decisions

- **Keep entitlement-stack wiring in an in-domain builder separate from app/common Wire registries.** — Lets tests (subscription/testutils, test/app, test/billing, test/customer) build a real entitlement system from one call without importing the full application DI graph, avoiding import cycles.
- **Reuse a single credit connector for both balance and grant roles.** — credit.NewCreditConnector implements both credit.BalanceConnector and credit.GrantConnector, so one instance keeps grant accounting and balance snapshotting consistent.

## Example: Assemble a fully-wired entitlement registry from external dependencies.

```
import (
	registrybuilder "github.com/openmeterio/openmeter/openmeter/registry/builder"
	"github.com/openmeterio/openmeter/openmeter/registry"
)

reg := registrybuilder.GetEntitlementRegistry(registrybuilder.EntitlementOptions{
	DatabaseClient:            dbClient,
	EntitlementsConfiguration: cfg.Entitlements,
	StreamingConnector:        streamingConn,
	Logger:                    logger,
	MeterService:              meterSvc,
	CustomerService:           customerSvc,
	Publisher:                 publisher,
	Tracer:                    tracer,
	Locker:                    locker,
// ...
```

<!-- archie:ai-end -->
