# featurecost

<!-- archie:ai-start -->

> Single-operation v3 HTTP handler that queries feature cost by resolving a feature, validating its meter association, building streaming query params via the shared meters query builder, and delegating to cost.Service. Body is optional.

## Patterns

**HandlerWithArgs for path-param endpoint** — Uses httptransport.NewHandlerWithArgs with featureID as the third (path-param) type argument; plain NewHandler is only for body-only endpoints. (`QueryFeatureCostHandler httptransport.HandlerWithArgs[QueryFeatureCostRequest, QueryFeatureCostResponse, QueryFeatureCostParams]`)
**Shared query param construction** — Reuses query.BuildQueryParams from api/v3/handlers/meters/query for streaming param construction, passing query.NewCustomerResolver(h.customerService); never duplicate that logic. (`params, err := query.BuildQueryParams(ctx, m, req.Body, query.NewCustomerResolver(h.customerService))`)
**Domain validation before service call** — Validate domain preconditions (feature must have a meter) in the operation func with models.NewGenericValidationError before invoking the cost service. (`if feat.MeterID == nil { return QueryFeatureCostResponse{}, models.NewGenericValidationError(fmt.Errorf("feature %s has no meter associated", feat.Key)) }`)
**ParseOptionalBody for query endpoint** — Use request.ParseOptionalBody (not ParseBody) because the request body is optional; ParseBody would break bodyless requests. (`if err := request.ParseOptionalBody(r, &body); err != nil { return QueryFeatureCostRequest{}, err }`)
**Nil-safe conversion with empty-slice fallback** — ToAPIFeatureCostQueryResult returns an empty Data slice (not nil) when the result is nil to avoid JSON null, and uses nullable.NewNullNullable[api.Numeric]() for absent cost vs NewNullableWithValue when present. (`if row.Cost != nil { apiRow.Cost = nullable.NewNullableWithValue(row.Cost.String()) } else { apiRow.Cost = nullable.NewNullNullable[api.Numeric]() }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler interface (single QueryFeatureCost method) + handler struct holding resolveNamespace, costService, featureConnector, meterService, customerService. | All injected services must be present in New(); a missing one is a Wire compile failure. |
| `query.go` | QueryFeatureCost operation: resolve namespace, parse optional body, look up feature, validate meter association, build query params, call cost service. | ParseOptionalBody (not ParseBody); feature lookup uses feature.IncludeArchivedFeatureFalse. |
| `convert.go` | Converts cost.CostQueryResult to api.FeatureCostQueryResult with nullable cost field. | nullable.NewNullNullable vs NewNullableWithValue must match presence/absence or JSON nullability is silently wrong; nil result -> empty Data slice. |

## Anti-Patterns

- Duplicating query.BuildQueryParams inline instead of calling the shared meters/query function
- Returning nil instead of an empty slice for Data when result is nil (causes JSON null)
- Using context.Background() instead of propagating the request ctx
- Calling request.ParseBody instead of request.ParseOptionalBody for this query endpoint

## Decisions

- **Cost querying reuses the meter query param builder** — Feature cost uses the same streaming query structure as meter queries; sharing query.BuildQueryParams avoids drift between the two filter/window handling paths.

## Example: Query operation with optional body and path param

```
func (h *handler) QueryFeatureCost() QueryFeatureCostHandler {
  return httptransport.NewHandlerWithArgs(
    func(ctx context.Context, r *http.Request, featureID QueryFeatureCostParams) (QueryFeatureCostRequest, error) {
      ns, err := h.resolveNamespace(ctx)
      if err != nil { return QueryFeatureCostRequest{}, err }
      var body api.MeterQueryRequest
      if err := request.ParseOptionalBody(r, &body); err != nil { return QueryFeatureCostRequest{}, err }
      return QueryFeatureCostRequest{Namespace: ns, FeatureID: featureID, Body: body}, nil
    },
    func(ctx context.Context, req QueryFeatureCostRequest) (QueryFeatureCostResponse, error) {
      feat, err := h.featureConnector.GetFeature(ctx, req.Namespace, req.FeatureID, feature.IncludeArchivedFeatureFalse)
      if err != nil { return QueryFeatureCostResponse{}, err }
      if feat.MeterID == nil {
        return QueryFeatureCostResponse{}, models.NewGenericValidationError(fmt.Errorf("feature %s has no meter associated", feat.Key))
      }
// ...
```

<!-- archie:ai-end -->
