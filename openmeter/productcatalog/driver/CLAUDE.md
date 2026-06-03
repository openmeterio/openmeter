# driver

<!-- archie:ai-start -->

> v1 HTTP handler package for the feature sub-domain: implements FeatureHandler (GetFeature, CreateFeature, ListFeatures, DeleteFeature) via the httptransport pattern with response mapping and centralised error encoding. Primary constraint: MeterSlug→MeterID resolution happens in the HTTP decoder, never in the connector.

## Patterns

**httptransport.Handler per operation with typed aliases** — Each operation returns a typed alias (e.g. GetFeatureHandler = httptransport.HandlerWithArgs[...]); decoder, op func, encoder injected at construction. Never implement ServeHTTP directly. (`return httptransport.NewHandlerWithArgs(decoderFn, operationFn, commonhttp.JSONResponseEncoder, httptransport.AppendOptions(h.options, httptransport.WithErrorEncoder(getErrorEncoder()), httptransport.WithOperationName("getFeature"))...)`)
**Namespace resolution before any connector call** — Every handler calls h.resolveNamespace(ctx) (delegates to namespacedriver.NamespaceDecoder); failure returns 500. (`ns, err := h.resolveNamespace(ctx); if err != nil { return models.NamespacedID{}, err }`)
**Centralised error encoder via getErrorEncoder()** — errors.go chains commonhttp.HandleErrorIfTypeMatches per domain error type; always pass httptransport.WithErrorEncoder(getErrorEncoder()). (`commonhttp.HandleErrorIfTypeMatches[*feature.FeatureNotFoundError](ctx, http.StatusNotFound, err, w) || ...`)
**MeterSlug→MeterID resolution in decoder** — In CreateFeature decoder, if parsedBody.MeterSlug != nil, call meterService.GetMeterByIDOrSlug before MapFeatureCreateInputsRequest; the connector takes MeterID only. (`m, err := h.meterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{Namespace: ns, IDOrSlug: *parsedBody.MeterSlug}); meterID = &m.ID`)
**LLM pricing post-enrichment on GetFeature** — After GetFeature, if UnitCost.Type == UnitCostTypeLLM and llmcostService != nil, call resolveLLMPricing then enrichFeatureResponseWithPricing. (`if feat.UnitCost != nil && feat.UnitCost.Type == feature.UnitCostTypeLLM && h.llmcostService != nil { pricing := resolveLLMPricing(ctx, h.llmcostService, feat); enrichFeatureResponseWithPricing(&resp, pricing) }`)
**Union response for paginated vs flat list** — ListFeatures returns commonhttp.Union[[]api.Feature, pagination.Result[api.Feature]] — Option1 for flat list when page.IsZero(), Option2 for paginated. (`if params.Page.IsZero() { response.Option1 = &mapped } else { response.Option2 = &pagination.Result[api.Feature]{...} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `feature.go` | FeatureHandler interface and featureHandlers struct with GetFeature/CreateFeature/ListFeatures/DeleteFeature. | MeterSlug→MeterID must be resolved in the decoder; LLM pricing enrichment only fires when llmcostService is non-nil. |
| `errors.go` | getErrorEncoder chains feature and meter domain errors to HTTP status codes. | New domain error types must be registered here; unregistered errors fall through to 500. |
| `parser.go` | MapFeatureToResponse, MapFeatureCreateInputsRequest, domainUnitCostToAPI, apiUnitCostToDomain, resolveLLMPricing, enrichFeatureResponseWithPricing. | apiUnitCostToDomain dispatches via u.Discriminator(); a new UnitCostType requires a case here and in the domain type-switch. |

## Anti-Patterns

- Calling connector methods before namespace resolution.
- Returning namespace errors with a status other than 500.
- Implementing ServeHTTP directly instead of httptransport.NewHandler/NewHandlerWithArgs.
- Omitting httptransport.WithOperationName — breaks OTel span naming.
- Hardcoding page-size limits instead of reusing commonhttp constants.

## Decisions

- **MeterSlug→MeterID translation in the HTTP decoder, not the connector** — The v1 API accepts meterSlug for backward compat while the domain works with MeterID; translating at the boundary keeps the connector independent of API versioning.

## Example: Add a new v1 feature operation with LLM pricing enrichment

```
func (h *featureHandlers) MyOp() MyOpHandler {
    return httptransport.NewHandlerWithArgs(
        func(ctx context.Context, r *http.Request, id string) (MyOpRequest, error) {
            ns, err := h.resolveNamespace(ctx)
            if err != nil { return MyOpRequest{}, err }
            return MyOpRequest{Namespace: ns, ID: id}, nil
        },
        func(ctx context.Context, req MyOpRequest) (api.Feature, error) {
            feat, err := h.connector.GetFeature(ctx, req.Namespace, req.ID, feature.IncludeArchivedFeatureFalse)
            if err != nil { return api.Feature{}, err }
            resp, err := MapFeatureToResponse(*feat)
            // ... LLM enrichment as above ...
            return resp, err
        },
        commonhttp.JSONResponseEncoder,
// ...
```

<!-- archie:ai-end -->
