# Feature Reference Integrity Repair

Status: the plan/add-on data repair and paired database constraints are
implemented and tested. The `FeatureReference` refactor described below remains
follow-up work. Subscription items are explicitly out of scope until their
feature lifecycle has been researched separately.

## Problem

Plan and add-on rate cards persist a feature reference as two nullable columns:
`feature_key` and `feature_id`. A former bug in the product-catalog feature
resolution path allowed only one half of that reference to be written:

- requests through the v1 API could persist only `feature_key`
- requests through the v3 API could persist only `feature_id`

The [resolver](../../openmeter/productcatalog/featureresolver/ratecard.go) now
populates both values before persistence, but existing rows remain incomplete.
The current schemas and adapters still represent the pair as independent
values, so the database cannot prevent the same defect from being reintroduced.

An ID-only row is unambiguous: the referenced feature identifies its immutable
key. It is still operationally inconvenient because a reader must load the
feature to reconstruct the key.

A key-only row is harder. Feature keys can be reused after archiving, so looking
up the currently active feature can attach the rate card to a different feature
version. The repair must find the feature in the same namespace that was active
when the rate card was last written. This proposal uses `rate_card.updated_at`
as that timestamp and defines a feature's active interval as
`[created_at, archived_at)`, with `deleted_at` as an equivalent terminal
boundary for legacy rows.

Using `updated_at` is an important data assumption. Before production rollout,
verify that affected rate cards were not updated for unrelated reasons after
their feature reference was selected. If that happened, a different provenance
signal or a per-row manual mapping is required.

## Target invariant

For plan and add-on rate cards:

- either both `feature_key` and `feature_id` are absent, meaning the rate card
  has no feature, or both are present and non-empty
- the feature belongs to the rate card's namespace
- `feature_id` and `feature_key` identify the same feature version
- persisted identity remains usable without resolving the full feature; an
  optional runtime-only resolved feature may be attached when needed

A two-column `CHECK` constraint can enforce only the first invariant. The
existing `feature_id` foreign key establishes that the ID exists, but does not
by itself prove namespace ownership or key/ID consistency. Those properties
must remain code-level validation unless a later schema change introduces a
namespace-scoped composite relationship.

## Rollout

### 1. Preflight production data

Measure incomplete rows by table and namespace before changing data:

```sql
SELECT 'plan_rate_cards' AS source, namespace,
       COUNT(*) FILTER (WHERE feature_id IS NOT NULL AND NULLIF(feature_key, '') IS NULL) AS id_only,
       COUNT(*) FILTER (WHERE feature_id IS NULL AND NULLIF(feature_key, '') IS NOT NULL) AS key_only
FROM plan_rate_cards
GROUP BY namespace
UNION ALL
SELECT 'addon_rate_cards', namespace,
       COUNT(*) FILTER (WHERE feature_id IS NOT NULL AND NULLIF(feature_key, '') IS NULL),
       COUNT(*) FILTER (WHERE feature_id IS NULL AND NULLIF(feature_key, '') IS NOT NULL)
FROM addon_rate_cards
GROUP BY namespace;
```

Also check complete rows for key/ID mismatches, cross-namespace references,
key-only rows with no temporal match, and key-only rows with multiple temporal
matches. The migration is best-effort: it must skip unresolved and ambiguous
rows rather than abort or choose an arbitrary feature. Production verification
remains a separate, read-only operational step.

Run the migration against recent copies of production datasets. Record runtime,
rows changed, lock time, generated WAL, and the number of unresolved or
ambiguous rows. Use a read-only equivalent of the update with `EXPLAIN
(ANALYZE, BUFFERS)` on a clone or replica to evaluate whether supporting indexes
are necessary.

### 2. Backfill plan and add-on rate cards

Run the repair only through OpenMeter's versioned migration runner. It must
execute these complete migration artifacts, in order, from the same OpenMeter
release:

1. [`20260814113138_add_rate_card_annotations.up.sql`](../../tools/migrate/migrations/20260814113138_add_rate_card_annotations.up.sql)
   adds the marker field.
2. [`20260814113139_backfill_rate_card_feature_references.up.sql`](../../tools/migrate/migrations/20260814113139_backfill_rate_card_feature_references.up.sql)
   performs the repair and writes its ownership marker in the same transaction.

Do not copy or execute individual repair `UPDATE` statements, split the
backfill artifact, or mark either version as applied without executing its
complete file. An unmarked pre-apply makes affected rows appear complete, so the
versioned backfill skips them and its normal rollback cannot identify the
manually written values. Such a partial repair is unsupported and must be
treated as irreversible until it has been assessed and reconciled manually.

The backfill is namespace-scoped and preserves the historical rate-card
`updated_at` value. ID-only rows receive the key of their referenced feature.
Key-only rows receive an ID only when exactly one feature in the same namespace
was active at `updated_at`. The same rules apply to `plan_rate_cards` and
`addon_rate_cards`. Rows with neither value, rows with both values, unresolved
references, and ambiguous references remain unchanged and unmarked.

Each updated row is marked in `annotations` with the original rate-card ID,
complete post-backfill feature pair, backfilled field, and a UTC migration
timestamp. Database-backed coverage is in
[`rate_card_feature_references_test.go`](../../tools/migrate/rate_card_feature_references_test.go).
The versioned backfill rollback is
[`20260814113139_backfill_rate_card_feature_references.down.sql`](../../tools/migrate/migrations/20260814113139_backfill_rate_card_feature_references.down.sql).
It clears the backfilled field only if the marker still belongs to the same row
and its complete feature pair is unchanged. If a later edit changed the
reference or recreated the rate card while copying annotations, rollback
removes the stale marker without changing either feature field. Pre-existing
annotations are preserved. Run rollback through the migration runner as well;
do not run the annotations schema rollback when only reverting the backfill.

### 3. Choose automated or controlled manual execution

Use the normal automated migration only after production-shaped tests show that
it finishes comfortably inside every environment's migration timeout and lock
budget. This includes Konnect, where a long-running migration can time out even
when the SQL is logically correct. The supported bundled command is:

```bash
openmeter-jobs migrate
```

If that margin is not demonstrated, stop legacy writers and run the same
command from the exact target OpenMeter release as a controlled maintenance
operation with a sufficient timeout. This still uses the embedded versioned
migration history; it is not a separate repair path. Installations that manage
migrations externally must follow the
[normal versioned migration procedure](../database-migration.md#running-migrations)
using the complete migration directory from that release.

In every execution mode, the migration runner must advance `schema_om` through
versions `20260814113138` and `20260814113139` only by executing their complete
artifacts. Never pre-apply feature-column updates, execute only part of the
backfill file, or manually mark either version as applied.

In either mode:

1. take and verify a recoverable backup
2. prevent legacy writers from creating new incomplete references
3. run the versioned migration through `20260814113139`
4. verify zero incomplete, ambiguous, mismatched, and cross-namespace rows
5. monitor runtime, locks, errors, and database growth

### 4. Enforce paired persistence in Ent

Deploy the constraint migration only after the backfill and production
verification succeed, and after every deployed writer is known to persist both
values. Otherwise a rolling deployment can make an older process fail writes
as soon as the constraint becomes active.

The authoritative Ent schemas define `feature_key` and `feature_id` as
non-empty when present and install `plan_rate_card_feature_reference` and
`addon_rate_card_feature_reference` checks. Each check accepts only a rate card
with no feature or with both non-empty identifiers. The generated migration is
[`20260818090123_rate_card_feature_reference_constraints.up.sql`](../../tools/migrate/migrations/20260818090123_rate_card_feature_reference_constraints.up.sql),
with rollback in
[`20260818090123_rate_card_feature_reference_constraints.down.sql`](../../tools/migrate/migrations/20260818090123_rate_card_feature_reference_constraints.down.sql).

The database-backed
[`rate_card_feature_reference_constraints_test.go`](../../tools/migrate/rate_card_feature_reference_constraints_test.go)
proves that existing incomplete rows block the migration, both-null and
both-populated rows remain valid, and half-populated or empty references are
rejected in both tables. For large datasets, decide during production testing
whether constraint validation needs a staged operational procedure.

### 5. Replace parallel fields with `FeatureReference`

The domain model should follow the same separation used by
[`CurrencyReference`](../../openmeter/currencies/currency.go): persisted
identity is explicit, while the resolved resource is runtime-only and ignored
by equality.

Illustrative API, not final code:

```go
type FeatureReference struct {
	ID  string `json:"id"`
	Key string `json:"key"`

	resolved *Feature
}

func (r FeatureReference) Validate() error
func (r FeatureReference) Equal(other FeatureReference) bool
func (r FeatureReference) IsResolved() bool
func (r FeatureReference) Feature() (*Feature, bool)
func (r FeatureReference) WithFeature(feature *Feature) (FeatureReference, error)
func (r FeatureReference) Clone() FeatureReference
func (f Feature) Reference() FeatureReference
```

`RateCardMeta` would carry `Feature *feature.FeatureReference` instead of two
independent pointers. A nil pointer means no feature; a non-nil reference always
contains both persisted identifiers. Partial v1 key input and v3 ID input are
authoring shapes at the API boundary, not valid persisted domain references.
The resolver converts either shape into `Feature.Reference()` before validation
and persistence, and rejects a supplied key/ID pair when the values disagree.

Persistence mappings must always write both scalar columns. Hydration rebuilds
the reference from those columns and attaches the sideloaded feature only when
the caller requests expanded feature data. Existing v1 and v3 serialization
must remain wire-compatible during the refactor. Tests should cover input by
key, input by ID, matching and mismatching pairs, archived features, namespace
isolation, round-trip persistence, cloning, equality, and resolved-resource
consistency.

## Subscription items: separate investigation

Subscription items need the same long-term identity model, but not the same
immediate migration. The current schema persists only `feature_key`, and
[`MapDBSubscriptionItem`](../../openmeter/subscription/repo/mapping.go)
explicitly returns a nil feature ID. [Add-on
compatibility](../../openmeter/subscription/addon/extend.go) also has feature-ID
validation disabled under `OM-1337`.

Before proposing a subscription migration, determine:

- whether `feature_id` should reference the original catalog feature version or
  the feature resolved when the subscription item was materialized
- which timestamp reliably identifies that decision; subscription items can be
  deleted and recreated during reconciliation, so `updated_at` may not have the
  same meaning as on catalog rate cards
- how direct subscription edits, plan changes, add-on apply/remove, restoration,
  cancellation, and entitlement recreation preserve feature identity
- the required Ent edge, namespace enforcement, historical backfill, and
  rolling-deployment compatibility

Until that research is complete, do not apply the rate-card constraint or
backfill mechanically to `subscription_items`.

## Completion criteria

- production datasets have no incomplete, ambiguous, mismatched, or
  cross-namespace plan/add-on rate-card references
- the migration execution mode is selected from measured runtime and lock data
- database constraints reject future half-populated rate-card references
- product-catalog code persists and validates a single `FeatureReference`
- v1 and v3 behavior remains compatible
- subscription-item work has a separately reviewed lifecycle design and rollout
