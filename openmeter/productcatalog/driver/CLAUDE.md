# driver

<!-- archie:ai-start -->

> v1 HTTP handlers for the feature sub-domain of productcatalog: implements FeatureHandler (GetFeature, CreateFeature, ListFeatures, DeleteFeature) using the httptransport pattern, plus response mapping and error encoding.

## Patterns

**httptransport.Handler per operation** — Each operation is returned as a typed alias (e.g. GetFeatureHandler = httptransport.HandlerWithArgs[Req,Resp,Params]). Decoder, operation func, and encoder are injected at construction; never implement ServeHTTP directly. (`return httptransport.NewHandlerWithArgs(decoderFn, operationFn, commonhttp.JSONResponseEncoder, opts...)`)
**Namespace resolution via namespaceDecoder** — All handlers call h.resolveNamespace(ctx) which delegates to namespacedriver.NamespaceDecoder.GetNamespace; failure returns 500. Never hardcode or skip namespace resolution. (`ns, err := h.resolveNamespace(ctx); if err != nil { return ... }`)
**Centralised error encoder via getErrorEncoder()** — errors.go defines getErrorEncoder() that chains commonhttp.HandleErrorIfTypeMatches for each domain error type. Always pass httptransport.WithErrorEncoder(getErrorEncoder()) in handler options. (`httptransport.WithErrorEncoder(getErrorEncoder())`)
**LLM pricing enrichment post-operation** — GetFeature checks feat.UnitCost.Type == UnitCostTypeLLM and calls resolveLLMPricing + enrichFeatureResponseWithPricing after the main operation. New handlers with LLM features must follow this post-enrichment pattern. (`if feat.UnitCost != nil && feat.UnitCost.Type == feature.UnitCostTypeLLM { pricing := resolveLLMPricing(...) }`)
**Union response for paginated vs flat list** — ListFeatures returns commonhttp.Union[[]api.Feature, pagination.Result[api.Feature]] — Option1 for flat list (page.IsZero()), Option2 for paginated. New list handlers must follow this dual-mode pattern. (`if params.Page.IsZero() { response.Option1 = &mapped } else { response.Option2 = &pagination.Result{...} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `feature.go` | FeatureHandler interface and featureHandlers struct with all four operation implementations. | MeterSlug-to-MeterID resolution in CreateFeature decoder — must call meterService.GetMeterByIDOrSlug before calling connector.CreateFeature. |
| `errors.go` | getErrorEncoder chains feature and meter domain errors to HTTP status codes. | New domain error types must be registered here; unregistered errors fall through to 500. |
| `parser.go` | Bidirectional mapping: MapFeatureToResponse, MapFeatureCreateInputsRequest, domainUnitCostToAPI, apiUnitCostToDomain, resolveLLMPricing, enrichFeatureResponseWithPricing. | apiUnitCostToDomain uses Discriminator() to dispatch; adding a new UnitCostType requires a new case here and in the domain switch. |

## Anti-Patterns

- Returning namespace errors with status other than 500 — internal server error is the correct code when namespace cannot be resolved.
- Calling connector methods before namespace resolution.
- Omitting httptransport.WithOperationName — breaks OTel span naming.
- Hardcoding page size limit (1000) without reusing commonhttp constants.

## Decisions

- **MeterSlug-to-MeterID translation lives in the HTTP decoder, not the connector** — The v1 API accepts meterSlug for backward compat; the domain works with MeterID. Translation at the boundary keeps the connector clean.

<!-- archie:ai-end -->
