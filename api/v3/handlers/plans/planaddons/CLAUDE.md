# planaddons

<!-- archie:ai-start -->

> HTTP handler package for the plan-addons sub-resource of the v3 Plans API. Implements CRUD operations (list, get, create, update, delete) by bridging generated api/v3 request/response types to the planaddon.Service domain interface via the httptransport.HandlerWithArgs pattern.

## Patterns

**HandlerWithArgs for path-param endpoints** — Every operation uses httptransport.NewHandlerWithArgs[Request, Response, Params] — never the bare httptransport.NewHandler — because all endpoints carry at least one path parameter (planID and/or planAddonID). (`type CreatePlanAddonHandler httptransport.HandlerWithArgs[CreatePlanAddonRequest, CreatePlanAddonResponse, CreatePlanAddonParams]`)
**Type-alias triplets per operation** — Each operation file defines exactly three type aliases (Request, Response, Params) and one Handler type alias before the factory method. Request types alias domain input structs directly (e.g. planaddon.CreatePlanAddonInput). (`type (\n\tCreatePlanAddonRequest  = planaddon.CreatePlanAddonInput\n\tCreatePlanAddonResponse = api.PlanAddon\n\tCreatePlanAddonParams   = string\n\tCreatePlanAddonHandler  httptransport.HandlerWithArgs[...]\n)`)
**Namespace always resolved in decoder** — Every decoder closure calls h.resolveNamespace(ctx) as its first domain step and returns early on error. The namespace is always set on the NamespacedModel embedded in the domain input. (`ns, err := h.resolveNamespace(ctx)\nif err != nil { return CreatePlanAddonRequest{}, err }`)
**apierrors.GenericErrorEncoder on every handler** — All options blocks pass httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()) — never a custom encoder — so domain errors map uniformly to RFC 7807 responses. (`httptransport.AppendOptions(h.options, httptransport.WithOperationName("create-plan-addon"), httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()))...`)
**ToAPIPlanAddon for domain-to-API conversion** — All operations that return a single entity call ToAPIPlanAddon(*domainResult) from convert.go. Validation issues are surfaced via ToAPIProductCatalogValidationErrors — never inlined. (`return ToAPIPlanAddon(*a)`)
**labels.ToMetadata / labels.FromMetadata for label round-tripping** — Labels on create/update are converted with labels.ToMetadata(body.Labels) in the decoder; the reverse labels.FromMetadata(a.Metadata) is called in ToAPIPlanAddon in convert.go. (`meta, err := labels.ToMetadata(body.Labels)\nif err != nil { return CreatePlanAddonRequest{}, err }`)
**Pagination via pagination.NewPage with defaults** — List operations construct the page with pagination.NewPage(number, size) defaulting to page 1 / size 20, then call page.Validate() and return apierrors.NewBadRequestError on failure. (`page := pagination.NewPage(1, 20)\nif params.Params.Page != nil { page = pagination.NewPage(lo.FromPtrOr(..., 1), lo.FromPtrOr(..., 20)) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Defines the Handler interface (all five operation methods) and the private handler struct with resolveNamespace, plan.Service, planaddon.Service, and shared httptransport options. New operations must be declared on the Handler interface here. | handler holds both plan.Service and planaddon.Service — plan.Service appears unused in the current operation files (only addonService is called); don't remove it without checking the v3 server mount. |
| `convert.go` | Single translation layer from planaddon.PlanAddon to api.PlanAddon. Also owns ToAPIProductCatalogValidationErrors for models.ValidationIssues. | ValidationErrors are retrieved via a.AsProductCatalogPlanAddon().ValidationErrors() — if that method changes signature, convert.go breaks silently because the error return is discarded. |
| `lists.go` | Handles ListPlanAddons including pagination parameter parsing and response wrapping with response.NewPagePaginationResponse. | PlanIDs filter is populated with []string{params.PlanID} — it is always scoped to a single plan; do not widen to multi-plan queries without changing the Params struct. |
| `create.go` | Parses api.CreatePlanAddonRequest body, resolves namespace and labels, calls addonService.CreatePlanAddon, returns 201. | Nil-pointer guard on create/update response (if a == nil) — this should not happen in normal flow but is kept as a safety net. |
| `update.go` | Parses api.UpsertPlanAddonRequest body; only sets req.Metadata when body.Labels is non-nil to preserve existing metadata on partial updates. | FromPlanPhase is always set as a pointer (&body.FromPlanPhase), making it always present even if the client sends an empty string — intentional upsert semantics. |

## Anti-Patterns

- Calling domain service methods directly from the response encoder closure — all service calls must happen in the operation (second) closure.
- Using httptransport.NewHandler instead of NewHandlerWithArgs when the endpoint has path parameters.
- Introducing context.Background() anywhere in decoder or operation closures — always propagate the ctx argument.
- Adding business logic (validation, enrichment) outside the decoder and operation closures — do not add logic in the handler struct methods themselves.
- Skipping apierrors.GenericErrorEncoder in the options block — omitting it prevents domain errors from mapping to correct HTTP status codes.

## Decisions

- **planaddon.Service is injected separately from plan.Service** — Plan addon operations are owned by the planaddon sub-domain; plan.Service is present for potential plan-level reads but addon mutations go through their own service to keep domain boundaries clean.
- **Request types alias domain input structs directly rather than defining new HTTP-layer structs** — Reduces the translation surface; the decoder closure maps HTTP fields onto the domain struct directly, so any domain input change is immediately visible as a compile error in the decoder.

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
