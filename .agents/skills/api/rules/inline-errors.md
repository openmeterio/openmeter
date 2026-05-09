# Inline (partial / non-fatal) errors

Some endpoints return errors **inside** a 2xx response body. These are **not** RFC-7807 problem details — RFC-7807 (see `aip-193-errors.md`) covers errors that fail the whole request. Inline errors live as a field on the response or resource model and describe per-item or pre-flight failures that did not stop the response from being produced.

## When to use

- **Partially successful responses** — the operation processed some items and reports per-item failures alongside the successful results (typically a `data[]` plus `errors[]` pair on the response model).
- **Not-yet-finalized resources** — the resource itself is in a non-final state (e.g. a draft) and exposes pre-flight findings that block promotion to the final state. The GET itself succeeds; the findings are part of the resource shape.

If the whole request fails, return an RFC-7807 response instead — see `aip-193-errors.md`.

## Required shape: `Shared.BaseError<T>`

All inline errors compose `Shared.BaseError<T>` from `api/spec/packages/aip/src/shared/errors.tsp`. It defines the three uniform fields every inline error must carry, so SDKs and UIs can render them generically:

| Field        | Type              | Required | Meaning                                                            |
| ------------ | ----------------- | -------- | ------------------------------------------------------------------ |
| `code`       | `T`               | yes      | Machine-readable error code                                        |
| `message`    | `string`          | yes      | Human-readable description                                         |
| `attributes` | `Record<unknown>` | optional | Additional structured context (field path, offending value, IDs)   |

`T` is the type of `code`. Pick one:

- **Domain-scoped enum (preferred)** — codes are discoverable and stable, and clients can switch on them. Include an `Unknown: "unknown"` zero member for forward compatibility.
- **`string`** — only when the set of codes is genuinely open-ended (e.g. validation rules that grow over time without API churn).

## Composition pattern

Use `model … is Shared.BaseError<T>` and add domain-specific identifying fields as needed (e.g. the input that produced the error, a JSON path, a resource id). Do **not** redeclare `code`, `message`, or `attributes`.

```tsp
import "../shared/index.tsp";

namespace MyDomain;

/** Machine-readable code for a {operation} error. */
@friendlyName("MyOperationErrorCode")
enum MyOperationErrorCode {
  Unknown: "unknown",
  // domain-specific codes...
}

/** Inline error returned by {operation}. */
@friendlyName("MyOperationError")
model MyOperationError is Shared.BaseError<MyOperationErrorCode> {
  /** Identifier from the request that produced this error. */
  @visibility(Lifecycle.Read)
  @summary("Subject")
  subject?: string;
}
```

For an open-ended code set, pass `string`:

```tsp
@friendlyName("MyValidationError")
model MyValidationError is Shared.BaseError<string> {
  /** JSON path to the offending field. */
  @visibility(Lifecycle.Read)
  @summary("Field")
  field: string;
}
```

## Wiring into the response

Add the inline errors as a field on the response or resource model — typically named `errors` (partial responses) or a domain-specific name like `validation_errors` (pre-flight findings on a draft resource).

```tsp
@friendlyName("MyOperationResponse")
model MyOperationResponse {
  @visibility(Lifecycle.Read)
  data: MyOperationResult[];

  /** Per-item failures encountered while processing the request. */
  @visibility(Lifecycle.Read)
  errors: MyOperationError[];
}
```

## Naming

- Model: `<Domain><Operation>Error` or `<Domain>ValidationError`.
- Code enum (when used): `<ModelName>Code` (drop the trailing `Error`, append `Code`).
- All fields use `@visibility(Lifecycle.Read)` — inline errors are always server-produced.
