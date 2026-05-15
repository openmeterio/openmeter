# featurecost

<!-- archie:ai-start -->

> Single-operation v3 HTTP handler for querying feature cost by delegating to cost.Service using meter query params built via shared query.BuildQueryParams logic. Bridges feature lookup, meter resolution, and cost calculation into one endpoint with optional request body.

## Patterns

**HandlerWithArgs for path-param endpoints** — Use httptransport.NewHandlerWithArgs when the URL has a path parameter (featureID here). The third type param is the path param type (string). Plain httptransport.NewHandler is for body-only endpoints. (`QueryFeatureCostHandler httptransport.HandlerWithArgs[QueryFeatureCostRequest, QueryFeatureCostResponse, QueryFeatureCostParams]`)
**Shared query param construction** — Reuse query.BuildQueryParams from api/v3/handlers/meters/query for streaming query param construction rather than duplicating the logic. Pass a query.NewCustomerResolver wrapping customerService. (`params, err := query.BuildQueryParams(ctx, m, req.Body, query.NewCustomerResolver(h.customerService))`)
**Domain validation before service call** — Validate domain preconditions (e.g., feature has a meter) in the operation func using models.NewGenericValidationError before calling the service. (`if feat.MeterID == nil { return QueryFeatureCostResponse{}, models.NewGenericValidationError(fmt.Errorf("feature %s has no meter associated", feat.Key)) }`)
**Nil-safe conversion with empty slice fallback** — ToAPIFeatureCostQueryResult returns an empty Data slice (not nil) when result is nil, preventing JSON null. (`if result == nil { return api.FeatureCostQueryResult{..., Data: []api.FeatureCostQueryRow{}} }`)
**ParseOptionalBody for query endpoints** — Use request.ParseOptionalBody (not ParseBody) because the request body is optional for this query endpoint. (`if err := request.ParseOptionalBody(r, &body); err != nil { return QueryFeatureCostRequest{}, err }`)
**nullable.NewNullNullable vs nullable.NewNullableWithValue** — In convert.go, use nullable.NewNullableWithValue when cost is present and nullable.NewNullNullable[api.Numeric]() when absent — wrong choice silently produces unexpected JSON nullability. (`if row.Cost != nil { apiRow.Cost = nullable.NewNullableWithValue(row.Cost.String()) } else { apiRow.Cost = nullable.NewNullNullable[api.Numeric]() }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Defines Handler interface (one method: QueryFeatureCost) and handler struct with resolveNamespace, costService, featureConnector, meterService, customerService. | All injected services must be present in New() signature; missing one causes Wire compile failure. |
| `query.go` | Implements QueryFeatureCost operation: resolves namespace, parses optional body, looks up feature, validates meter association, builds query params, calls cost service. | ParseOptionalBody is used (not ParseBody) because request body is optional; switching to ParseBody breaks bodyless requests. |
| `convert.go` | Converts cost.CostQueryResult to api.FeatureCostQueryResult. Uses nullable.NewNullNullable for absent cost field. | nullable.NewNullNullable vs nullable.NewNullableWithValue must be chosen correctly; using wrong one silently produces unexpected JSON nullability. |

## Anti-Patterns

- Duplicating query.BuildQueryParams logic inline instead of calling the shared function from api/v3/handlers/meters/query
- Returning nil instead of empty slice for Data when result is nil — causes JSON null
- Using context.Background() instead of propagating the request ctx
- Calling request.ParseBody instead of request.ParseOptionalBody for this query endpoint

## Decisions

- **Cost querying reuses meter query param builder (query.BuildQueryParams)** — Feature cost uses the same streaming query structure as meter queries; sharing the builder avoids drift between the two filter/window handling paths.

## Example: Implementing a query operation with optional body and path param

```
func (h *handler) QueryFeatureCost() QueryFeatureCostHandler {
  return httptransport.NewHandlerWithArgs(
    func(ctx context.Context, r *http.Request, featureID QueryFeatureCostParams) (QueryFeatureCostRequest, error) {
      ns, err := h.resolveNamespace(ctx)
      if err != nil { return QueryFeatureCostRequest{}, err }
      var body api.MeterQueryRequest
      if err := request.ParseOptionalBody(r, &body); err != nil {
        return QueryFeatureCostRequest{}, err
      }
      return QueryFeatureCostRequest{Namespace: ns, FeatureID: featureID, Body: body}, nil
    },
    func(ctx context.Context, req QueryFeatureCostRequest) (QueryFeatureCostResponse, error) {
      feat, err := h.featureConnector.GetFeature(ctx, req.Namespace, req.FeatureID, feature.IncludeArchivedFeatureFalse)
      if err != nil { return QueryFeatureCostResponse{}, err }
      if feat.MeterID == nil {
// ...
```

<!-- archie:ai-end -->
