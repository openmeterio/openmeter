# adapter

<!-- archie:ai-start -->

> Implements cost.Adapter by querying ClickHouse meter data via streaming.Connector and resolving per-unit costs from feature configuration (manual or LLM-based). No Ent/PostgreSQL — all persistence goes through streaming.Connector and injected domain services.

## Patterns

**Interface compliance assertion** — Package-level var enforces compile-time satisfaction of cost.Adapter. (`var _ cost.Adapter = (*adapter)(nil)`)
**Pre-resolve and cache expensive lookups** — getLLMPrices scans all meter rows to collect unique (provider, model) pairs and resolves prices once into a map[llmPriceKey]llmPriceResult; never call llmcostService.ResolvePrice inside the per-row loop. (`priceCache := a.getLLMPrices(ctx, feat, rows)`)
**Internal group-by key injection then stripping** — addLLMGroupByKeys appends LLM dimension properties to QueryParams.GroupBy when absent and returns the injected keys so computeCostRows can strip them and aggregate across them. (`internalGroupByKeys := addLLMGroupByKeys(feat, &params)`)
**costResolverFunc separates resolution from aggregation** — computeCostRows accepts a costResolverFunc; makeCostResolver produces a closure that turns GenericNotFoundError into non-fatal detail strings rather than hard errors. (`costRows, currency, err := computeCostRows(rows, internalGroupByKeys, a.makeCostResolver(ctx, feat, priceCache))`)
**Generic domain errors for HTTP mapping** — models.NewGenericValidationError for missing feature config, models.NewGenericNotFoundError for unavailable pricing — mapping to 400/404 at the HTTP layer. (`return nil, models.NewGenericValidationError(fmt.Errorf("feature %s has no meter associated", feat.Key))`)
**Defensive clone before mutation** — slices.Clone GroupBy and build a new merged map for FilterGroupBy before mutating — never mutate the caller's streaming.QueryParams in place. (`params.GroupBy = slices.Clone(params.GroupBy)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Primary QueryFeatureCost: resolves feature + meter, injects LLM group-by keys, pre-resolves prices, delegates to computeCostRows. | llmcostService nil-check in resolveLLMUnitCost is required (service is optional). Don't add Ent/DB access — ClickHouse-only via streaming.Connector. |
| `compute.go` | Pure computation: costResolverFunc, costRowAccumulator, computeCostRows, filterGroupBy, buildDirectCostRow, costPerTokenForType, buildCacheKey — no I/O. | computeCostRows preserves insertion order via aggregationKeys []string — don't replace with map iteration (randomises output). Output is sorted by cost descending after aggregation. |
| `compute_test.go` | Table-driven unit tests for computeCostRows: aggregation, stripping, window separation, partial pricing, mixed resolved/unresolved rows. | Tests are in package adapter (not adapter_test) to access unexported helpers. Use alpacadecimal.Decimal.Equal for cost comparisons, not == or assert.Equal. |

## Anti-Patterns

- Calling llmcostService.ResolvePrice inside the per-row cost loop instead of pre-resolving via getLLMPrices.
- Passing *entdb.Client or touching Ent — this adapter is ClickHouse-only via streaming.Connector.
- Mutating the caller's streaming.QueryParams.GroupBy/FilterGroupBy in place instead of cloning.
- Returning a hard error for pricing-not-found in costResolverFunc instead of converting to a detail string so the row is still emitted.
- Adding a cost resolution type without a branch in resolveUnitCost's switch (and a costPerTokenForType case if token-type based).

## Decisions

- **Price cache keyed on (provider, model), not (provider, model, token_type).** — ResolvePrice depends only on provider+model; token_type is resolved from ModelPricing after lookup, avoiding redundant network calls when a model has multiple token types.
- **Internal group-by keys injected into query params and stripped from output.** — ClickHouse must group by LLM dimensions to produce per-token-type rows for accurate cost resolution, but output must aggregate across them when the caller didn't request those dimensions.
- **costResolverFunc is a function type, not an interface.** — Lets makeCostResolver close over feat and priceCache without a new struct, keeping compute.go free of I/O and trivially testable.

## Example: Adding a new unit cost type to resolveUnitCost

```
// adapter.go resolveUnitCost switch:
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
