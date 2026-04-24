# httphandler

<!-- archie:ai-start -->

> HTTP handler package (package httpdriver) for portal token endpoints — implements the httptransport.Handler pattern for CreateToken, ListTokens, and InvalidateToken, delegating to portal.Service and meter.Service for validation.

## Patterns

**Composite Handler interface** — Top-level Handler embeds sub-interfaces (TokenHandler); each operation returns a typed handler alias. var _ Handler = (*handler)(nil) enforces compile-time compliance. (`type Handler interface { TokenHandler }
type TokenHandler interface { CreateToken() CreateTokenHandler; ListTokens() ListTokensHandler; InvalidateToken() InvalidateTokenHandler }`)
**httptransport.NewHandler / NewHandlerWithArgs** — Each endpoint is a closure pair: (decoder func) + (operation func) + ResponseEncoder + options. Params-bearing endpoints use NewHandlerWithArgs; param-less use NewHandler. (`return httptransport.NewHandlerWithArgs(decoderFn, operationFn, commonhttp.JSONResponseEncoderWithStatus[T](http.StatusOK), opts...)`)
**Namespace resolved via namespaceDecoder** — resolveNamespace(ctx) wraps namespaceDecoder.GetNamespace; returns HTTP 500 on failure. Always call this at the start of every decoder func. (`ns, err := h.resolveNamespace(ctx)
if err != nil { return ..., err }`)
**Domain input type as request type alias** — Request types reuse domain input structs directly (CreateTokenRequest = portal.CreateTokenInput) rather than duplicating fields. (`type CreateTokenRequest = portal.CreateTokenInput`)
**Token field added post-mapping in CreateToken operation** — toAPIPortalToken deliberately omits the raw token string; CreateToken operation manually sets portalToken.Token = token.Token after calling the mapping helper. (`portalToken := toAPIPortalToken(token)
portalToken.Token = token.Token`)
**WithOperationName option on every handler** — httptransport.AppendOptions appends WithOperationName (e.g. "createPortalToken") to h.options for every handler — required for OTel span naming. (`httptransport.AppendOptions(h.options, httptransport.WithOperationName("listPortalTokens"))...`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler/TokenHandler interfaces, handler struct, New() constructor, resolveNamespace helper | New() takes variadic HandlerOption — callers pass error handler and tracer options here; forgetting them silences errors. |
| `portal.go` | All three endpoint implementations: ListTokens, CreateToken, InvalidateToken | CreateToken validates AllowedMeterSlugs against live meter list — this is an operation-level side call to meterService.ListMeters, not a decode-phase check. InvalidateToken returns 204 (EmptyResponseEncoder), not 200. |
| `mapping.go` | toAPIPortalToken maps portal.PortalToken to api.PortalToken; sets Expired flag if ExpiresAt is in the past | Token field is intentionally NOT mapped here — always set it explicitly in the create handler after calling toAPIPortalToken. |

## Anti-Patterns

- Returning the raw JWT token string via toAPIPortalToken — it must be set manually only in CreateToken
- Adding database calls directly in handlers — delegate to portal.Service or meter.Service
- Omitting WithOperationName from AppendOptions — breaks OTel tracing span names
- Using context.Background() instead of the request context in decoder or operation closures

## Decisions

- **Meter slug validation in CreateToken operation rather than inside portal.Service** — portal.Service is JWT-only and has no knowledge of meters; cross-domain validation belongs in the HTTP handler layer which has access to both services.
- **Request type aliases to domain input structs** — Avoids duplicating field definitions; keeps the handler thin and the domain type as the single source of truth for input shape.

## Example: Adding a new portal endpoint following the existing pattern

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
