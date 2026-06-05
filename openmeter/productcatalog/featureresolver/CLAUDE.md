# featureresolver

<!-- archie:ai-start -->

> Thin resolution layer implementing productcatalog.FeatureResolver over feature.FeatureConnector. Resolves rate-card feature ID/key references to concrete features and back-fills resolved IDs onto rate cards.

## Patterns

**Constructor returns the productcatalog interface** — New(service feature.FeatureConnector) returns productcatalog.FeatureResolver; nil service is an error. (`func New(service feature.FeatureConnector) (productcatalog.FeatureResolver, error)`)
**Namespaced wrapper over namespace-explicit core** — resolver.WithNamespace returns namespacedResolver that forwards to resolver.Resolve/BatchResolve with the bound namespace; core methods always take namespace explicitly. (`func (r *resolver) WithNamespace(ns string) productcatalog.NamespacedFeatureResolver`)
**BatchResolve via pagination.CollectAll** — BatchResolve pages through ListFeatures (IncludeArchived=false) with pagination.NewPaginator/CollectAll and returns a map keyed by both ID and Key, nil for misses. (`pagination.CollectAll(ctx, pagination.NewPaginator(func(ctx, page){ return r.service.ListFeatures(ctx, feature.ListFeaturesParams{IDsOrKeys: idsOrKeys, ...}) }), min(len(idsOrKeys), 100))`)
**Resolve enforces ID/key slot correctness** — Resolve returns GenericNotFoundError for misses and GenericConflictError when a value resolves to a feature whose ID/Key doesn't match the requested slot (id-is-actually-a-key etc.). (`if f.ID != *id { return nil, models.NewGenericConflictError(...) }`)
**ResolveFeaturesForRateCards mutates in place with field-prefixed errors** — Iterates *RateCards, BatchResolves all feature IDs+keys, then rc.SetFeature(&f.ID, &f.Key); collects models.ErrorWithFieldPrefix(ratecards[key]) into a NillableGenericValidationError. (`rc.SetFeature(&(f).ID, &(f).Key)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `resolver.go` | resolver/namespacedResolver implementing FeatureResolver/NamespacedFeatureResolver. | Header note says this should live under feature/ after refactor; Resolve distinguishes NotFound vs Conflict — preserve both error types. |
| `ratecard.go` | ResolveFeaturesForRateCards: batch-resolve and back-fill feature refs on rate cards. | Uses productcatalog.ErrRateCardFeatureNotFound / ErrRateCardFeatureMismatch; skips rate cards without HasFeature(); errors carry the ratecard.key field selector. |
| `ratecard_test.go` | DB-backed tests via pctestutils.NewTestEnv covering success/not-found/mismatch/wrong-slot. | Exercises the id-in-key-slot and key-in-id-slot mismatch cases — keep those guards intact. |

## Anti-Patterns

- Reimplementing feature lookup against the repo instead of going through FeatureResolver
- Collapsing NotFound and Conflict into one error (tests assert each distinctly)
- Resolving rate cards that lack a feature (must skip when !rc.HasFeature())
- Dropping the field-prefix on rate-card resolution errors (callers rely on ratecard.key context)

## Decisions

- **Resolver lives outside the feature package** — Explicit NOTE: should move under feature/ post-refactor; kept separate to avoid the connector's pending service migration.
- **BatchResolve keys results by both ID and key** — Callers (rate cards) reference features by either slot, and a single map lets the caller look up whichever it holds.

## Example: Resolving and back-filling features onto rate cards

```
resolver, _ := featureresolver.New(featureConnector)
if err := featureresolver.ResolveFeaturesForRateCards(ctx, resolver, namespace, rateCards); err != nil {
  return err // NillableGenericValidationError wrapping ErrRateCardFeatureNotFound/Mismatch
}
```

<!-- archie:ai-end -->
