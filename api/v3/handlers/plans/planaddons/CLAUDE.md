# planaddons

<!-- archie:ai-start -->

> v3 HTTP handler package for the plan-addons sub-resource of the Plans API. Bridges generated api/v3 request/response types to planaddon.Service using the httptransport.HandlerWithArgs pattern for all CRUD operations.

## Patterns

**HandlerWithArgs for all endpoints** — Every operation uses httptransport.NewHandlerWithArgs (never bare NewHandler) because all endpoints carry path parameters (planID and/or planAddonID). (`CreatePlanAddonHandler httptransport.HandlerWithArgs[CreatePlanAddonRequest, CreatePlanAddonResponse, CreatePlanAddonParams]`)
**Type-alias triplet per operation file** — Each file defines Request (aliasing the domain input struct, e.g. planaddon.CreatePlanAddonInput), Response (api.PlanAddon), Params, and a Handler alias before the factory method. (`type ( CreatePlanAddonRequest = planaddon.CreatePlanAddonInput; CreatePlanAddonResponse = api.PlanAddon; CreatePlanAddonParams = string )`)
**Namespace resolved first in every decoder** — Every decoder calls h.resolveNamespace(ctx) as its first domain step, returns early on error, and sets it on the NamespacedModel of the domain input. (`ns, err := h.resolveNamespace(ctx); if err != nil { return CreatePlanAddonRequest{}, err }`)
**apierrors.GenericErrorEncoder on every handler** — Every options block passes httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()) plus WithOperationName — never a custom encoder — so domain errors map uniformly to RFC 7807. (`httptransport.AppendOptions(h.options, httptransport.WithOperationName("create-plan-addon"), httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()))`)
**ToAPIPlanAddon for all entity responses** — Single-entity operations return ToAPIPlanAddon(*result) from convert.go; validation issues surface via ToAPIProductCatalogValidationErrors — never inlined in operation files. (`return ToAPIPlanAddon(*a)`)
**labels.ToMetadata / labels.FromMetadata round-tripping** — Inbound labels convert via labels.ToMetadata(body.Labels) in the decoder; the reverse labels.FromMetadata(a.Metadata) runs inside ToAPIPlanAddon. ToMetadata errors must be returned from the decoder. (`meta, err := labels.ToMetadata(body.Labels); if err != nil { return CreatePlanAddonRequest{}, err }`)
**Nil-guard on mutating responses** — Create and Update guard against a nil service result with an explicit error rather than dereferencing. (`if a == nil { return CreatePlanAddonResponse{}, fmt.Errorf("failed to create plan addon") }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler interface (5 methods) and handler struct with resolveNamespace, plan.Service (service), planaddon.Service (addonService), options. | plan.Service is currently unused in operation files; all mutations go through addonService. Do not remove service without checking the v3 server mount. |
| `convert.go` | ToAPIPlanAddon translation and ToAPIProductCatalogValidationErrors for models.ValidationIssues. | ValidationErrors come from a.AsProductCatalogPlanAddon().ValidationErrors() with the error discarded — a signature change there breaks convert.go silently. |
| `lists.go` | ListPlanAddons with pagination (default page 1 / size 20) and response.NewPagePaginationResponse wrapping. | PlanIDs filter is always []string{params.PlanID} (single plan). Do not widen without changing the Params struct and domain input. |
| `update.go` | Parses api.UpsertPlanAddonRequest; only sets req.Metadata when body.Labels is non-nil to preserve metadata on partial updates. | FromPlanPhase is always set as a pointer (&body.FromPlanPhase) — present even for empty string; intentional upsert semantics. |
| `create.go` | Parses api.CreatePlanAddonRequest, resolves namespace + labels, calls CreatePlanAddon, returns HTTP 201. | labels.ToMetadata must run before building the request; its error must be returned from the decoder, not ignored. |
| `get.go / delete.go` | HandlerWithArgs keyed on a struct {PlanID, PlanAddonID}; delete returns 204 via EmptyResponseEncoder. | Get builds GetPlanAddonInput from PlanAddonID only; delete passes both PlanID and ID. |

## Anti-Patterns

- Using httptransport.NewHandler instead of NewHandlerWithArgs when the endpoint has path parameters.
- Calling domain service methods from the response encoder closure — all service calls go in the operation closure.
- Introducing context.Background() in decoder or operation closures — always propagate the ctx argument.
- Adding business logic (validation, enrichment) in handler struct methods instead of decoder/operation closures.
- Omitting apierrors.GenericErrorEncoder from the options block.

## Decisions

- **planaddon.Service is injected separately from plan.Service.** — Plan-addon operations are owned by the planaddon sub-domain; addon mutations go through their own service to keep domain boundaries clean.
- **Request types alias domain input structs directly.** — Reduces the translation surface — any domain input change surfaces immediately as a compile error in the decoder closure.
- **Pagination defaults to page 1 / size 20 and validates before the domain call.** — Centralizes page-validation as a 400 before invoking the domain layer, matching all v3 list handlers.

## Example: A create operation with namespace + label conversion and HTTP 201

```
func (h *handler) CreatePlanAddon() CreatePlanAddonHandler {
  return httptransport.NewHandlerWithArgs(
    func(ctx context.Context, r *http.Request, planID CreatePlanAddonParams) (CreatePlanAddonRequest, error) {
      var body api.CreatePlanAddonRequest
      if err := request.ParseBody(r, &body); err != nil { return CreatePlanAddonRequest{}, err }
      ns, err := h.resolveNamespace(ctx); if err != nil { return CreatePlanAddonRequest{}, err }
      meta, err := labels.ToMetadata(body.Labels); if err != nil { return CreatePlanAddonRequest{}, err }
      return CreatePlanAddonRequest{NamespacedModel: models.NamespacedModel{Namespace: ns}, PlanID: planID, AddonID: body.Addon.Id, Metadata: meta}, nil
    },
    func(ctx context.Context, req CreatePlanAddonRequest) (CreatePlanAddonResponse, error) {
      a, err := h.addonService.CreatePlanAddon(ctx, req); if err != nil { return CreatePlanAddonResponse{}, err }
      if a == nil { return CreatePlanAddonResponse{}, fmt.Errorf("failed to create plan addon") }
      return ToAPIPlanAddon(*a)
    },
    commonhttp.JSONResponseEncoderWithStatus[CreatePlanAddonResponse](http.StatusCreated),
// ...
```

<!-- archie:ai-end -->
