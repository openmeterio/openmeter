# adapter

<!-- archie:ai-start -->

> Ent-backed persistence layer for the productcatalog feature domain. Implements feature.FeatureRepo (NewPostgresFeatureRepo) over openmeter/ent/db, translating feature domain types to/from the db.Feature entity.

## Patterns

**Repo constructor returns the domain interface** — NewPostgresFeatureRepo(db, logger) returns feature.FeatureRepo, not the concrete *featureDBAdapter. Wire other adapters the same way. (`func NewPostgresFeatureRepo(db *db.Client, logger *slog.Logger) feature.FeatureRepo`)
**Entity->domain mapping via MapFeatureEntity** — All read paths end in MapFeatureEntity(*db.Feature) feature.Feature. Do not hand-build feature.Feature elsewhere in this package. (`return MapFeatureEntity(entity), nil`)
**Re-fetch with WithMeter for MeterSlug backcompat** — When MeterID is set, re-query with WithMeter(Select(FieldID,FieldKey)) so MapFeatureEntity can populate the deprecated MeterSlug field. (`.WithMeter(func(mq *db.MeterQuery){ mq.Select(dbmeter.FieldID, dbmeter.FieldKey) })`)
**Archive is soft + cross-aggregate guard** — ArchiveFeature sets ArchivedAt (no hard delete) and first checks Plan and Subscription references, returning feature.ForbiddenError if the feature is still in use. (`if planReferencesIt { return &feature.ForbiddenError{...} }`)
**Filter helpers + pagination dual-mode** — List uses filter.ApplyToQuery for Key/Name/MeterIDs and supports both Page (Paginate) and legacy Limit/Offset; ArchivedAt filtering gated on IncludeArchived. (`query = filter.ApplyToQuery(query, params.Key, dbfeature.FieldKey)`)
**UnitCost type-switch with explicit clears** — Create/Update map UnitCost by Type (Manual/LLM), and Update always clears the opposite type's columns so type switches persist cleanly. (`switch unitCost.Type { case feature.UnitCostTypeManual: query = query.ClearUnitCostLlm...().SetUnitCostType(...) }`)
**Transactionality via entutils TxUser/TxCreator** — transaction.go implements Tx/WithTx/Self by hijacking the ent tx and rebuilding the repo from the raw tx config. (`func (e *featureDBAdapter) WithTx(ctx, tx) feature.FeatureRepo { txClient := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return NewPostgresFeatureRepo(txClient.Client(), e.logger) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `feature.go` | featureDBAdapter implementing FeatureRepo CRUD/list and MapFeatureEntity mapping. | MeterSlug is deprecated/v1-only — only populated from the Meter edge; ArchiveFeature's plan/subscription guards (FIXME OM-1055) live in the DB layer intentionally. |
| `transaction.go` | TxCreator/TxUser implementation for FeatureRepo. | WithTx must rebuild via NewPostgresFeatureRepo so the new repo binds to the tx client; do not return the original receiver. |
| `feature_test.go` | Postgres-backed repo tests using testutils.InitPostgresDB and a serializing sync.Mutex. | Tests pre-create the Meter row to satisfy the FK; they run t.Parallel but lock a shared mutex. |

## Anti-Patterns

- Constructing feature.Feature manually in read paths instead of calling MapFeatureEntity
- Setting UnitCost columns without clearing the opposite cost type on Update (stale LLM/manual fields survive)
- Hard-deleting features instead of setting ArchivedAt
- Skipping the WithMeter re-fetch and leaving MeterSlug empty for v1 consumers

## Decisions

- **Cross-aggregate plan/subscription reference checks live inside the DB adapter** — FIXME OM-1055 notes features are referenced by ID with no versioning, so the guard sits at the persistence boundary until productcatalog/plan and feature are unified.
- **Keep MeterSlug derived from the Meter edge** — v1 API backward compatibility; MeterID is the source of truth going forward.

## Example: Constructing the Postgres feature repository

```
import (
  "github.com/openmeterio/openmeter/openmeter/ent/db"
  "github.com/openmeterio/openmeter/openmeter/productcatalog/adapter"
)

repo := adapter.NewPostgresFeatureRepo(dbClient, logger) // returns feature.FeatureRepo
```

<!-- archie:ai-end -->
