# commonhttp

<!-- archie:ai-start -->

> Shared HTTP request/response plumbing for all v1/v3 handlers: typed JSON/CSV/PlainText/Redirect response encoders, a request body decoder, and the error-encoder chain that maps domain errors and models.ValidationIssue HTTP-status attributes to RFC-7807 status problems.

## Patterns

**Generic typed encoders returning encoder.ResponseEncoder[T]** — Encoders are generic over the response type and either are direct encoder functions (JSONResponseEncoder) or factories returning encoder.ResponseEncoder[Response] (JSONResponseEncoderWithStatus, EmptyResponseEncoder, RedirectResponseEncoder). Use these instead of hand-writing w.Write. (`func JSONResponseEncoderWithStatus[Response any](statusCode int) encoder.ResponseEncoder[Response]`)
**Composed error encoder via short-circuit OR** — GenericErrorEncoder chains HandleIssueIfHTTPStatusKnown and HandleErrorIfTypeMatches[*models.GenericXxxError] calls with ||; the first matching mapper writes the response and returns true. Add new mappings by inserting another HandleErrorIfTypeMatches term. (`return HandleIssueIfHTTPStatusKnown(ctx, err, w) || HandleErrorIfTypeMatches[*models.GenericConflictError](ctx, http.StatusConflict, err, w) || ...`)
**ValidationIssue HTTP-status attribute mapping** — Handlers attach commonhttp.WithHTTPStatusCodeAttribute(code) to a models.ValidationIssue; HandleIssueIfHTTPStatusKnown reads that private attribute, groups issues by code, and emits them under the legacy 'validationErrors' extension key while stripping the private status attribute from the response. (`models.NewValidationIssue(code, msg, commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest))`)
**Singular-status prioritization default** — When validation issues carry multiple distinct HTTP status codes, the default HTTPStatusAttributePriorizationBehaviorSingular returns false (declines to map) rather than guessing a status; only a single distinct code is mapped. (`if len(issuesByCodeMap) > 1 { return false }`)
**Errors wrapped as ErrorWithHTTPStatusCode** — NewHTTPError(statusCode, err, extensions...) embeds the error and renders via models.NewStatusProblem(ctx, err, code).Respond(w); ExtendProblem(name, details) adds extension fields. (`NewHTTPError(http.StatusBadRequest, fmt.Errorf("decode json: %w", err))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `errors.go` | Error-to-HTTP mapping: NewHTTPError/ErrorWithHTTPStatusCode, HandleErrorIfTypeMatches[T], HandleIssueIfHTTPStatusKnown, WithHTTPStatusCodeAttribute. | The 'validationErrors' extension key is kept only for backwards compatibility (see FIXME). HandleErrorIfTypeMatches uses lo.ErrorsAs so T must implement error or errors.As would panic. |
| `encoder.go` | All response encoders + GetMediaType (Accept header parse, defaults to application/json) + the two error-encoder helpers DummyErrorEncoder/GenericErrorEncoder. | GetMediaType returns a non-nil error AND a defaulted mediatype on invalid Accept; callers usually ignore the error. RedirectResponseEncoder enforces a 3xx status range. |
| `decoder.go` | JSONRequestBodyDecoder wrapping go-chi/render.DecodeJSON, returning a 400 HTTPError on failure. | Always wrap decode failures as a 400 — do not return the raw render error. |
| `union.go` | Union[Primary, Secondary] JSON marshaller where Option1 takes precedence and an empty union marshals to empty bytes. | MarshalJSON returns []byte{} (not 'null') when both options are nil, which is invalid JSON if embedded directly. |
| `sort.go` | GetSortOrder mapping an optional input value to sortx.Order (asc/desc/none). | Nil input maps to sortx.OrderNone via defaultx.WithDefault. |
| `pagination.go` | Pagination constants: DefaultPageSize=100, MaxPageSize=1000, DefaultPage=1. | Constants only — no validation logic lives here. |

## Anti-Patterns

- Hand-writing w.WriteHeader + w.Write in handlers instead of using the typed encoders here.
- Returning raw domain errors from handlers without routing them through GenericErrorEncoder / NewHTTPError so they become RFC-7807 problems.
- Mapping multiple distinct HTTP status codes from one validation-issue set — the singular behavior intentionally declines that.
- Exposing the private httpStatusCodeErrorAttribute to clients (it is stripped before responding; don't re-add it).

## Decisions

- **Validation issues carry HTTP status as a private attribute rather than a typed error** — Lets domain layers stay HTTP-agnostic while the encoder layer derives status codes from accumulated issues.
- **Multi-status issue sets are not mapped (return false)** — Avoids arbitrarily picking a status when issues disagree; an outer/default handler decides instead.

## Example: Wire encoders into a v3 handler

```
import "github.com/openmeterio/openmeter/pkg/framework/commonhttp"

httptransport.NewHandlerWithArgs(resolve, exec,
  commonhttp.JSONResponseEncoderWithStatus[CreateResponse](http.StatusCreated),
  httptransport.WithErrorEncoder(commonhttp.GenericErrorEncoder()),
)
```

<!-- archie:ai-end -->
