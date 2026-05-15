# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing meter.Service and meter.ManageService interfaces. Owns all DB reads and writes for meter entities, enforcing namespace isolation and soft-delete semantics via DeletedAt.

## Patterns

**TransactingRepo wrapping on every mutating method** — Every mutating DB operation wraps its body in transaction.Run + entutils.TransactingRepo (or WithNoValue). Reads-only methods use entutils.TransactingRepo directly without transaction.Run. (`return transaction.Run(ctx, a, func(ctx context.Context) (meterpkg.Meter, error) { return entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *Adapter) (meterpkg.Meter, error) { ... }) })`)
**Tx/WithTx/Self triad required by TransactingRepo** — Adapter implements entutils.TxCreator: Tx() hijacks transaction via HijackTx+NewTxDriver, WithTx() creates txClient from raw config, Self() returns self. Omitting any method breaks TransactingRepo rebinding. (`func (a *Adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *Adapter { txClient := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &Adapter{db: txClient.Client(), logger: a.logger} }`)
**Config-validated constructor** — New(Config) validates all required fields (Client, Logger) before constructing the Adapter; returns error on invalid config. (`func New(config Config) (*Adapter, error) { if err := config.Validate(); err != nil { return nil, err } return &Adapter{db: config.Client, logger: config.Logger}, nil }`)
**Interface compliance assertion** — var _ meter.Service = (*Adapter)(nil) at package level ensures compile-time interface satisfaction. Add similar assertions for any new interface this adapter implements. (`var _ meter.Service = (*Adapter)(nil)`)
**MapFromEntityFactory for all domain mapping** — All query paths call MapFromEntityFactory to convert *db.Meter to meter.Meter. Never construct domain objects inline from Ent entities. (`meter, err := MapFromEntityFactory(entity)`)
**Soft-delete via DeletedAt filter** — Meters are soft-deleted by setting DeletedAt; queries default to filtering DeletedAt IS NULL. Only omit the filter when IncludeDeleted is explicitly set. (`if !params.IncludeDeleted { query = query.Where(meterdb.DeletedAtIsNil()) }`)
**Typed error wrapping** — db.IsConstraintError -> models.NewGenericConflictError; db.IsNotFound -> meter.NewMeterNotFoundError; other errors wrapped with fmt.Errorf. (`if db.IsConstraintError(err) { return meterpkg.Meter{}, models.NewGenericConflictError(fmt.Errorf("meter with the same slug already exists")) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Core Adapter struct, Config, constructor, and Tx/WithTx/Self triad implementation. | Forgetting to implement any of Tx/WithTx/Self breaks TransactingRepo for this adapter. |
| `manage.go` | Create/Update/Delete meter + cross-entity checks (HasActiveFeatureForMeter, HasEntitlementForMeter, ListFeaturesForMeter). Uses featureadapter.MapFeatureEntity for feature mapping. | Cross-entity queries use entitlementdb and featuredb predicates; missing namespace Where clauses can leak cross-namespace data. |
| `meter.go` | ListMeters (pagination, filtering, ordering) and GetMeterByIDOrSlug plus MapFromEntityFactory. | Ordering switch must handle all meter.OrderBy constants or return GenericValidationError; missing cases silently use default ordering. |
| `adapter_test.go` | Integration tests using testutils.InitPostgresDB and schema.Create to provision a real schema. | Tests use t.Context(); using context.Background() breaks test-scoped cancellation and transaction cleanup. |

## Anti-Patterns

- Calling a.db directly in a helper method body without wrapping in entutils.TransactingRepo — bypasses ctx-carried transaction.
- Hand-writing SQL instead of using Ent query builders.
- Editing openmeter/ent/db/ generated files directly.
- Returning raw Ent entities from public functions instead of mapping through MapFromEntityFactory.
- Omitting DeletedAt IS NULL filter in queries where soft-deleted meters should be excluded.

## Decisions

- **Adapter implements both meter.Service and meter.ManageService at the DB layer; service/ wraps it for business logic.** — Keeps persistence concerns isolated; ManageService in service/ layer adds hooks, event publishing, and cross-domain validation without mixing them into SQL.
- **Soft-delete via DeletedAt rather than hard DELETE.** — Meters referenced by historical billing/usage data must not be physically removed to preserve audit trails and FK integrity.

## Example: Create a meter inside a transaction using TransactingRepo

```
func (a *Adapter) CreateMeter(ctx context.Context, input meterpkg.CreateMeterInput) (meterpkg.Meter, error) {
	return transaction.Run(ctx, a, func(ctx context.Context) (meterpkg.Meter, error) {
		return entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *Adapter) (meterpkg.Meter, error) {
			entity, err := repo.db.Meter.Create().
				SetNamespace(input.Namespace).
				SetKey(input.Key).
				SetAggregation(input.Aggregation).
				Save(ctx)
			if err != nil {
				if db.IsConstraintError(err) {
					return meterpkg.Meter{}, models.NewGenericConflictError(fmt.Errorf("meter already exists"))
				}
				return meterpkg.Meter{}, fmt.Errorf("failed to create meter: %w", err)
			}
			return MapFromEntityFactory(entity)
// ...
```

<!-- archie:ai-end -->
