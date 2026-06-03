# httphandler

<!-- archie:ai-start -->

> v1 HTTP handler package (package httpdriver) for portal token endpoints — implements the httptransport.Handler pattern for CreateToken, ListTokens, InvalidateToken, delegating to portal.Service and meter.Service for cross-domain meter slug validation.

## Patterns

**Composite Handler interface with sub-interfaces** — Top-level Handler embeds TokenHandler; each operation returns a typed handler alias. var _ Handler = (*handler)(nil) enforces compliance. (`type TokenHandler interface { CreateToken() CreateTokenHandler; ListTokens() ListTokensHandler; InvalidateToken() InvalidateTokenHandler }`)
**httptransport.NewHandler / NewHandlerWithArgs per endpoint** — Each endpoint is a decoder + operation + ResponseEncoder + options closure. Params-bearing endpoints use NewHandlerWithArgs; param-less use NewHandler. (`return httptransport.NewHandlerWithArgs(decoderFn, operationFn, commonhttp.JSONResponseEncoderWithStatus[T](http.StatusOK), opts...)`)
**Namespace resolved at the start of every decoder** — resolveNamespace(ctx) wraps namespaceDecoder.GetNamespace; returns HTTP 500 on failure. Always call at the very start of every decoder before other logic. (`ns, err := h.resolveNamespace(ctx); if err != nil { return ..., err }`)
**Domain input type as request type alias** — Request types reuse domain input structs directly rather than duplicating fields, keeping the handler thin. (`type CreateTokenRequest = portal.CreateTokenInput`)
**Token field added post-mapping in CreateToken only** — toAPIPortalToken omits the raw JWT string; CreateToken manually sets portalToken.Token = token.Token after the mapping helper. (`portalToken := toAPIPortalToken(token); portalToken.Token = token.Token`)
**WithOperationName appended on every handler** — httptransport.AppendOptions always appends WithOperationName for OTel span naming — never omit. (`httptransport.AppendOptions(h.options, httptransport.WithOperationName("listPortalTokens"))...`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler/TokenHandler interfaces, handler struct, New() constructor, resolveNamespace helper. | New() takes variadic HandlerOption — callers pass error handler and tracer options; forgetting them silences errors and breaks OTel tracing. |
| `portal.go` | All three endpoint implementations: ListTokens, CreateToken, InvalidateToken. | CreateToken validates AllowedMeterSlugs against the live meter list via meterService.ListMeters in the operation phase, not decode. InvalidateToken returns 204 (EmptyResponseEncoder), not 200. |
| `mapping.go` | toAPIPortalToken maps portal.PortalToken to api.PortalToken; sets Expired when ExpiresAt is in the past. | Token field is intentionally NOT mapped here — set it explicitly in CreateToken after toAPIPortalToken; adding it here is a security risk. |

## Anti-Patterns

- Returning the raw JWT token string via toAPIPortalToken — set it manually only in the CreateToken operation.
- Adding database calls directly in handlers — delegate to portal.Service or meter.Service.
- Omitting WithOperationName from AppendOptions — breaks OTel span names.
- Using context.Background() instead of the request context in decoder or operation closures.

## Decisions

- **Meter slug validation in CreateToken operation rather than inside portal.Service.** — portal.Service is JWT-only and has no meter knowledge; cross-domain validation belongs in the HTTP layer that has both services.
- **Request type aliases to domain input structs.** — Avoids duplicating field definitions; keeps the handler thin with the domain type as the single source of truth.

## Example: Add a new portal endpoint following the existing handler pattern

```
// In handler.go — add to TokenHandler interface:
GetToken() GetTokenHandler

// In portal.go:
type (
  GetTokenRequest  struct { namespace, id string }
  GetTokenResponse = *api.PortalToken
  GetTokenHandler  httptransport.Handler[GetTokenRequest, GetTokenResponse]
)

func (h *handler) GetToken() GetTokenHandler {
  return httptransport.NewHandler(
    func(ctx context.Context, r *http.Request) (GetTokenRequest, error) {
      ns, err := h.resolveNamespace(ctx)
      if err != nil { return GetTokenRequest{}, err }
// ...
```

<!-- archie:ai-end -->
