# apierrors

<!-- archie:ai-start -->

> Provides the RFC 7807-style error type system for the v3 API: a single BaseAPIError struct, typed constructors for every HTTP status, and a GenericErrorEncoder that maps domain errors to HTTP responses. All v3 error responses must flow through this package.

## Patterns

**BaseAPIError as the single error wire type** — All v3 HTTP error responses are produced by constructing a *BaseAPIError and calling HandleAPIError(w, r). Never write raw http.Error or json.Marshal in a handler — always go through this type. (`apierrors.NewNotFoundError(ctx, err, "customer").HandleAPIError(w, r)`)
**Named constructors for every status** — Each HTTP error status has a dedicated constructor in errors_ctors.go (NewInternalError, NewNotFoundError, NewBadRequestError, NewForbiddenError, etc.). Use the correct constructor — never set Status manually on a BaseAPIError you build by hand. (`return apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{{Field: "name", Rule: "required", Reason: "is required", Source: apierrors.InvalidParamSourceBody}})`)
**GenericErrorEncoder chain integration** — GenericErrorEncoder() checks for *BaseAPIError first (via lo.ErrorsAs), then falls through to commonhttp.HandleErrorIfTypeMatches for domain-specific errors, then commonhttp.HandleIssueIfHTTPStatusKnown. Register it as the first ErrorEncoder when building httptransport.Handler instances for v3. (`httptransport.NewHandler(op, dec, enc, httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()))`)
**NewV3ErrorHandlerFunc for oapi-codegen ChiServerOptions** — Pass NewV3ErrorHandlerFunc(logger) as ChiServerOptions.ErrorHandlerFunc so generated router binding errors (InvalidParamFormatError, RequiredParamError, etc.) are translated to InvalidParameters and rendered as 400 Bad Request in v3 format. (`api.HandlerWithOptions(srv, api.ChiServerOptions{ErrorHandlerFunc: apierrors.NewV3ErrorHandlerFunc(logger)})`)
**InvalidParameters for field-level validation errors** — When input fields fail validation, populate InvalidParameters ([]InvalidParameter) with Field, Rule, Reason, and Source fields before passing to NewBadRequestError. The Source must be one of the four InvalidParameterSource constants. (`apierrors.InvalidParameters{{Field: "page.size", Rule: "max_value", Reason: "must be <= 1000", Source: apierrors.InvalidParamSourceQuery}}`)
**instance() sets the urn:request: correlation field** — The Instance field is populated from chi middleware.GetReqID(ctx) via the package-private instance() helper. Always pass the request context (not context.Background()) to constructors so the correlation ID is present. (`apierrors.NewInternalError(r.Context(), err)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `errors.go` | Defines BaseAPIError struct, InvalidParameter/InvalidParameters types, and the HandleAPIError write method. The source of the wire type. | UnderlyingError is excluded from JSON (json:"-") — it is for logging only, never returned to clients. |
| `errors_ctors.go` | All public constructors plus URL constants for error types. Add new constructors here when new HTTP status codes are needed. | NewEmptySetResponse uses Status 200 with a synthetic Type — it is not an error but is modeled as BaseAPIError for uniform rendering. |
| `encoder.go` | GenericErrorEncoder: the httptransport ErrorEncoder that bridges domain errors to HTTP responses. Must remain the chain entry point for all v3 handlers. | Order matters: *BaseAPIError checked first; then feature/meter not-found; then ValidationIssue with HTTP status attribute. |
| `handler.go` | NewV3ErrorHandlerFunc: the oapi-codegen ChiServerOptions.ErrorHandlerFunc implementation. Translates generated router binding errors to InvalidParameters. | enrichFieldFromBindError extracts sub-field paths from deepObject bind errors — preserve this behavior when updating parameter parsing. |

## Anti-Patterns

- Writing http.Error or calling w.WriteHeader directly in a v3 handler instead of BaseAPIError.HandleAPIError
- Constructing BaseAPIError literals with Status set by hand instead of using a named constructor
- Passing context.Background() to constructors — the instance (correlation ID) will be missing
- Adding new error types outside this package and bypassing GenericErrorEncoder
- Returning UnderlyingError content in the Detail field — it is for logs only

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
