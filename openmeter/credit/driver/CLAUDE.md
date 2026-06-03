# driver

<!-- archie:ai-start -->

> v1 HTTP handlers (creditdriver) for grant operations (ListGrants, VoidGrant, ListGrantsV2) on the v1 Chi router; a pure translation layer adapting HTTP request/response to credit.GrantConnector + grant.Repo with no business logic.

## Patterns

**httptransport.HandlerWithArgs per endpoint** — Each handler is a type alias of httptransport.HandlerWithArgs[Request, Response, Params] returned from a method on grantHandler, separating URL/query params from body decode. (`type ListGrantsHandler httptransport.HandlerWithArgs[ListGrantsHandlerRequest, ListGrantsHandlerResponse, ListGrantsHandlerParams]`)
**Namespace via namespaceDecoder, never from request** — Every handler resolves namespace through the injected NamespaceDecoder; failure returns an error. Namespace is never read from path/query/body. (`ns, err := h.resolveNamespace(ctx); if err != nil { return ListGrantsHandlerRequest{}, err }`)
**Dual pagination union for v1 backward compatibility** — ListGrants returns commonhttp.Union[[]api.EntitlementGrant, pagination.Result[...]]: Option1 (plain array) when Page.IsZero(), else Option2; ListGrantsV2 always returns a paginated result. (`if request.params.Page.IsZero() { response.Option1 = &apiGrants } else { response.Option2 = &pagination.Result{...} }`)
**Per-handler error encoder appended to shared options** — Domain errors map to HTTP statuses inside each handler's error encoder via httptransport.AppendOptions over shared h.options. (`httptransport.AppendOptions(h.options, httptransport.WithErrorEncoder(func(...) bool { return commonhttp.HandleErrorIfTypeMatches[*pagination.InvalidError](ctx, http.StatusBadRequest, err, w) }))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `grant.go` | GrantHandler interface + grantHandler impl: ListGrants, VoidGrant, ListGrantsV2. Uses grantRepo for reads and grantConnector for VoidGrant mutation. | ListGrantsV2 resolves customer IDOrKey→ID via customerService.GetCustomer (cross-domain) and silently skips deleted customers. v2 always returns api.GrantV2PaginatedResponse with api.EntitlementGrantV2 (no dual format). |

## Anti-Patterns

- Adding business logic here — the driver only translates HTTP to domain types.
- Reading namespace from HTTP query/path/body instead of namespaceDecoder.
- Returning raw Ent errors from the error encoder instead of commonhttp.NewHTTPError.
- Using the v1 api.EntitlementGrant type in a v2 endpoint (v2 uses api.EntitlementGrantV2).
- Adding a new endpoint here without first adding it to the TypeSpec source in api/spec/.

## Decisions

- **ListGrants returns a union (array OR paginated) while ListGrantsV2 is always paginated.** — The v1 API predates pagination and existing clients expect a plain array without page params; v2 was designed paginated from the start.

## Example: Adding a new v1 grant handler following the HandlerWithArgs pattern

```
type MyGrantHandler httptransport.HandlerWithArgs[MyGrantHandlerRequest, interface{}, MyGrantHandlerParams]

func (h *grantHandler) MyGrant() MyGrantHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, p MyGrantHandlerParams) (MyGrantHandlerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil { return MyGrantHandlerRequest{}, err }
			return MyGrantHandlerRequest{id: models.NamespacedID{Namespace: ns, ID: p.ID}}, nil
		},
		func(ctx context.Context, req MyGrantHandlerRequest) (interface{}, error) {
			return nil, h.grantConnector.MyOperation(ctx, req.id)
		},
	)
}
```

<!-- archie:ai-end -->
