# driver

<!-- archie:ai-start -->

> v1 HTTP handler package for the feature sub-domain of productcatalog: implements FeatureHandler (GetFeature, CreateFeature, ListFeatures, DeleteFeature) using the httptransport pattern, response mapping, and error encoding. Primary constraint: MeterSlug-to-MeterID resolution must happen in the HTTP decoder, not in the connector.

## Patterns

**httptransport.Handler per operation with typed aliases** — Each operation returns a typed alias (e.g. GetFeatureHandler = httptransport.HandlerWithArgs[Req,Resp,Params]). Decoder, operation func, and encoder injected at construction; never implement ServeHTTP directly. (`return httptransport.NewHandlerWithArgs(decoderFn, operationFn, commonhttp.JSONResponseEncoder, httptransport.AppendOptions(h.options, httptransport.WithErrorEncoder(getErrorEncoder()), httptransport.WithOperationName("getFeature"))...)`)
**Namespace resolution before any connector call** — All handlers call h.resolveNamespace(ctx) which delegates to namespacedriver.NamespaceDecoder.GetNamespace; failure returns 500. Never skip or hardcode namespace resolution. (`ns, err := h.resolveNamespace(ctx); if err != nil { return models.NamespacedID{}, err }`)
**Centralised error encoder via getErrorEncoder()** — errors.go defines getErrorEncoder() chaining commonhttp.HandleErrorIfTypeMatches for each domain error type. Always pass httptransport.WithErrorEncoder(getErrorEncoder()) in handler options; unregistered errors fall to 500. (`commonhttp.HandleErrorIfTypeMatches[*feature.FeatureNotFoundError](ctx, http.StatusNotFound, err, w) || commonhttp.HandleErrorIfTypeMatches[*feature.ForbiddenError](ctx, http.StatusBadRequest, err, w) || ...`)
**MeterSlug-to-MeterID resolution in decoder** — In CreateFeature decoder, if parsedBody.MeterSlug != nil, call meterService.GetMeterByIDOrSlug before calling MapFeatureCreateInputsRequest. The connector works with MeterID only. (`m, err := h.meterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{Namespace: ns, IDOrSlug: *parsedBody.MeterSlug}); meterID = &m.ID`)
**LLM pricing post-enrichment on GetFeature** — After calling connector.GetFeature, if feat.UnitCost.Type == UnitCostTypeLLM, call resolveLLMPricing and enrichFeatureResponseWithPricing. New handlers with LLM features must follow this post-enrichment pattern. (`if feat.UnitCost != nil && feat.UnitCost.Type == feature.UnitCostTypeLLM { pricing := resolveLLMPricing(ctx, h.llmcostService, feat); enrichFeatureResponseWithPricing(&resp, pricing) }`)
**Union response for paginated vs flat list** — ListFeatures returns commonhttp.Union[[]api.Feature, pagination.Result[api.Feature]] — Option1 for flat list when page.IsZero(), Option2 for paginated. (`if params.Page.IsZero() { response.Option1 = &mapped } else { response.Option2 = &pagination.Result[api.Feature]{Items: mapped, TotalCount: paged.TotalCount} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `feature.go` | FeatureHandler interface and featureHandlers struct with GetFeature, CreateFeature, ListFeatures, DeleteFeature implementations. | MeterSlug-to-MeterID must be resolved in the decoder before connector call; LLM pricing enrichment only fires when llmcostService is non-nil. |
| `errors.go` | getErrorEncoder chains feature and meter domain errors to HTTP status codes. | New domain error types must be registered here; unregistered errors fall through to 500 Internal Server Error. |
| `parser.go` | MapFeatureToResponse, MapFeatureCreateInputsRequest, domainUnitCostToAPI, apiUnitCostToDomain, resolveLLMPricing, enrichFeatureResponseWithPricing. | apiUnitCostToDomain uses u.Discriminator() to dispatch; adding a new UnitCostType requires a new case here and in the domain type-switch. |

## Anti-Patterns

- Calling connector methods before namespace resolution — namespace context must be established first.
- Returning namespace errors with status other than 500 — internal server error is the correct code.
- Implementing ServeHTTP directly rather than using httptransport.NewHandler/NewHandlerWithArgs.
- Omitting httptransport.WithOperationName — breaks OTel span naming.
- Hardcoding page size limit without reusing commonhttp constants.

## Decisions

- **MeterSlug-to-MeterID translation in the HTTP decoder, not the connector** — The v1 API accepts meterSlug for backward compat; the domain works with MeterID. Translation at the boundary keeps the connector clean and independent of API versioning concerns.

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
            if err != nil { return api.Feature{}, err }
            if feat.UnitCost != nil && feat.UnitCost.Type == feature.UnitCostTypeLLM {
                if pricing := resolveLLMPricing(ctx, h.llmcostService, feat); pricing != nil {
                    enrichFeatureResponseWithPricing(&resp, pricing)
// ...
```

<!-- archie:ai-end -->
