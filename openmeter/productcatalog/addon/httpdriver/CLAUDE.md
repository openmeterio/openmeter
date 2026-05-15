# httpdriver

<!-- archie:ai-start -->

> HTTP handler layer for the addon domain, adapting addon.Service to HTTP request/response via the generic httptransport.Handler pipeline. Exposes a Handler interface consumed by the v1 Chi router; each operation is an independently typed handler value.

## Patterns

**httptransport.NewHandler / NewHandlerWithArgs per operation** — Each operation returns a typed handler via httptransport.NewHandler (no path params) or NewHandlerWithArgs (with path params). Three closures: decoder, operation, encoder. (`return httptransport.NewHandlerWithArgs(decoderFn, operationFn, commonhttp.JSONResponseEncoderWithStatus[Resp](http.StatusOK), opts...)`)
**Type aliases for Request/Response/Handler per operation** — Each handler function block declares type aliases at the top mapping to domain input/output and api.* generated types. Makes the handler's type contract explicit. (`type (
	ListAddonsRequest  = addon.ListAddonsInput
	ListAddonsResponse = api.AddonPaginatedResponse
	ListAddonsHandler  httptransport.HandlerWithArgs[ListAddonsRequest, ListAddonsResponse, ListAddonsParams]
)`)
**Namespace resolved via h.resolveNamespace(ctx)** — Every decoder calls h.resolveNamespace(ctx) which uses namespacedriver.NamespaceDecoder.GetNamespace. Failure returns 500. Never pass namespace as a path param — it comes from context. (`ns, err := h.resolveNamespace(ctx); if err != nil { return ..., err }`)
**ValidationErrorEncoder per handler** — Every handler appends httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindAddon)) to map domain validation errors to 400. (`httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindAddon))`)
**All api<->domain conversions in mapping.go** — FromAddon (domain->api), AsCreateAddonRequest, AsUpdateAddonRequest (api->domain) live in mapping.go. Handler closures call these helpers, never inline-convert. (`return FromAddon(*a)`)
**Handler interface composed from sub-interfaces** — Handler interface in driver.go embeds AddonHandler listing all operation handler methods. var _ Handler = (*handler)(nil) provides compile-time assertion. (`var _ Handler = (*handler)(nil)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `driver.go` | Handler + AddonHandler interface definitions, handler struct, New constructor, and resolveNamespace helper. | New operations must be added to both the AddonHandler interface and the handler struct. resolveNamespace must never be bypassed. |
| `addon.go` | One function per HTTP operation (ListAddons, CreateAddon, UpdateAddon, DeleteAddon, GetAddon, PublishAddon, ArchiveAddon), each returning its typed handler. | IgnoreNonCriticalIssues=true is set on create/update — preserve this. Delete uses http.StatusNoContent with EmptyResponseEncoder. Create uses http.StatusCreated. |
| `mapping.go` | api<->domain type conversion with Currency.Validate() after currency assignment and RateCards mapped via productcatalog/http package helpers. | Currency.Validate() must be called after assignment. Use http.AsRateCards and http.FromRateCard for rate card conversion, not custom logic. |

## Anti-Patterns

- Calling domain services inside decoder functions — decoders must only parse/validate HTTP input.
- Inline type conversion in handler closures instead of using mapping.go helpers.
- Omitting WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(...)) — validation errors will not map to 400.
- Adding handler methods to handler struct without adding them to the Handler/AddonHandler interface.
- Using context.Background() inside handler closures instead of the ctx passed to the operation closure.

## Decisions

- **Handler methods return typed handler values rather than implementing http.Handler directly.** — The httptransport.Handler[Req,Resp] pattern separates decoding, operation, and encoding, making each independently testable and composable with middleware via Chain.

## Example: Add a new addon HTTP operation with path param

```
type (
	CloneAddonRequest  = addon.CloneAddonInput
	CloneAddonResponse = api.Addon
	CloneAddonHandler  httptransport.HandlerWithArgs[CloneAddonRequest, CloneAddonResponse, string]
)

func (h *handler) CloneAddon() CloneAddonHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, addonID string) (CloneAddonRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil { return CloneAddonRequest{}, err }
			return CloneAddonRequest{NamespacedID: models.NamespacedID{Namespace: ns, ID: addonID}}, nil
		},
		func(ctx context.Context, request CloneAddonRequest) (CloneAddonResponse, error) {
			a, err := h.service.CloneAddon(ctx, request)
// ...
```

<!-- archie:ai-end -->
