# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter for the feature entity: implements feature.FeatureRepo (CRUD + archive + list) against openmeter/ent/db and exposes the Tx/WithTx/Self triad for ctx-transaction propagation. Primary constraint: every query returning a Feature must eager-load the Meter edge for MeterSlug backward compat.

## Patterns

**TransactingRepo on every method** — Wrap each adapter method body with entutils.TransactingRepo so it rebinds to the ctx-bound Ent transaction instead of the raw client. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *featureDBAdapter) (feature.Feature, error) { ... })`)
**WithMeter edge on every Feature query** — All queries returning Feature rows must call .WithMeter(func(mq *db.MeterQuery){ mq.Select(dbmeter.FieldID, dbmeter.FieldKey) }) — omitting breaks MeterSlug for v1 callers. (`query.WithMeter(func(mq *db.MeterQuery) { mq.Select(dbmeter.FieldID, dbmeter.FieldKey) })`)
**UnitCost type-switch clears opposite columns** — When writing UnitCost, clear the other type's DB columns so type switches are clean; MapFeatureEntity applies the same switch on read. (`case feature.UnitCostTypeManual: query = query.ClearUnitCostLlmProvider()...SetUnitCostType(...)`)
**Dual MeterGroupByFilters column write** — On write populate both AdvancedMeterGroupByFilters (typed) and MeterGroupByFilters (legacy map[string]string); on read prefer the advanced column if non-empty. (`query.SetAdvancedMeterGroupByFilters(feat.MeterGroupByFilters).SetMeterGroupByFilters(feature.ConvertMeterGroupByFiltersToMapString(feat.MeterGroupByFilters))`)
**ArchiveFeature cross-checks active references** — Before setting ArchivedAt, query active plans and active subscriptions referencing the feature; return ForbiddenError if any exist. (`planReferencesIt, err := c.db.Plan.Query().Where(dbplan.EffectiveFromNotNil(), dbplan.HasPhasesWith(...)).Exist(ctx)`)
**TxCreator + TxUser triad in transaction.go** — transaction.go implements Tx() via HijackTx + NewTxDriver, WithTx() via db.NewTxClientFromRawConfig + NewPostgresFeatureRepo, Self() returning self — pure plumbing only. (`func (e *featureDBAdapter) WithTx(ctx, tx *entutils.TxDriver) feature.FeatureRepo { txClient := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return NewPostgresFeatureRepo(txClient.Client(), e.logger) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `feature.go` | All feature CRUD: CreateFeature, UpdateFeature, GetByIdOrKey, ArchiveFeature, HasActiveFeatureForMeter, ListFeatures, MapFeatureEntity. | Missing WithMeter edge returns broken MeterSlug; failing to clear the opposite UnitCost type's columns on update corrupts stored cost. FIXME: prefer models.NewGenericNotFoundError over feature.FeatureNotFoundError in new code. |
| `transaction.go` | Implements entutils.TxCreator and entutils.TxUser[FeatureRepo] via HijackTx and NewTxClientFromRawConfig. | Never add business logic here; must remain pure transaction plumbing. |
| `feature_test.go` | Integration tests using testutils.InitPostgresDB + dbClient.Schema.Create; run in parallel behind a sync.Mutex. | Tests must create a real meter row first to satisfy the Feature.MeterID FK; skipping it fails the save. |

## Anti-Patterns

- Using c.db directly in a helper accepting *db.Client without entutils.TransactingRepo — bypasses the ctx transaction.
- Omitting WithMeter on a new Feature query — breaks MeterSlug for v1 callers.
- Adding validation or event publishing inside the adapter — belongs in featureConnector.
- Returning feature.FeatureNotFoundError in new code — use models.NewGenericNotFoundError.
- Editing files under openmeter/ent/db/ — generated; regenerate with make generate.

## Decisions

- **Dual MeterGroupByFilters columns (legacy map + advanced typed filters)** — The v1 API predates typed filter support; both columns are maintained for backward compat while new code uses the advanced column.
- **TxCreator/TxUser split into a separate transaction.go** — Ent transactions propagate via ctx; Tx/WithTx/Self lets callers use entutils.TransactingRepo without leaking transaction types into domain interfaces.

## Example: Add a new Feature field: write on create, clear/set on update, read in MapFeatureEntity

```
// CreateFeature: query = query.SetMyField(feat.MyField)
// UpdateFeature: query = query.ClearMyField() // or .SetMyField(...)
// MapFeatureEntity: f.MyField = entity.MyField
```

<!-- archie:ai-end -->
