# adapter

<!-- archie:ai-start -->

> Implements cost.Adapter: queries metered usage for a feature, resolves per-unit costs (manual or LLM provider/model/token-type pricing), and aggregates usage rows into priced CostQueryRows. It is the only place that translates feature unit-cost config + streaming meter rows into monetary cost.

## Patterns

**Adapter struct with constructor New and interface assertion** — New(...) returns cost.Adapter wrapping a private adapter struct; a compile-time `var _ cost.Adapter = (*adapter)(nil)` enforces the contract. New takes exactly its four collaborators (feature.FeatureConnector, meter.Service, streaming.Connector, llmcost.Service) as positional args. (`func New(featureConnector feature.FeatureConnector, meterService meterpkg.Service, streamingConnector streaming.Connector, llmcostService llmcost.Service) cost.Adapter`)
**Feature filters take precedence over caller filters** — In QueryFeatureCost, feat.MeterGroupByFilters are merged on top of params.FilterGroupBy so callers cannot widen usage beyond the feature's filter scope. Always overlay feature filters last. (`for k, v := range feat.MeterGroupByFilters { merged[k] = v }`)
**Internal group-by keys added for resolution, then aggregated away** — addLLMGroupByKeys appends provider/model/token_type properties to params.GroupBy only if missing, returning the internally-added keys. computeCostRows aggregates rows back across those internal keys (filterGroupBy strips them) so the user-visible grouping is preserved. (`internalGroupByKeys := addLLMGroupByKeys(feat, &params)`)
**Pre-resolve LLM prices into a cache keyed by (provider, model)** — getLLMPrices scans all rows once and resolves each unique llmPriceKey{provider, modelID} via llmcostService.ResolvePrice; resolveLLMUnitCost only reads the cache and must never call ResolvePrice itself. Missing provider/model caches a PriceNotFoundError instead of leaving the key absent. (`cache[key] = llmPriceResult{price: price, err: err}`)
**Not-found pricing is a non-fatal detail, not an error** — makeCostResolver converts models.IsGenericNotFoundError into a (nil cost, detail string, nil err) result; only non-not-found errors are fatal. costResolverFunc's contract: non-nil error is fatal, nil cost + non-empty detail means pricing unavailable. (`if models.IsGenericNotFoundError(err) { return nil, err.Error(), nil }`)
**Resolve dimensions from static value OR group-by property** — resolveDimension returns the static value if set, else looks up the configured ...Property key in groupByValues. resolveUnitCost switches on feat.UnitCost.Type (UnitCostTypeManual / UnitCostTypeLLM) and errors on unknown types. Always NormalizeModelID(provider, modelID) before lookups in both getLLMPrices and resolveLLMUnitCost. (`provider, modelID = llmcost.NormalizeModelID(provider, modelID)`)
**Defensive copy of caller slices/maps before mutation** — QueryFeatureCost clones params.GroupBy via slices.Clone and builds a fresh merged FilterGroupBy map so it never mutates the caller's input.QueryParams. (`params.GroupBy = slices.Clone(params.GroupBy)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Adapter implementation: QueryFeatureCost orchestration, LLM price pre-resolution (getLLMPrices), group-by key injection (addLLMGroupByKeys), and unit-cost resolution (resolveUnitCost / resolveLLMUnitCost / resolveDimension). | resolveLLMUnitCost assumes the price is already in priceCache and errors hard if not — any new row source must also flow through getLLMPrices. Validation of feat.MeterID and feat.UnitCost happens here, not in the service. |
| `compute.go` | Pure aggregation/pricing math: computeCostRows, costRowAccumulator (addUsage/finalize), buildDirectCostRow, filterGroupBy, costPerTokenForType, buildCacheKey, and the costResolverFunc type. | Aggregation key is built from WindowStart/End + Subject + CustomerID + external group-by values; changing it changes grouping semantics. Cost uses alpacadecimal (usage.Mul(amount)); rows are sorted by cost descending with nil-cost rows last. buildCacheKey must stay deterministic (sorted keys, NUL separator). |
| `compute_test.go` | Table-driven unit tests for computeCostRows covering no-aggregation, internal-key aggregation, per-subject/per-window grouping, external-key preservation, and price-not-found detail handling. | Tests drive computeCostRows directly with a hand-built costResolverFunc (makeResolver); they do not exercise the LLM cache path. Decimal assertions use .Equal, not float compare. |

## Anti-Patterns

- Calling llmcostService.ResolvePrice from inside resolveLLMUnitCost or per-row — prices must be batch-resolved once in getLLMPrices and read from the cache.
- Mutating input.QueryParams.GroupBy or FilterGroupBy in place instead of cloning/merging into fresh structures.
- Returning a fatal error when a price/token-type is unavailable — surface it as a row Detail via the not-found path so partial results still return.
- Skipping llmcost.NormalizeModelID before building llmPriceKey or calling ResolvePrice, causing cache misses against canonical DB forms.
- Adding business validation (feature has no meter / no unit cost) anywhere other than the adapter's QueryFeatureCost entry.

## Decisions

- **Pre-resolve prices keyed only by (provider, model), not token_type.** — ResolvePrice depends only on provider+model; token-type cost is extracted later via costPerTokenForType, avoiding redundant service calls for input/output/cache token types of the same model.
- **Inject LLM dimension keys into GroupBy, then aggregate them away.** — Per-row unit cost differs by provider/model/token_type, but users may not request those dimensions; internal-then-aggregate keeps pricing correct while honoring the requested grouping.

## Example: Resolve a per-row LLM unit cost from the pre-resolved price cache

```
key := llmPriceKey{provider, modelID}
cached, ok := priceCache[key]
if !ok {
    return nil, fmt.Errorf("resolving LLM price for provider=%s model=%s: price not in cache", provider, modelID)
}
if cached.err != nil {
    return nil, cached.err
}
amount, err := costPerTokenForType(cached.price.Pricing, feature.LLMTokenType(tokenTypeStr))
if err != nil {
    return nil, models.NewGenericNotFoundError(err)
}
return &cost.ResolvedUnitCost{Amount: amount, Currency: currencyx.Code(cached.price.Currency)}, nil
```

<!-- archie:ai-end -->
