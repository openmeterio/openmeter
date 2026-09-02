# Billing feature-meter resolution

- This package overlaps with the newer [product catalog feature resolvers](../../productcatalog/featureresolver).
- It predates those resolvers and should eventually converge with them rather than remain a separate resolution path.
- Convergence is only safe after every product catalog layer, including subscriptions, snapshots feature IDs and the existing database records have been backfilled. Until then, this package's historical lookup behavior is what allows billing to invoice archived features.

## Purpose and ownership

This package resolves the feature and meter definitions needed to rate and
invoice billing lines. It is a billing compatibility boundary, not the
canonical product catalog feature-resolution model.

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

By default, every requested feature must exist. Missing features are returned
as not-found errors, and a feature whose caller requires a meter but has no
resolved meter is returned as a validation error. Missing-feature errors take
precedence when a batch contains both categories.

`WithAllowMissingFeatures` relaxes only feature existence. It does not suppress
invalid references or missing required meter associations. This allows callers
to retain a usable collection while handling absent features at their own
boundary without hiding other catalog inconsistencies.

Billing-specific translation of these failures into invoice validation issues
belongs to the consuming billing service or line engine; this package only
resolves catalog state and classifies resolution failures.
