# featurecost

<!-- archie:ai-start -->

> Single-operation v3 HTTP handler for querying feature cost by delegating to cost.Service using meter query params built via shared query.BuildQueryParams logic. Bridges feature lookup, meter resolution, and cost calculation into one endpoint.

## Patterns

**HandlerWithArgs for path-param endpoints** — Use httptransport.NewHandlerWithArgs when the URL has a path parameter (featureID here). The third type param is the path param type (string). Plain httptransport.NewHandler is for body-only endpoints. (`QueryFeatureCostHandler httptransport.HandlerWithArgs[QueryFeatureCostRequest, QueryFeatureCostResponse, QueryFeatureCostParams]`)
**Shared query param construction** — Reuse query.BuildQueryParams from api/v3/handlers/meters/query for streaming query param construction rather than duplicating the logic. Pass a query.NewCustomerResolver wrapping customerService. (`params, err := query.BuildQueryParams(ctx, m, req.Body, query.NewCustomerResolver(h.customerService))`)
**Domain validation before service call** — Validate domain preconditions (e.g., feature has a meter) in the operation func using models.NewGenericValidationError before calling the service. (`if feat.MeterID == nil { return QueryFeatureCostResponse{}, models.NewGenericValidationError(...) }`)
**Nil-safe conversion with empty slice fallback** — ToAPIFeatureCostQueryResult returns an empty Data slice (not nil) when result is nil, preventing null in JSON. (`if result == nil { return api.FeatureCostQueryResult{..., Data: []api.FeatureCostQueryRow{}} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Defines Handler interface (one method: QueryFeatureCost) and handler struct with resolveNamespace, costService, featureConnector, meterService, customerService. | All injected services must be present in New() signature; missing one causes Wire compile failure. |
| `query.go` | Implements QueryFeatureCost operation: resolves namespace, parses optional body, looks up feature, validates meter association, builds query params, calls cost service. | ParseOptionalBody is used (not ParseBody) because the request body is optional for this query endpoint. |
| `convert.go` | Converts cost.CostQueryResult to api.FeatureCostQueryResult. Uses nullable.NewNullNullable for absent cost field. | nullable.NewNullNullable vs nullable.NewNullableWithValue must be chosen correctly; using wrong one silently produces unexpected JSON nullability. |

## Anti-Patterns

- Duplicating query.BuildQueryParams logic inline instead of calling the shared function from api/v3/handlers/meters/query
- Returning nil instead of empty slice for Data when result is nil
- Using context.Background() instead of propagating the request ctx

## Decisions

- **Cost querying reuses meter query param builder (query.BuildQueryParams)** — Feature cost uses the same streaming query structure as meter queries; sharing the builder avoids drift between the two filter/window handling paths.

<!-- archie:ai-end -->
