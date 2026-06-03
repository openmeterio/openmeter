# internal

<!-- archie:ai-start -->

> Internal Svix error translation layer that converts raw *svix.Error HTTP responses into typed domain webhook error categories (retryable, unrecoverable, not-found, validation). Single-responsibility: error mapping only, no Svix API calls.

## Patterns

**WrapSvixError is the sole entry-point** — All Svix API call sites pass errors through WrapSvixError(err), which type-asserts to *svix.Error via lo.ErrorsAs and dispatches to status-specific JSON unmarshal + SvixError construction. Never construct SvixError directly from outside this package. (`if err := internal.WrapSvixError(svixAPIErr); err != nil { return err }`)
**SvixError.Wrap() maps HTTP status to domain error** — 4xx (400, 409, 422) -> webhook.NewUnrecoverableError; 404 -> webhook.NewNotFoundError; 429 and 5xx -> webhook.NewRetryableError with RetryAfter. Unrecognized statuses return the raw SvixError. Always call .Wrap() after constructing SvixError. (`return SvixError{HTTPStatus: svixErr.Status(), Code: &body.Code, Details: []string{body.Detail}}.Wrap()`)
**422 uses SvixValidationErrorBody, others use SvixErrorBody** — HTTP 422 responses carry a detail array of SvixValidationError structs; all other statuses carry a flat SvixErrorBody with code + detail string. The unmarshal branch in WrapSvixError must stay in sync with Svix response shapes. (`case http.StatusUnprocessableEntity: var body SvixValidationErrorBody; json.Unmarshal(svixErr.Body(), &body)`)
**Rate-limit carries RetryAfter duration** — 429 and 5xx are wrapped as webhook.NewRetryableError(e, e.RetryAfter). RetryAfter must be populated by callers parsing the Retry-After header before constructing SvixError. (`case http.StatusTooManyRequests: return webhook.NewRetryableError(e, e.RetryAfter)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `error.go` | Defines SvixError value type, its Wrap() HTTP-status-to-domain mapping, JSON body structs (SvixErrorBody, SvixValidationErrorBody, SvixValidationError), and the WrapSvixError entry-point. | A new HTTP status mapping needs a new case in both SvixError.Wrap() and WrapSvixError's switch. Missing a status silently returns the raw SvixError without domain wrapping, bypassing retry/unrecoverable classification. |

## Anti-Patterns

- Returning *svix.Error directly to callers outside this package — always go through WrapSvixError.
- Adding business logic or Svix API calls here — it is error-mapping only.
- Unmarshaling Svix response bodies outside this package and duplicating the JSON struct definitions.
- Treating all Svix errors as retryable — 4xx validation errors must be unrecoverable to avoid infinite retry loops.
- Constructing SvixError{} struct literals from outside this package — use WrapSvixError exclusively.

## Decisions

- **Separate internal package for Svix error translation.** — Keeps *svix.Error (an SDK type) from leaking into the domain webhook package; callers only see webhook.RetryableError/UnrecoverableError/NotFoundError — domain-portable types independent of the Svix SDK.
- **HTTP status drives retry vs unrecoverable classification.** — Svix 4xx (400, 409, 422) indicate caller-side data problems that will not resolve on retry; 429 and 5xx are transient. Encoding this in Wrap() centralizes classification and keeps retry logic out of the notification consumer.

## Example: Wrap a Svix API error at a call site in the svix webhook handler

```
import "github.com/openmeterio/openmeter/openmeter/notification/webhook/svix/internal"

_, err := svixClient.Message.Create(ctx, appID, &svix.MessageIn{...})
if err != nil {
    return internal.WrapSvixError(err)
}
```

<!-- archie:ai-end -->
