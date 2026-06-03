# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing meter.Service and meter.ManageService. Owns all DB reads/writes for meter entities, enforcing namespace isolation and soft-delete via DeletedAt.

## Patterns

**TransactingRepo wrapping on every method** — Mutating methods wrap their body in transaction.Run + entutils.TransactingRepo (or WithNoValue); read-only methods use entutils.TransactingRepo directly. (`return transaction.Run(ctx, a, func(ctx) (Meter, error) { return entutils.TransactingRepo(ctx, a, func(ctx, repo *Adapter) (Meter, error) { ... }) })`)
**Tx/WithTx/Self triad** — Adapter implements entutils.TxCreator: Tx() hijacks via HijackTx+NewTxDriver, WithTx() builds txClient from raw config, Self() returns self. Omitting any breaks TransactingRepo rebinding. (`func (a *Adapter) WithTx(ctx, tx *entutils.TxDriver) *Adapter { return &Adapter{db: db.NewTxClientFromRawConfig(ctx, *tx.GetConfig()).Client(), logger: a.logger} }`)
**Config-validated constructor** — New(Config) validates Client and Logger before constructing; returns error on invalid config. (`func New(config Config) (*Adapter, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**Interface compliance assertion** — var _ meter.Service = (*Adapter)(nil) asserts compile-time satisfaction; add similar assertions for new interfaces. (`var _ meter.Service = (*Adapter)(nil)`)
**MapFromEntityFactory for all domain mapping** — All query paths convert *db.Meter to meter.Meter via MapFromEntityFactory — never build domain objects inline from Ent entities. (`m, err := MapFromEntityFactory(entity)`)
**Soft-delete via DeletedAt filter** — Meters are soft-deleted by SetDeletedAt(clock.Now()); queries default to meterdb.DeletedAtIsNil() unless IncludeDeleted is set. (`if !params.IncludeDeleted { query = query.Where(meterdb.DeletedAtIsNil()) }`)
**Typed error wrapping** — db.IsConstraintError -> models.NewGenericConflictError; db.IsNotFound -> meter.NewMeterNotFoundError; otherwise fmt.Errorf-wrapped. (`if db.IsConstraintError(err) { return Meter{}, models.NewGenericConflictError(...) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Adapter struct, Config, constructor, and Tx/WithTx/Self triad. | Forgetting any of Tx/WithTx/Self breaks TransactingRepo for this adapter. |
| `manage.go` | Create/Update/Delete plus cross-entity checks (HasActiveFeatureForMeter, HasEntitlementForMeter, ListFeaturesForMeter); uses featureadapter.MapFeatureEntity. | Cross-entity queries use entitlementdb/featuredb predicates — missing namespace Where clauses can leak cross-namespace data. |
| `meter.go` | ListMeters (pagination, filtering, ordering), GetMeterByIDOrSlug, and MapFromEntityFactory. | The OrderBy switch must handle all meter.OrderBy constants or return GenericValidationError on default. |
| `adapter_test.go` | Integration tests using testutils.InitPostgresDB + Schema.Create on a real DB. | Use t.Context(); context.Background() breaks test-scoped cancellation and transaction cleanup. |

## Anti-Patterns

- Calling a.db directly in a helper without wrapping in entutils.TransactingRepo — bypasses ctx-carried transaction
- Hand-writing SQL instead of using Ent query builders
- Editing openmeter/ent/db/ generated files directly
- Returning raw Ent entities instead of mapping via MapFromEntityFactory
- Omitting DeletedAt IS NULL where soft-deleted meters should be excluded

## Decisions

- **Adapter implements both meter.Service and meter.ManageService at the DB layer; service/ wraps it for business logic** — Keeps persistence isolated; hooks, event publishing, and cross-domain validation live in service/ without mixing into SQL.
- **Soft-delete via DeletedAt rather than hard DELETE** — Meters referenced by historical billing/usage data must not be physically removed, preserving audit trails and FK integrity.

## Example: Create a meter inside a transaction

```
func (a *Adapter) CreateMeter(ctx context.Context, input meterpkg.CreateMeterInput) (meterpkg.Meter, error) {
	return transaction.Run(ctx, a, func(ctx context.Context) (meterpkg.Meter, error) {
		return entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *Adapter) (meterpkg.Meter, error) {
			entity, err := repo.db.Meter.Create().SetNamespace(input.Namespace).SetKey(input.Key).SetAggregation(input.Aggregation).Save(ctx)
			if err != nil {
				if db.IsConstraintError(err) { return meterpkg.Meter{}, models.NewGenericConflictError(fmt.Errorf("meter already exists")) }
				return meterpkg.Meter{}, fmt.Errorf("failed to create meter: %w", err)
			}
			return MapFromEntityFactory(entity)
		})
	})
}
```

<!-- archie:ai-end -->
