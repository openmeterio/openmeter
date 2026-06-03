# llmcost

<!-- archie:ai-start -->

> v3 HTTP handler package for the LLM cost price catalog: listing system and manual prices, getting a price by ID, and managing per-namespace manual overrides (create/list/delete). Bridges llmcost.Service to HTTP via the httptransport pipeline.

## Patterns

**Decimal-as-string via decimalFromString** — All per-token pricing amounts are api strings parsed with decimalFromString (alpacadecimal.NewFromString wrapped in models.NewGenericValidationError). Outbound, decimals are serialized with .String(). Never accept a decimal string without going through decimalFromString. (`inputPerToken, err := decimalFromString(p.InputPerToken)`)
**Provider display-name mapping with title-case fallback** — providerDisplayNames (static map in convert.go) maps known provider IDs to display names; formatProviderName falls back to splitting on hyphen/underscore and capitalizing. Never hardcode provider display names in handler files. (`Provider: api.LLMCostProvider{Id: providerID, Name: formatProviderName(providerID)}`)
**Source enum one-directional mapping** — Internal PriceSourceManual maps to api.LLMCostPriceSourceManual; everything else maps to api.LLMCostPriceSourceSystem. Reads only ever expose system/manual. (`source := api.LLMCostPriceSourceSystem; if p.Source == llmcost.PriceSourceManual { source = api.LLMCostPriceSourceManual }`)
**Sort-field allowlist validation** — list_prices validates parsed sort fields against validPriceSortField before calling the service, returning apierrors.NewBadRequestError with the full valid-field list on failure. (`if !validPriceSortField(sort.Field) { return req, apierrors.NewBadRequestError(ctx, ..., {Field: "sort", Reason: "...must be one of: id, provider.id, model.id, effective_from, effective_to"}) }`)
**Nil-guarded optional pricing fields** — Optional fields (CacheReadPerToken, CacheWritePerToken, ReasoningPerToken) are *alpacadecimal.Decimal in domain and *string in API; use lo.ToPtr(d.String()) only when non-nil. (`if p.CacheReadPerToken != nil { out.CacheReadPerToken = lo.ToPtr(p.CacheReadPerToken.String()) }`)
**Filter parsing via filters.FromAPIFilterString** — All string filters (provider, model_id, model_name, currency, and source) are parsed with filters.FromAPIFilterString, each wrapped in its own bad-request error path. Source is a string filter, not an enum filter. (`provider, err := filters.FromAPIFilterString(params.Filter.Provider)`)
**Asymmetric list capabilities (prices vs overrides)** — list_prices supports sort; list_overrides does not. Overrides are per-namespace and small — do not add sort to list_overrides without first updating the service layer. (`// list_overrides.go has no Sort params block — intentionally omitted`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler interface (5 methods) and handler struct with resolveNamespace, llmcost.Service, options; New() constructor. | Only one domain service (llmcost.Service) is injected — no secondary services. |
| `convert.go` | All domain<->API conversions: domainPriceToAPI, domainPricingToAPI, apiPricingToDomain, apiCreateOverrideToDomain, decimalFromString, formatProviderName, validPriceSortField, providerDisplayNames. | decimalFromString returns a wrapped validation error — propagate it, never ignore. Adding a provider requires updating the static providerDisplayNames map. |
| `list_prices.go` | Pagination (default page 1 / size 20), sort with allowlist, and five filters (provider, model_id, model_name, currency, source) for the global catalog. | Sort field must pass validPriceSortField. Source filter is a string filter via FromAPIFilterString. |
| `list_overrides.go` | Pagination and four filters (provider, model_id, model_name, currency) for per-namespace overrides; no sort. | No sort support — do not add without service-layer changes. |
| `create_override.go` | Parses api.LLMCostOverrideCreate via request.ParseBody, resolves namespace, converts to domain via apiCreateOverrideToDomain, returns HTTP 201. | Uses bare httptransport.NewHandler (no path arg); validation errors from apiCreateOverrideToDomain must surface from the decoder. |
| `delete_override.go / get_price.go` | HandlerWithArgs[..., api.ULID] keyed on the path ULID; delete returns 204 via EmptyResponseEncoder. | Both set ID and Namespace on the domain input from the path arg + resolveNamespace. |

## Anti-Patterns

- Hardcoding provider display names in handler logic instead of using formatProviderName.
- Accepting decimal strings without decimalFromString — bypasses NewGenericValidationError wrapping.
- Exposing internal PriceSource values directly instead of mapping through the system/manual enum.
- Adding sort to list_overrides without updating the service layer.
- Using commonhttp.GenericErrorEncoder instead of apierrors.GenericErrorEncoder.

## Decisions

- **Provider display names in a static map with title-case fallback.** — API consumers need human-readable names; a central map avoids scatter while the fallback handles unknown providers without code changes.
- **list_prices supports sort while list_overrides does not.** — Overrides are per-namespace and typically small; sort complexity would need service-layer changes not yet warranted.

## Example: Converting a domain Price to API response with optional pricing fields

```
func domainPriceToAPI(p llmcost.Price) api.LLMCostPrice {
  source := api.LLMCostPriceSourceSystem
  if p.Source == llmcost.PriceSourceManual { source = api.LLMCostPriceSourceManual }
  return api.LLMCostPrice{
    Id: p.ID,
    Provider: api.LLMCostProvider{Id: string(p.Provider), Name: formatProviderName(string(p.Provider))},
    Model: api.LLMCostModel{Id: p.ModelID, Name: p.ModelName},
    Currency: p.Currency, Source: source,
    Pricing: domainPricingToAPI(p.Pricing),
  }
}
```

<!-- archie:ai-end -->
