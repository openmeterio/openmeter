# httpdriver

<!-- archie:ai-start -->

> HTTP transport layer for plan-addon assignment endpoints (list/create/get/update/delete). Adapts api.* request/response types to the planaddon.Service and back.

## Patterns

**httptransport.NewHandlerWithArgs triple** — Each endpoint returns a typed Handler built from (decode→request, service call, response encoder) plus options with WithOperationName and WithErrorEncoder. (`return httptransport.NewHandlerWithArgs(decodeFn, businessFn, commonhttp.JSONResponseEncoderWithStatus[...](http.StatusOK), httptransport.AppendOptions(...))`)
**Request type aliases to service inputs** — Request types are aliases of the service input structs (e.g. CreatePlanAddonRequest = planaddon.CreatePlanAddonInput), avoiding a duplicate struct. (`type ( CreatePlanAddonRequest = planaddon.CreatePlanAddonInput; ... )`)
**Namespace resolution per request** — Decode funcs call h.resolveNamespace(ctx) (via namespacedriver.NamespaceDecoder) and 500 if absent; never trust a body-supplied namespace. (`ns, err := h.resolveNamespace(ctx); req.Namespace = ns`)
**As*/From* mapping functions** — Body→input uses AsCreatePlanAddonRequest/AsUpdatePlanAddonRequest; domain→api uses FromPlanAddon, which also surfaces ValidationErrors via AsProductCatalogPlanAddon().ValidationErrors(). (`resp := api.PlanAddon{ ..., ValidationErrors: http.FromValidationErrors(validationIssues) }`)
**ValidationErrorEncoder per resource kind** — All handlers attach httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindAddon)) so validation issues render as structured HTTP errors. (`httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindAddon))`)
**Plan/addon identity via path args** — Handlers carry PlanID/AddonID or *IDOrKey through the HandlerWithArgs params struct, not the body; List packs PlanIDOrKey into both PlanIDs and PlanKeys. (`PlanIDs: []string{params.PlanIDOrKey}, PlanKeys: []string{params.PlanIDOrKey}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `driver.go` | Handler/PlanAddonHandler interfaces, the handler struct (service+namespaceDecoder+options), New(), and resolveNamespace. | New takes namespaceDecoder first, then service; resolveNamespace returns 500 (not 401) when namespace missing. |
| `planaddon.go` | The five endpoint constructors (ListPlanAddons/CreatePlanAddon/UpdatePlanAddon/DeletePlanAddon/GetPlanAddon) with request/response type aliases and DefaultPageSize/DefaultPageNumber. | Create uses StatusCreated, Delete uses EmptyResponseEncoder+StatusNoContent; List maps each item through FromPlanAddon and aborts on first cast error. |
| `mapping.go` | FromPlanAddon (domain→api), AsCreatePlanAddonRequest/AsUpdatePlanAddonRequest (api→input), AsMetadata helper. | Update maps a.FromPlanPhase to a pointer (&a.FromPlanPhase) while Create passes it by value — the nil-vs-set semantics matter downstream in the service/adapter merge. |

## Anti-Patterns

- Reading namespace from the request body instead of resolveNamespace(ctx).
- Adding an endpoint without WithOperationName and the ValidationErrorEncoder option.
- Defining a new Request struct instead of aliasing the planaddon.* service input.
- Encoding domain objects directly instead of mapping through FromPlanAddon (loses ValidationErrors/Metadata/Annotations translation).

## Decisions

- **Request types alias service inputs and the handler delegates straight to planaddon.Service.** — Keeps the HTTP layer a thin transport adapter with no business logic, all validation living in service/adapter.
- **FromPlanAddon attaches AsProductCatalogPlanAddon().ValidationErrors() to the API payload.** — Plan-addon compatibility issues are returned as non-fatal validation errors on GET/LIST rather than failing the request.

<!-- archie:ai-end -->
