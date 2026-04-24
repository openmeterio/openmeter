# llmcost

<!-- archie:ai-start -->

> v3 HTTP handler package for LLM cost price catalog: listing system and manual prices, getting a price by ID, and managing per-namespace manual overrides (create, list, delete). Bridges llmcost.Service to HTTP using the standard httptransport pattern.

## Patterns

**Decimal-as-string for pricing fields** — All per-token pricing amounts are transmitted as api.Numeric strings and parsed/serialized via alpacadecimal. Use decimalFromString (wraps alpacadecimal.NewFromString + models.NewGenericValidationError) for all inbound conversions. (`inputPerToken, err := decimalFromString(p.InputPerToken)`)
**Provider display name mapping** — providerDisplayNames map in convert.go provides canonical display names for known provider IDs. Unknown providers are title-cased by splitting on hyphen/underscore. Never hardcode provider display names in handler files. (`Provider: api.LLMCostProvider{Id: providerID, Name: formatProviderName(providerID)}`)
**Source mapping: internal vs API enum** — Internal PriceSourceManual maps to api.LLMCostPriceSourceManual; everything else maps to api.LLMCostPriceSourceSystem. This is a one-directional mapping (reads only expose system/manual). (`source := api.LLMCostPriceSourceSystem; if p.Source == llmcost.PriceSourceManual { source = api.LLMCostPriceSourceManual }`)
**Sort field allowlist validation** — Validate sort fields against an explicit allowlist (validPriceSortField) before passing to the service. Return apierrors.NewBadRequestError with a descriptive Reason listing all valid fields. (`if !validPriceSortField(sort.Field) { return req, apierrors.NewBadRequestError(...) }`)
**Nil pointer optional pricing fields** — Optional pricing fields (CacheReadPerToken, CacheWritePerToken, ReasoningPerToken) are *alpacadecimal.Decimal in domain and *string in API; use lo.ToPtr(d.String()) for non-nil values. (`if p.CacheReadPerToken != nil { out.CacheReadPerToken = lo.ToPtr(p.CacheReadPerToken.String()) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Defines Handler interface (5 methods) and handler struct with resolveNamespace, llmcost.Service, options. | Only one domain service (llmcost.Service) is injected; unlike features handler, no secondary services needed. |
| `convert.go` | All domain<->API conversions including domainPriceToAPI, apiPricingToDomain, apiCreateOverrideToDomain, provider display name formatting, and validPriceSortField. | decimalFromString wraps validation error; callers must propagate the error, not ignore it. |
| `list_prices.go` | Handles pagination, sort (with field allowlist), and filters (provider, model_id, model_name, currency, source) for the global price catalog. | Source filter uses filters.FromAPIFilterString, same as other string fields — not an enum filter. |
| `list_overrides.go` | Handles pagination and filters (provider, model_id, model_name, currency) for per-namespace overrides. No sort support unlike list_prices. | Overrides do not support sort — do not add sort without updating the service layer. |

## Anti-Patterns

- Hardcoding provider display names in handler response logic instead of using formatProviderName
- Accepting decimal strings without using decimalFromString — bypasses validation error wrapping
- Exposing internal PriceSource values directly to API instead of mapping through the source enum

## Decisions

- **Provider display names in a static map with title-case fallback** — API consumers need human-readable provider names; a central map avoids scatter while the fallback handles new/unknown providers without code changes.

<!-- archie:ai-end -->
