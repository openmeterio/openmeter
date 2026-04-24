# httpdriver

<!-- archie:ai-start -->

> HTTP handler layer for plan-addon assignment CRUD endpoints, bridging the v1 REST API (api.PlanAddon*) to planaddon.Service via the httptransport.HandlerWithArgs generic pattern.

## Patterns

**httptransport.NewHandlerWithArgs for every endpoint** — Each handler method returns a typed HandlerWithArgs[Request, Response, Params] constructed inline with a decoder closure, an operation closure, and a ResponseEncoder. Never implement ServeHTTP directly. (`return httptransport.NewHandlerWithArgs(decoderFn, operationFn, commonhttp.JSONResponseEncoderWithStatus[T](http.StatusOK), ...options)`)
**Namespace resolved via h.resolveNamespace(ctx)** — All decoders call h.resolveNamespace(ctx) which delegates to h.namespaceDecoder.GetNamespace(ctx). Never read namespace from URL params. (`ns, err := h.resolveNamespace(ctx); if err != nil { return ..., err }`)
**ValidationErrorEncoder per operation** — Every operation appends httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindAddon)) so domain validation errors are encoded as Problem Details. (`httptransport.AppendOptions(h.options, httptransport.WithOperationName("createPlanAddon"), httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindAddon)))...`)
**Request/Response type aliases in planaddon.go** — Each handler defines local type aliases (e.g. CreatePlanAddonRequest = planaddon.CreatePlanAddonInput) next to the handler type declaration for self-documentation and to avoid import shadowing. (`type (CreatePlanAddonRequest = planaddon.CreatePlanAddonInput; CreatePlanAddonResponse = api.PlanAddon; CreatePlanAddonHandler httptransport.HandlerWithArgs[...])`)
**Handler interface embeds sub-interfaces** — Handler interface embeds PlanAddonHandler, which declares one factory method per endpoint (ListPlanAddons(), CreatePlanAddon(), etc.) returning typed handler values. (`type Handler interface { PlanAddonHandler }; type PlanAddonHandler interface { ListPlanAddons() ListPlanAddonsHandler; ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `driver.go` | Defines Handler and PlanAddonHandler interfaces, handler struct, New constructor, and resolveNamespace helper. | All new endpoint methods must appear in both the PlanAddonHandler interface and the handler struct for the var _ Handler = (*handler)(nil) compile check to pass. |
| `planaddon.go` | One function per HTTP operation; each builds and returns a HandlerWithArgs using inline decoder/operation closures. | Status codes must match API spec: 200 for list/get/update, 201 for create, 204 (EmptyResponseEncoder) for delete. |
| `mapping.go` | Converts between api.* types and domain input/output types. FromPlanAddon is the primary domain→API mapper; AsCreatePlanAddonRequest/AsUpdatePlanAddonRequest are the API→domain mappers. | FromPlanAddon calls a.AsProductCatalogPlanAddon().ValidationErrors() — validation issues are always propagated to the response, never swallowed. |

## Anti-Patterns

- Implementing net/http.Handler directly on handler methods — always use httptransport.NewHandlerWithArgs.
- Resolving namespace from URL path parameters instead of h.resolveNamespace(ctx).
- Omitting WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(...)) from an operation's options — validation errors will not be serialized correctly.
- Adding business logic (plan/addon status checks, conflict detection) in the decoder or mapping layer — keep decoders to parsing only.
- Returning raw domain errors (planaddon.NotFoundError) without letting the error encoder chain handle them.

## Decisions

- **httptransport.HandlerWithArgs generic pattern for all operations** — Separates HTTP concerns (decode, encode, error handling) from business logic and enables consistent middleware chaining across all endpoints.
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
			return MyNewRequest{
// ...
```

<!-- archie:ai-end -->
