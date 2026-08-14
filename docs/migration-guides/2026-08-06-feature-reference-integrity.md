# Feature Reference Integrity Repair

Status: proposal. The plan/add-on data repair is implemented and tested; the
database constraints and `FeatureReference` refactor described below are
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

First apply the schema-only
[`20260813093105_add_rate_card_annotations.up.sql`](../../tools/migrate/migrations/20260813093105_add_rate_card_annotations.up.sql)
migration. It adds the marker field independently of the potentially
long-running data repair.

The repair must be namespace-scoped and must not change the historical rate-card
`updated_at` value. The core operations are:

```sql
-- ID-only: the ID directly identifies the key, including for archived features.
UPDATE plan_rate_cards rc
SET feature_key = f.key
FROM features f
WHERE rc.feature_id = f.id
  AND rc.namespace = f.namespace
  AND NULLIF(rc.feature_key, '') IS NULL;

-- Key-only: select the feature version active when the rate card was written.
UPDATE plan_rate_cards rc
SET feature_id = f.id
FROM features f
WHERE rc.feature_id IS NULL
  AND NULLIF(rc.feature_key, '') IS NOT NULL
  AND f.namespace = rc.namespace
  AND f.key = rc.feature_key
  AND f.created_at <= rc.updated_at
  AND (f.archived_at IS NULL OR f.archived_at > rc.updated_at)
  AND (f.deleted_at IS NULL OR f.deleted_at > rc.updated_at);
```

Apply the same operations to `addon_rate_cards`. Only uniquely resolved rows are
updated. Each updated rate card is marked in its `annotations` field with the
original rate-card ID, complete post-backfill feature pair, backfilled field,
and a UTC migration timestamp. Rows with neither value, rows with both values,
unresolved references, and ambiguous references are left unchanged and
unmarked.

The complete implementation is
[`20260813093106_backfill_rate_card_feature_references.up.sql`](../../tools/migrate/migrations/20260813093106_backfill_rate_card_feature_references.up.sql),
with database-backed coverage in
[`rate_card_feature_references_test.go`](../../tools/migrate/rate_card_feature_references_test.go).
Rollback clears the backfilled field only if the marker still belongs to the
same row and its complete feature pair is unchanged. If a later edit changed
the reference or recreated the rate card while copying annotations, rollback
removes the stale marker without changing either feature field. Pre-existing
annotations are preserved. The annotations columns belong to the preceding
schema migration and remain in place when only the backfill is rolled back.

### 3. Choose automated or controlled manual execution

Use the normal automated migration only after production-shaped tests show that
it finishes comfortably inside every environment's migration timeout and lock
budget. This includes Konnect, where a long-running migration can time out even
when the SQL is logically correct.

If that margin is not demonstrated, execute the same versioned migration
artifact in a controlled maintenance operation before deploying the schema
constraint. Prefer the normal migration binary or migration CLI so `schema_om`
records the version. Do not mark the version as applied without executing it.
If operators pre-apply only the idempotent repair SQL, the versioned migration
must still run later to record its version. The schema-only annotations
migration must already be applied before either execution path runs the
backfill.

In either mode:

1. take and verify a recoverable backup
2. prevent legacy writers from creating new incomplete references
3. apply the repair
4. verify zero incomplete, ambiguous, mismatched, and cross-namespace rows
5. monitor runtime, locks, errors, and database growth

### 4. Enforce paired persistence in Ent

Only add constraints after the backfill and production verification succeed,
and after every deployed writer is known to persist both values. Otherwise a
rolling deployment can make an older process fail writes as soon as the
constraint becomes active.

Example additions to the existing annotations on both rate-card schemas:

```go
func (PlanRateCard) Annotations() []entschema.Annotation {
	return []entschema.Annotation{
		entsql.Checks(map[string]string{
			// Existing currency checks omitted.
			"plan_rate_card_feature_reference":
				`(feature_key IS NULL AND feature_id IS NULL) OR ` +
					`(feature_key IS NOT NULL AND feature_key <> '' AND ` +
						`feature_id IS NOT NULL AND feature_id <> '')`,
		}),
	}
}

func (AddonRateCard) Annotations() []entschema.Annotation {
	return []entschema.Annotation{
		entsql.Checks(map[string]string{
			// Existing currency checks omitted.
			"addon_rate_card_feature_reference":
				`(feature_key IS NULL AND feature_id IS NULL) OR ` +
					`(feature_key IS NOT NULL AND feature_key <> '' AND ` +
						`feature_id IS NOT NULL AND feature_id <> '')`,
		}),
	}
}
```

Add `.NotEmpty()` to the optional Ent fields as application-side defense, but
do not rely on it for database enforcement. Generate the schema migration from
the authoritative Ent schema. Migration tests must prove that both-null and
both-populated rows are accepted and either half-populated shape is rejected in
both tables. For large datasets, decide during production testing whether
constraint validation needs a staged operational procedure.

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
