# handlers

<!-- archie:ai-start -->

> Structural umbrella for all v3 HTTP handler sub-packages; each child owns one API resource slice and bridges generated api/v3 request/response types to domain services via the httptransport.Handler[Request,Response] decode/operate/encode pipeline. Every handler in this tree must follow the same namespace-resolution, error-encoder, and convert.go conventions.

## Patterns

**Type-alias triad per operation file** — Each operation file declares Request, Response, and optionally Params type aliases from api/v3 generated types, keeping the file self-contained and searchable. (`type ListMetersRequest = api.ListMetersRequestObject; type ListMetersResponse = api.ListMetersResponseObject`)
**NewHandlerWithArgs for path-param endpoints; NewHandler for no-param** — Endpoints that take URL path parameters (meterID, planID, addonID) must use httptransport.NewHandlerWithArgs; parameter-less list endpoints use httptransport.NewHandler. (`httptransport.NewHandlerWithArgs(op, decode, encode, httptransport.AppendOptions(opts, httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()))...)`)
**Namespace resolved in decoder closure, never in operation closure** — Every decoder closure must call h.resolveNamespace(ctx) (or the injected namespaceDecoder.GetNamespace) before building the domain input. Namespace must never be extracted in the operation or encoder closures. (`ns, err := h.resolveNamespace(ctx); if err != nil { return req, err }`)
**apierrors.GenericErrorEncoder always last in error chain** — Every handler's AppendOptions block must include httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()) so domain model errors map to correct HTTP status codes. Domain-specific encoders chain before it, not instead of it. (`httptransport.AppendOptions(opts, httptransport.WithErrorEncoder(errorEncoder(), apierrors.GenericErrorEncoder()))`)
**convert.go owns all domain<->API mappings; convert.gen.go is Goverter-generated** — Hand-written bidirectional conversion lives in convert.go. Goverter-driven generation targets convert.gen.go. Discriminated unions (price types, app types) are always hand-coded. Never edit convert.gen.go directly. (`// convert.go: func ToAPIAddon(a addon.Addon) apiv3.Addon { ... }`)
**IgnoreNonCriticalIssues=true on mutating inputs** — Create and Update decoders must set req.IgnoreNonCriticalIssues = true so resources with only non-critical validation issues are not rejected. This flag is set in the decoder, not in the domain service. (`input.IgnoreNonCriticalIssues = true // set in decoder closure`)
**var _ Handler = (*handler)(nil) compile-time check in handler.go** — Every handler.go must include a blank-identifier compile-time assertion that the unexported handler struct satisfies the exported Handler interface. (`var _ Handler = (*handler)(nil)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `api/v3/handlers/*/handler.go` | Declares the public Handler interface and unexported handler struct; constructs httptransport.Handler instances and wires them to injected domain service fields. | Missing var _ Handler compile-time check; omitting apierrors.GenericErrorEncoder from any operation's AppendOptions; storing a domain service under the wrong interface type. |
| `api/v3/handlers/*/convert.go` | Owns all bidirectional API<->domain struct transformations for the resource including discriminated unions (price types, app types, source enums). | Adding a new union variant in TypeSpec without adding matching cases; partial round-trips (ToAPI without FromAPI); mixing conversion logic into operation files. |
| `api/v3/handlers/*/convert.gen.go` | Goverter-generated conversion code. DO NOT EDIT. Regenerate with make generate after modifying goverter annotations in convert.go. | Any manual edit will be silently overwritten on next make generate. |
| `api/v3/handlers/*/error_encoder.go` | Present in packages with domain-specific error types (billingprofiles, features). Maps domain error structs to HTTP status codes before GenericErrorEncoder runs. | Packages that add a new domain error type without updating this file will silently return 500. |
| `api/v3/handlers/meters/query/` | Sub-package that builds streaming.QueryParams from MeterQueryRequest fields; shared by meters and featurecost handlers. | Never duplicate this logic inline; always call query.BuildQueryParams. |
| `api/v3/handlers/plans/convert.go` | Rich bidirectional transformer for the Plan > Phase > RateCard > Price hierarchy; price type switch must be exhaustive with an unsupported-type guard. | Adding a new price type in TypeSpec without updating both ToAPIBillingPrice and FromAPIBillingPrice plus convert_test.go round-trip tests. |
| `api/v3/handlers/apps/convert.go` | Polymorphic app mapping; each app type requires its own case in the ToAPIBillingApp switch via type assertion on app.App. | Adding a new app type without a matching case causes a runtime panic or silent nil return. |
| `api/v3/handlers/customers/billing/handler.go` | Entry point for customer billing-override sub-resource; wired separately from core customer CRUD. | Mixing billing-override operations with core customer operations in the parent package. |

## Anti-Patterns

- Using httptransport.NewHandler instead of NewHandlerWithArgs for any endpoint that takes a URL path parameter.
- Resolving the namespace inside the operation closure instead of the decoder closure — breaks decode/operate separation and testability.
- Hand-editing any convert.gen.go file — it is always overwritten by make generate.
- Omitting apierrors.GenericErrorEncoder() from an operation's AppendOptions — domain errors will return 500 to callers.
- Adding business logic (validation, defaults, service calls) inside the encoder closure (third argument) — all domain calls belong in the operation closure.

## Decisions

- **Each resource lives in its own sub-package under api/v3/handlers/ rather than a single flat handlers package.** — Isolates generated type aliases, conversion logic, and error encoders per resource; prevents import cycles and keeps each package small and independently regenerable.
- **Goverter is used only for struct-level field mappings; discriminated unions (price types, app types) are always hand-coded in convert.go.** — Goverter cannot express type-assertion switches or one-of union dispatch; hand-coding these makes the exhaustiveness check explicit and reviewable.
- **Domain-specific error encoders (billingprofiles, features) chain before GenericErrorEncoder rather than replacing it.** — GenericErrorEncoder handles common model error types (NotFound, Validation, Conflict); domain encoders only need to add cases for errors GenericErrorEncoder does not know about.

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

func (h *handler) GetAddon() httptransport.Handler[GetAddonRequest, GetAddonResponse] {
// ...
```

<!-- archie:ai-end -->
