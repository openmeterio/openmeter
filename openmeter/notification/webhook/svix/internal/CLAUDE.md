# internal

<!-- archie:ai-start -->

> Private helper package for the Svix webhook integration that translates raw `*svix.Error` HTTP failures into the notification domain's `webhook` error taxonomy (recoverable/retryable/validation/not-found). It is the single boundary where Svix SDK error shapes are decoded and reclassified for the delivery layer.

## Patterns

**Single public entrypoint WrapSvixError** — External callers go through `WrapSvixError(err)`; it returns `nil` for nil input, passes through errors that are not `*svix.Error`, and otherwise decodes and re-wraps. Do not export new ad-hoc wrappers — extend this function. (`svixErr, ok := lo.ErrorsAs[*svix.Error](err); if !ok { return err }`)
**HTTP status -> webhook error class mapping in SvixError.Wrap** — `SvixError.Wrap()` is the authoritative status->class switch: 400/409/422 -> `webhook.NewUnrecoverableError(webhook.NewValidationError(...))`, 404 -> `webhook.NewNotFoundError`, 429/5xx -> `webhook.NewRetryableError(..., RetryAfter)`. New status handling belongs in this switch, not at call sites. (`case http.StatusTooManyRequests: return webhook.NewRetryableError(e, e.RetryAfter)`)
**Two-shape body decoding by status** — 422 responses are unmarshalled into `SvixValidationErrorBody` (list of `SvixValidationError{Loc,Message,Type}`); all other statuses into the flat `SvixErrorBody{Code,Detail}`. Match the JSON tag shape Svix actually returns when adding a status branch. (`var body SvixValidationErrorBody; json.Unmarshal(svixErr.Body(), &body)`)
**Compile-time error interface assertion** — `var _ error = (*SvixError)(nil)` enforces that `SvixError` implements `error`; keep `Error()` defined so this assertion holds. (`var _ error = (*SvixError)(nil)`)
**samber/lo for nil-safe pointer and slice ops** — Use `lo.FromPtrOr`, `lo.ToPtr`, `lo.Map`, `lo.ErrorsAs` rather than manual nil checks or local helpers, consistent with repo conventions. (`buf.WriteString(lo.FromPtrOr(e.Code, "unknown svix error"))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `error.go` | The entire package: defines `SvixError` value type, its `Error()`/`Wrap()` methods, the `SvixErrorBody`/`SvixValidationErrorBody`/`SvixValidationError` decode structs, and `WrapSvixError` translator. | JSON-decode failures fall back to wrapping the ORIGINAL `err` (not the parse error) via `fmt.Errorf("failed to parse Svix error response: %w", err)` — keep this so the underlying Svix error is preserved. `Wrap()`'s `default` branch returns the raw `SvixError` unwrapped, so unmapped statuses are neither retryable nor validation errors. |

## Anti-Patterns

- Classifying Svix HTTP statuses (retryable vs unrecoverable) at delivery call sites instead of inside `SvixError.Wrap()`.
- Returning a raw `*svix.Error` from this package — always funnel through `WrapSvixError` so it becomes a `webhook` domain error.
- Importing this `internal` package from outside `openmeter/notification/webhook/svix` (Go internal visibility rules forbid it).
- On JSON unmarshal failure, wrapping the parse error instead of the original `err`, which would hide the real Svix failure.

## Decisions

- **Reclassify Svix errors into the `webhook` package's recoverable/retryable/validation/not-found taxonomy here rather than in the consumer.** — The Kafka/delivery layer needs a stable retry signal independent of Svix's HTTP encoding; centralizing the mapping keeps retry policy in one place.
- **Separate 422 validation bodies from generic error bodies.** — Svix returns a structured per-field `loc/msg/type` list only for validation (422), so a distinct struct and human-readable `[location=... type=...]` formatting is required.

## Example: Translate a Svix SDK error into a domain webhook error

```
import (
	"net/http"
	svix "github.com/svix/svix-webhooks/go"
	"github.com/openmeterio/openmeter/openmeter/notification/webhook"
)

// WrapSvixError turns *svix.Error into a webhook.* error; non-svix errors pass through.
func WrapSvixError(err error) error {
	svixErr, ok := lo.ErrorsAs[*svix.Error](err)
	if !ok {
		return err
	}
	return SvixError{HTTPStatus: svixErr.Status(), Code: &body.Code, Details: []string{body.Detail}}.Wrap()
}

// ...
```

<!-- archie:ai-end -->
