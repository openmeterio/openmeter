# planaddons

<!-- archie:ai-start -->

> HTTP handler package for the plan-addons sub-resource of the v3 Plans API. Bridges generated api/v3 request/response types to the planaddon.Service domain interface using the httptransport.HandlerWithArgs pattern for all CRUD operations (list, get, create, update, delete).

## Patterns

**HandlerWithArgs for all endpoints** — Every operation uses httptransport.NewHandlerWithArgs[Request, Response, Params] — never bare httptransport.NewHandler — because all endpoints carry path parameters (planID and/or planAddonID). (`type CreatePlanAddonHandler httptransport.HandlerWithArgs[CreatePlanAddonRequest, CreatePlanAddonResponse, CreatePlanAddonParams]`)
**Type-alias triplets per operation file** — Each operation file defines exactly three type aliases (Request, Response, Params) and one Handler type alias before the factory method. Request types alias domain input structs directly (e.g. planaddon.CreatePlanAddonInput). (`type (
	CreatePlanAddonRequest  = planaddon.CreatePlanAddonInput
	CreatePlanAddonResponse = api.PlanAddon
	CreatePlanAddonParams   = string
	CreatePlanAddonHandler  httptransport.HandlerWithArgs[...]
)`)
**Namespace resolved first in every decoder** — Every decoder closure calls h.resolveNamespace(ctx) as its first domain step and returns early on error. The namespace is set on the NamespacedModel embedded in the domain input. (`ns, err := h.resolveNamespace(ctx)
if err != nil { return CreatePlanAddonRequest{}, err }`)
**apierrors.GenericErrorEncoder on every handler** — All options blocks pass httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()) — never a custom encoder — so domain errors map uniformly to RFC 7807 responses. (`httptransport.AppendOptions(h.options, httptransport.WithOperationName("create-plan-addon"), httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()))...`)
**ToAPIPlanAddon for all entity responses** — All operations that return a single entity call ToAPIPlanAddon(*domainResult) from convert.go. Validation issues surface via ToAPIProductCatalogValidationErrors — never inlined in operation files. (`return ToAPIPlanAddon(*a)`)
**labels.ToMetadata / labels.FromMetadata for label round-tripping** — Labels on create/update are converted with labels.ToMetadata(body.Labels) in the decoder; the reverse labels.FromMetadata(a.Metadata) is called inside ToAPIPlanAddon in convert.go. (`meta, err := labels.ToMetadata(body.Labels)
if err != nil { return CreatePlanAddonRequest{}, err }`)
**Nil-pointer guard on mutating operation responses** — Create and Update operations guard against a nil service response with an explicit error rather than dereferencing directly, even though nil should not occur in normal flow. (`if a == nil { return CreatePlanAddonResponse{}, fmt.Errorf("failed to create plan addon") }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Defines the Handler interface (all five operation methods) and the private handler struct with resolveNamespace, plan.Service, planaddon.Service, and shared httptransport options. New operations must be declared on the Handler interface here. | handler holds both plan.Service (field: service) and planaddon.Service (field: addonService) — plan.Service is currently unused in operation files; do not remove it without checking the v3 server mount. All mutation calls go through addonService. |
| `convert.go` | Single translation layer from planaddon.PlanAddon to api.PlanAddon. Also owns ToAPIProductCatalogValidationErrors for models.ValidationIssues. | ValidationErrors are retrieved via a.AsProductCatalogPlanAddon().ValidationErrors() — the error return is discarded. If that method changes signature, convert.go breaks silently. |
| `lists.go` | Handles ListPlanAddons including pagination parameter parsing (default page 1 / size 20) and response wrapping with response.NewPagePaginationResponse. | PlanIDs filter is always []string{params.PlanID} — scoped to one plan. Do not widen to multi-plan queries without changing the Params struct and domain input. |
| `update.go` | Parses api.UpsertPlanAddonRequest body; only sets req.Metadata when body.Labels is non-nil to preserve existing metadata on partial updates. | FromPlanPhase is always set as a pointer (&body.FromPlanPhase), making it present even for empty-string — intentional upsert semantics. Do not change to conditional nil-guard without checking domain behavior. |
| `create.go` | Parses api.CreatePlanAddonRequest body, resolves namespace and labels, calls addonService.CreatePlanAddon, returns HTTP 201. | Labels conversion via labels.ToMetadata must happen before constructing CreatePlanAddonRequest — error from ToMetadata must be returned from the decoder, not silently ignored. |

## Anti-Patterns

- Using httptransport.NewHandler instead of NewHandlerWithArgs when the endpoint has path parameters.
- Calling domain service methods from the response encoder closure — all service calls must happen in the operation (second) closure.
- Introducing context.Background() in decoder or operation closures — always propagate the ctx argument.
- Adding business logic (validation, enrichment) in handler struct methods instead of inside decoder and operation closures.
- Omitting apierrors.GenericErrorEncoder from the options block — domain errors will not map to correct HTTP status codes.

## Decisions

- **planaddon.Service is injected separately from plan.Service** — Plan addon operations are owned by the planaddon sub-domain; plan.Service is present for potential plan-level reads but addon mutations go through their own service to keep domain boundaries clean.
- **Request types alias domain input structs directly rather than defining new HTTP-layer structs** — Reduces the translation surface — the decoder maps HTTP fields onto the domain struct directly, so any domain input change is immediately visible as a compile error in the decoder closure.
- **Pagination defaults to page 1 / size 20 and validates before issuing domain call** — Centralises page-validation error as a 400 apierrors.NewBadRequestError before the domain layer is invoked, matching the pattern used across all v3 list handlers.

## Example: Adding a new operation (e.g. ClonePlanAddon) following the established pattern

```
// clone.go
package planaddons

import (
	"context"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)
// ...
```

<!-- archie:ai-end -->
