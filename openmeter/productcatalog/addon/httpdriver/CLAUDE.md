# httpdriver

<!-- archie:ai-start -->

> HTTP handler layer for the addon domain, adapting addon.Service to HTTP request/response using the generic httptransport.Handler pattern. Exposes a Handler interface consumed by the v1 Chi router.

## Patterns

**httptransport.NewHandler / NewHandlerWithArgs** — Each operation returns a typed handler constructed with httptransport.NewHandler (no path params) or NewHandlerWithArgs (with path params). Three closures: decoder, operation, encoder. (`return httptransport.NewHandlerWithArgs(decoderFn, operationFn, commonhttp.JSONResponseEncoderWithStatus[Resp](http.StatusOK), opts...)`)
**Request/Response type aliases** — Each handler declares type aliases for Request, Response, and Handler at the top of the function block, mapping to domain input/output types and api.* generated types. (`type (\n\tListAddonsRequest  = addon.ListAddonsInput\n\tListAddonsResponse = api.AddonPaginatedResponse\n\tListAddonsHandler  httptransport.HandlerWithArgs[ListAddonsRequest, ListAddonsResponse, ListAddonsParams]\n)`)
**Namespace resolved from context** — Every decoder calls h.resolveNamespace(ctx) via namespacedriver.NamespaceDecoder.GetNamespace; failure returns 500 internal server error. (`ns, err := h.resolveNamespace(ctx); if err != nil { return ..., err }`)
**ValidationErrorEncoder per resource kind** — Every handler appends httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindAddon)) so domain validation errors map to correct HTTP status codes. (`httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindAddon))`)
**Mapping in mapping.go only** — All api<->domain conversions (FromAddon, AsCreateAddonRequest, AsUpdateAddonRequest) live in mapping.go. Handler closures call these helpers, never inline-convert. (`return FromAddon(*a)`)
**Handler interface composed from sub-interfaces** — Handler interface in driver.go embeds AddonHandler which lists all operation handler methods. Compile-time assertion var _ Handler = (*handler)(nil) ensures implementation. (`var _ Handler = (*handler)(nil)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `driver.go` | Handler + AddonHandler interface definition, handler struct, New constructor, resolveNamespace helper. | New operations must be added to both the AddonHandler interface and the handler struct. resolveNamespace must not be bypassed. |
| `addon.go` | One function per HTTP operation (ListAddons, CreateAddon, UpdateAddon, DeleteAddon, GetAddon, PublishAddon, ArchiveAddon), each returning its typed handler. | IgnoreNonCriticalIssues=true is set on create/update requests — preserve this for non-breaking validation. Delete returns http.StatusNoContent with EmptyResponseEncoder. |
| `mapping.go` | api<->domain type conversion: FromAddon (domain->api), AsCreateAddonRequest / AsUpdateAddonRequest (api->domain). Currency validation happens here. | Currency.Validate() must be called after assignment. RateCards mapped via productcatalog/http package helpers (http.AsRateCards, http.FromRateCard). |

## Anti-Patterns

- Calling domain services directly in decoder functions — decoders must only parse/validate HTTP input and return a domain input type.
- Inline type conversion in handler closures instead of using mapping.go helpers.
- Omitting WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(...)) — validation errors will not map to 400.
- Adding new handler methods to handler struct without adding them to the Handler/AddonHandler interface.
- Using context.Background() inside handler closures instead of the ctx passed to the operation closure.

## Decisions

- **Handler methods return typed handler values rather than implementing http.Handler directly.** — The httptransport.Handler[Req,Resp] pattern separates decoding, operation, and encoding, making each independently testable and composable with middleware via Chain.

## Example: Add a new addon sub-resource HTTP operation

```
type (\n\tCloneAddonRequest  = addon.CloneAddonInput\n\tCloneAddonResponse = api.Addon\n\tCloneAddonHandler  httptransport.HandlerWithArgs[CloneAddonRequest, CloneAddonResponse, string]\n)\n\nfunc (h *handler) CloneAddon() CloneAddonHandler {\n\treturn httptransport.NewHandlerWithArgs(\n\t\tfunc(ctx context.Context, r *http.Request, addonID string) (CloneAddonRequest, error) {\n\t\t\tns, err := h.resolveNamespace(ctx)\n\t\t\tif err != nil { return CloneAddonRequest{}, err }\n\t\t\treturn CloneAddonRequest{NamespacedID: models.NamespacedID{Namespace: ns, ID: addonID}}, nil\n\t\t},\n\t\tfunc(ctx context.Context, request CloneAddonRequest) (CloneAddonResponse, error) {\n\t\t\ta, err := h.service.CloneAddon(ctx, request)\n\t\t\tif err != nil { return CloneAddonResponse{}, err }\n\t\t\treturn FromAddon(*a)\n\t\t},\n\t\tcommonhttp.JSONResponseEncoderWithStatus[CloneAddonResponse](http.StatusOK),\n\t\thttptransport.AppendOptions(h.options,\n\t\t\thttptransport.WithOperationName("cloneAddon"),\n\t\t\thttptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindAddon)),\n\t\t)...,\n\t)\n}
```

<!-- archie:ai-end -->
