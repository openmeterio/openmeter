# apierrors

<!-- archie:ai-start -->

> RFC 7807-style error type system for all v3 API responses: a single BaseAPIError struct, typed constructors for every HTTP status, and a GenericErrorEncoder mapping domain errors to HTTP responses. Every v3 error response must flow through this package.

## Patterns

**BaseAPIError via named constructors only** — All v3 error responses are produced by a named constructor in errors_ctors.go (NewInternalError, NewNotFoundError, NewBadRequestError, NewForbiddenError, NewConflictError, etc.). Never set Status/Type by hand on a struct literal. (`return apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{{Field: "name", Rule: "required", Reason: "is required", Source: apierrors.InvalidParamSourceBody}})`)
**GenericErrorEncoder first for all v3 handlers** — Register apierrors.GenericErrorEncoder() as the ErrorEncoder. It checks *BaseAPIError first via lo.ErrorsAs, then falls through to commonhttp handlers for feature/meter not-found and ValidationIssues with HTTP status attributes. (`httptransport.NewHandler(op, dec, enc, httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()))`)
**NewV3ErrorHandlerFunc for ChiServerOptions** — Pass NewV3ErrorHandlerFunc(logger) as ChiServerOptions.ErrorHandlerFunc so generated router binding errors (InvalidParamFormatError, RequiredParamError, etc.) become InvalidParameters rendered as 400. (`api.HandlerWithOptions(srv, api.ChiServerOptions{ErrorHandlerFunc: apierrors.NewV3ErrorHandlerFunc(logger)})`)
**Always pass request context to constructors** — instance() extracts chi middleware.GetReqID(ctx) into the Instance field as 'urn:request:<id>'. Passing context.Background() silently drops the correlation ID. (`return apierrors.NewInternalError(r.Context(), err)`)
**InvalidParameters for field-level validation** — Populate InvalidParameters with Field, Rule, Reason and a Source from the four InvalidParameterSource constants before NewBadRequestError. (`apierrors.InvalidParameters{{Field: "page.size", Rule: "max_value", Reason: "must be <= 1000", Source: apierrors.InvalidParamSourceQuery}}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `errors.go` | Defines BaseAPIError, InvalidParameter/InvalidParameters, InvalidParameterSource enum, and HandleAPIError (renders via render.RenderJSON as application/problem+json). | UnderlyingError is json:"-" — never copy it into Detail or any caller-visible field. |
| `errors_ctors.go` | All public constructors plus Type URL constants (InternalType, NotFoundType, BadRequestType...) and the instance() helper. | NewEmptySetResponse uses Status 200 with a synthetic Type — not an error, but modeled as BaseAPIError for uniform rendering. |
| `encoder.go` | GenericErrorEncoder: the httptransport ErrorEncoder bridging domain errors to HTTP — *BaseAPIError first, then feature/meter not-found, then ValidationIssue HTTP-status mapping. | Check order matters: *BaseAPIError must be first so custom v3 errors short-circuit. |
| `handler.go` | NewV3ErrorHandlerFunc translates oapi-codegen router binding errors to InvalidParameters with correct Source; enrichFieldFromBindError extracts deepObject sub-field paths. | Preserve enrichFieldFromBindError behavior (e.g. 'page.sizee' from 'page') when updating param parsing. |

## Anti-Patterns

- Writing http.Error or w.WriteHeader directly in a v3 handler instead of BaseAPIError.HandleAPIError
- Constructing BaseAPIError struct literals with Status set by hand instead of a named constructor
- Passing context.Background() to constructors — the instance correlation ID will be missing
- Adding new error types outside this package and bypassing GenericErrorEncoder
- Returning UnderlyingError content in the Detail field — it is for logs only

## Decisions

- **Single BaseAPIError struct for all v3 errors rendered as application/problem+json** — Uniform wire type gives consistent shape across endpoints and lets the encoder short-circuit on the first type assertion.
- **Named constructors with pre-wired Type URL constants (kongapi.info/konnect/*)** — Prevents ad-hoc status/type drift across handlers; reviewers can grep NewBadRequestError to audit all 400 paths.

## Example: Returning a 400 with InvalidParameters from a v3 handler

```
import "github.com/openmeterio/openmeter/api/v3/apierrors"

func (h *Handler) CreateFoo(ctx context.Context, req FooRequest) error {
    if req.Name == "" {
        return apierrors.NewBadRequestError(ctx, nil, apierrors.InvalidParameters{
            {Field: "name", Rule: "required", Reason: "is required", Source: apierrors.InvalidParamSourceBody},
        })
    }
    // ...
}
```

<!-- archie:ai-end -->
