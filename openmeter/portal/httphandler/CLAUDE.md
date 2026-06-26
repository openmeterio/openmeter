# httphandler

<!-- archie:ai-start -->

> HTTP transport layer (package httpdriver) exposing portal token endpoints — create, list, invalidate — by wrapping portal.Service and meter.Service in the project's httptransport.Handler request/response pattern.

## Patterns

**httptransport three-function handler** — Each endpoint returns a typed handler built from a request decoder, business logic func, and response encoder, with WithOperationName set. (`httptransport.NewHandler(decode, exec, commonhttp.JSONResponseEncoderWithStatus[CreateTokenResponse](http.StatusOK), httptransport.AppendOptions(h.options, httptransport.WithOperationName("createPortalToken"))...)`)
**Handler interface composition** — handler implements TokenHandler (CreateToken/ListTokens/InvalidateToken) with `var _ Handler = (*handler)(nil)`; constructed via New(namespaceDecoder, portalService, meterService, options...). (`type Handler interface { TokenHandler }`)
**Namespace resolved from decoder** — Every handler calls h.resolveNamespace(ctx) which reads namespaceDecoder.GetNamespace; missing namespace is a 500. (`ns, err := h.resolveNamespace(ctx); if err != nil { return ListTokensRequest{}, err }`)
**Type aliases for request/response/handler** — Each endpoint declares aliases (e.g. CreateTokenRequest = portal.CreateTokenInput, CreateTokenResponse = *api.PortalToken) tying transport types to domain/API types. (`type CreateTokenHandler httptransport.Handler[CreateTokenRequest, CreateTokenResponse]`)
**Centralized API mapping** — mapping.go toAPIPortalToken converts portal.PortalToken -> api.PortalToken and deliberately omits Token; CreateToken re-attaches token.Token after mapping. (`portalToken := toAPIPortalToken(token); portalToken.Token = token.Token`)
**Meter-slug validation before token creation** — CreateToken validates every AllowedMeterSlug exists via meterService.ListMeters (filter.FilterString{In:...}) returning meter.NewMeterNotFoundError on miss. (`if _, ok := metersBySlug[slug]; !ok { return nil, meter.NewMeterNotFoundError(slug) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler/TokenHandler interfaces, handler struct (namespaceDecoder, options, portalService, meterService), New() constructor, resolveNamespace helper. | Package is named httpdriver despite the folder being httphandler — match the package name. resolveNamespace returns a 500 (commonhttp.NewHTTPError) not 400 when namespace missing. |
| `portal.go` | CreateToken/ListTokens/InvalidateToken handlers with their request/response type aliases and operation names. | CreateToken still returns http.StatusOK (TODO says should be 201). ListTokens hardcodes page 1 with default pageSize 25. InvalidateToken uses EmptyResponseEncoder with 204. Empty (but non-nil) AllowedMeterSlugs array is a validation error. |
| `mapping.go` | toAPIPortalToken domain->API mapper; sets Expired=true when ExpiresAt is in the past. | Never map Token here — it is a security risk; only CreateToken sets it explicitly afterward. |

## Anti-Patterns

- Mapping the raw token into the API response inside toAPIPortalToken (leaks the secret on list/get).
- Skipping resolveNamespace / hardcoding a namespace instead of reading the namespace decoder.
- Creating a token without validating AllowedMeterSlugs against meterService.ListMeters.
- Writing inline JSON encoding instead of commonhttp encoders + httptransport handler wrappers.
- Allowing an empty-but-present AllowedMeterSlugs array through to the service.

## Decisions

- **Token string is excluded from the standard mapper and only attached in CreateToken.** — The signed token is sensitive and must be returned exactly once at creation, never on list/invalidate responses.
- **Handlers depend on meter.Service in addition to portal.Service.** — AllowedMeterSlugs must reference real meters, so creation validates slugs against the meter catalog before issuing a token.

## Example: Validate meter slugs then create a token, attaching the secret once

```
meterList, err := h.meterService.ListMeters(ctx, meter.ListMetersParams{
  Namespace: request.Namespace,
  Key:       &filter.FilterString{In: request.AllowedMeterSlugs},
})
metersBySlug := lo.KeyBy(meterList.Items, func(m meter.Meter) string { return m.Key })
for _, slug := range *request.AllowedMeterSlugs {
  if _, ok := metersBySlug[slug]; !ok { return nil, meter.NewMeterNotFoundError(slug) }
}
token, err := h.portalService.CreateToken(ctx, request)
portalToken := toAPIPortalToken(token)
portalToken.Token = token.Token
```

<!-- archie:ai-end -->
