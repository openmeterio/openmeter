# featureresolver

<!-- archie:ai-start -->

> Adapts feature.FeatureConnector into the productcatalog.FeatureResolver/NamespacedFeatureResolver interface and resolves rate-card feature references (by ID and/or key) to concrete features, attaching field-prefixed validation errors. Breaks the import cycle between productcatalog and the feature package.

## Patterns

**Resolver wraps FeatureConnector** — New(service feature.FeatureConnector) returns a resolver; WithNamespace returns a namespacedResolver. Both satisfy productcatalog.FeatureResolver via var _ assertions. (`var _ productcatalog.FeatureResolver = (*resolver)(nil)`)
**BatchResolve via CollectAll over ListFeatures** — BatchResolve fans the id/key list into one paginated feature.ListFeaturesParams query (pagination.CollectAll + NewPaginator), building a map keyed by both ID and key. (`features, err := pagination.CollectAll(ctx, pagination.NewPaginator(func(ctx, page) (pagination.Result[feature.Feature], error) { return r.service.ListFeatures(ctx, feature.ListFeaturesParams{...}) }))`)
**ID/key conflict detection** — Resolve checks f.ID == *id and f.Key == *key after lookup, returning GenericConflictError on mismatch and GenericNotFoundError on absence. (`if f.ID != *id { return nil, models.NewGenericConflictError(...) }`)
**Field-prefixed aggregated rate-card errors** — ResolveFeaturesForRateCards accumulates per-ratecard errors wrapped with models.ErrorWithFieldPrefix(NewFieldSelectorGroup(...)) and joins them via NewNillableGenericValidationError. (`errs = append(errs, models.ErrorWithFieldPrefix(fieldSelector, fmt.Errorf("feature not found ...: %w", productcatalog.ErrRateCardFeatureNotFound)))`)
**SetFeature back onto the rate card** — On successful resolution each rate card is mutated in place via rc.SetFeature(&f.ID, &f.Key) so downstream code sees both ID and key populated. (`rc.SetFeature(&(f).ID, &(f).Key)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `resolver.go` | New, resolver, namespacedResolver; Resolve, BatchResolve, WithNamespace. | BatchResolve returns (nil, nil) for an empty input list; Resolve requires at least one of id/key non-empty. |
| `ratecard.go` | ResolveFeaturesForRateCards — bulk-resolves features for a RateCards collection and reports field-scoped validation errors. | Returns nil for nil/empty rate cards; mutates rate cards in place via SetFeature; uses productcatalog.ErrRateCardFeatureNotFound / ErrRateCardFeatureMismatch sentinels. |

## Anti-Patterns

- Calling feature.FeatureConnector directly from productcatalog plan/addon code instead of through this resolver — reintroduces the import direction this package exists to break.
- Returning a bare error from ResolveFeaturesForRateCards instead of the joined NewNillableGenericValidationError — loses field-path context.
- Resolving with neither id nor key set — Resolve returns an error.

## Decisions

- **Resolver lives in featureresolver, not feature (NOTE comment: should move under feature after refactor)** — productcatalog needs to resolve features without importing the feature package's connector directly; a thin adapter keeps the dependency direction clean.

<!-- archie:ai-end -->
