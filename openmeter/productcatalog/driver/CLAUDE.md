# driver

<!-- archie:ai-start -->

> HTTP transport layer (package productcatalogdriver) for the feature API. Wires feature.FeatureConnector + meter/llmcost services into httptransport handlers and maps between api.* and feature.* types.

## Patterns

**Handler interface + handlers struct** — FeatureHandler exposes Get/Create/List/DeleteFeature; featureHandlers holds connector, namespaceDecoder, meterService, llmcostService. New handlers go through NewFeatureHandler. (`func NewFeatureHandler(connector feature.FeatureConnector, namespaceDecoder, meterService, llmcostService, options...) FeatureHandler`)
**httptransport three-stage handlers** — Each endpoint is NewHandler/NewHandlerWithArgs(decodeRequest, businessLogic, responseEncoder, options...) with WithOperationName and an error encoder. (`httptransport.NewHandlerWithArgs(decode, exec, commonhttp.JSONResponseEncoder, ...WithOperationName("getFeature"))`)
**Namespace resolution from context** — Every decode step calls h.resolveNamespace(ctx) via namespaceDecoder.GetNamespace; missing namespace is a 500. (`ns, err := h.resolveNamespace(ctx)`)
**Centralized error encoder** — getErrorEncoder() in errors.go maps each feature.*Error / meter error to an HTTP status via commonhttp.HandleErrorIfTypeMatches; attach it with WithErrorEncoder. (`HandleErrorIfTypeMatches[*feature.FeatureNotFoundError](ctx, http.StatusNotFound, err, w)`)
**API<->domain mapping in parser.go** — MapFeatureToResponse, MapFeatureCreateInputsRequest, and domainUnitCostToAPI/apiUnitCostToDomain own all api.Feature translation; handlers call these, not inline conversions. (`resp, err := MapFeatureToResponse(*feat)`)
**MeterSlug resolved to MeterID at the boundary** — CreateFeature decode resolves parsedBody.MeterSlug via meterService.GetMeterByIDOrSlug into a meterID before building inputs. (`m, err := h.meterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{Namespace: ns, IDOrSlug: *parsedBody.MeterSlug})`)
**List dual-shape response** — ListFeatures returns commonhttp.Union of []api.Feature (legacy) vs pagination.Result; selection driven by params.Page.IsZero(). (`if params.Page.IsZero() { response.Option1 = &mapped } else { response.Option2 = ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `feature.go` | FeatureHandler interface, featureHandlers, and all four endpoint definitions. | DeleteFeature uses operation.AsNoResponseOperation(connector.ArchiveFeature); LLM pricing enrichment only runs when UnitCost.Type==LLM and llmcostService!=nil; list rejects PageSize>1000. |
| `parser.go` | api<->feature mapping (response + create inputs + unit cost). | AdvancedMeterGroupByFilters takes precedence over legacy MeterGroupByFilters; unit cost conversion returns errors that must propagate. |
| `errors.go` | getErrorEncoder mapping domain errors to HTTP status codes. | Add new feature error types here or they fall through to a generic 500. |

## Anti-Patterns

- Building api.Feature inline instead of MapFeatureToResponse / parser.go helpers
- Reading the namespace directly instead of resolveNamespace(ctx)
- Adding endpoints without WithOperationName or without registering errors in getErrorEncoder
- Passing meter slug through to the connector instead of resolving it to a meter ID first

## Decisions

- **Package is named productcatalogdriver, not driver** — Avoids collisions and matches the project's *driver HTTP-layer naming convention.
- **List endpoint returns a Union of slice vs paginated result** — Preserves the legacy un-paginated v1 response shape while supporting page-based pagination.

## Example: Defining an httptransport handler with namespace + error encoder

```
func (h *featureHandlers) GetFeature() GetFeatureHandler {
  return httptransport.NewHandlerWithArgs(
    func(ctx context.Context, r *http.Request, featureID string) (models.NamespacedID, error) {
      ns, err := h.resolveNamespace(ctx)
      if err != nil { return models.NamespacedID{}, err }
      return models.NamespacedID{Namespace: ns, ID: featureID}, nil
    },
    func(ctx context.Context, id models.NamespacedID) (api.Feature, error) {
      feat, err := h.connector.GetFeature(ctx, id.Namespace, id.ID, feature.IncludeArchivedFeatureFalse)
      if err != nil { return api.Feature{}, err }
      return MapFeatureToResponse(*feat)
    },
    commonhttp.JSONResponseEncoder,
    httptransport.AppendOptions(h.options, httptransport.WithErrorEncoder(getErrorEncoder()), httptransport.WithOperationName("getFeature"))...,
  )
// ...
```

<!-- archie:ai-end -->
