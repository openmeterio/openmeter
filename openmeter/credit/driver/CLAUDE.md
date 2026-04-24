# driver

<!-- archie:ai-start -->

> HTTP handlers (v1 creditdriver) for grant operations (ListGrants, VoidGrant, ListGrantsV2) mounted on the v1 Chi router. Adapts between HTTP request/response and credit.GrantConnector + grant.Repo.

## Patterns

**httptransport.HandlerWithArgs pattern for all endpoints** — Each handler is defined as a type alias httptransport.HandlerWithArgs[Request, Response, Params] and returned from a method on grantHandler. The three-argument form is used when URL params or query params must be decoded separately from the body. (`type ListGrantsHandler httptransport.HandlerWithArgs[ListGrantsHandlerRequest, ListGrantsHandlerResponse, ListGrantsHandlerParams]`)
**Namespace resolved via namespaceDecoder, never from request body** — Every handler calls h.resolveNamespace(ctx) using the injected NamespaceDecoder. If resolution fails it returns 500. The namespace is never read from query params or path in this driver. (`ns, err := h.resolveNamespace(ctx); if !ok { return commonhttp.NewHTTPError(http.StatusInternalServerError, ...) }`)
**Dual pagination: both page/pageSize and limit/offset supported** — ListGrants and ListGrantsV2 support both cursor-based (Page/PageSize) and legacy limit/offset pagination. When Page.IsZero(), Option1 (plain array) is returned; otherwise Option2 (paginated result) is returned for ListGrants. (`if request.params.Page.IsZero() { response.Option1 = &apiGrants } else { response.Option2 = &pagination.Result{...} }`)
**Per-handler error encoder appended to shared options** — Domain errors (GenericValidationError, GrantNotFoundError, pagination.InvalidError) are mapped to HTTP status codes inside each handler's error encoder via httptransport.AppendOptions. The shared h.options provide cross-cutting concerns (logging, tracing). (`httptransport.AppendOptions(h.options, httptransport.WithErrorEncoder(func(...) bool { ... }))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `grant.go` | Defines GrantHandler interface + grantHandler implementation with ListGrants, VoidGrant, ListGrantsV2. Uses both grantRepo (for read path) and grantConnector (for VoidGrant mutation). | ListGrantsV2 resolves customer IDOrKey to ID by calling customerService.GetCustomer — this is a cross-domain call. Deleted customers are silently skipped. The v2 response type is api.GrantV2PaginatedResponse (always paginated, no dual format). |

## Anti-Patterns

- Adding business logic here — the driver only translates HTTP to domain types.
- Reading namespace from HTTP query params instead of namespaceDecoder.
- Returning raw Ent errors from the error encoder — convert to commonhttp.NewHTTPError first.
- Using the v1 api.EntitlementGrant type in a v2 endpoint — v2 uses api.EntitlementGrantV2 via entitlement_httpdriverv2.MapEntitlementGrantToAPIV2.

## Decisions

- **ListGrants returns a union type (array OR paginated result) for backward compatibility; ListGrantsV2 always returns a paginated result.** — The v1 API predates pagination; existing clients expect a plain array when no page parameters are provided. The v2 endpoint was designed with pagination from the start.

<!-- archie:ai-end -->
