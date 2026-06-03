# httpdriver

<!-- archie:ai-start -->

> HTTP handler layer for plan-addon assignment CRUD endpoints, bridging the v1 REST API (api.PlanAddon*) to planaddon.Service via httptransport.HandlerWithArgs. Owns request decoding, response mapping, and error encoding only — no business logic.

## Patterns

**httptransport.NewHandlerWithArgs per endpoint** — Each handler method returns a typed HandlerWithArgs[Request, Response, Params] built inline with decoder, operation, and ResponseEncoder closures. Never implement ServeHTTP directly. (`return httptransport.NewHandlerWithArgs(decoderFn, operationFn, commonhttp.JSONResponseEncoderWithStatus[T](http.StatusOK), ...options)`)
**Namespace via h.resolveNamespace(ctx)** — All decoders call h.resolveNamespace(ctx) delegating to h.namespaceDecoder.GetNamespace(ctx). Never read namespace from URL params or headers directly. (`ns, err := h.resolveNamespace(ctx); if err != nil { return ..., err }`)
**ValidationErrorEncoder per operation** — Every operation appends httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindAddon)) so domain validation errors render as RFC 7807. (`httptransport.AppendOptions(h.options, httptransport.WithOperationName("createPlanAddon"), httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindAddon)))...`)
**Local type aliases in planaddon.go** — Each endpoint declares local type aliases (e.g. CreatePlanAddonRequest = planaddon.CreatePlanAddonInput) next to its handler type for self-documentation. (`type (CreatePlanAddonRequest = planaddon.CreatePlanAddonInput; CreatePlanAddonResponse = api.PlanAddon; CreatePlanAddonHandler httptransport.HandlerWithArgs[...])`)
**Handler interface embeds sub-interfaces** — Handler embeds PlanAddonHandler, which declares one factory method per endpoint. var _ Handler = (*handler)(nil) enforces compile-time satisfaction. (`type Handler interface { PlanAddonHandler }; type PlanAddonHandler interface { ListPlanAddons() ListPlanAddonsHandler; ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `driver.go` | Handler and PlanAddonHandler interfaces, handler struct, New constructor, resolveNamespace helper. | New endpoints must appear in both PlanAddonHandler and the struct for var _ Handler = (*handler)(nil). resolveNamespace returns 500 (not 400) on missing namespace. |
| `planaddon.go` | One function per HTTP operation, each returning a HandlerWithArgs with inline decoder/operation closures. | Status codes must match the spec: 200 list/get/update, 201 create, 204 (EmptyResponseEncoder) delete. No business logic here. |
| `mapping.go` | FromPlanAddon converts domain to api.PlanAddon; AsCreatePlanAddonRequest/AsUpdatePlanAddonRequest convert API bodies to domain inputs. | FromPlanAddon calls a.AsProductCatalogPlanAddon().ValidationErrors() — propagate validation issues into the response ValidationErrors field, never swallow. |

## Anti-Patterns

- Implementing net/http.Handler directly — always use httptransport.NewHandlerWithArgs.
- Resolving namespace from URL path params instead of h.resolveNamespace(ctx).
- Omitting WithErrorEncoder(ValidationErrorEncoder(...)) — validation errors will not serialize correctly.
- Adding business logic (status checks, conflict detection) in decoder or mapping — decoders parse only.
- Returning raw domain errors without letting the error encoder chain handle them.

## Decisions

- **httptransport.HandlerWithArgs for all operations.** — Separates HTTP concerns from logic and enables consistent middleware (OTel, error encoding) without boilerplate.
- **Namespace injected via NamespaceDecoder, not path/query param.** — Self-hosted uses a static default namespace; the decoder abstraction supports both static and multi-tenant resolvers without changing handlers.

## Example: Add a new read-only endpoint following the httpdriver pattern

```
type (MyNewRequest = planaddon.MyNewInput; MyNewResponse = api.PlanAddon; MyNewParams struct{ PlanIDOrKey string }; MyNewHandler httptransport.HandlerWithArgs[MyNewRequest, MyNewResponse, MyNewParams])

func (h *handler) MyNew() MyNewHandler {
  return httptransport.NewHandlerWithArgs(
    func(ctx context.Context, r *http.Request, params MyNewParams) (MyNewRequest, error) {
      ns, err := h.resolveNamespace(ctx)
      if err != nil { return MyNewRequest{}, err }
      return MyNewRequest{NamespacedModel: models.NamespacedModel{Namespace: ns}, PlanIDOrKey: params.PlanIDOrKey}, nil
    }, /* op */, /* enc */)
}
```

<!-- archie:ai-end -->
