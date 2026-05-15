# apierrors

<!-- archie:ai-start -->

> Provides the RFC 7807-style error type system for all v3 API responses: a single BaseAPIError struct, typed constructors for every HTTP status, and a GenericErrorEncoder that maps domain errors to HTTP responses. Every v3 error response must flow through this package.

## Patterns

**BaseAPIError via named constructors only** — All v3 HTTP error responses are produced by a named constructor in errors_ctors.go (NewInternalError, NewNotFoundError, NewBadRequestError, NewForbiddenError, NewConflictError, etc.). Never set Status or Type manually on a hand-constructed BaseAPIError struct. (`return apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{{Field: "name", Rule: "required", Reason: "is required", Source: apierrors.InvalidParamSourceBody}})`)
**GenericErrorEncoder as first ErrorEncoder for all v3 handlers** — Register apierrors.GenericErrorEncoder() as the ErrorEncoder when building httptransport.Handler instances. It checks *BaseAPIError first via lo.ErrorsAs, then falls through to commonhttp handlers for feature/meter not-found errors and ValidationIssues with HTTP status attributes. (`httptransport.NewHandler(op, dec, enc, httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()))`)
**NewV3ErrorHandlerFunc for oapi-codegen ChiServerOptions** — Pass NewV3ErrorHandlerFunc(logger) as ChiServerOptions.ErrorHandlerFunc so generated router binding errors (InvalidParamFormatError, RequiredParamError, etc.) are translated to InvalidParameters and rendered as 400 Bad Request in v3 format. (`api.HandlerWithOptions(srv, api.ChiServerOptions{ErrorHandlerFunc: apierrors.NewV3ErrorHandlerFunc(logger)})`)
**Always pass request context to constructors** — The instance() helper extracts chi middleware.GetReqID(ctx) to populate the Instance field as 'urn:request:<id>'. Passing context.Background() silently omits the correlation ID. (`return apierrors.NewInternalError(r.Context(), err)`)
**InvalidParameters for field-level validation errors** — When input fields fail validation, populate InvalidParameters ([]InvalidParameter) with Field, Rule, Reason, and Source before passing to NewBadRequestError. Source must be one of the four InvalidParameterSource constants (InvalidParamSourcePath, InvalidParamSourceQuery, InvalidParamSourceBody, InvalidParamSourceHeader). (`apierrors.InvalidParameters{{Field: "page.size", Rule: "max_value", Reason: "must be <= 1000", Source: apierrors.InvalidParamSourceQuery}}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `errors.go` | Defines BaseAPIError struct, InvalidParameter/InvalidParameters types, and HandleAPIError write method. UnderlyingError is the underlying error stack for logging — it is excluded from JSON (json:"-") and must never appear in Detail. | UnderlyingError is json:"-" — never copy it into Detail or any response field visible to callers. |
| `errors_ctors.go` | All public constructors plus Type URL constants (InternalType, NotFoundType, BadRequestType, etc.). Add new constructors here when new HTTP status codes are needed. | NewEmptySetResponse uses Status 200 with a synthetic Type — it is not an error but modeled as BaseAPIError for uniform rendering. |
| `encoder.go` | GenericErrorEncoder: the httptransport ErrorEncoder bridging domain errors to HTTP responses. *BaseAPIError checked first; then feature/meter not-found; then ValidationIssue with HTTP status attribute. | Order of checks in the encoder matters — *BaseAPIError must be first so custom v3 errors short-circuit before domain error matching. |
| `handler.go` | NewV3ErrorHandlerFunc: translates oapi-codegen generated router binding errors (InvalidParamFormatError, RequiredParamError, TooManyValuesForParamError, etc.) to InvalidParameters with correct Source fields. | enrichFieldFromBindError extracts sub-field paths from deepObject bind errors (e.g. 'page.sizee' from 'page') — preserve this behavior when updating parameter parsing. |

## Anti-Patterns

- Writing http.Error or calling w.WriteHeader directly in a v3 handler instead of BaseAPIError.HandleAPIError
- Constructing BaseAPIError struct literals with Status set by hand instead of using a named constructor
- Passing context.Background() to constructors — the instance correlation ID will be missing
- Adding new error types outside this package and bypassing GenericErrorEncoder
- Returning UnderlyingError content in the Detail field — it is for logs only, never clients

## Decisions

- **Single BaseAPIError struct for all v3 error responses rendered as application/problem+json** — Uniform error wire type ensures consistent shape across all v3 endpoints and lets the error encoder short-circuit on the first type assertion rather than pattern-matching many types.
- **Named constructors with pre-wired Type URL constants (kongapi.info/konnect/*)** — Prevents ad-hoc status/type combinations drifting across handlers; code reviewers can grep for NewBadRequestError to audit all 400 paths.

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
