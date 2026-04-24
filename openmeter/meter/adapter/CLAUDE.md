# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing meter.Service and meter.ManageService interfaces. Owns all DB reads and writes for meter entities, enforcing namespace isolation and soft-delete semantics via DeletedAt.

## Patterns

**TransactingRepo wrapping** — Every mutating DB operation wraps its body in transaction.Run + entutils.TransactingRepo (or WithNoValue) so ctx-carried transactions are honored. (`transaction.Run(ctx, a, func(ctx context.Context) (meter.Meter, error) { return entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *Adapter) (meter.Meter, error) { ... }) })`)
**Config-validated constructor** — New(Config) validates all required fields (Client, Logger) before constructing the Adapter; returns error on invalid config. (`func New(config Config) (*Adapter, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**Interface compliance assertion** — var _ meter.Service = (*Adapter)(nil) at package level ensures compile-time interface satisfaction. (`var _ meter.Service = (*Adapter)(nil)`)
**WithTx / Self / Tx triad** — Adapter implements entutils.TxCreator by exposing Tx (hijack), WithTx (rebind to tx client), and Self (return self) — required by TransactingRepo machinery. (`func (a *Adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *Adapter { txClient := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &Adapter{db: txClient.Client(), logger: a.logger} }`)
**Soft-delete via DeletedAt** — Meters are soft-deleted by setting DeletedAt timestamp; queries filter DeletedAt IS NULL unless IncludeDeleted is explicitly set. (`query.Where(meterdb.DeletedAtIsNil()) // default; omit only when IncludeDeleted=true`)
**MapFromEntityFactory mapper** — MapFromEntityFactory converts *db.Meter to meter.Meter domain type; all query paths call this function rather than constructing domain objects inline. (`meter, err := MapFromEntityFactory(entity)`)
**Error wrapping conventions** — db.IsConstraintError -> models.NewGenericConflictError; db.IsNotFound -> meter.NewMeterNotFoundError or models.NewGenericNotFoundError; other errors wrapped with fmt.Errorf. (`if db.IsConstraintError(err) { return meter.Meter{}, models.NewGenericConflictError(...) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Core Adapter struct definition, Config, constructor, and TxCreator triad (Tx/WithTx/Self). | Forgetting to implement Tx/WithTx/Self breaks TransactingRepo for this adapter. |
| `manage.go` | Create/Update/Delete meter + cross-entity checks (HasActiveFeatureForMeter, HasEntitlementForMeter, ListFeaturesForMeter). Uses featureadapter.MapFeatureEntity for feature mapping. | Cross-entity queries use entitlementdb and featuredb predicates; missing Where clauses can leak cross-namespace data. |
| `meter.go` | ListMeters (pagination, filtering, ordering) and GetMeterByIDOrSlug plus MapFromEntityFactory. | Ordering switch must handle all meter.OrderBy constants or fall through to default; missing cases silently use createdAt ordering. |
| `adapter_test.go` | Integration tests using testutils.InitPostgresDB and e.db.EntDriver.Client().Schema.Create to provision a real schema. | Tests call t.Context(); using context.Background() breaks test-scoped cancellation. |

## Anti-Patterns

- Calling a.db directly in a helper without wrapping in entutils.TransactingRepo — bypasses ctx-carried transaction.
- Hand-writing SQL instead of using Ent query builders.
- Editing openmeter/ent/db/ generated files.
- Returning raw Ent entities from public functions instead of mapping through MapFromEntityFactory.
- Omitting DeletedAt IS NULL filter in queries where soft-deleted meters should be excluded.

## Decisions

- **Adapter implements both meter.Service and meter.ManageService at the DB layer; service/ wraps it for business logic.** — Keeps persistence concerns isolated; ManageService in service/ layer adds hooks, event publishing, and cross-domain validation without mixing them into SQL.
- **Soft-delete via DeletedAt rather than hard DELETE.** — Meters referenced by historical billing/usage data must not be physically removed to preserve audit trails.

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
