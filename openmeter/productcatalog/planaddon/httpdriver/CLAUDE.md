# httpdriver

<!-- archie:ai-start -->

> HTTP handler layer for plan-addon assignment CRUD endpoints, bridging the v1 REST API (api.PlanAddon*) to planaddon.Service via the httptransport.HandlerWithArgs generic pattern. Owns request decoding, response mapping, and error encoding only — no business logic.

## Patterns

**httptransport.NewHandlerWithArgs for every endpoint** — Each handler method returns a typed HandlerWithArgs[Request, Response, Params] constructed inline with a decoder closure, an operation closure, and a ResponseEncoder. Never implement ServeHTTP directly. (`return httptransport.NewHandlerWithArgs(decoderFn, operationFn, commonhttp.JSONResponseEncoderWithStatus[T](http.StatusOK), ...options)`)
**Namespace resolved via h.resolveNamespace(ctx)** — All decoders call h.resolveNamespace(ctx) which delegates to h.namespaceDecoder.GetNamespace(ctx). Never read namespace from URL params or request headers directly. (`ns, err := h.resolveNamespace(ctx); if err != nil { return ..., err }`)
**ValidationErrorEncoder per operation** — Every operation appends httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindAddon)) so domain validation errors are encoded as RFC 7807 Problem Details. (`httptransport.AppendOptions(h.options, httptransport.WithOperationName("createPlanAddon"), httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindAddon)))...`)
**Local type aliases in planaddon.go** — Each endpoint declares local type aliases (e.g. CreatePlanAddonRequest = planaddon.CreatePlanAddonInput) next to the handler type declaration for self-documentation. (`type (CreatePlanAddonRequest = planaddon.CreatePlanAddonInput; CreatePlanAddonResponse = api.PlanAddon; CreatePlanAddonHandler httptransport.HandlerWithArgs[...])`)
**Handler interface embeds sub-interfaces** — Handler interface embeds PlanAddonHandler, which declares one factory method per endpoint returning typed handler values. var _ Handler = (*handler)(nil) enforces compile-time interface satisfaction. (`type Handler interface { PlanAddonHandler }; type PlanAddonHandler interface { ListPlanAddons() ListPlanAddonsHandler; ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `driver.go` | Defines Handler and PlanAddonHandler interfaces, handler struct, New constructor, and resolveNamespace helper. | All new endpoint methods must appear in both the PlanAddonHandler interface and the handler struct for var _ Handler = (*handler)(nil) to pass. resolveNamespace returns 500 (not 400) on missing namespace. |
| `planaddon.go` | One function per HTTP operation; each builds and returns a HandlerWithArgs using inline decoder/operation closures. | Status codes must match API spec: 200 for list/get/update, 201 for create, 204 (EmptyResponseEncoder) for delete. Do not add business logic here. |
| `mapping.go` | FromPlanAddon converts domain planaddon.PlanAddon to api.PlanAddon. AsCreatePlanAddonRequest/AsUpdatePlanAddonRequest convert API body types to domain input types. | FromPlanAddon calls a.AsProductCatalogPlanAddon().ValidationErrors() — validation issues must always be propagated to the response ValidationErrors field, never swallowed. |

## Anti-Patterns

- Implementing net/http.Handler directly on handler methods — always use httptransport.NewHandlerWithArgs.
- Resolving namespace from URL path parameters instead of h.resolveNamespace(ctx).
- Omitting WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(...)) from an operation's options — validation errors will not serialize correctly.
- Adding business logic (plan/addon status checks, conflict detection) in decoder or mapping layer — decoders parse only.
- Returning raw domain errors without letting the error encoder chain handle them.

## Decisions

- **httptransport.HandlerWithArgs generic pattern for all operations** — Separates HTTP concerns (decode, encode, error handling) from business logic and enables consistent middleware chaining (OTel, error encoding) across all endpoints without boilerplate.
- **Namespace injected via NamespaceDecoder rather than path/query param** — Self-hosted deployments use a static default namespace; the decoder abstraction allows both self-hosted (static) and multi-tenant (header/JWT) resolvers without changing handler code.

## Example: Add a new read-only endpoint following the httpdriver pattern

```
type (
	MyNewRequest  = planaddon.MyNewInput
	MyNewResponse = api.PlanAddon
	MyNewParams   struct{ PlanIDOrKey string }
	MyNewHandler  httptransport.HandlerWithArgs[MyNewRequest, MyNewResponse, MyNewParams]
)

func (h *handler) MyNew() MyNewHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params MyNewParams) (MyNewRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return MyNewRequest{}, err
			}
			return MyNewRequest{NamespacedModel: models.NamespacedModel{Namespace: ns}, PlanIDOrKey: params.PlanIDOrKey}, nil
// ...
```

<!-- archie:ai-end -->
