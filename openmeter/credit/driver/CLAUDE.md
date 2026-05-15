# driver

<!-- archie:ai-start -->

> v1 HTTP handlers (creditdriver) for grant operations (ListGrants, VoidGrant, ListGrantsV2) mounted on the v1 Chi router; adapts between HTTP request/response types and credit.GrantConnector + grant.Repo. This is a pure translation layer — no business logic.

## Patterns

**httptransport.HandlerWithArgs for all endpoints** — Each handler is defined as a type alias of httptransport.HandlerWithArgs[Request, Response, Params] and returned from a method on grantHandler. The three-argument form separates URL/query params from body decoding. (`type ListGrantsHandler httptransport.HandlerWithArgs[ListGrantsHandlerRequest, ListGrantsHandlerResponse, ListGrantsHandlerParams]`)
**Namespace resolved via namespaceDecoder, never from request body** — Every handler calls h.resolveNamespace(ctx) using the injected NamespaceDecoder. If resolution fails it returns 500. Namespace is never read from query params or path. (`ns, err := h.resolveNamespace(ctx); if err != nil { return ListGrantsHandlerRequest{}, err }`)
**Dual pagination: array OR paginated result for backward compatibility** — ListGrants returns commonhttp.Union[[]api.EntitlementGrant, pagination.Result[api.EntitlementGrant]]: when Page.IsZero() returns Option1 (plain array), otherwise Option2 (paginated result). ListGrantsV2 always returns a paginated result. (`if request.params.Page.IsZero() { response.Option1 = &apiGrants } else { response.Option2 = &pagination.Result{...} }`)
**Per-handler error encoder appended to shared options** — Domain errors are mapped to HTTP status codes inside each handler's error encoder via httptransport.AppendOptions. Shared h.options provide cross-cutting concerns. (`httptransport.AppendOptions(h.options, httptransport.WithErrorEncoder(func(...) bool { return commonhttp.HandleErrorIfTypeMatches[*pagination.InvalidError](ctx, http.StatusBadRequest, err, w) }))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `grant.go` | Defines GrantHandler interface + grantHandler implementation with ListGrants, VoidGrant, ListGrantsV2. Uses both grantRepo (for read path) and grantConnector (for VoidGrant mutation). | ListGrantsV2 resolves customer IDOrKey to ID by calling customerService.GetCustomer (cross-domain call). Deleted customers are silently skipped. The v2 response type is api.GrantV2PaginatedResponse (always paginated, no dual format). v2 uses api.EntitlementGrantV2 via entitlement_httpdriverv2.MapEntitlementGrantToAPIV2. |

## Anti-Patterns

- Adding business logic here — the driver only translates HTTP to domain types.
- Reading namespace from HTTP query params instead of namespaceDecoder.
- Returning raw Ent errors from the error encoder — convert to commonhttp.NewHTTPError first.
- Using the v1 api.EntitlementGrant type in a v2 endpoint — v2 uses api.EntitlementGrantV2.
- Implementing a new endpoint here without first adding it to the TypeSpec source in api/spec/.

## Decisions

- **ListGrants returns a union type (array OR paginated result) while ListGrantsV2 always returns a paginated result.** — The v1 API predates pagination; existing clients expect a plain array when no page parameters are provided. v2 was designed with pagination from the start.

## Example: Adding a new v1 grant handler following the existing pattern

```
type MyGrantHandlerRequest struct{ id models.NamespacedID }
type MyGrantHandlerResponse = interface{}
type MyGrantHandlerParams struct{ ID string }
type MyGrantHandler httptransport.HandlerWithArgs[MyGrantHandlerRequest, MyGrantHandlerResponse, MyGrantHandlerParams]

func (h *grantHandler) MyGrant() MyGrantHandler {
	return httptransport.NewHandlerWithArgs[MyGrantHandlerRequest, MyGrantHandlerResponse, MyGrantHandlerParams](
		func(ctx context.Context, r *http.Request, p MyGrantHandlerParams) (MyGrantHandlerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil { return MyGrantHandlerRequest{}, err }
			return MyGrantHandlerRequest{id: models.NamespacedID{Namespace: ns, ID: p.ID}}, nil
		},
		func(ctx context.Context, req MyGrantHandlerRequest) (interface{}, error) {
			return nil, h.grantConnector.MyOperation(ctx, req.id)
		},
// ...
```

<!-- archie:ai-end -->
