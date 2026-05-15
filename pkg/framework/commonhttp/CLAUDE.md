# commonhttp

<!-- archie:ai-start -->

> Shared HTTP encoding/decoding primitives for all domain httpdriver packages: JSON/CSV/plain-text response encoders, RFC 7807 problem-detail error encoders, ValidationIssue-to-HTTP-status mapping, cursor-pagination constants, sort-order helpers, and a Union[Primary,Secondary] JSON type.

## Patterns

**GenericErrorEncoder chain — first match wins** — GenericErrorEncoder() returns an encoder.ErrorEncoder that calls HandleIssueIfHTTPStatusKnown first, then HandleErrorIfTypeMatches for each models.Generic* error type in priority order. Return value bool short-circuits the chain. (`return HandleIssueIfHTTPStatusKnown(ctx, err, w) || HandleErrorIfTypeMatches[*models.GenericNotFoundError](ctx, http.StatusNotFound, err, w)`)
**WithHTTPStatusCodeAttribute on ValidationIssue — domain type stays clean** — Domain code attaches an HTTP status to a ValidationIssue via commonhttp.WithHTTPStatusCodeAttribute(code); HandleIssueIfHTTPStatusKnown reads the openmeter.http.status_code attribute and strips it before serialization. ValidationIssue must not import net/http. (`models.NewValidationIssue(code, msg, commonhttp.WithHTTPStatusCodeAttribute(http.StatusUnprocessableEntity))`)
**Generic response encoders — use JSONResponseEncoderWithStatus for non-200** — JSONResponseEncoder always writes 200. Use JSONResponseEncoderWithStatus[Response](statusCode) when the success code differs (e.g., 201 Created). (`httptransport.NewHandler(decode, op, commonhttp.JSONResponseEncoderWithStatus[MyResp](http.StatusCreated))`)
**CSVResponse interface for CSV encoding** — To use CSVResponseEncoder a type must implement FileName() string and Records() [][]string. Content-Disposition is set automatically with the filename. (`type MyCSV struct{}; func (c MyCSV) FileName() string { return "report" }; func (c MyCSV) Records() [][]string { return rows }`)
**NewHTTPError for decoder errors — not plain errors** — JSONRequestBodyDecoder wraps render.DecodeJSON errors in NewHTTPError(http.StatusBadRequest, ...). Always return NewHTTPError from custom decoders so the error encoder chain knows the status. (`return NewHTTPError(http.StatusBadRequest, fmt.Errorf("decode json: %w", err))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `errors.go` | Core error encoding: ErrorWithHTTPStatusCode, HandleErrorIfTypeMatches[T], HandleIssueIfHTTPStatusKnown. | HTTPStatusAttributePriorizationBehaviorSingular is the only supported mode — multiple ValidationIssues with different status codes cause HandleIssueIfHTTPStatusKnown to return false, falling through to the next encoder. |
| `encoder.go` | Response encoders (JSON, plain-text, CSV, empty, redirect) and GenericErrorEncoder(). | JSONResponseEncoder always writes HTTP 200; use JSONResponseEncoderWithStatus for 201/204 etc. Union.MarshalJSON returns []byte{} (not 'null') when both options are nil. |
| `decoder.go` | JSONRequestBodyDecoder wraps render.DecodeJSON and wraps parse errors in NewHTTPError(400). | Returns ErrorWithHTTPStatusCode, not a plain error — callers must not compare with errors.Is(err, someErr) without unwrapping. |
| `union.go` | Union[Primary,Secondary] JSON union type; Option1 takes precedence over Option2 during marshaling. | Returns empty []byte{} (not 'null') when both options are nil — downstream decoders may choke on empty bytes. |
| `sort.go` | GetSortOrder helper maps an API sort-direction enum pointer to sortx.Order. | Returns sortx.OrderNone when the input pointer is nil; never returns an error — check the zero value. |

## Anti-Patterns

- Returning a plain error from a request decoder instead of NewHTTPError — GenericErrorEncoder won't know the status code and will fall through to 500
- Adding a new domain error type to GenericErrorEncoder without a corresponding models.Generic* sentinel type in pkg/models
- Calling HandleIssueIfHTTPStatusKnown without WithHTTPStatusCodeAttribute on the ValidationIssue — it returns false and falls through
- Writing status code before headers in a custom encoder — must set headers first, then WriteHeader

## Decisions

- **Error encoder returns bool (chain pattern)** — Multiple encoders compose in a chain; returning true short-circuits to prevent double-write of the response body, matching the httptransport ErrorEncoder interface contract.
- **ValidationIssue carries HTTP status as an attribute rather than a field** — ValidationIssue is a domain type that must not import net/http; the attribute key lives in commonhttp and is read only at the HTTP boundary, keeping the domain layer clean.

## Example: Attach HTTP status to a domain ValidationIssue and encode it through the error chain

```
import (
    "github.com/openmeterio/openmeter/pkg/framework/commonhttp"
    "github.com/openmeterio/openmeter/pkg/models"
)

// In service/adapter layer:
err := models.NewValidationIssue(
    models.ErrorCode("invalid_field"),
    "field x is required",
    commonhttp.WithHTTPStatusCodeAttribute(http.StatusUnprocessableEntity),
)

// In HTTP handler error encoder chain (added automatically by httptransport defaults):
// commonhttp.HandleIssueIfHTTPStatusKnown(ctx, err, w) -> returns true, writes 422
```

<!-- archie:ai-end -->
