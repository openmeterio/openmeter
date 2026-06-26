# httpdriver

<!-- archie:ai-start -->

> HTTP transport layer for add-on endpoints, translating api.* request/response types to addon.Service calls via the httptransport framework. The only place add-on REST shape is mapped to/from domain.

## Patterns

**httptransport handler triple (decode, business, encode)** — Each endpoint method returns a typed handler built with httptransport.NewHandler / NewHandlerWithArgs taking a request decoder, a service-calling func, and a response encoder. Request/Response/Handler types are declared as type aliases in a `type (...)` block. (`func (h *handler) CreateAddon() CreateAddonHandler { return httptransport.NewHandler(decode, func(ctx, req) { h.service.CreateAddon(ctx, req) }, encoder, opts...) }`)
**Namespace resolution before building request** — Every decode func calls h.resolveNamespace(ctx) first (wrapping namespaceDecoder.GetNamespace) and stamps NamespacedModel/NamespacedID onto the request. (`ns, err := h.resolveNamespace(ctx); req.NamespacedID = models.NamespacedID{Namespace: ns, ID: addonID}`)
**Mapping via FromAddon / AsCreateAddonRequest / AsUpdateAddonRequest** — Domain→API uses FromAddon (which also emits ValidationErrors and switches on a.Status()); API→domain uses AsCreateAddonRequest/AsUpdateAddonRequest delegating ratecard conversion to productcatalog/http AsRateCards. (`return FromAddon(*a)`)
**Shared productcatalog ValidationErrorEncoder per operation** — Every handler appends httptransport.WithOperationName(...) and WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindAddon)) to h.options. (`httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindAddon))`)
**Handler interface composition + private struct** — driver.go exposes a Handler interface embedding AddonHandler (one method per endpoint); the concrete *handler{service, namespaceDecoder, options} is private with var _ Handler = (*handler)(nil) and a New(...) constructor. (`type Handler interface { AddonHandler }`)
**IgnoreNonCriticalIssues on create/update** — Create/Update decoders set req.IgnoreNonCriticalIssues = true so non-critical validation issues surface as response ValidationErrors rather than hard failures. (`req.IgnoreNonCriticalIssues = true`)
**Publish/Archive synthesize EffectivePeriod from clock.Now()** — PublishAddon sets EffectiveFrom = clock.Now(); ArchiveAddon sets EffectiveTo = clock.Now() (API spec does not yet carry these — see TODO(chrisgacsal)). (`EffectivePeriod: productcatalog.EffectivePeriod{EffectiveFrom: lo.ToPtr(clock.Now())}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `addon.go` | All seven endpoint handlers: ListAddons, CreateAddon, UpdateAddon, DeleteAddon, GetAddon, PublishAddon, ArchiveAddon. | GetAddon uses ref.ParseIDOrKey to split ULID vs key; list filters map api slices into filter.FilterString/FilterULID with In set. Status param maps api.AddonStatus -> productcatalog.AddonStatus. |
| `driver.go` | Handler/AddonHandler interfaces, *handler struct, resolveNamespace, and New constructor. | resolveNamespace returns 500 (commonhttp.NewHTTPError) when namespace missing; New takes namespaceDecoder, addon.Service, options. |
| `mapping.go` | FromAddon (domain->api, sets Status + ValidationErrors), AsCreateAddonRequest/AsUpdateAddonRequest (api->domain). | FromAddon must map every productcatalog.AddonStatus or returns an error on default; currency validated via currency.Code(a.Currency).Validate(). |

## Anti-Patterns

- Building a request without first calling h.resolveNamespace and stamping the namespace.
- Calling adapter/Ent directly instead of going through h.service (addon.Service).
- Hand-rolling JSON encoding instead of commonhttp.JSONResponseEncoderWithStatus / EmptyResponseEncoder.
- Omitting the productcataloghttp.ValidationErrorEncoder error encoder on a new endpoint.
- Mapping ratecards locally instead of reusing productcatalog/http AsRateCards/FromRateCard.

## Decisions

- **Request/Response types are aliases of addon.*Input / api.* rather than new structs.** — Keeps the transport layer a thin adapter; the service owns the canonical input shape.
- **EffectivePeriod for publish/archive is set server-side from clock.Now().** — TypeSpec API does not yet expose effective dates (documented TODOs); clock injection keeps it testable.

## Example: Endpoint handler with namespace resolution and validation error encoder

```
func (h *handler) CreateAddon() CreateAddonHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreateAddonRequest, error) {
			body := api.AddonCreate{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil { return CreateAddonRequest{}, err }
			ns, err := h.resolveNamespace(ctx)
			if err != nil { return CreateAddonRequest{}, err }
			req, err := AsCreateAddonRequest(body, ns)
			if err != nil { return CreateAddonRequest{}, err }
			req.IgnoreNonCriticalIssues = true
			return req, nil
		},
		func(ctx context.Context, request CreateAddonRequest) (CreateAddonResponse, error) {
			a, err := h.service.CreateAddon(ctx, request)
			if err != nil { return CreateAddonResponse{}, err }
// ...
```

<!-- archie:ai-end -->
