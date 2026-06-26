# adapter

<!-- archie:ai-start -->

> Ent-backed persistence layer for meter definitions. Implements meter.Service against db.Meter and answers meter-dependency queries (active features/entitlements) used by the meter service before allowing deletes/group-by drops.

## Patterns

**Adapter implements meter.Service** — Adapter is constructed via New(Config) with a *db.Client and *slog.Logger, both validated as non-nil in Config.Validate(). Compile-time assert `var _ meter.Service = (*Adapter)(nil)`. (`func New(config Config) (*Adapter, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**TxCreator wiring** — Adapter exposes Tx/WithTx/Self so it plugs into entutils.TransactingRepo. Tx uses a.db.HijackTx; WithTx rebuilds via db.NewTxClientFromRawConfig. Every mutating method wraps its body in transaction.Run(ctx, a, ...) + entutils.TransactingRepo(ctx, a, func(ctx, repo *Adapter)...). (`return transaction.Run(ctx, a, func(ctx) (Meter, error) { return entutils.TransactingRepo(ctx, a, func(ctx, repo *Adapter)...) })`)
**Entity->domain via MapFromEntityFactory** — Every read maps *db.Meter to meter.Meter through MapFromEntityFactory; list paths use pagination.MapResultErr(entities, MapFromEntityFactory). Never hand-build meter.Meter inline. (`resp, err := pagination.MapResultErr(entities, MapFromEntityFactory)`)
**Constraint/not-found error translation** — Translate db errors to domain errors: db.IsConstraintError -> models.NewGenericConflictError; db.IsNotFound -> meterpkg.NewMeterNotFoundError(key) or models.NewGenericValidationError. Validation errors are wrapped with models.NewGenericValidationError. (`if db.IsConstraintError(err) { return Meter{}, models.NewGenericConflictError(fmt.Errorf("meter with the same slug already exists")) }`)
**Soft delete and live-row filtering** — DeleteMeter sets DeletedAt rather than removing rows; queries gate on meterdb.DeletedAtIsNil and feature/entitlement queries use Or(DeletedAtIsNil, DeletedAtGT(clock.Now())) and Or(ArchivedAtIsNil, ArchivedAtGT(clock.Now())). (`repo.db.Meter.UpdateOneID(meter.ID).SetDeletedAt(time.Now()).Save(ctx)`)
**Filter/order via pkg helpers** — ListMeters applies filter.ApplyToQuery(query, params.Key, meterdb.FieldKey) for FilterString fields and entutils.GetOrdering(params.Order) + meterdb.ByKey/ByName/etc for ordering. Add new order keys by extending the switch on params.OrderBy. (`query = filter.ApplyToQuery(query, params.Key, meterdb.FieldKey)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Adapter struct, Config+Validate, New, and Tx/WithTx/Self transaction plumbing | Both Client and Logger must be non-nil; do not inject slog.Default() as fallback |
| `manage.go` | Mutations (CreateMeter/UpdateMeter/DeleteMeter) and dependency checks (HasActiveFeatureForMeter, HasEntitlementForMeter, ListFeaturesForMeter) | UpdateMeter only sets mutable fields (Name, Description, GroupBy, Metadata, Annotations); Key/Aggregation/EventType are immutable. Use SetOrClearAnnotations for nillable annotation updates |
| `meter.go` | Reads (ListMeters, GetMeterByIDOrSlug) and MapFromEntityFactory mapper | GetMeterByIDOrSlug matches ID OR (Key AND DeletedAtIsNil); MapFromEntityFactory returns error on nil entity |
| `adapter_test.go` | TestEnv harness (NewTestEnv, DBSchemaMigrate) over real Postgres via testutils.InitPostgresDB | Requires Postgres; use env.Meter (Adapter) directly, not service wiring |

## Anti-Patterns

- Building meter.Meter inline instead of MapFromEntityFactory
- Performing mutations without transaction.Run + entutils.TransactingRepo (loses tx context binding)
- Returning raw ent errors instead of translating via db.IsConstraintError/db.IsNotFound to domain errors
- Hard-deleting meter rows instead of SetDeletedAt
- Adding business rules (event-type/reserved-type validation, publish events) here — that belongs in meter/service

## Decisions

- **Adapter directly satisfies meter.Service (read+CRUD) but business orchestration lives in meter/service ManageService** — Keeps persistence pure; service layer owns hooks, namespace provisioning, event publishing and reserved-event-type validation
- **Feature/entitlement existence checks live in the meter adapter** — DeleteMeter and group-by-drop validation in the service need to query foreign tables; doing it via Ent here keeps it transaction-aware

## Example: Transaction-aware mutation returning a mapped domain value

```
func (a *Adapter) CreateMeter(ctx context.Context, input meterpkg.CreateMeterInput) (meterpkg.Meter, error) {
	if err := input.Validate(); err != nil { return meterpkg.Meter{}, err }
	return transaction.Run(ctx, a, func(ctx context.Context) (meterpkg.Meter, error) {
		return entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *Adapter) (meterpkg.Meter, error) {
			entity, err := repo.db.Meter.Create().SetNamespace(input.Namespace).SetKey(input.Key).Save(ctx)
			if err != nil {
				if db.IsConstraintError(err) { return meterpkg.Meter{}, models.NewGenericConflictError(fmt.Errorf("meter with the same slug already exists")) }
				return meterpkg.Meter{}, fmt.Errorf("failed to create meter: %w", err)
			}
			return MapFromEntityFactory(entity)
		})
	})
}
```

<!-- archie:ai-end -->
