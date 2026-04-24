# adapter

<!-- archie:ai-start -->

> Implements cost.Adapter by querying ClickHouse meter data via streaming.Connector and resolving per-unit costs from feature configuration (manual or LLM-based). Single exported constructor New() returns the concrete adapter bound to four injected dependencies.

## Patterns

**Interface compliance assertion** — var _ cost.Adapter = (*adapter)(nil) at package level ensures the struct satisfies the interface at compile time. (`var _ cost.Adapter = (*adapter)(nil)`)
**Pre-resolve and cache expensive lookups** — getLLMPrices scans all meter rows first to collect unique (provider, model) pairs and resolves prices once into a map[llmPriceKey]llmPriceResult before per-row cost computation. Never call llmcostService.ResolvePrice inside the per-row loop. (`priceCache := a.getLLMPrices(ctx, feat, rows)`)
**Internal group-by key injection and post-aggregation stripping** — addLLMGroupByKeys appends LLM dimension properties to streaming.QueryParams.GroupBy when not already present, returns the injected keys so computeCostRows can strip them from output rows and aggregate across them. (`internalGroupByKeys := addLLMGroupByKeys(feat, &params)`)
**costResolverFunc abstraction separates cost resolution from aggregation** — computeCostRows accepts a costResolverFunc instead of calling adapter methods directly; makeCostResolver produces the closure that converts GenericNotFoundError into non-fatal detail strings rather than hard errors. (`costRows, currency, err := computeCostRows(rows, internalGroupByKeys, a.makeCostResolver(ctx, feat, priceCache))`)
**GenericValidationError / GenericNotFoundError for domain errors** — Use models.NewGenericValidationError for missing feature configuration and models.NewGenericNotFoundError (via models.IsGenericNotFoundError check) for unavailable pricing — these map to 400/404 in the HTTP layer. (`return nil, models.NewGenericValidationError(fmt.Errorf("feature %s has no meter associated", feat.Key))`)
**Feature MeterGroupByFilters merged with precedence** — Feature-level filters are merged into params.FilterGroupBy after copying user-supplied values; feature values overwrite caller values to prevent scope escape. (`for k, v := range feat.MeterGroupByFilters { merged[k] = v }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Primary QueryFeatureCost implementation: resolves feature + meter, injects LLM group-by keys, pre-resolves prices, delegates computation to computeCostRows. | llmcostService nil-check in resolveLLMUnitCost is required — the service is injected as an optional dependency. Don't add DB access here; this adapter uses streaming.Connector only. |
| `compute.go` | Pure computation layer: costResolverFunc type, costRowAccumulator, computeCostRows, filterGroupBy, buildDirectCostRow, costPerTokenForType, buildCacheKey — no I/O. | computeCostRows preserves insertion order of aggregation keys via aggregationKeys []string; don't replace with map iteration as that would randomise output order. Output is sorted by cost descending after aggregation. |
| `compute_test.go` | Table-driven unit tests for computeCostRows covering aggregation, stripping, window separation, partial pricing, and mixed resolved/unresolved rows. | Tests are in package adapter (not adapter_test) so they access unexported helpers. Use alpacadecimal.Decimal.Equal for cost comparisons, not == or assert.Equal on the raw struct. |

## Anti-Patterns

- Calling llmcostService.ResolvePrice inside the per-row cost loop — always pre-resolve via getLLMPrices first.
- Passing *entdb.Client or touching Ent directly — this adapter is ClickHouse-only via streaming.Connector.
- Mutating the caller's streaming.QueryParams.GroupBy or FilterGroupBy slices/maps in place — always slices.Clone and build a new merged map.
- Returning a hard error for pricing-not-found in costResolverFunc — convert via models.IsGenericNotFoundError to a detail string so the row is still emitted.
- Adding new cost resolution types without a corresponding branch in resolveUnitCost's switch and a costPerTokenForType case if token-type based.

## Decisions

- **Price cache keyed on (provider, model) not (provider, model, token_type)** — llmcostService.ResolvePrice only depends on provider+model; token_type is resolved from ModelPricing after the lookup, avoiding redundant network calls when a model has multiple token types.
- **Internal group-by keys injected into query params and stripped from output** — ClickHouse must group by LLM dimensions to produce per-token-type rows needed for accurate cost resolution, but if the caller didn't request those dimensions the output must aggregate across them.
- **costResolverFunc is a function type rather than an interface** — Allows makeCostResolver to close over feat and priceCache without a new struct, keeping compute.go free of I/O dependencies and trivially testable with makeResolver in compute_test.go.

## Example: Adding a new unit cost type to resolveUnitCost

```
// In adapter.go resolveUnitCost switch:
case feature.UnitCostTypeCustom:
    if feat.UnitCost.Custom == nil {
        return nil, fmt.Errorf("feature %s has custom unit cost type but no custom configuration", feat.Key)
    }
    return &cost.ResolvedUnitCost{
        Amount:   feat.UnitCost.Custom.Amount,
        Currency: currencyx.Code(globlCurrency.USD),
    }, nil
```

<!-- archie:ai-end -->
