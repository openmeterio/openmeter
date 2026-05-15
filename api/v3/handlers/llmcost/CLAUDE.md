# llmcost

<!-- archie:ai-start -->

> v3 HTTP handler package for LLM cost price catalog: listing system and manual prices, getting a price by ID, and managing per-namespace manual overrides (create, list, delete). Bridges llmcost.Service to HTTP using the standard httptransport pattern.

## Patterns

**Decimal-as-string for pricing fields** — All per-token pricing amounts are transmitted as api.Numeric strings and parsed/serialized via alpacadecimal. Use decimalFromString (wraps alpacadecimal.NewFromString + models.NewGenericValidationError) for all inbound conversions. (`inputPerToken, err := decimalFromString(p.InputPerToken)`)
**Provider display name mapping** — providerDisplayNames map in convert.go provides canonical display names for known provider IDs. Unknown providers are title-cased by splitting on hyphen/underscore. Never hardcode provider display names in handler files. (`Provider: api.LLMCostProvider{Id: providerID, Name: formatProviderName(providerID)}`)
**Source mapping: internal vs API enum** — Internal PriceSourceManual maps to api.LLMCostPriceSourceManual; everything else maps to api.LLMCostPriceSourceSystem. One-directional mapping (reads only expose system/manual). (`source := api.LLMCostPriceSourceSystem; if p.Source == llmcost.PriceSourceManual { source = api.LLMCostPriceSourceManual }`)
**Sort field allowlist validation** — Validate sort fields against an explicit allowlist (validPriceSortField) before passing to the service. Return apierrors.NewBadRequestError with a descriptive Reason listing all valid fields. (`if !validPriceSortField(sort.Field) { return req, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{{Field: "sort", Reason: "unsupported sort field, must be one of: id, provider.id, model.id, effective_from, effective_to"}}) }`)
**Nil pointer optional pricing fields** — Optional pricing fields (CacheReadPerToken, CacheWritePerToken, ReasoningPerToken) are *alpacadecimal.Decimal in domain and *string in API; use lo.ToPtr(d.String()) for non-nil values. (`if p.CacheReadPerToken != nil { out.CacheReadPerToken = lo.ToPtr(p.CacheReadPerToken.String()) }`)
**No sort support for list_overrides** — Overrides do not support sort — do not add sort without first updating the service layer. list_prices.go supports sort; list_overrides.go does not — they are deliberately asymmetric. (`// list_overrides.go has no sort params block — intentionally omitted`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Defines Handler interface (5 methods) and handler struct with resolveNamespace, llmcost.Service, options. | Only one domain service (llmcost.Service) is injected; unlike features handler, no secondary services needed. |
| `convert.go` | All domain<->API conversions including domainPriceToAPI, apiPricingToDomain, apiCreateOverrideToDomain, provider display name formatting, and validPriceSortField. | decimalFromString wraps validation error; callers must propagate the error, not ignore it. Adding a new provider to providerDisplayNames requires updating the static map. |
| `list_prices.go` | Handles pagination, sort (with field allowlist), and filters (provider, model_id, model_name, currency, source) for the global price catalog. | Source filter uses filters.FromAPIFilterString, same as other string fields — not an enum filter. Sort field must pass validPriceSortField check. |
| `list_overrides.go` | Handles pagination and filters (provider, model_id, model_name, currency) for per-namespace overrides. No sort support unlike list_prices. | Overrides do not support sort — do not add sort without updating the service layer. |

## Anti-Patterns

- Hardcoding provider display names in handler response logic instead of using formatProviderName from convert.go
- Accepting decimal strings without using decimalFromString — bypasses validation error wrapping with models.NewGenericValidationError
- Exposing internal PriceSource values directly to API instead of mapping through the source enum (system/manual)
- Adding sort to list_overrides without updating the service layer to support it

## Decisions

- **Provider display names in a static map with title-case fallback** — API consumers need human-readable provider names; a central map avoids scatter while the fallback handles new/unknown providers without code changes.
- **list_prices supports sort; list_overrides does not** — Overrides are per-namespace and typically small; adding sort complexity would require service-layer changes not yet needed.

## Example: Converting a domain Price to API response with optional pricing fields

```
func domainPriceToAPI(p llmcost.Price) api.LLMCostPrice {
  source := api.LLMCostPriceSourceSystem
  if p.Source == llmcost.PriceSourceManual { source = api.LLMCostPriceSourceManual }
  providerID := string(p.Provider)
  out := api.LLMCostPrice{
    Id: p.ID,
    Provider: api.LLMCostProvider{Id: providerID, Name: formatProviderName(providerID)},
    Model: api.LLMCostModel{Id: p.ModelID, Name: p.ModelName},
    Currency: p.Currency,
    Source: source,
    Pricing: domainPricingToAPI(p.Pricing),
  }
  return out
}

// ...
```

<!-- archie:ai-end -->
