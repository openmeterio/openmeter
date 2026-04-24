# cost

<!-- archie:ai-start -->

> Computes feature cost by querying ClickHouse meter usage via streaming.Connector and resolving per-unit costs from feature configuration (manual or LLM-based). Thin two-layer domain: adapter owns ClickHouse queries and cost resolution; service owns input validation and delegation.

## Patterns

**Pre-resolve LLM prices once per query** — Call getLLMPrices to build a price cache keyed on (provider, model) before iterating rows — never call llmcostService.ResolvePrice inside the per-row loop. (`prices, err := a.getLLMPrices(ctx, namespace, feature); // then pass prices to resolveUnitCost`)
**Clone GroupBy/FilterGroupBy before merging** — Never mutate the caller's streaming.QueryParams.GroupBy or FilterGroupBy slices/maps; always slices.Clone and build a new merged map. (`merged := maps.Clone(feature.MeterGroupByFilters); for k, v := range params.FilterGroupBy { merged[k] = v }`)
**costResolverFunc abstraction** — Cost resolution is injected as a function type (costResolverFunc), not an interface, separating LLM vs. manual resolution from aggregation logic. (`var resolver costResolverFunc = func(tokenType string) (*ResolvedUnitCost, error) { ... }`)
**pricing-not-found → detail string, not hard error** — In costResolverFunc, convert models.IsGenericNotFoundError to a detail string so the row is still emitted rather than aborting the query. (`if models.IsGenericNotFoundError(err) { return nil, nil /* populate row.Detail instead */ }`)
**Service validates then delegates** — Service.QueryFeatureCost calls input.Validate() then calls cost.Adapter.QueryFeatureCost — no business logic or data access in the service. (`if err := input.Validate(); err != nil { return nil, err }; return s.adapter.QueryFeatureCost(ctx, input)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/cost/adapter.go` | Defines cost.Adapter interface with single method QueryFeatureCost. | Interface has only one method; keep adapter.go as the contract, implementation lives in adapter/ sub-package. |
| `openmeter/cost/service.go` | Defines cost.Service interface plus all input/output types (QueryFeatureCostInput, CostQueryResult, CostQueryRow, ResolvedUnitCost). | QueryFeatureCostInput.Validate() uses models.NewNillableGenericValidationError — replicate this pattern for any new input types. |
| `openmeter/cost/adapter/adapter.go` | Concrete adapter; wires streaming.Connector, feature resolver, llmcost service, and costResolverFunc together. | Internal group-by keys injected into query params must be stripped from output rows — omitting the strip leaks internal fields to callers. |
| `openmeter/cost/adapter/compute.go` | Per-row cost computation: resolveUnitCost switch + costPerTokenForType. | New cost resolution types require a new switch branch in resolveUnitCost AND a costPerTokenForType case if token-type based. |

## Anti-Patterns

- Calling llmcostService.ResolvePrice inside the per-row cost loop — always pre-resolve via getLLMPrices first.
- Passing *entdb.Client or touching Ent directly — this adapter is ClickHouse-only via streaming.Connector.
- Mutating the caller's streaming.QueryParams.GroupBy or FilterGroupBy slices/maps in place — always clone before merging.
- Returning a hard error for pricing-not-found in costResolverFunc — convert to a detail string so the row is still emitted.
- Adding computation or data access directly in service methods — delegate entirely to cost.Adapter.

## Decisions

- **Price cache keyed on (provider, model) not (provider, model, token_type).** — Token-type breakdown happens inside resolveUnitCost after the cache lookup; caching at the coarser key reduces LLM-cost service calls.
- **Internal group-by keys injected into query params and stripped from output.** — ClickHouse queries require additional group-by dimensions for cost resolution that must not surface in the public CostQueryRow.GroupBy map.
- **costResolverFunc is a function type rather than an interface.** — Keeps the resolution strategy as a simple closure without requiring a named type, making it easy to swap LLM vs. manual resolution at construction time.

## Example: Implementing a new cost adapter method that queries ClickHouse

```
import (
	"context"
	"github.com/openmeterio/openmeter/openmeter/cost"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

func (a *adapter) QueryFeatureCost(ctx context.Context, input cost.QueryFeatureCostInput) (*cost.CostQueryResult, error) {
	prices, err := a.getLLMPrices(ctx, input.Namespace, feature)
	if err != nil { return nil, err }
	// Clone params to avoid mutating caller's slice
	params := input.QueryParams
	params.GroupBy = slices.Clone(input.QueryParams.GroupBy)
	// ... inject internal keys, query, strip internal keys from result rows
}
```

<!-- archie:ai-end -->
