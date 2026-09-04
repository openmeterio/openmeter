# Billing feature-meter resolution

- This package overlaps with the newer [product catalog feature resolvers](../../productcatalog/featureresolver).
- It predates those resolvers and should eventually converge with them rather than remain a separate resolution path.
- Convergence is only safe after feature IDs are snapshotted throughout every product catalog layer, including subscriptions, and existing database records have been backfilled. Until then, this package's historical lookup behavior is what allows billing to invoice archived features.

## Purpose and ownership

This package defines the feature-meter contracts used to rate and invoice
billing lines. The [service](./service) subpackage owns catalog access and the
concrete resolved collection. This keeps billing's domain contracts independent
from the resolver implementation while retaining a billing-specific
compatibility boundary rather than a canonical product catalog resolver.

Billing data may still refer to a feature by key. Keys identify the current
logical feature, while IDs identify a specific persisted version. The resolver
therefore returns collections addressable by both key and ID and lets callers
declare whether a meter association is required.

## Historical resolution

Resolution deliberately includes archived features and deleted meters. This
preserves the definitions required by historical or pending invoices after the
corresponding catalog resources have been archived.

For a key, the collection selects the active feature when one exists. If all
features with that key are archived, it selects the most recently archived
one. Every returned feature remains addressable by its exact ID, including
older archived versions. A meter is attached by its snapshotted meter ID even
when that meter has since been deleted.

These semantics are compatibility behavior. Once all billing inputs carry
snapshotted feature IDs and existing records are backfilled, billing should use
the product catalog resolver path and this package can be converged into it.

## Validation contract

By default, every requested feature must exist. A missing feature or required
meter is returned as a critical validation issue. Persisted gathering lines
and charges additionally implement the optional owner contract, which scopes
the issue to that entity. Pre-persistence intents and static references do not
claim an owner and therefore produce issues without an entity path.

Resolution accepts billing entities that provide an optional feature-meter
reference. Entities without a feature dependency are skipped. Duplicate ID or
key references are fetched once, then every original entity is validated so
each failure retains its line or charge identity. The resolved collection is
returned alongside validation failures. Callers may continue with that partial
collection only after splitting the error into validation issues and confirming
that no system error remains. Feature and meter service failures remain fatal.

`FeatureMeterRef` is only the dependency value returned by those entities; it
is intentionally not itself a resolution or lookup target. This prevents
callers from discarding the originating line or charge identity while mapping
references for resolution.

Collection lookups accept the same entity references. An explicit feature ID
takes precedence when a reference also contains a key, preserving access to the
exact historical feature version. `Get` rejects an empty reference, while `Has`
returns false for one and checks only feature existence, not the meter requirement.
When a feature exists but its required meter is absent, `Get` returns both the
resolved feature and the validation error so callers can retain usable billing
output.

The service subpackage constructs billing validation issues directly. Higher
level billing services own the validation component and remain responsible for
deciding whether a workflow can continue with the returned collection and
validation issues.
