namespace Shared;

/**
 * Shared properties for inline errors returned inside a 2xx response.
 *
 * Use for partially successful responses (e.g. per-item failures alongside
 * a `data` array) or for pre-flight findings on resources that are not yet
 * finalized (e.g. validation errors on a draft resource).
 *
 * This is **not** an RFC-7807 problem detail. RFC-7807 covers errors that
 * fail the whole request; inline errors live as fields on the response or
 * resource model.
 *
 * Compose with `is` and add domain-specific identifying fields as needed:
 *
 * ```tsp
 * @friendlyName("MyOperationError")
 * model MyOperationError is Shared.BaseError<MyOperationErrorCode> {
 *   // domain-specific identifiers, e.g. the input that produced the error
 *   subject?: string;
 * }
 * ```
 *
 * `T` is the type of the `code` field. Prefer a domain-scoped enum with an
 * `Unknown: "unknown"` zero member so codes stay discoverable and stable.
 * Use `string` only when the set of codes is open-ended.
 */
@friendlyName("BaseError")
model BaseError<T> {
  /**
   * Machine-readable error code.
   */
  @visibility(Lifecycle.Read)
  @summary("Code")
  code: T;

  /**
   * Human-readable description of the error.
   */
  @visibility(Lifecycle.Read)
  @summary("Message")
  message: string;

  /**
   * Additional structured context.
   */
  @visibility(Lifecycle.Read)
  @summary("Attributes")
  attributes?: Record<unknown>;
}
