# handlers

<!-- archie:ai-start -->

> Structural umbrella for all v3 HTTP handler sub-packages; each child owns one API resource slice and bridges generated api/v3 request/response types to domain services via the httptransport.Handler pattern. Every handler in this tree must follow the same decode/operate/encode contract and error-encoder chain.

## Patterns

**Type-alias triad per operation** — Each operation file declares Request, Response, and optionally Params type aliases from api/v3 generated types, keeping operation files self-contained. (`type ListMetersRequest = api.ListMetersRequestObject; type ListMetersResponse = api.ListMetersResponseObject`)
**httptransport.NewHandlerWithArgs for path-param endpoints** — Endpoints that take URL path parameters (e.g. meterID, planID) must use NewHandlerWithArgs; parameter-less list endpoints use NewHandler. (`httptransport.NewHandlerWithArgs(op, decode, encode, httptransport.AppendOptions(opts, httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()))...)`)
**Namespace resolved in decoder, never in operation** — Every decoder closure must call h.resolveNamespace(ctx) (or equivalent injected resolver) to obtain the namespace string before building the domain input. (`ns, err := h.resolveNamespace(ctx); if err != nil { return req, err }`)
**apierrors.GenericErrorEncoder always last in error chain** — Every handler's AppendOptions block must include httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()) so domain errors map to correct HTTP status codes. (`httptransport.AppendOptions(opts, httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()))`)
**Domain-specific errorEncoder chained before GenericErrorEncoder** — Packages with domain-specific errors (billing, features) define a local errorEncoder() and chain it before GenericErrorEncoder so billing.NotFoundError → 404 etc. (`httptransport.WithErrorEncoder(errorEncoder(), apierrors.GenericErrorEncoder())`)
**convert.go owns all domain↔API mappings; convert.gen.go is generated** — Hand-written bidirectional conversion lives in convert.go; Goverter-driven generation targets convert.gen.go. Never edit convert.gen.go directly. (`// convert.go: func ToAPIAddon(a addon.Addon) apiv3.Addon { ... }`)
**IgnoreNonCriticalIssues=true on mutating inputs** — Create and Update decoders must set req.IgnoreNonCriticalIssues = true so plans/addons with only non-critical validation issues are not rejected. (`input.IgnoreNonCriticalIssues = true // set in decoder, not in domain service`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `api/v3/handlers/*/handler.go` | Declares the Handler interface and unexported handler struct; constructs httptransport.Handler instances and wires them to domain service fields. var _ Handler = (*handler)(nil) compile-time check is required. | Missing var _ Handler check; storing domain service under a wrong field type; omitting error encoder from any operation's AppendOptions. |
| `api/v3/handlers/*/convert.go` | Owns all bidirectional API↔domain struct transformations for the resource. Discriminated unions (price types, app types) are hand-coded here. | Adding a new union variant in TypeSpec without adding matching cases here; partial round-trips (ToAPI without FromAPI). |
| `api/v3/handlers/*/convert.gen.go` | Goverter-generated conversion code. DO NOT EDIT. Regenerate with make generate. | Any manual edit will be overwritten on next make generate. |
| `api/v3/handlers/*/error_encoder.go` | Present in packages with domain-specific error types (billing, features). Maps domain error structs to HTTP status codes before GenericErrorEncoder runs. | Packages that add a new domain error type without updating this file will silently return 500. |
| `api/v3/handlers/meters/query/` | Sub-package that builds streaming.QueryParams from MeterQueryRequest fields; shared by meters and featurecost handlers. | Never duplicate this logic inline; always call query.BuildQueryParams. |
| `api/v3/handlers/plans/convert.go` | Rich bidirectional transformer for the Plan → Phase → RateCard → Price hierarchy. Price type switch must be exhaustive with an unsupported-type guard. | Adding a new price type in TypeSpec without updating both ToAPIBillingPrice and FromAPIBillingPrice plus round-trip tests in convert_test.go. |
| `api/v3/handlers/customers/billing/handler.go` | Entry point for customer billing-override sub-resource; wired separately from core customer CRUD. | Mixing billing-override operations with core customer operations in the parent package. |
| `api/v3/handlers/apps/convert.go` | Polymorphic app mapping; each app type requires its own case in the ToAPIBillingApp switch via type assertion on app.App. | Adding a new app type without a matching case causes a runtime panic or silent nil return. |

## Anti-Patterns

- Using httptransport.NewHandler instead of NewHandlerWithArgs for any endpoint that takes a URL path parameter.
- Resolving the namespace inside the operation closure instead of the decoder closure — breaks testability and violates the decode/operate separation.
- Hand-editing any convert.gen.go file — it is always overwritten by make generate.
- Omitting apierrors.GenericErrorEncoder() from an operation's AppendOptions — domain errors will return 500 to callers.
- Adding business logic (validation, defaults, service calls) inside the encoder closure (third argument) — all domain calls belong in the operation closure.

## Decisions

- **Each resource lives in its own sub-package under api/v3/handlers/ rather than a single flat handlers package.** — Isolates generated type aliases, conversion logic, and error encoders per resource; prevents import cycles and keeps each package small and independently regenerable.
- **Goverter is used only for struct-level field mappings; discriminated unions (price types, app types, app type strings) are always hand-coded in convert.go.** — Goverter cannot express type-assertion switches or one-of union dispatch; hand-coding these makes the exhaustiveness check explicit and reviewable.
- **Domain-specific error encoders (billingprofiles, features) chain before GenericErrorEncoder rather than replacing it.** — GenericErrorEncoder handles the common model error types (NotFound, Validation, Conflict); domain encoders only need to add cases for errors GenericErrorEncoder does not know about.

## Example: Adding a new path-param v3 handler operation (e.g. GetAddon)

```
// api/v3/handlers/addons/get.go
package addons

import (
	"context"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport/apierrors"
)

type GetAddonRequest = apiv3.GetAddonRequestObject
type GetAddonResponse = apiv3.GetAddonResponseObject

func (h *handler) GetAddon(ctx context.Context, req GetAddonRequest) (GetAddonResponse, error) {
// ...
```

<!-- archie:ai-end -->
