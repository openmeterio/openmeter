# commonhttp

<!-- archie:ai-start -->

> Shared HTTP encoding/decoding primitives for all domain httpdriver packages: JSON/CSV/plain-text response encoders, RFC 7807 problem-detail error encoders, ValidationIssue-to-HTTP-status mapping, pagination constants, sort order helpers, and a Union[Primary,Secondary] JSON type.

## Patterns

**GenericErrorEncoder chain** — GenericErrorEncoder() returns an encoder.ErrorEncoder that calls HandleIssueIfHTTPStatusKnown first, then HandleErrorIfTypeMatches for each models.Generic* error type. First match wins and returns true. (`return HandleIssueIfHTTPStatusKnown(ctx, err, w) || HandleErrorIfTypeMatches[*models.GenericNotFoundError](ctx, http.StatusNotFound, err, w)`)
**WithHTTPStatusCodeAttribute on ValidationIssue** — Domain code attaches an HTTP status to a ValidationIssue via commonhttp.WithHTTPStatusCodeAttribute(code); HandleIssueIfHTTPStatusKnown reads the attribute and renders the correct status. The attribute is stripped from the JSON response. (`models.NewValidationIssue(code, msg, commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest))`)
**ResponseEncoder generics** — All encoder functions are generic over the response type: JSONResponseEncoder[Response], CSVResponseEncoder[Response CSVResponse], EmptyResponseEncoder[Response], etc. (`func JSONResponseEncoderWithStatus[Response any](statusCode int) encoder.ResponseEncoder[Response]`)
**CSVResponse interface** — To use CSVResponseEncoder a type must implement FileName() string and Records() [][]string. (`type CSVResponse interface { FileName() string; Records() [][]string }`)
**ExtendProblemFunc for RFC 7807 extensions** — NewHTTPError accepts variadic ExtendProblemFunc values; each is applied to problem.Extensions map before Respond(w) is called. (`NewHTTPError(code, err, ExtendProblem("field", "reason")).EncodeError(ctx, w)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `errors.go` | Core error encoding: ErrorWithHTTPStatusCode, HandleErrorIfTypeMatches[T error], HandleIssueIfHTTPStatusKnown. | HTTPStatusAttributePriorizationBehaviorSingular is the only supported mode; multiple ValidationIssues with different status codes cause HandleIssueIfHTTPStatusKnown to return false (falls through to next encoder). |
| `encoder.go` | Response encoders (JSON, plain-text, CSV, empty, redirect) and GenericErrorEncoder(). | JSONResponseEncoder always writes 200; use JSONResponseEncoderWithStatus for non-200 success codes. |
| `decoder.go` | JSONRequestBodyDecoder wraps render.DecodeJSON and wraps parse errors in NewHTTPError(400,...). | Returns commonhttp.ErrorWithHTTPStatusCode, not a plain error. |
| `union.go` | Union[Primary,Secondary] JSON union type; Option1 takes precedence over Option2 during marshaling. | Returns empty []byte{} (not 'null') when both options are nil — callers must handle this. |
| `sort.go` | GetSortOrder helper maps an API sort-direction enum to sortx.Order. | Returns sortx.OrderNone when input pointer is nil; never returns an error. |

## Anti-Patterns

- Returning a plain error instead of NewHTTPError from request decoders — the error encoder chain won't know the status code
- Adding new domain error types to GenericErrorEncoder without a corresponding models.Generic* sentinel type
- Calling HandleIssueIfHTTPStatusKnown without WithHTTPStatusCodeAttribute on the ValidationIssue — it will return false and fall through
- Writing status code before headers in custom encoders — breaks header-only responses

## Decisions

- **Error encoder returns bool (chain pattern)** — Multiple encoders can be composed in a chain; each returns true to short-circuit or false to pass to the next, matching the httptransport ErrorEncoder interface.
- **ValidationIssue carries HTTP status as an attribute rather than a field** — ValidationIssue is a domain type that must not import net/http; the attribute key is defined in commonhttp and read only at the HTTP boundary.

## Example: Attach HTTP status to a domain ValidationIssue and encode it

```
import (
    "github.com/openmeterio/openmeter/pkg/framework/commonhttp"
    "github.com/openmeterio/openmeter/pkg/models"
)

err := models.NewValidationIssue(
    models.ErrorCode("invalid_field"),
    "field x is required",
    commonhttp.WithHTTPStatusCodeAttribute(http.StatusUnprocessableEntity),
)
// In the handler's error encoder chain:
commonhttp.HandleIssueIfHTTPStatusKnown(ctx, err, w) // returns true, writes 422
```

<!-- archie:ai-end -->
