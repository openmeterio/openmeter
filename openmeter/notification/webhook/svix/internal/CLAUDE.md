# internal

<!-- archie:ai-start -->

> Internal Svix error translation layer: converts raw *svix.Error HTTP responses into typed webhook error categories (retryable, unrecoverable, not-found, validation) consumed by the svix webhook handler. Single-responsibility package — error mapping only.

## Patterns

**SvixError wraps to domain error via Wrap()** — SvixError.Wrap() maps HTTP status codes to webhook.NewRetryableError, webhook.NewUnrecoverableError, webhook.NewNotFoundError, or returns raw SvixError for unrecognized statuses. Always call .Wrap() after constructing SvixError. (`return SvixError{HTTPStatus: svixErr.Status(), Code: &body.Code, Details: []string{body.Detail}}.Wrap()`)
**WrapSvixError is the entry-point for callers** — All Svix API call sites must pass errors through WrapSvixError(err) which type-asserts to *svix.Error via lo.ErrorsAs and dispatches to status-specific JSON unmarshal + SvixError construction. Never construct SvixError directly from outside this package. (`if err := WrapSvixError(svixAPIErr); err != nil { return err }`)
**422 UnprocessableEntity uses SvixValidationErrorBody** — HTTP 422 responses from Svix carry a detail array of SvixValidationError structs; all other statuses carry a flat SvixErrorBody. Unmarshal branch must be kept in sync with Svix's actual response shapes. (`case http.StatusUnprocessableEntity: var body SvixValidationErrorBody; json.Unmarshal(svixErr.Body(), &body)`)
**Rate-limit carries RetryAfter duration** — 429 Too Many Requests and 5xx server errors are wrapped as webhook.NewRetryableError(e, e.RetryAfter). The RetryAfter field must be populated by callers parsing the Retry-After response header before constructing SvixError. (`case http.StatusTooManyRequests: return webhook.NewRetryableError(e, e.RetryAfter)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `error.go` | Defines SvixError value type, its Wrap() HTTP-status-to-domain-error mapping, JSON body structs for Svix error responses, and the WrapSvixError public entry-point. | Adding a new HTTP status mapping requires a new case in both SvixError.Wrap() and WrapSvixError's switch. Missing a status silently returns the raw SvixError without domain wrapping, bypassing retry/unrecoverable classification. |

## Anti-Patterns

- Returning *svix.Error directly to callers outside this package — always go through WrapSvixError
- Adding business logic or Svix API calls to this package — it is error-mapping only
- Unmarshaling Svix response body outside this package and duplicating the JSON struct definitions
- Treating all Svix errors as retryable — 4xx validation errors must be wrapped as unrecoverable to avoid infinite retry loops

## Decisions

- **Separate internal package for Svix error translation** — Keeps *svix.Error (an SDK type) from leaking into the domain webhook package. The internal boundary enforces that callers only see webhook.RetryableError / webhook.UnrecoverableError / webhook.NotFoundError — domain-portable error types.
- **HTTP status drives retry vs unrecoverable classification** — Svix 4xx (400, 409, 422) indicate caller-side data problems that will not resolve on retry; 429 and 5xx are transient. Encoding this in Wrap() keeps retry logic out of the notification consumer.

## Example: Wrapping a Svix API error at a call site in the svix webhook handler

```
import (
    "github.com/openmeterio/openmeter/openmeter/notification/webhook/svix/internal"
)

_, err := svixClient.Message.Create(ctx, appID, &svix.MessageIn{...})
if err != nil {
    return internal.WrapSvixError(err)
}
```

<!-- archie:ai-end -->
