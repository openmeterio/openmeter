# cost

<!-- archie:ai-start -->

> Computes feature cost by querying ClickHouse meter usage via streaming.Connector and resolving per-unit costs from feature configuration (manual or LLM-based). Two-layer domain: adapter/ owns ClickHouse queries and cost resolution, service/ owns input validation and delegation.

## Patterns

**Service validates then delegates** — cost.Service.QueryFeatureCost calls input.Validate() then cost.Adapter.QueryFeatureCost — no business logic or data access in the service layer. (`if err := input.Validate(); err != nil { return nil, err }; return s.adapter.QueryFeatureCost(ctx, input)`)
**Pre-resolve LLM prices once per query** — The adapter builds a price cache keyed on (provider, model) via getLLMPrices before iterating rows — never call llmcostService.ResolvePrice inside the per-row loop. (`prices, err := a.getLLMPrices(ctx, namespace, feature); // then pass prices to resolveUnitCost`)
**Clone GroupBy/FilterGroupBy before merging** — Never mutate the caller's streaming.QueryParams.GroupBy or FilterGroupBy; clone before injecting internal keys. (`params.GroupBy = slices.Clone(input.QueryParams.GroupBy)`)
**pricing-not-found → detail string, not hard error** — In costResolverFunc, convert models.IsGenericNotFoundError to a detail string so the row is still emitted rather than aborting the query. (`if models.IsGenericNotFoundError(err) { /* populate row.Detail instead of returning err */ }`)
**QueryFeatureCostInput.Validate via NewNillableGenericValidationError** — All cost input types validate using models.NewNillableGenericValidationError over errors.Join. (`return models.NewNillableGenericValidationError(errors.Join(errs...))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Defines cost.Adapter interface with the single method QueryFeatureCost; the contract for the adapter/ sub-package. | Keep this as the contract only; implementation lives in adapter/. |
| `service.go` | Defines cost.Service interface plus all input/output types (QueryFeatureCostInput, CostQueryResult, CostQueryRow, ResolvedUnitCost) with Validate(). | Replicate NewNillableGenericValidationError for any new input type; cost is alpacadecimal and currency currencyx.Code. |

## Anti-Patterns

- Calling llmcostService.ResolvePrice inside the per-row cost loop instead of pre-resolving via getLLMPrices
- Passing *entdb.Client or touching Ent — this domain is ClickHouse-only via streaming.Connector
- Mutating the caller's streaming.QueryParams.GroupBy/FilterGroupBy in place instead of cloning
- Returning a hard error for pricing-not-found in costResolverFunc instead of a detail string
- Adding computation or data access in service methods — delegate entirely to cost.Adapter

## Decisions

- **Price cache keyed on (provider, model), not (provider, model, token_type)** — Token-type breakdown happens inside resolveUnitCost after the cache lookup; caching at the coarser key reduces LLM-cost service calls.
- **costResolverFunc is a function type, not an interface** — Keeps resolution strategy as a swappable closure (LLM vs. manual) without a named type.

<!-- archie:ai-end -->
