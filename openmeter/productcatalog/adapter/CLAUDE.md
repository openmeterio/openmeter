# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter for the feature entity: implements feature.FeatureRepo (CRUD + archive + list) against the openmeter/ent/db schema and exposes Tx/WithTx/Self for transaction propagation.

## Patterns

**FeatureRepo constructor** — NewPostgresFeatureRepo(db *db.Client, logger *slog.Logger) returns feature.FeatureRepo; always inject *db.Client from Wire, never construct inline. (`featureAdapter := adapter.NewPostgresFeatureRepo(client, logger)`)
**TxCreator + TxUser implementation** — transaction.go implements Tx(), WithTx(), Self() on featureDBAdapter so callers can use entutils.TransactingRepo without bypassing the ctx-bound transaction. (`func (e *featureDBAdapter) WithTx(ctx context.Context, tx *entutils.TxDriver) feature.FeatureRepo { ... }`)
**WithMeter edge for MeterSlug backward compat** — Every query that returns a Feature must WithMeter(mq.Select(dbmeter.FieldID, dbmeter.FieldKey)) to populate the deprecated MeterSlug field for v1 API compatibility. (`query.WithMeter(func(mq *db.MeterQuery) { mq.Select(dbmeter.FieldID, dbmeter.FieldKey) })`)
**Dual-column MeterGroupByFilters storage** — On write, store advanced filters in AdvancedMeterGroupByFilters and legacy equality-only in MeterGroupByFilters; on read, prefer AdvancedMeterGroupByFilters if non-empty. (`if len(entity.AdvancedMeterGroupByFilters) > 0 { f.MeterGroupByFilters = entity.AdvancedMeterGroupByFilters } else { ... }`)
**Archive guard cross-checks plan and subscription references** — ArchiveFeature queries active plans and active subscriptions referencing the feature before setting ArchivedAt; return ForbiddenError if any active reference exists. (`planReferencesIt, err := c.db.Plan.Query()...Exist(ctx)`)
**UnitCost type-switch on read and write** — All UnitCost mutations must clear the other type's DB columns (e.g. when switching manual→llm, clear UnitCostManualAmount). MapFeatureEntity applies the same type-switch on read. (`case feature.UnitCostTypeManual: query = query.ClearUnitCostLlmProviderProperty()...`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `feature.go` | All feature CRUD: CreateFeature, UpdateFeature, GetByIdOrKey, ArchiveFeature, HasActiveFeatureForMeter, ListFeatures, MapFeatureEntity. | Missing WithMeter edge on any new query returns broken MeterSlug; failing to clear opposite type's columns on UnitCost update corrupts the stored cost. |
| `transaction.go` | Implements entutils.TxCreator and entutils.TxUser[FeatureRepo] via HijackTx + db.NewTxClientFromRawConfig. | Do not add business logic here; this file must remain pure transaction plumbing. |
| `feature_test.go` | Integration tests using testutils.InitPostgresDB and dbClient.Schema.Create; tests run in parallel behind a sync.Mutex. | Tests create a real meter row first to satisfy FK constraints; skip this and inserts will fail. |

## Anti-Patterns

- Using c.db directly in a helper function that accepts *db.Client instead of calling entutils.TransactingRepo — bypasses ctx transaction.
- Omitting WithMeter edge on a new query returning Feature — breaks MeterSlug for v1 callers.
- Editing files under openmeter/ent/db/ — generated code, never edit manually.
- Returning feature.FeatureNotFoundError instead of models.NewGenericNotFoundError — noted as FIXME in code but new code should use generic errors.

## Decisions

- **Dual MeterGroupByFilters columns (legacy map[string]string + advanced typed filters)** — v1 API predates typed filter support; both columns are maintained for backward compatibility while new code uses the advanced column.
- **TxCreator/TxUser embedded in the adapter** — Ent transactions propagate implicitly via ctx; exposing Tx/WithTx/Self lets callers use entutils.TransactingRepo without leaking transaction types into domain interfaces.

## Example: Add a new field to Feature: write it on create, clear it on update, read it in MapFeatureEntity

```
// In CreateFeature:
query = query.SetMyField(feat.MyField)
// In UpdateFeature (clear opposite):
query = query.ClearMyField() // or .SetMyField(...)
// In MapFeatureEntity:
f.MyField = entity.MyField
```

<!-- archie:ai-end -->
