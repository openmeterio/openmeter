# httpdriver

<!-- archie:ai-start -->

> HTTP handler layer for the addon domain, adapting addon.Service to request/response via the generic httptransport.Handler pipeline. Exposes a Handler interface consumed by the v1 Chi router; each operation is an independently typed handler value.

## Patterns

**httptransport.NewHandler(WithArgs) per operation** — Each operation returns a typed handler via NewHandler (no path params) or NewHandlerWithArgs (with path params): decoder, operation, encoder closures. (`return httptransport.NewHandlerWithArgs(decoderFn, operationFn, commonhttp.JSONResponseEncoderWithStatus[Resp](http.StatusOK), opts...)`)
**Type aliases per operation** — Each handler block declares Request/Response/Handler type aliases mapping domain input/output to api.* generated types. (`type ( ListAddonsRequest = addon.ListAddonsInput; ListAddonsResponse = api.AddonPaginatedResponse; ListAddonsHandler httptransport.HandlerWithArgs[ListAddonsRequest, ListAddonsResponse, ListAddonsParams] )`)
**Namespace via h.resolveNamespace(ctx)** — Every decoder calls h.resolveNamespace(ctx) (namespacedriver.NamespaceDecoder). Namespace comes from context, never a path param. (`ns, err := h.resolveNamespace(ctx); if err != nil { return ..., err }`)
**ValidationErrorEncoder per handler** — Every handler appends WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(ResourceKindAddon)) to map validation errors to 400. (`httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindAddon))`)
**All api<->domain conversions in mapping.go** — FromAddon, AsCreateAddonRequest, AsUpdateAddonRequest live in mapping.go; closures call them, never inline-convert. (`return FromAddon(*a)`)
**Handler interface with compile-time assertion** — Handler embeds AddonHandler listing all operation methods; var _ Handler = (*handler)(nil) asserts compliance. (`var _ Handler = (*handler)(nil)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `driver.go` | Handler + AddonHandler interface definitions, handler struct, New constructor, resolveNamespace helper. | New operations must be added to both the AddonHandler interface and the handler struct; never bypass resolveNamespace. |
| `addon.go` | One function per HTTP operation, each returning its typed handler. | IgnoreNonCriticalIssues=true on create/update — preserve. Delete uses StatusNoContent + EmptyResponseEncoder; Create uses StatusCreated. |
| `mapping.go` | api<->domain conversion using productcatalog/http rate-card helpers. | Call Currency.Validate() after assignment. Use http.AsRateCards/http.FromRateCard, not custom rate-card logic. |

## Anti-Patterns

- Calling domain services inside decoder functions — decoders only parse/validate HTTP input.
- Inline type conversion in handler closures instead of mapping.go.
- Omitting WithErrorEncoder(ValidationErrorEncoder(...)) — validation errors won't map to 400.
- Adding handler methods to the struct without adding them to the Handler/AddonHandler interface.
- Using context.Background() inside closures instead of the passed ctx.

## Decisions

- **Handlers return typed handler values rather than implementing http.Handler directly.** — The httptransport.Handler[Req,Resp] pattern separates decode/operation/encode, making each independently testable and middleware-composable via Chain.

## Example: Add an addon HTTP operation with a path param

```
type ( CloneAddonRequest = addon.CloneAddonInput; CloneAddonResponse = api.Addon; CloneAddonHandler httptransport.HandlerWithArgs[CloneAddonRequest, CloneAddonResponse, string] )

func (h *handler) CloneAddon() CloneAddonHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, addonID string) (CloneAddonRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil { return CloneAddonRequest{}, err }
			return CloneAddonRequest{NamespacedID: models.NamespacedID{Namespace: ns, ID: addonID}}, nil
		},
		func(ctx context.Context, request CloneAddonRequest) (CloneAddonResponse, error) { a, err := h.service.CloneAddon(ctx, request); ... },
	)
}
```

<!-- archie:ai-end -->
