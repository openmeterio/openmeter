# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter for the feature entity: implements feature.FeatureRepo (CRUD + archive + list) against openmeter/ent/db and exposes Tx/WithTx/Self for transaction propagation. Primary constraint: every query that returns a Feature must load the Meter edge for MeterSlug backward compat.

## Patterns

**TransactingRepo on every method** — Every adapter method must wrap its body with entutils.TransactingRepo or TransactingRepoWithNoValue so it rebinds to the ctx-bound Ent transaction instead of using the raw client. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *featureDBAdapter) (feature.Feature, error) { return toDomain(tx.db.Feature.Create()...Save(ctx)) })`)
**WithMeter edge on every Feature query** — All queries returning Feature rows must call .WithMeter(func(mq *db.MeterQuery) { mq.Select(dbmeter.FieldID, dbmeter.FieldKey) }) — omitting this breaks MeterSlug for v1 API callers. (`query.WithMeter(func(mq *db.MeterQuery) { mq.Select(dbmeter.FieldID, dbmeter.FieldKey) })`)
**UnitCost type-switch with opposite-column clear** — When writing UnitCost, always clear the other type's DB columns to handle type switches cleanly. MapFeatureEntity applies the same switch on read. (`case feature.UnitCostTypeManual: query = query.ClearUnitCostLlmProviderProperty().ClearUnitCostLlmProvider()...SetUnitCostType(...)`)
**Dual MeterGroupByFilters column write** — On write, populate both AdvancedMeterGroupByFilters (typed) and MeterGroupByFilters (legacy map[string]string) for backward compat. On read in MapFeatureEntity, prefer AdvancedMeterGroupByFilters if non-empty. (`query.SetAdvancedMeterGroupByFilters(feat.MeterGroupByFilters).SetMeterGroupByFilters(feature.ConvertMeterGroupByFiltersToMapString(feat.MeterGroupByFilters))`)
**ArchiveFeature cross-checks active plan and subscription references** — Before setting ArchivedAt, query active plans and active subscriptions that reference the feature; return ForbiddenError if any reference exists. (`planReferencesIt, err := c.db.Plan.Query().Where(dbplan.EffectiveFromNotNil(), dbplan.HasPhasesWith(...)).Exist(ctx)`)
**TxCreator + TxUser triad in transaction.go** — transaction.go implements Tx() via HijackTx + NewTxDriver, WithTx() via db.NewTxClientFromRawConfig + NewPostgresFeatureRepo, and Self() returning self — never add business logic here. (`func (e *featureDBAdapter) WithTx(ctx context.Context, tx *entutils.TxDriver) feature.FeatureRepo { txClient := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return NewPostgresFeatureRepo(txClient.Client(), e.logger) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `feature.go` | All feature CRUD: CreateFeature, UpdateFeature, GetByIdOrKey, ArchiveFeature, HasActiveFeatureForMeter, ListFeatures, MapFeatureEntity. | Missing WithMeter edge on any new query returns broken MeterSlug; failing to clear the opposite UnitCost type's columns on update corrupts stored cost; FIXME note says use models.NewGenericNotFoundError instead of feature.FeatureNotFoundError in new code. |
| `transaction.go` | Implements entutils.TxCreator and entutils.TxUser[FeatureRepo] via HijackTx and NewTxClientFromRawConfig. | Never add business logic here; this file must remain pure transaction plumbing. |
| `feature_test.go` | Integration tests using testutils.InitPostgresDB and dbClient.Schema.Create; tests run in parallel behind a sync.Mutex. | Tests must create a real meter row first to satisfy FK constraints on Feature.MeterID; skipping this step causes save to fail. |

## Anti-Patterns

- Using c.db directly in a helper that accepts *db.Client without calling entutils.TransactingRepo — bypasses ctx transaction.
- Omitting WithMeter edge on a new query returning Feature — breaks MeterSlug for v1 callers.
- Adding business logic (validation, event publishing) inside the adapter — that belongs in featureConnector.
- Returning feature.FeatureNotFoundError in new code — use models.NewGenericNotFoundError instead.
- Editing files under openmeter/ent/db/ — generated code, regenerate with make generate.

## Decisions

- **Dual MeterGroupByFilters columns (legacy map[string]string + advanced typed filters)** — v1 API predates typed filter support; both columns are maintained for backward compatibility while new code uses the advanced column.
- **TxCreator/TxUser embedded in the adapter via a separate transaction.go** — Ent transactions propagate implicitly via ctx; Tx/WithTx/Self lets callers use entutils.TransactingRepo without leaking transaction types into domain interfaces.

## Example: Add a new field to Feature: write on create, clear on update, read in MapFeatureEntity

```
// CreateFeature: query = query.SetMyField(feat.MyField)
// UpdateFeature (clear opposite type if applicable): query = query.ClearMyField() // or .SetMyField(...)
// MapFeatureEntity: f.MyField = entity.MyField
```

<!-- archie:ai-end -->
