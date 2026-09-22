# Credit filters

`Filters.Matches(route)` matches credit restrictions against the dimensions
recorded on a ledger route. Feature and plan dimensions combine with AND;
entries within either list combine with OR. Empty lists impose no restriction.
A restricted dimension does not match a route without that dimension.

Plans use a catalog key and an optional integer version comparison: exactly one
of `eq`, `in`, `gte`, or `lte`. Omitting the version matches all versions,
including future versions. Spend routes carry a concrete version (`eq`).
`Equal` compares complete normalized filter sets for bucket identity; it is not
spend matching. Collection, advance backfill, and live balance allocation share
this matcher. Corrections preserve the original source route.

Spend routes use the immutable plan attribution recorded on
[charges](../../billing/charges/README.md). Routes without recorded attribution
do not match plan-restricted credits.

`Filters.Version` is retained by Ent, normalization, and JSON round trips.
Readers and writers switch on it: v1 supports features; v2 adds plans.
New feature-only filters use v1; new plan filters use v2. Encoding honors the
selected version, and v1 rejects plans. Unknown versions and fields fail decoding.

Adding a dimension requires a new storage version, its validation, normalization,
and matching semantics, plus any route attribution it needs. Storage versions
do not affect equality or exact route lookup.

Grants and ledger routes use JSON for filter reads and writes. Their legacy
feature columns remain in the schema, deprecated and unused. Feature-only
routing keys remain unchanged; plan-filtered routes use V5.

Legacy lineage keeps its existing `advance_features` storage and reads/writes.
The matcher adapts those features in memory; plan-restricted grants do not match
legacy advances without plan attribution.
